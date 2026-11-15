package storage

// Pemindaian antivirus untuk unggahan (M8.2, R-SEC-04).
//
// Berkas yang diunggah dipindai sebelum masuk ke folder penyimpanan. Pemindai
// bawaannya ClamAV, lewat daemon clamd dan perintah INSTREAM: isi berkas
// dikirim lewat TCP dalam potongan, dan clamd menjawab "stream: OK" atau
// "stream: <nama> FOUND".
//
// Bila pemindai dipasang tetapi tidak bisa dihubungi, unggahan ditolak
// (gagal tertutup): berkas yang belum dipindai tidak pernah tersimpan.

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// ErrInfected berarti pemindai menemukan malware di berkas.
var ErrInfected = errors.New("berkas terdeteksi berbahaya")

// ErrScanUnavailable berarti pemindai tidak bisa dihubungi atau jawabannya tidak dikenal.
var ErrScanUnavailable = errors.New("pemindai antivirus tidak bisa dihubungi")

// Scanner memindai isi satu berkas. Hasil nil berarti bersih.
type Scanner interface {
	Scan(ctx context.Context, r io.Reader) error
}

// ClamAV adalah pemindai lewat daemon clamd di alamat Addr, misalnya 127.0.0.1:3310.
type ClamAV struct {
	Addr    string
	Timeout time.Duration // batas waktu satu pemindaian; nol berarti 60 detik
}

// chunkSize adalah ukuran satu potongan INSTREAM. Harus di bawah
// StreamMaxLength clamd (bawaannya 25 MB).
const chunkSize = 64 << 10

// Scan mengirim isi r ke clamd dan membaca jawabannya.
func (c ClamAV) Scan(ctx context.Context, r io.Reader) error {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.Addr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrScanUnavailable, err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	// Awalan z berarti perintah dan jawaban diakhiri byte nol.
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return fmt.Errorf("%w: %v", ErrScanUnavailable, err)
	}
	buf := make([]byte, chunkSize)
	size := make([]byte, 4)
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			// Setiap potongan diawali panjangnya: 4 byte, big-endian.
			binary.BigEndian.PutUint32(size, uint32(n))
			if _, err := conn.Write(size); err != nil {
				return fmt.Errorf("%w: %v", ErrScanUnavailable, err)
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return fmt.Errorf("%w: %v", ErrScanUnavailable, err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	// Potongan berpanjang nol menutup kiriman.
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return fmt.Errorf("%w: %v", ErrScanUnavailable, err)
	}
	reply, err := io.ReadAll(io.LimitReader(conn, 4096))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrScanUnavailable, err)
	}
	answer := strings.TrimSpace(strings.TrimRight(string(reply), "\x00"))
	switch {
	case strings.HasSuffix(answer, "OK"):
		return nil
	case strings.HasSuffix(answer, "FOUND"):
		return fmt.Errorf("%w: %s", ErrInfected, strings.TrimSuffix(strings.TrimPrefix(answer, "stream: "), " FOUND"))
	default:
		return fmt.Errorf("%w: jawaban %q", ErrScanUnavailable, answer)
	}
}
