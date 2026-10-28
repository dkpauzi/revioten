package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// newToken membuat token sesi: 32 byte acak dari crypto/rand (256 bit; ASVS
// 7.2.3 meminta minimal 128 bit), ditulis base64url tanpa padding supaya aman
// dipakai di cookie. Yang disimpan di database hanya hash-nya.
func newToken() (token string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("membuat token sesi: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, hashToken(token), nil
}

// hashToken menghitung SHA-256 dari token. SHA-256 cukup di sini (tidak perlu
// argon2id), karena token berisi 256 bit acak dan tidak bisa ditebak dari kamus.
func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
