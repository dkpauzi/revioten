package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argonParams adalah parameter argon2id.
type argonParams struct {
	memory     uint32 // KiB
	iterations uint32
	threads    uint8
	saltLength uint32 // byte
	keyLength  uint32 // byte
}

// defaultParams mengikuti konfigurasi minimum OWASP Password Storage Cheat
// Sheet (dicek 28 Oktober 2026): m=19456 (19 MiB), t=2, p=1. Salt 16 byte dan
// hash 32 byte. E-08 (D-85).
var defaultParams = argonParams{memory: 19 * 1024, iterations: 2, threads: 1, saltLength: 16, keyLength: 32}

// errBrokenHash berarti teks hash di database tidak bisa dibaca.
var errBrokenHash = errors.New("hash password tidak bisa dibaca")

// hashPassword membuat hash argon2id dengan salt acak, dalam format PHC:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<salt base64>$<hash base64>
//
// Parameter ikut tersimpan, jadi hash lama tetap bisa diperiksa walau
// parameternya nanti dinaikkan.
func hashPassword(password string) (string, error) {
	return hashWithParams(password, defaultParams)
}

func hashWithParams(password string, p argonParams) (string, error) {
	salt := make([]byte, p.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("membuat salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.threads, p.keyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memory, p.iterations, p.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// verifyPassword memeriksa password terhadap hash tersimpan. needsRehash true
// bila hash memakai parameter lama, supaya bisa dibuat ulang setelah login berhasil.
func verifyPassword(password, encoded string) (match, needsRehash bool, err error) {
	p, salt, key, err := decodeHash(encoded)
	if err != nil {
		return false, false, err
	}
	other := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.threads, p.keyLength)
	// ConstantTimeCompare butuh waktu yang sama walau byte pertama sudah beda,
	// jadi lama proses tidak membocorkan seberapa mirip password-nya.
	if subtle.ConstantTimeCompare(key, other) != 1 {
		return false, false, nil
	}
	return true, p != defaultParams, nil
}

// decodeHash membaca teks hash format PHC.
func decodeHash(encoded string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// parts[0] kosong karena teks diawali $.
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, errBrokenHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argonParams{}, nil, nil, errBrokenHash
	}
	var p argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.threads); err != nil {
		return argonParams{}, nil, nil, errBrokenHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, errBrokenHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return argonParams{}, nil, nil, errBrokenHash
	}
	p.saltLength = uint32(len(salt))
	p.keyLength = uint32(len(key))
	return p, salt, key, nil
}
