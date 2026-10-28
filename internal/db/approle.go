package db

import (
	"context"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AppRole adalah role database yang dipakai server untuk semua operasi setelah
// migrasi (E-06). Role ini dibuat migrasi 00004 dengan hak sekecil mungkin per
// tabel: audit_log hanya bisa dibaca dan ditambah.
const AppRole = "revioten_app"

// appPasswordPattern: hanya huruf dan angka, 16 sampai 128 karakter. Dengan
// huruf dan angka saja, password sama persis di .env, di URL koneksi, dan di
// PostgreSQL (tanpa aturan normalisasi SASLprep untuk karakter khusus).
var appPasswordPattern = regexp.MustCompile(`^[A-Za-z0-9]{16,128}$`)

// scramIterations adalah jumlah putaran PBKDF2, sama dengan bawaan PostgreSQL.
const scramIterations = 4096

// SetAppRolePassword memasang password role aplikasi. Dipanggil dengan pool
// milik pemilik database, setelah migrasi.
//
// Yang dikirim ke PostgreSQL adalah verifier SCRAM-SHA-256, bukan password-nya,
// sama seperti perintah \password di psql. Dengan begitu password tidak pernah
// sampai ke server database, juga tidak ke log-nya bila perintah ini gagal (E-18).
func SetAppRolePassword(ctx context.Context, owner *pgxpool.Pool, password string) error {
	if !appPasswordPattern.MatchString(password) {
		// Isi password tidak disebut (E-18).
		return errors.New("APP_DB_PASSWORD harus berisi huruf dan angka saja, 16 sampai 128 karakter")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("membuat salt: %w", err)
	}
	verifier, err := scramVerifier(password, salt, scramIterations)
	if err != nil {
		return err
	}

	// ALTER ROLE tidak menerima parameter $1, jadi perintahnya disusun
	// PostgreSQL sendiri dengan format(): %I mengutip nama, %L mengutip nilai.
	var statement string
	err = owner.QueryRow(ctx, "SELECT format('ALTER ROLE %I PASSWORD %L', $1::text, $2::text)",
		AppRole, verifier).Scan(&statement)
	if err != nil {
		return fmt.Errorf("menyusun perintah password role %s: %w", AppRole, err)
	}
	if _, err := owner.Exec(ctx, statement); err != nil {
		return fmt.Errorf("memasang password role %s: %w", AppRole, err)
	}
	return nil
}

// scramVerifier menyusun verifier SCRAM-SHA-256 (RFC 5802 dan RFC 7677) dalam
// bentuk yang disimpan PostgreSQL:
//
//	SCRAM-SHA-256$<putaran>:<salt>$<StoredKey>:<ServerKey>
//
// PostgreSQL hanya menyimpan StoredKey dan ServerKey. Saat login, klien
// membuktikan tahu password tanpa mengirim password-nya.
func scramVerifier(password string, salt []byte, iterations int) (string, error) {
	salted, err := pbkdf2.Key(sha256.New, password, salt, iterations, sha256.Size)
	if err != nil {
		return "", fmt.Errorf("menghitung PBKDF2: %w", err)
	}
	clientKey := hmacSHA256(salted, "Client Key")
	storedKey := sha256.Sum256(clientKey)
	serverKey := hmacSHA256(salted, "Server Key")

	b64 := base64.StdEncoding.EncodeToString
	return fmt.Sprintf("SCRAM-SHA-256$%d:%s$%s:%s",
		iterations, b64(salt), b64(storedKey[:]), b64(serverKey)), nil
}

func hmacSHA256(key []byte, message string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return mac.Sum(nil)
}
