// Package storage menyimpan berkas lampiran di folder privat (M5.2, R-SEC-04,
// G9). Nama berkas adalah SHA-256 isinya, jadi berkas yang sama hanya
// disimpan sekali dan isinya bisa dibuktikan tidak berubah. Folder ini tidak
// pernah dilayani langsung oleh web server; unduhan selalu lewat API yang
// memeriksa hak akses.
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// ErrTooLarge berarti berkas lebih besar dari batas.
var ErrTooLarge = errors.New("berkas terlalu besar")

// ErrEmpty berarti berkas kosong.
var ErrEmpty = errors.New("berkas kosong")

var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Store adalah folder penyimpanan berkas.
type Store struct {
	dir string
}

// New membuat penyimpanan di folder dir. Folder dibuat bila belum ada.
func New(dir string) (*Store, error) {
	if dir == "" {
		return nil, errors.New("folder penyimpanan lampiran kosong")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("membuat folder lampiran: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Put menyimpan isi r dan memberi SHA-256 serta ukurannya. Isi yang lebih
// dari maxBytes ditolak dengan ErrTooLarge dan tidak tersimpan.
func (s *Store) Put(r io.Reader, maxBytes int64) (string, int64, error) {
	tmp, err := os.CreateTemp(s.dir, "upload-*")
	if err != nil {
		return "", 0, fmt.Errorf("membuat berkas sementara: %w", err)
	}
	defer os.Remove(tmp.Name()) // tidak berpengaruh bila sudah dipindah
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(r, maxBytes+1))
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", 0, fmt.Errorf("menyimpan berkas: %w", err)
	}
	if n == 0 {
		return "", 0, ErrEmpty
	}
	if n > maxBytes {
		return "", 0, ErrTooLarge
	}
	sum := hex.EncodeToString(h.Sum(nil))
	final := s.path(sum)
	if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
		return "", 0, err
	}
	if _, err := os.Stat(final); err == nil {
		return sum, n, nil // isi yang sama sudah tersimpan
	}
	if err := os.Rename(tmp.Name(), final); err != nil {
		return "", 0, fmt.Errorf("memindahkan berkas: %w", err)
	}
	return sum, n, nil
}

// Open membuka berkas menurut SHA-256-nya.
func (s *Store) Open(sum string) (*os.File, error) {
	if !hashPattern.MatchString(sum) {
		return nil, os.ErrNotExist
	}
	return os.Open(s.path(sum))
}

// path memberi letak berkas: dua huruf pertama hash menjadi subfolder,
// supaya satu folder tidak berisi terlalu banyak berkas.
func (s *Store) path(sum string) string {
	return filepath.Join(s.dir, sum[:2], sum)
}
