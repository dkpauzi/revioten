package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/backup"
	"github.com/dkpauzi/revioten/internal/db"
)

// Perintah backup dan restore (M1.7, D-119). Keduanya dijalankan di komputer
// server, dari folder repo (tempat compose.yaml), saat PostgreSQL menyala lewat
// docker compose up -d. pg_dump dan pg_restore dijalankan di dalam container
// db, jadi tidak perlu dipasang di komputer.

// runBackup menjalankan `revioten backup`: satu backup terenkripsi ke
// BACKUP_DIR, beserta manifest-nya.
func runBackup(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	fs.SetOutput(out)
	envFile := fs.String("env", ".env", "file berisi pengaturan database dan backup")
	if err := fs.Parse(args); err != nil {
		return errors.New("flag backup tidak dikenal; yang tersedia: -env")
	}
	cfg, err := loadConfig(os.Getenv, *envFile)
	if err != nil {
		return err
	}
	if cfg.BackupDir == "" || cfg.BackupPassphrase == "" {
		return errors.New("BACKUP_DIR dan BACKUP_PASSPHRASE belum diisi di file .env: lihat contoh dan penjelasannya di .env.example")
	}

	ctx := context.Background()
	owner, err := db.Connect(ctx, cfg.OwnerURL)
	if err != nil {
		return err
	}
	defer owner.Close()
	info, err := backup.ReadInfo(ctx, owner)
	if err != nil {
		return err
	}
	// Folder lampiran ikut terarsip terenkripsi (D-163).
	m, err := backup.CreateWithFiles(ctx, composeDumper{user: cfg.OwnerUser, database: info.Database},
		cfg.BackupDir, cfg.BackupPassphrase, info, time.Now(), cfg.AttachmentDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Backup dibuat: %s (%d byte)\n", filepath.Join(cfg.BackupDir, m.File), m.Size)
	fmt.Fprintf(out, "SHA-256: %s\n", m.SHA256)
	fmt.Fprintf(out, "Versi skema %d, ujung rantai audit: entri %d\n", m.SchemaVersion, m.AuditLastSeq)
	if m.Files != "" {
		fmt.Fprintf(out, "Lampiran: %s (%d berkas)\n", filepath.Join(cfg.BackupDir, m.Files), m.FilesCount)
	}
	return nil
}

// restoreOptions adalah isian perintah restore.
type restoreOptions struct {
	file     string
	database string
	envFile  string
	files    string // folder baru untuk lampiran hasil restore; "" berarti tidak
}

// parseRestoreFlags membaca flag perintah restore. Nama database bawaannya
// revioten_restore, jadi restore tidak pernah menimpa database yang dipakai.
func parseRestoreFlags(args []string, out io.Writer) (restoreOptions, error) {
	var opts restoreOptions
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&opts.file, "file", "", "file backup berakhiran .dump.age (wajib)")
	fs.StringVar(&opts.database, "db", "revioten_restore", "nama database baru untuk hasil restore")
	fs.StringVar(&opts.envFile, "env", ".env", "file berisi pengaturan database dan backup")
	fs.StringVar(&opts.files, "files", "", "folder baru (kosong) untuk lampiran hasil restore (D-163)")
	if err := fs.Parse(args); err != nil {
		return restoreOptions{}, errors.New("flag restore tidak dikenal; yang tersedia: -file, -db, -env, -files")
	}
	if opts.file == "" {
		return restoreOptions{}, errors.New("isi -file, contoh: revioten restore -file D:/backup-revioten/revioten-20261030-134500.dump.age")
	}
	return opts, nil
}

// runRestore menjalankan `revioten restore`: backup dipulihkan ke database
// baru, lalu rantai audit-nya diperiksa terhadap manifest.
func runRestore(args []string, out io.Writer) error {
	opts, err := parseRestoreFlags(args, out)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(os.Getenv, opts.envFile)
	if err != nil {
		return err
	}
	if cfg.BackupPassphrase == "" {
		return errors.New("BACKUP_PASSPHRASE belum diisi di file .env: isi dengan passphrase saat backup dibuat")
	}

	ctx := context.Background()
	owner, err := db.Connect(ctx, cfg.OwnerURL)
	if err != nil {
		return err
	}
	defer owner.Close()
	target := backup.Target{
		Restorer: composeRestorer{user: cfg.OwnerUser},
		Owner:    owner,
		Connect: func(ctx context.Context, database string) (*pgxpool.Pool, error) {
			return db.Connect(ctx, withDatabase(cfg.OwnerURL, database))
		},
	}
	result, err := backup.Restore(ctx, target, opts.file, cfg.BackupPassphrase, opts.database)
	if err != nil {
		return err
	}

	m := result.Manifest
	fmt.Fprintf(out, "Database %s dipulihkan dari backup %s (versi skema %d, dibuat %s UTC).\n",
		result.Database, m.File, m.SchemaVersion, m.CreatedAt.Format("2006-01-02 15:04:05"))
	switch {
	case !result.Chain.Valid:
		fmt.Fprintf(out, "PERINGATAN: rantai audit rusak di entri %d (%s).\n", result.Chain.BrokenSeq, result.Chain.Reason)
	case !result.TipMatches:
		fmt.Fprintf(out, "PERINGATAN: entri audit %d tidak cocok dengan manifest. Isi audit log berbeda dari saat backup dibuat.\n", m.AuditLastSeq)
	default:
		fmt.Fprintf(out, "Rantai audit utuh: %d entri. Ujung rantai yang dicatat manifest (entri %d) cocok.\n", result.Chain.Entries, m.AuditLastSeq)
	}
	fmt.Fprintf(out, "Database yang sedang dipakai (%s) tidak diubah.\n", cfg.Database)
	if opts.files != "" {
		n, err := backup.RestoreFiles(opts.file, cfg.BackupPassphrase, opts.files)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "Lampiran dipulihkan ke %s: %d berkas.\n", opts.files, n)
	}
	return nil
}

// composeDumper menjalankan pg_dump di dalam container db lewat docker compose.
type composeDumper struct {
	user, database string
}

func (d composeDumper) Dump(ctx context.Context, w io.Writer) error {
	return runCompose(ctx, nil, w, "pg_dump", "--username="+d.user, "--dbname="+d.database, "--format=custom")
}

// composeRestorer menjalankan pg_restore di dalam container db lewat docker compose.
type composeRestorer struct {
	user string
}

func (r composeRestorer) Restore(ctx context.Context, database string, in io.Reader) error {
	args := append([]string{"pg_restore", "--username=" + r.user, "--dbname=" + database}, backup.RestoreFlags...)
	return runCompose(ctx, in, nil, args...)
}

// runCompose menjalankan satu perintah di container db: docker compose exec
// -T (tanpa terminal, supaya data biner tidak diubah). Di dalam container,
// pg_dump dan pg_restore login lewat socket lokal, jadi tidak ada password
// yang dikirim atau tertulis di pesan error (E-18, D-95).
func runCompose(ctx context.Context, stdin io.Reader, stdout io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose", "exec", "-T", "db"}, args...)...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s lewat docker compose gagal: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
