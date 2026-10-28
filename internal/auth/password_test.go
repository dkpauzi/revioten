package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	const password = "kuda terbang di atas sawah"

	encoded, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	// Format PHC: algoritma, versi, parameter, salt, dan hash dalam satu teks.
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("format hash %q tidak sesuai parameter OWASP", encoded)
	}
	if strings.Contains(encoded, password) {
		t.Error("password asli ikut tersimpan di hash")
	}

	ok, rehash, err := verifyPassword(password, encoded)
	if err != nil || !ok || rehash {
		t.Errorf("verifyPassword benar = %v, %v, %v; ingin true, false, nil", ok, rehash, err)
	}
	// Password dibandingkan apa adanya, termasuk huruf besar dan kecil (ASVS 6.2.8).
	if ok, _, _ := verifyPassword("Kuda terbang di atas sawah", encoded); ok {
		t.Error("password dengan huruf berbeda dianggap benar")
	}

	// Salt acak membuat dua hash dari password yang sama berbeda.
	again, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword kedua: %v", err)
	}
	if again == encoded {
		t.Error("dua hash dari password yang sama identik, salt tidak acak")
	}
}

func TestVerifyPasswordBrokenHash(t *testing.T) {
	broken := []string{
		"",
		"bukan hash",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaA",
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaA",
		"$argon2id$v=19$m=x,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$bukan-base64!$aGFzaA",
	}
	for _, encoded := range broken {
		ok, _, err := verifyPassword("password apa saja", encoded)
		if err == nil || ok {
			t.Errorf("hash rusak %q: ok=%v err=%v, ingin false dan error", encoded, ok, err)
		}
	}
}

// TestVerifyPasswordOldParams: hash lama dengan parameter yang lebih lemah tetap
// bisa diperiksa, dan ditandai perlu dibuat ulang dengan parameter sekarang.
func TestVerifyPasswordOldParams(t *testing.T) {
	weaker := argonParams{memory: 8 * 1024, iterations: 1, threads: 1, saltLength: 16, keyLength: 32}
	encoded, err := hashWithParams("kuda terbang di atas sawah", weaker)
	if err != nil {
		t.Fatalf("hashWithParams: %v", err)
	}
	ok, rehash, err := verifyPassword("kuda terbang di atas sawah", encoded)
	if err != nil || !ok || !rehash {
		t.Errorf("verifyPassword = %v, %v, %v; ingin true, true, nil", ok, rehash, err)
	}
}
