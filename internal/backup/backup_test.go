package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
)

// TestMain menyalakan PostgreSQL untuk uji restore. Test lain di paket ini
// tetap berjalan dengan -short.
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// passphraseUji panjangnya lebih dari 15 karakter (CheckPassphrase).
const passphraseUji = "sapi perah di ladang jagung"

// waktuUji adalah waktu backup di test.
var waktuUji = time.Date(2026, 10, 30, 13, 45, 0, 0, time.UTC)

// fakeDumper menulis isi yang sudah ditentukan, sebagai pengganti pg_dump.
type fakeDumper struct {
	content string
	err     error
}

func (d fakeDumper) Dump(_ context.Context, w io.Writer) error {
	if d.err != nil {
		return d.err
	}
	_, err := io.WriteString(w, d.content)
	return err
}

// TestEncryptRoundTrip: isi yang dienkripsi hanya bisa dibaca kembali dengan
// passphrase yang sama, dan teks aslinya tidak terlihat di file.
func TestEncryptRoundTrip(t *testing.T) {
	var sealed bytes.Buffer
	if err := Encrypt(&sealed, strings.NewReader("isi database rahasia"), passphraseUji); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if strings.Contains(sealed.String(), "rahasia") {
		t.Error("teks asli terlihat di hasil enkripsi")
	}

	plain, err := Decrypt(bytes.NewReader(sealed.Bytes()), passphraseUji)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	got, err := io.ReadAll(plain)
	if err != nil || string(got) != "isi database rahasia" {
		t.Errorf("hasil Decrypt %q, %v", got, err)
	}

	if _, err := Decrypt(bytes.NewReader(sealed.Bytes()), "passphrase yang salah sama sekali"); err == nil {
		t.Error("passphrase salah harus ditolak")
	}
}

// TestCheckPassphrase: passphrase paling sedikit 15 karakter. Sebelum D-132 sama dengan
// panjang minimal password (D-85).
func TestCheckPassphrase(t *testing.T) {
	if err := CheckPassphrase("pendek"); err == nil {
		t.Error("passphrase pendek harus ditolak")
	}
	if err := CheckPassphrase(""); err == nil {
		t.Error("passphrase kosong harus ditolak")
	}
	if err := CheckPassphrase(passphraseUji); err != nil {
		t.Errorf("passphrase panjang ditolak: %v", err)
	}
}

// TestCreate: backup ditulis ke folder tujuan sebagai file terenkripsi beserta
// manifest-nya, dan manifest mencatat hash file serta ujung rantai audit.
func TestCreate(t *testing.T) {
	dir := t.TempDir()
	info := Info{Database: "revioten", SchemaVersion: 9, AuditLastSeq: 42, AuditLastHash: "abc123"}
	m, err := Create(t.Context(), fakeDumper{content: "isi dump"}, dir, passphraseUji, info, waktuUji)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.File != "revioten-20261030-134500.dump.age" || m.AuditLastSeq != 42 || m.AuditLastHash != "abc123" ||
		m.SchemaVersion != 9 || m.Database != "revioten" || !m.CreatedAt.Equal(waktuUji) || m.Format != 1 {
		t.Errorf("manifest %+v", m)
	}

	// Manifest di samping file backup, dan isinya sama dengan hasil Create.
	read, err := ReadManifest(filepath.Join(dir, m.File))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if read != m {
		t.Errorf("manifest dari file %+v, ingin %+v", read, m)
	}
	if err := VerifyFile(filepath.Join(dir, m.File), m); err != nil {
		t.Errorf("VerifyFile: %v", err)
	}

	// Tidak ada file sementara yang tertinggal.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if fmt.Sprint(names) != "[revioten-20261030-134500.dump.age revioten-20261030-134500.manifest.json]" {
		t.Errorf("isi folder: %v", names)
	}
}

// TestCreateFailures: folder tujuan harus sudah ada (misalnya disk lain yang
// terpasang), dan dump yang gagal tidak meninggalkan file setengah jadi.
func TestCreateFailures(t *testing.T) {
	info := Info{Database: "revioten"}
	missing := filepath.Join(t.TempDir(), "disk-yang-tidak-terpasang")
	if _, err := Create(t.Context(), fakeDumper{content: "x"}, missing, passphraseUji, info, waktuUji); err == nil {
		t.Error("folder yang tidak ada harus ditolak")
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Error("Create tidak boleh membuat folder tujuan sendiri")
	}

	dir := t.TempDir()
	if _, err := Create(t.Context(), fakeDumper{err: errors.New("pg_dump gagal")}, dir, passphraseUji, info, waktuUji); err == nil {
		t.Error("dump yang gagal harus menjadi error")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("file tertinggal setelah dump gagal: %v", entries)
	}
	if _, err := Create(t.Context(), fakeDumper{content: "x"}, dir, "pendek", info, waktuUji); err == nil {
		t.Error("passphrase pendek harus ditolak")
	}
}

// TestVerifyFileDetectsDamage: file backup yang berubah satu byte saja
// ketahuan dari hash di manifest.
func TestVerifyFileDetectsDamage(t *testing.T) {
	dir := t.TempDir()
	m, err := Create(t.Context(), fakeDumper{content: "isi dump"}, dir, passphraseUji, Info{Database: "revioten"}, waktuUji)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, m.File)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] ^= 0xff
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFile(path, m); err == nil {
		t.Error("file yang rusak harus ditolak")
	}
}

// containerDumper menjalankan pg_dump di dalam container test, menggantikan
// docker compose exec di perintah backup.
type containerDumper struct{ database string }

func (d containerDumper) Dump(ctx context.Context, w io.Writer) error {
	path := "/tmp/" + d.database + ".dump"
	if err := dbtest.Exec(ctx, "pg_dump", "-U", dbtest.OwnerUser(), "-d", d.database, "--format=custom", "--file="+path); err != nil {
		return err
	}
	r, err := dbtest.CopyFromContainer(ctx, path)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(w, r)
	return err
}

// containerRestorer menjalankan pg_restore di dalam container test.
type containerRestorer struct{}

func (containerRestorer) Restore(ctx context.Context, database string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	path := "/tmp/restore-" + database + ".dump"
	if err := dbtest.CopyToContainer(ctx, data, path); err != nil {
		return err
	}
	return dbtest.Exec(ctx, append([]string{"pg_restore", "-U", dbtest.OwnerUser(), "-d", database}, append(RestoreFlags, path)...)...)
}

// TestBackupAndRestore adalah kriteria selesai M1.7: backup database yang
// berisi data dan audit log dipulihkan ke database kosong, rantai auditnya
// utuh, dan ujung rantainya cocok dengan manifest (D-119).
func TestBackupAndRestore(t *testing.T) {
	app, owner := dbtest.NewPools(t)
	ctx := t.Context()
	source := owner.Config().ConnConfig.Database

	// Isi database: admin pertama dan satu customer, masing-masing dengan entri audit.
	checker := access.NewChecker(app)
	authSvc := auth.NewService(app, checker)
	admin, err := authSvc.CreateFirstAdmin(ctx, "Admin Uji", "admin.uji", "kuda terbang di atas sawah")
	if err != nil {
		t.Fatal(err)
	}
	customers := customer.NewService(app, checker)
	group, err := customers.CreateProjectGroup(ctx, admin, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := customers.Create(ctx, admin, customer.NewCustomer{Code: "AB", Name: "Customer Contoh", ProjectGroupID: group.ID}); err != nil {
		t.Fatal(err)
	}

	info, err := ReadInfo(ctx, owner)
	if err != nil {
		t.Fatalf("ReadInfo: %v", err)
	}
	if info.Database != source || info.AuditLastSeq != 3 || info.AuditLastHash == "" || info.SchemaVersion < 9 {
		t.Errorf("info %+v", info)
	}
	dir := t.TempDir()
	m, err := Create(ctx, containerDumper{database: source}, dir, passphraseUji, info, waktuUji)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	target := Target{
		Restorer: containerRestorer{},
		Owner:    owner,
		Connect: func(ctx context.Context, database string) (*pgxpool.Pool, error) {
			return dbtest.ConnectOwner(ctx, database)
		},
	}
	restored := source + "_restore"
	result, err := Restore(ctx, target, filepath.Join(dir, m.File), passphraseUji, restored)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if !result.Chain.Valid || result.Chain.LastSeq != 3 || !result.TipMatches {
		t.Errorf("hasil restore %+v", result)
	}

	// Isi database hasil restore sama: customer dan admin ada.
	pool, err := dbtest.ConnectOwner(ctx, restored)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var name string
	if err := pool.QueryRow(ctx, "SELECT name FROM customers WHERE code = 'AB'").Scan(&name); err != nil || name != "Customer Contoh" {
		t.Errorf("customer hasil restore: %q, %v", name, err)
	}

	// Database yang sudah ada tidak pernah ditimpa.
	if _, err := Restore(ctx, target, filepath.Join(dir, m.File), passphraseUji, restored); !errors.Is(err, ErrDatabaseExists) {
		t.Errorf("restore ke database yang ada: %v, ingin ErrDatabaseExists", err)
	}
	// Passphrase salah: ditolak, dan database yang baru dibuat dibuang lagi.
	if _, err := Restore(ctx, target, filepath.Join(dir, m.File), "passphrase yang salah sama sekali", restored+"_2"); err == nil {
		t.Error("passphrase salah harus ditolak")
	}
	var exists bool
	if err := owner.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", restored+"_2").Scan(&exists); err != nil || exists {
		t.Errorf("database dari restore yang gagal masih ada: %v, %v", exists, err)
	}
	// Nama database yang tidak aman ditolak sebelum menyentuh server.
	if _, err := Restore(ctx, target, filepath.Join(dir, m.File), passphraseUji, "Nama Aneh;"); err == nil {
		t.Error("nama database yang tidak aman harus ditolak")
	}
}
