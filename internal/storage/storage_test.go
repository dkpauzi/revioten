package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// TestStore: berkas disimpan dengan nama SHA-256 isinya, isi yang sama
// disimpan sekali, dan berkas yang terlalu besar atau kosong ditolak.
func TestStore(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	content := "8D report for claim 2026-031"
	sum, n, err := s.Put(strings.NewReader(content), 1024)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256([]byte(content))
	if sum != hex.EncodeToString(want[:]) || n != int64(len(content)) {
		t.Errorf("hash %s, ukuran %d", sum, n)
	}
	again, _, err := s.Put(strings.NewReader(content), 1024)
	if err != nil || again != sum {
		t.Errorf("isi sama: %s, %v", again, err)
	}
	f, err := s.Open(sum)
	if err != nil {
		t.Fatal(err)
	}
	read, _ := io.ReadAll(f)
	f.Close()
	if string(read) != content {
		t.Errorf("isi dibaca: %q", read)
	}

	if _, _, err := s.Put(strings.NewReader(strings.Repeat("x", 11)), 10); !errors.Is(err, ErrTooLarge) {
		t.Errorf("terlalu besar: %v", err)
	}
	if _, _, err := s.Put(strings.NewReader(""), 10); !errors.Is(err, ErrEmpty) {
		t.Errorf("kosong: %v", err)
	}
	// Nama yang bukan hash tidak pernah dibuka, jadi tidak bisa keluar folder.
	if _, err := s.Open("../../etc/passwd"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("nama aneh dibuka: %v", err)
	}
	// Berkas sementara tidak tertinggal.
	entries, _ := os.ReadDir(s.dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "upload-") {
			t.Errorf("berkas sementara tertinggal: %s", e.Name())
		}
	}
}
