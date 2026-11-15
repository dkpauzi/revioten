package storage

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
)

// fakeClamd meniru daemon clamd: membaca satu kiriman INSTREAM, lalu menjawab
// FOUND bila isinya memuat kata "EICAR", selain itu OK.
func fakeClamd(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				command := make([]byte, len("zINSTREAM\x00"))
				if _, err := io.ReadFull(conn, command); err != nil || string(command) != "zINSTREAM\x00" {
					return
				}
				var content []byte
				for {
					size := make([]byte, 4)
					if _, err := io.ReadFull(conn, size); err != nil {
						return
					}
					n := binary.BigEndian.Uint32(size)
					if n == 0 {
						break
					}
					chunk := make([]byte, n)
					if _, err := io.ReadFull(conn, chunk); err != nil {
						return
					}
					content = append(content, chunk...)
				}
				if strings.Contains(string(content), "EICAR") {
					conn.Write([]byte("stream: Eicar-Test-Signature FOUND\x00"))
					return
				}
				conn.Write([]byte("stream: OK\x00"))
			}()
		}
	}()
	return ln.Addr().String()
}

// M8.2 · Unggahan dipindai sebelum disimpan (R-SEC-04): berkas bersih
// tersimpan, berkas berbahaya ditolak dan tidak tersimpan, dan pemindai yang
// mati membuat unggahan ditolak.
func TestScanOnPut(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	s = s.WithScanner(ClamAV{Addr: fakeClamd(t)})

	sum, _, err := s.PutScanned(ctx, strings.NewReader(strings.Repeat("laporan 8D ", 20000)), 1<<20)
	if err != nil {
		t.Fatalf("berkas bersih: %v", err)
	}
	if f, err := s.Open(sum); err != nil {
		t.Errorf("berkas bersih tidak tersimpan: %v", err)
	} else {
		f.Close()
	}

	if _, _, err := s.PutScanned(ctx, strings.NewReader("X5O!P%@AP EICAR test"), 1<<20); !errors.Is(err, ErrInfected) || !strings.Contains(err.Error(), "Eicar-Test-Signature") {
		t.Errorf("berkas berbahaya: %v", err)
	}
	// Tidak ada berkas sementara atau berkas berbahaya yang tertinggal: hanya subfolder berkas bersih.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || !entries[0].IsDir() {
		t.Errorf("isi folder: %v", entries)
	}

	dead := s.WithScanner(ClamAV{Addr: "127.0.0.1:1"})
	if _, _, err := dead.PutScanned(ctx, strings.NewReader("isi"), 1<<20); !errors.Is(err, ErrScanUnavailable) {
		t.Errorf("pemindai mati: %v", err)
	}
	// Tanpa pemindai, berkas disimpan seperti biasa.
	plain, _ := New(t.TempDir())
	if _, _, err := plain.PutScanned(ctx, strings.NewReader("EICAR"), 1<<20); err != nil {
		t.Errorf("tanpa pemindai: %v", err)
	}
}
