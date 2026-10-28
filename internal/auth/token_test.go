package auth

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestNewToken(t *testing.T) {
	token, hash, err := newToken()
	if err != nil {
		t.Fatalf("newToken: %v", err)
	}
	// 32 byte acak (256 bit, ASVS 7.2.3 minta minimal 128 bit) ditulis base64url
	// tanpa padding, jadi panjangnya 43 karakter.
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 {
		t.Errorf("token %q bukan 32 byte base64url: %v", token, err)
	}
	if !bytes.Equal(hash, hashToken(token)) || len(hash) != 32 {
		t.Errorf("hash token tidak sama dengan hashToken(token)")
	}

	other, _, err := newToken()
	if err != nil {
		t.Fatalf("newToken kedua: %v", err)
	}
	if other == token {
		t.Error("dua token berturut-turut sama")
	}
}
