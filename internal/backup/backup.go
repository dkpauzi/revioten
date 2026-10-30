// Package backup membuat dan memulihkan backup database terenkripsi (M1.7,
// D-119, RK-49). Data asli dipakai di localhost sejak awal (D-29, D-36), jadi
// backup dibuat sebelum register nomor Tahap 2 mulai terisi.
//
// Satu backup terdiri dari dua file di folder tujuan (sebaiknya di disk lain):
//   - revioten-<waktu>.dump.age: hasil pg_dump (format custom) yang dienkripsi
//     dengan age memakai passphrase. File ini juga bisa dibuka tanpa aplikasi
//     ini, dengan perintah `age -d`.
//   - revioten-<waktu>.manifest.json: catatan backup yang tidak rahasia: hash
//     SHA-256 file terenkripsi, versi skema, dan ujung rantai audit log. Ujung
//     rantai yang tersimpan di luar database membuat penulisan ulang seluruh
//     rantai oleh admin database tetap ketahuan (R-SEC-03, E-29).
//
// Paket ini tidak menjalankan pg_dump dan pg_restore sendiri. Pemanggil
// memberikan Dumper dan Restorer: perintah CLI memakai docker compose exec,
// dan test memakai container test.
package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"filippo.io/age"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
)

// Akhiran nama file satu backup.
const (
	dumpSuffix     = ".dump.age"
	manifestSuffix = ".manifest.json"
)

// manifestFormat adalah versi bentuk manifest. Dinaikkan bila field berubah.
const manifestFormat = 1

// minPassphraseLength sama dengan panjang minimal password (D-85).
const minPassphraseLength = 15

// RestoreFlags adalah pilihan pg_restore untuk setiap restore:
//   - --no-owner: tabel dimiliki user yang menjalankan restore, jadi backup
//     bisa dipulihkan di server dengan nama pemilik lain;
//   - --single-transaction: semua atau tidak sama sekali, jadi restore yang
//     gagal tidak meninggalkan database setengah terisi;
//   - --exit-on-error: berhenti pada error pertama.
var RestoreFlags = []string{"--no-owner", "--single-transaction", "--exit-on-error"}

// ErrDatabaseExists berarti database tujuan restore sudah ada. Restore tidak
// pernah menimpa database, supaya data yang sedang dipakai tidak hilang.
var ErrDatabaseExists = errors.New("database tujuan sudah ada")

// Info adalah keadaan database saat backup dibuat, untuk manifest.
type Info struct {
	Database      string
	SchemaVersion int64  // nomor migrasi terakhir yang sudah dijalankan
	AuditLastSeq  int64  // ujung rantai audit; 0 bila audit log masih kosong
	AuditLastHash string // hash entri AuditLastSeq
}

// Manifest adalah catatan satu backup. Isinya tidak rahasia: tidak ada isi
// database, password, atau passphrase (E-18, D-95).
type Manifest struct {
	Format        int       `json:"format"`
	CreatedAt     time.Time `json:"created_at"`
	File          string    `json:"file"`
	SHA256        string    `json:"sha256"`
	Size          int64     `json:"size"`
	Database      string    `json:"database"`
	SchemaVersion int64     `json:"schema_version"`
	AuditLastSeq  int64     `json:"audit_last_seq"`
	AuditLastHash string    `json:"audit_last_hash"`
}

// Dumper menulis isi satu database dalam format custom pg_dump ke w.
type Dumper interface {
	Dump(ctx context.Context, w io.Writer) error
}

// Restorer memulihkan isi format custom pg_dump dari r ke database yang
// sudah ada dan masih kosong, dengan RestoreFlags.
type Restorer interface {
	Restore(ctx context.Context, database string, r io.Reader) error
}

// CheckPassphrase memeriksa passphrase backup. Passphrase yang lupa berarti
// backup tidak bisa dibuka lagi, jadi simpan salinannya di tempat lain.
func CheckPassphrase(passphrase string) error {
	if utf8.RuneCountInString(passphrase) < minPassphraseLength {
		return fmt.Errorf("BACKUP_PASSPHRASE harus paling sedikit %d karakter", minPassphraseLength)
	}
	return nil
}

// Encrypt mengenkripsi isi src ke dst dengan age, memakai passphrase
// (scrypt). Hasilnya bisa dibuka dengan Decrypt atau dengan `age -d`.
func Encrypt(dst io.Writer, src io.Reader, passphrase string) error {
	w, err := encryptTo(dst, passphrase)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, src); err != nil {
		return fmt.Errorf("mengenkripsi: %w", err)
	}
	// Close menulis potongan terakhir; tanpa Close, file tidak lengkap.
	return w.Close()
}

// encryptTo membuka penulis terenkripsi ke dst.
func encryptTo(dst io.Writer, passphrase string) (io.WriteCloser, error) {
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return nil, fmt.Errorf("menyiapkan enkripsi: %w", err)
	}
	w, err := age.Encrypt(dst, recipient)
	if err != nil {
		return nil, fmt.Errorf("menyiapkan enkripsi: %w", err)
	}
	return w, nil
}

// Decrypt membuka isi yang dienkripsi Encrypt. Passphrase yang salah ditolak
// di sini, sebelum isinya dibaca.
func Decrypt(src io.Reader, passphrase string) (io.Reader, error) {
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, fmt.Errorf("menyiapkan dekripsi: %w", err)
	}
	r, err := age.Decrypt(src, identity)
	if err != nil {
		// Isi passphrase tidak disebut (E-18).
		return nil, errors.New("backup tidak bisa dibuka: passphrase salah atau file bukan backup age")
	}
	return r, nil
}

// ReadInfo membaca keadaan database dari koneksi pemilik: nama database,
// versi skema, dan ujung rantai audit. Dibaca sebelum dump dimulai, jadi
// rantai di dalam dump paling sedikit sepanjang yang tercatat di manifest.
func ReadInfo(ctx context.Context, owner *pgxpool.Pool) (Info, error) {
	version, err := db.SchemaVersion(ctx, owner)
	if err != nil {
		return Info{}, err
	}
	seq, hash, err := audit.ChainHead(ctx, db.New(owner))
	if err != nil {
		return Info{}, err
	}
	return Info{Database: owner.Config().ConnConfig.Database, SchemaVersion: version, AuditLastSeq: seq, AuditLastHash: hash}, nil
}

// Create membuat satu backup di folder dir: dump dari dumper dienkripsi dan
// ditulis ke file sementara, lalu diganti nama setelah lengkap, sehingga file
// backup tidak pernah setengah jadi. Folder harus sudah ada; Create tidak
// membuatnya, supaya backup tidak diam-diam tertulis ke disk yang salah saat
// disk tujuan belum terpasang.
//
// ASUMSI E-60: backup lama tidak pernah dihapus otomatis, dan jadwal harian
// diatur di luar program (Task Scheduler di Windows, cron di Linux).
func Create(ctx context.Context, dumper Dumper, dir, passphrase string, info Info, now time.Time) (Manifest, error) {
	if err := CheckPassphrase(passphrase); err != nil {
		return Manifest{}, err
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return Manifest{}, fmt.Errorf("folder backup %s tidak ada: pasang disk tujuannya, atau isi BACKUP_DIR dengan folder yang ada", dir)
	}

	base := "revioten-" + now.UTC().Format("20060102-150405")
	m := Manifest{
		Format: manifestFormat, CreatedAt: now.UTC(), File: base + dumpSuffix, Database: info.Database,
		SchemaVersion: info.SchemaVersion, AuditLastSeq: info.AuditLastSeq, AuditLastHash: info.AuditLastHash,
	}
	final := filepath.Join(dir, m.File)
	if _, err := os.Stat(final); err == nil {
		return Manifest{}, fmt.Errorf("file backup %s sudah ada; tunggu satu detik lalu coba lagi", final)
	}

	sum, size, err := writeAtomic(dir, final, func(w io.Writer) error {
		sealed, err := encryptTo(w, passphrase)
		if err != nil {
			return err
		}
		if err := dumper.Dump(ctx, sealed); err != nil {
			return fmt.Errorf("membuat dump: %w", err)
		}
		return sealed.Close()
	})
	if err != nil {
		return Manifest{}, err
	}
	m.SHA256, m.Size = sum, size

	manifest, err := json.MarshalIndent(m, "", "  ")
	if err == nil {
		_, _, err = writeAtomic(dir, filepath.Join(dir, base+manifestSuffix), func(w io.Writer) error {
			_, err := w.Write(append(manifest, '\n'))
			return err
		})
	}
	if err != nil {
		// Backup tanpa manifest tidak bisa diperiksa, jadi ikut dibuang.
		_ = os.Remove(final)
		return Manifest{}, fmt.Errorf("menulis manifest: %w", err)
	}
	return m, nil
}

// writeAtomic menulis file final lewat file sementara di folder yang sama,
// lalu menggantinya nama. Hasilnya hash SHA-256 (heksadesimal) dan ukuran isi.
func writeAtomic(dir, final string, write func(io.Writer) error) (string, int64, error) {
	tmp, err := os.CreateTemp(dir, filepath.Base(final)+".tmp-*")
	if err != nil {
		return "", 0, fmt.Errorf("membuat file di %s: %w", dir, err)
	}
	// Bila ada yang gagal, file sementara dibuang. Setelah Rename berhasil,
	// Remove ini tidak menemukan apa-apa.
	defer os.Remove(tmp.Name())

	h := sha256.New()
	counter := &countingWriter{}
	if err := write(io.MultiWriter(tmp, h, counter)); err != nil {
		tmp.Close()
		return "", 0, err
	}
	// Sync memastikan isinya sudah di disk sebelum file dianggap jadi.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", 0, fmt.Errorf("menyimpan %s: %w", final, err)
	}
	if err := tmp.Close(); err != nil {
		return "", 0, fmt.Errorf("menyimpan %s: %w", final, err)
	}
	if err := os.Rename(tmp.Name(), final); err != nil {
		return "", 0, fmt.Errorf("menyimpan %s: %w", final, err)
	}
	return hex.EncodeToString(h.Sum(nil)), counter.n, nil
}

// countingWriter menghitung jumlah byte yang ditulis.
type countingWriter struct{ n int64 }

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}

// ReadManifest membaca manifest yang ada di samping file backup path.
func ReadManifest(path string) (Manifest, error) {
	if !strings.HasSuffix(path, dumpSuffix) {
		return Manifest{}, fmt.Errorf("file backup harus berakhiran %s", dumpSuffix)
	}
	manifestPath := strings.TrimSuffix(path, dumpSuffix) + manifestSuffix
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("membaca manifest %s: %w", manifestPath, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("manifest %s tidak bisa dibaca: %w", manifestPath, err)
	}
	if m.Format != manifestFormat {
		return Manifest{}, fmt.Errorf("manifest %s berformat %d; program ini membaca format %d", manifestPath, m.Format, manifestFormat)
	}
	return m, nil
}

// VerifyFile memeriksa bahwa file backup sama persis dengan yang dicatat
// manifest: ukuran dan hash SHA-256. File yang rusak atau diganti ketahuan di
// sini, sebelum restore dimulai.
func VerifyFile(path string, m Manifest) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("membuka file backup: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return fmt.Errorf("membaca file backup: %w", err)
	}
	if size != m.Size || hex.EncodeToString(h.Sum(nil)) != m.SHA256 {
		return fmt.Errorf("file backup %s tidak sama dengan manifest-nya: file rusak atau diganti", filepath.Base(path))
	}
	return nil
}

// Target adalah server tujuan restore.
type Target struct {
	Restorer Restorer
	// Owner adalah koneksi pemilik ke server tujuan, ke database mana saja.
	// Dipakai untuk membuat database baru dan role aplikasi.
	Owner *pgxpool.Pool
	// Connect membuka koneksi pemilik ke database hasil restore, untuk
	// memeriksa rantai audit-nya.
	Connect func(ctx context.Context, database string) (*pgxpool.Pool, error)
}

// Result adalah hasil restore.
type Result struct {
	Manifest Manifest
	Database string
	// Chain adalah hasil pemeriksaan rantai audit di database hasil restore.
	Chain audit.Verification
	// TipMatches true bila entri ujung rantai yang dicatat manifest ada di
	// database hasil restore dengan hash yang sama.
	TipMatches bool
}

// databaseNamePattern: huruf kecil, angka, dan garis bawah, paling banyak 63
// karakter (batas nama di PostgreSQL), diawali huruf atau garis bawah.
var databaseNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// Restore memulihkan backup path ke database baru bernama database:
//  1. file diperiksa terhadap manifest-nya (VerifyFile);
//  2. passphrase diperiksa dengan membuka file;
//  3. database baru dibuat (database yang sudah ada tidak pernah ditimpa),
//     dan role aplikasi disiapkan;
//  4. isinya dipulihkan oleh Restorer dalam satu transaksi;
//  5. rantai audit database baru diperiksa, dan ujungnya dicocokkan dengan
//     manifest.
//
// Bila langkah 4 gagal, database yang baru dibuat dibuang lagi.
func Restore(ctx context.Context, t Target, path, passphrase, database string) (Result, error) {
	if !databaseNamePattern.MatchString(database) {
		return Result{}, errors.New("nama database hanya boleh berisi huruf kecil a-z, angka, dan garis bawah, paling banyak 63 karakter")
	}
	m, err := ReadManifest(path)
	if err != nil {
		return Result{}, err
	}
	if err := VerifyFile(path, m); err != nil {
		return Result{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Result{}, fmt.Errorf("membuka file backup: %w", err)
	}
	defer f.Close()
	plain, err := Decrypt(f, passphrase)
	if err != nil {
		return Result{}, err
	}

	var exists bool
	if err := t.Owner.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", database).Scan(&exists); err != nil {
		return Result{}, fmt.Errorf("memeriksa database tujuan: %w", err)
	}
	if exists {
		return Result{}, fmt.Errorf("%w: %s; pilih nama lain dengan -db", ErrDatabaseExists, database)
	}
	// CREATE DATABASE tidak menerima parameter $1, jadi nama diamankan dengan
	// Sanitize (tanda kutip ganda), setelah diperiksa polanya di atas.
	name := pgx.Identifier{database}.Sanitize()
	if _, err := t.Owner.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		return Result{}, fmt.Errorf("membuat database %s: %w", database, err)
	}
	if err := restoreInto(ctx, t, database, plain); err != nil {
		if _, dropErr := t.Owner.Exec(ctx, "DROP DATABASE "+name); dropErr != nil {
			return Result{}, fmt.Errorf("%w (database %s juga gagal dibuang: %v)", err, database, dropErr)
		}
		return Result{}, err
	}

	pool, err := t.Connect(ctx, database)
	if err != nil {
		return Result{}, fmt.Errorf("membuka database hasil restore: %w", err)
	}
	defer pool.Close()
	q := db.New(pool)
	chain, err := audit.VerifyChain(ctx, q)
	if err != nil {
		return Result{}, err
	}
	tipMatches := m.AuditLastSeq == 0
	if !tipMatches {
		hash, err := audit.HashAt(ctx, q, m.AuditLastSeq)
		if err != nil {
			return Result{}, err
		}
		tipMatches = hash == m.AuditLastHash
	}
	return Result{Manifest: m, Database: database, Chain: chain, TipMatches: tipMatches}, nil
}

// restoreInto menyiapkan role aplikasi, lalu menjalankan Restorer.
func restoreInto(ctx context.Context, t Target, database string, plain io.Reader) error {
	if err := db.EnsureAppRole(ctx, t.Owner); err != nil {
		return err
	}
	if err := t.Restorer.Restore(ctx, database, plain); err != nil {
		return fmt.Errorf("memulihkan isi backup: %w", err)
	}
	return nil
}
