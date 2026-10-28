package db

import (
	"errors"
	"net/url"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testAppPassword adalah password role aplikasi di container test.
const testAppPassword = "PasswordAplikasiUji2026"

// connectAsApp membuka pool ke database test sebagai role aplikasi.
func connectAsApp(t *testing.T, password string) (*pgxpool.Pool, error) {
	t.Helper()
	u, err := url.Parse(testDatabaseURL)
	if err != nil {
		t.Fatalf("alamat database test: %v", err)
	}
	u.User = url.UserPassword(AppRole, password)
	pool, err := Connect(t.Context(), u.String())
	if err == nil {
		t.Cleanup(pool.Close)
	}
	return pool, err
}

// prepareAppRole menjalankan migrasi dan memasang password role aplikasi,
// lalu mengembalikan pool pemilik dan pool role aplikasi.
func prepareAppRole(t *testing.T) (owner, app *pgxpool.Pool) {
	t.Helper()
	owner = connectForTest(t)
	if err := Migrate(t.Context(), owner); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := SetAppRolePassword(t.Context(), owner, testAppPassword); err != nil {
		t.Fatalf("SetAppRolePassword: %v", err)
	}
	app, err := connectAsApp(t, testAppPassword)
	if err != nil {
		t.Fatalf("login sebagai %s: %v", AppRole, err)
	}
	return owner, app
}

// TestScramVerifier berjalan tanpa database: bentuk verifier sama dengan yang
// disimpan PostgreSQL, dan salt yang berbeda memberi verifier yang berbeda.
func TestScramVerifier(t *testing.T) {
	salt := []byte("salt-uji-16-byte")
	first, err := scramVerifier("PasswordAplikasiUji2026", salt, 4096)
	if err != nil {
		t.Fatal(err)
	}
	shape := regexp.MustCompile(`^SCRAM-SHA-256\$4096:[A-Za-z0-9+/]+=*\$[A-Za-z0-9+/]+=*:[A-Za-z0-9+/]+=*$`)
	if !shape.MatchString(first) {
		t.Errorf("bentuk verifier salah: %s", first)
	}
	again, _ := scramVerifier("PasswordAplikasiUji2026", salt, 4096)
	if again != first {
		t.Error("password dan salt yang sama memberi verifier berbeda")
	}
	other, _ := scramVerifier("PasswordAplikasiUji2026", []byte("salt-lain-16byte"), 4096)
	if other == first {
		t.Error("salt berbeda memberi verifier yang sama")
	}
}

// TestSetAppRolePassword membuktikan PostgreSQL menerima password role
// aplikasi yang dipasang lewat verifier, dan menolak password yang salah.
func TestSetAppRolePassword(t *testing.T) {
	owner := connectForTest(t)
	ctx := t.Context()
	if err := Migrate(ctx, owner); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Hanya huruf dan angka, minimal 16 karakter, supaya password sama persis
	// di semua tempat (tanpa aturan normalisasi SASLprep).
	for _, bad := range []string{"", "pendek123", "password dengan spasi", "Password-Aplikasi-2026"} {
		if err := SetAppRolePassword(ctx, owner, bad); err == nil {
			t.Errorf("password %q diterima, seharusnya ditolak", bad)
		}
	}

	if err := SetAppRolePassword(ctx, owner, testAppPassword); err != nil {
		t.Fatalf("SetAppRolePassword: %v", err)
	}
	if _, err := connectAsApp(t, testAppPassword); err != nil {
		t.Errorf("login dengan password yang benar gagal: %v", err)
	}
	if _, err := connectAsApp(t, "PasswordYangSalah2026"); err == nil {
		t.Error("login dengan password salah berhasil")
	}
}

// TestAppRolePrivileges membuktikan role aplikasi hanya punya hak yang
// dibutuhkan operasinya (E-06), termasuk tanpa UPDATE dan DELETE di audit_log.
func TestAppRolePrivileges(t *testing.T) {
	_, app := prepareAppRole(t)
	ctx := t.Context()

	tests := []struct {
		name    string
		sql     string
		allowed bool
	}{
		{"baca customers", "SELECT count(*) FROM customers", true},
		{"baca audit_log", "SELECT count(*) FROM audit_log", true},
		{"ubah sesi", "UPDATE sessions SET last_seen_at = last_seen_at WHERE false", true},
		{"hapus sesi", "DELETE FROM sessions WHERE false", true},
		{"ubah akun", "UPDATE accounts SET failed_logins = failed_logins WHERE false", true},
		// Akun tidak pernah dihapus (ASUMSI E-26).
		{"hapus akun", "DELETE FROM accounts WHERE false", false},
		// Katalog peran hanya dibaca sampai layar pengelolaannya ada (M8.1).
		{"ubah katalog peran", "UPDATE role_permissions SET role_id = role_id WHERE false", false},
		{"ubah audit_log", "UPDATE audit_log SET action = action WHERE false", false},
		{"hapus audit_log", "DELETE FROM audit_log WHERE false", false},
		{"kosongkan audit_log", "TRUNCATE audit_log", false},
		{"ubah catatan migrasi", "DELETE FROM goose_db_version WHERE false", false},
		{"buat tabel", "CREATE TABLE coba_tabel (id int)", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.Exec(ctx, tt.sql)
			if tt.allowed && err != nil {
				t.Errorf("ditolak, seharusnya boleh: %v", err)
			}
			if !tt.allowed && !isInsufficientPrivilege(err) {
				t.Errorf("ingin ditolak karena hak (42501), dapat: %v", err)
			}
		})
	}
}

// TestAuditLogRejectsChanges membuktikan trigger menolak UPDATE, DELETE, dan
// TRUNCATE di audit_log, juga dari pemilik tabel yang punya semua hak (R-SEC-03).
func TestAuditLogRejectsChanges(t *testing.T) {
	owner, _ := prepareAppRole(t)
	ctx := t.Context()

	_, err := owner.Exec(ctx, `INSERT INTO audit_log
		(seq, occurred_at, actor_type, action, details, prev_hash, hash)
		VALUES ((SELECT coalesce(max(seq), 0) + 1 FROM audit_log), now(), 'system', 'test.inserted', '{}',
		        repeat('0', 64), md5(random()::text) || md5(random()::text))`)
	if err != nil {
		t.Fatalf("menambah entri sebagai pemilik: %v", err)
	}

	for _, sql := range []string{
		"UPDATE audit_log SET action = 'test.changed'",
		"DELETE FROM audit_log",
		"TRUNCATE audit_log",
		// Statement tanpa baris yang cocok pun ditolak.
		"UPDATE audit_log SET action = action WHERE false",
	} {
		_, err := owner.Exec(ctx, sql)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "P0001" || pgErr.Where == "" {
			t.Errorf("%s: ingin ditolak trigger (P0001), dapat %v", sql, err)
		}
	}
}

// isInsufficientPrivilege melaporkan apakah err adalah penolakan hak (42501).
func isInsufficientPrivilege(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42501"
}
