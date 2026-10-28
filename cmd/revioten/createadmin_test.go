package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestReadPasswordFromPipe: bila input bukan terminal (misalnya dikirim lewat
// pipa oleh skrip E2E), satu baris dibaca apa adanya, tanpa akhiran baris baru.
func TestReadPasswordFromPipe(t *testing.T) {
	tests := map[string]string{
		"akhiran Linux":   "kuda terbang di atas sawah\n",
		"akhiran Windows": "kuda terbang di atas sawah\r\n",
		"tanpa akhiran":   "kuda terbang di atas sawah",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := readPassword(strings.NewReader(input), &out)
			if err != nil {
				t.Fatalf("readPassword: %v", err)
			}
			if got != "kuda terbang di atas sawah" {
				t.Errorf("password %q", got)
			}
			if strings.Contains(out.String(), got) {
				t.Error("password ikut ditulis ke layar")
			}
		})
	}

	if _, err := readPassword(strings.NewReader(""), &bytes.Buffer{}); err == nil {
		t.Error("input kosong harus ditolak")
	}
}

// TestCreateAdminFlags: -name dan -login wajib diisi, dan password tidak
// pernah diterima lewat flag (supaya tidak tersimpan di riwayat perintah).
func TestCreateAdminFlags(t *testing.T) {
	tests := map[string][]string{
		"tanpa -name":              {"-login", "admin.uji"},
		"tanpa -login":             {"-name", "Admin Uji"},
		"flag -password tidak ada": {"-name", "Admin Uji", "-login", "admin.uji", "-password", "rahasia"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			_, err := parseCreateAdminFlags(args, &out)
			if err == nil {
				t.Fatal("ingin error")
			}
			if strings.Contains(err.Error(), "rahasia") {
				t.Errorf("pesan error memuat password: %v", err)
			}
		})
	}

	opts, err := parseCreateAdminFlags([]string{"-name", "Admin Uji", "-login", "admin.uji"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parseCreateAdminFlags: %v", err)
	}
	if opts.name != "Admin Uji" || opts.login != "admin.uji" || opts.envFile != ".env" {
		t.Errorf("hasil %+v", opts)
	}
}
