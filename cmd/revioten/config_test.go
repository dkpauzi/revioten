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
`

func TestLoadConfigFromEnvFile(t *testing.T) {
	cfg, err := loadConfig(fakeEnv(nil), writeEnvFile(t, sampleEnv))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	want := "postgres://revioten:rahasia-uji@127.0.0.1:5433/revioten?sslmode=disable"
	if cfg.DatabaseURL != want {
		t.Errorf("DatabaseURL %q, ingin %q", cfg.DatabaseURL, want)
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
	if !strings.HasSuffix(cfg.DatabaseURL, "/revioten_e2e?sslmode=disable") {
		t.Errorf("DatabaseURL %q tidak memakai POSTGRES_DB dari environment", cfg.DatabaseURL)
	}
}

func TestLoadConfigDatabaseURL(t *testing.T) {
	env := fakeEnv(map[string]string{"DATABASE_URL": "postgres://u:p@db.example:6000/x"})
	cfg, err := loadConfig(env, filepath.Join(t.TempDir(), "tidak-ada.env"))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.DatabaseURL != "postgres://u:p@db.example:6000/x" {
		t.Errorf("DatabaseURL %q, ingin nilai DATABASE_URL apa adanya", cfg.DatabaseURL)
	}
}

func TestLoadConfigDefaultPort(t *testing.T) {
	content := strings.Replace(sampleEnv, "POSTGRES_PORT=5433\n", "", 1)
	cfg, err := loadConfig(fakeEnv(nil), writeEnvFile(t, content))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if !strings.Contains(cfg.DatabaseURL, "@127.0.0.1:5432/") {
		t.Errorf("DatabaseURL %q tidak memakai port bawaan 5432", cfg.DatabaseURL)
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
	parsed, err := pgxpool.ParseConfig(cfg.DatabaseURL)
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
		t.Fatal("ingin error karena POSTGRES_USER dan POSTGRES_DB kosong")
	}
	msg := err.Error()
	for _, want := range []string{"POSTGRES_USER", "POSTGRES_DB", ".env.example"} {
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
