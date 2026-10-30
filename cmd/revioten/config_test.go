package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// fakeEnv meniru os.Getenv dari sebuah map.
func fakeEnv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

// writeEnvFile menulis file .env sementara dan mengembalikan alamatnya.
func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const sampleEnv = `# contoh seperti .env.example
POSTGRES_USER=revioten
POSTGRES_PASSWORD=rahasia-uji
POSTGRES_DB=revioten
POSTGRES_PORT=5433
APP_DB_PASSWORD=PasswordAplikasiUji2026
`

// TestLoadConfigFromEnvFile: dua alamat database dari isian .env, satu untuk
// pemilik database (migrasi) dan satu untuk role aplikasi (E-06).
func TestLoadConfigFromEnvFile(t *testing.T) {
	cfg, err := loadConfig(fakeEnv(nil), writeEnvFile(t, sampleEnv))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if want := "postgres://revioten:rahasia-uji@127.0.0.1:5433/revioten?sslmode=disable"; cfg.OwnerURL != want {
		t.Errorf("OwnerURL %q, ingin %q", cfg.OwnerURL, want)
	}
	if want := "postgres://revioten_app:PasswordAplikasiUji2026@127.0.0.1:5433/revioten?sslmode=disable"; cfg.AppURL != want {
		t.Errorf("AppURL %q, ingin %q", cfg.AppURL, want)
	}
	if cfg.AppPassword != "PasswordAplikasiUji2026" {
		t.Error("AppPassword tidak diambil dari APP_DB_PASSWORD")
	}
}

// TestLoadConfigEnvWins: nilai di environment menang atas file .env, sama
// seperti Docker Compose. Test E2E memakai ini untuk database terpisah.
func TestLoadConfigEnvWins(t *testing.T) {
	env := fakeEnv(map[string]string{"POSTGRES_DB": "revioten_e2e"})
	cfg, err := loadConfig(env, writeEnvFile(t, sampleEnv))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	for _, u := range []string{cfg.OwnerURL, cfg.AppURL} {
		if !strings.HasSuffix(u, "/revioten_e2e?sslmode=disable") {
			t.Errorf("alamat %q tidak memakai POSTGRES_DB dari environment", u)
		}
	}
}

// TestLoadConfigDatabaseURL: DATABASE_URL dipakai apa adanya untuk pemilik
// database. Role aplikasi memakai server dan database yang sama.
func TestLoadConfigDatabaseURL(t *testing.T) {
	env := fakeEnv(map[string]string{
		"DATABASE_URL":    "postgres://u:p@db.example:6000/x",
		"APP_DB_PASSWORD": "PasswordAplikasiUji2026",
	})
	cfg, err := loadConfig(env, filepath.Join(t.TempDir(), "tidak-ada.env"))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.OwnerURL != "postgres://u:p@db.example:6000/x" {
		t.Errorf("OwnerURL %q, ingin nilai DATABASE_URL apa adanya", cfg.OwnerURL)
	}
	if cfg.AppURL != "postgres://revioten_app:PasswordAplikasiUji2026@db.example:6000/x" {
		t.Errorf("AppURL %q", cfg.AppURL)
	}
}

func TestLoadConfigDefaultPort(t *testing.T) {
	content := strings.Replace(sampleEnv, "POSTGRES_PORT=5433\n", "", 1)
	cfg, err := loadConfig(fakeEnv(nil), writeEnvFile(t, content))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if !strings.Contains(cfg.OwnerURL, "@127.0.0.1:5432/") {
		t.Errorf("OwnerURL %q tidak memakai port bawaan 5432", cfg.OwnerURL)
	}
}

// TestLoadConfigSpecialPassword: password dengan karakter khusus harus
// dikodekan di URL, lalu terbaca kembali utuh oleh pgx.
func TestLoadConfigSpecialPassword(t *testing.T) {
	password := "p@ss:w/rd#1 %?&="
	env := fakeEnv(map[string]string{"POSTGRES_PASSWORD": password})
	cfg, err := loadConfig(env, writeEnvFile(t, sampleEnv))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	parsed, err := pgxpool.ParseConfig(cfg.OwnerURL)
	if err != nil {
		t.Fatalf("URL tidak bisa dibaca pgx: %v", err)
	}
	if parsed.ConnConfig.Password != password || parsed.ConnConfig.User != "revioten" {
		t.Errorf("pgx membaca user %q dan password yang berbeda", parsed.ConnConfig.User)
	}
}

// TestLoadConfigMissing: pesan error menyebut apa yang kurang dan cara
// memperbaikinya, tanpa pernah memuat password (E-18).
func TestLoadConfigMissing(t *testing.T) {
	env := fakeEnv(map[string]string{"POSTGRES_PASSWORD": "rahasia-uji"})
	_, err := loadConfig(env, filepath.Join(t.TempDir(), "tidak-ada.env"))
	if err == nil {
		t.Fatal("ingin error karena POSTGRES_USER, POSTGRES_DB, dan APP_DB_PASSWORD kosong")
	}
	msg := err.Error()
	for _, want := range []string{"POSTGRES_USER", "POSTGRES_DB", "APP_DB_PASSWORD", ".env.example"} {
		if !strings.Contains(msg, want) {
			t.Errorf("pesan %q tidak menyebut %s", msg, want)
		}
	}
	if strings.Contains(msg, "rahasia-uji") {
		t.Errorf("password bocor ke pesan error: %q", msg)
	}
}

func TestParseEnvFile(t *testing.T) {
	content := `
# komentar
  POSTGRES_USER = revioten
QUOTED="nilai dengan # tanda pagar"
SINGLE='kutip tunggal'
INLINE=abc # komentar setelah spasi
NOT_COMMENT=abc#def
EMPTY=
`
	got, err := parseEnvFile(content)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
	}
	want := map[string]string{
		"POSTGRES_USER": "revioten",
		"QUOTED":        "nilai dengan # tanda pagar",
		"SINGLE":        "kutip tunggal",
		"INLINE":        "abc",
		"NOT_COMMENT":   "abc#def",
		"EMPTY":         "",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %q, ingin %q", key, got[key], value)
		}
	}
	if len(got) != len(want) {
		t.Errorf("ada %d kunci, ingin %d: %v", len(got), len(want), got)
	}
}

// TestParseEnvFileInvalidLine: baris yang salah disebut nomornya saja, tanpa
// isinya, karena isinya bisa berupa password.
func TestParseEnvFileInvalidLine(t *testing.T) {
	_, err := parseEnvFile("POSTGRES_USER=revioten\nrahasia-tanpa-nama\n")
	if err == nil {
		t.Fatal("ingin error untuk baris tanpa tanda =")
	}
	if !strings.Contains(err.Error(), "baris 2") || strings.Contains(err.Error(), "rahasia") {
		t.Errorf("pesan %q harus menyebut baris 2 tanpa isinya", err.Error())
	}
}

// TestLoadConfigBackup: pengaturan backup (M1.7) dibaca dari .env tapi tidak
// wajib untuk server. Nama user pemilik dan nama database diambil dari alamat
// pemilik, juga bila alamat itu ditulis lewat DATABASE_URL.
func TestLoadConfigBackup(t *testing.T) {
	cfg, err := loadConfig(fakeEnv(nil), writeEnvFile(t, sampleEnv+"BACKUP_DIR=D:/backup-revioten\nBACKUP_PASSPHRASE=sapi perah di ladang jagung\n"))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.BackupDir != "D:/backup-revioten" || cfg.BackupPassphrase != "sapi perah di ladang jagung" ||
		cfg.OwnerUser != "revioten" || cfg.Database != "revioten" {
		t.Errorf("hasil %+v", cfg)
	}

	env := fakeEnv(map[string]string{"DATABASE_URL": "postgres://pemilik:rahasia@127.0.0.1:5432/demo?sslmode=disable"})
	cfg, err = loadConfig(env, writeEnvFile(t, "APP_DB_PASSWORD=PasswordAplikasiUji2026\n"))
	if err != nil {
		t.Fatalf("loadConfig dengan DATABASE_URL: %v", err)
	}
	if cfg.OwnerUser != "pemilik" || cfg.Database != "demo" || cfg.BackupDir != "" {
		t.Errorf("hasil dengan DATABASE_URL %+v", cfg)
	}
	if got := withDatabase(cfg.OwnerURL, "revioten_restore"); got != "postgres://pemilik:rahasia@127.0.0.1:5432/revioten_restore?sslmode=disable" {
		t.Errorf("withDatabase %q", got)
	}
}

// TestBackupFlags: restore wajib diberi -file, dan nama database bawaannya
// tidak pernah sama dengan database yang sedang dipakai.
func TestBackupFlags(t *testing.T) {
	if _, err := parseRestoreFlags(nil, &strings.Builder{}); err == nil {
		t.Error("restore tanpa -file harus ditolak")
	}
	opts, err := parseRestoreFlags([]string{"-file", "D:/backup/revioten-20261030-134500.dump.age"}, &strings.Builder{})
	if err != nil {
		t.Fatalf("parseRestoreFlags: %v", err)
	}
	if opts.database != "revioten_restore" || opts.envFile != ".env" {
		t.Errorf("hasil %+v", opts)
	}
}
