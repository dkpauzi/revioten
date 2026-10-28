package db

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// testDatabaseURL diisi TestMain dengan alamat PostgreSQL sementara.
// Nilainya kosong bila test dijalankan dengan -short.
var testDatabaseURL string

// TestMain berjalan sekali sebelum semua test di package ini. Ia menyalakan
// satu container PostgreSQL 18.6 untuk semua test database, lalu membuangnya
// setelah semua test selesai.
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		// -short: test database dilewati, test lain tetap berjalan.
		os.Exit(m.Run())
	}

	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:18.6",
		postgres.WithDatabase("revioten_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal menyalakan PostgreSQL untuk test:", err)
		fmt.Fprintln(os.Stderr, "Pastikan Docker berjalan, atau pakai `go test -short ./...` untuk melewati test database.")
		_ = testcontainers.TerminateContainer(container)
		os.Exit(1)
	}

	testDatabaseURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal membaca alamat database test:", err)
		_ = testcontainers.TerminateContainer(container)
		os.Exit(1)
	}

	code := m.Run()

	if err := testcontainers.TerminateContainer(container); err != nil {
		fmt.Fprintln(os.Stderr, "gagal membuang container test:", err)
	}
	os.Exit(code)
}

// connectForTest membuka pool ke database test dan menutupnya otomatis
// setelah test selesai. Test dilewati bila dijalankan dengan -short.
func connectForTest(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("test database dilewati karena -short")
	}

	pool, err := Connect(t.Context(), testDatabaseURL)
	if err != nil {
		t.Fatalf("Connect ke database test: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// practiceTableSQL membuat tabel latihan. Tabel ini hanya ada di database
// test; tabel sungguhan dibuat lewat migrasi goose mulai M0.5.
const practiceTableSQL = `
CREATE TABLE IF NOT EXISTS practice_rows (
	id      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	content jsonb  NOT NULL
)`

// practiceContent adalah isi JSONB satu baris latihan.
type practiceContent struct {
	Process string `json:"process"`
	Point   string `json:"point"`
}

func createPracticeTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), practiceTableSQL); err != nil {
		t.Fatalf("membuat tabel latihan: %v", err)
	}
}

// countRowsWithPoint menghitung baris latihan dengan nilai "point" tertentu.
// Operator ->> mengambil satu nilai dari JSONB sebagai teks.
func countRowsWithPoint(t *testing.T, pool *pgxpool.Pool, point string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(t.Context(),
		"SELECT count(*) FROM practice_rows WHERE content->>'point' = $1", point,
	).Scan(&n)
	if err != nil {
		t.Fatalf("menghitung baris: %v", err)
	}
	return n
}

// TestConnect memeriksa bahwa Connect bisa menghubungi PostgreSQL 18.6.
func TestConnect(t *testing.T) {
	pool := connectForTest(t)

	var version string
	err := pool.QueryRow(t.Context(), "SHOW server_version").Scan(&version)
	if err != nil {
		t.Fatalf("membaca versi server: %v", err)
	}
	if !strings.HasPrefix(version, "18.6") {
		t.Errorf("versi server %q, seharusnya diawali 18.6", version)
	}
}

// TestConnectError memeriksa bahwa Connect gagal dengan alamat yang salah,
// dan pesan errornya tidak membocorkan password. Test ini tidak butuh Docker.
func TestConnectError(t *testing.T) {
	const password = "rahasia-uji"

	tests := []struct {
		name string
		url  string
	}{
		{name: "alamat tidak bisa dibaca", url: "postgres://test:" + password + "@127.0.0.1:bukan-port/x"},
		{name: "server tidak ada", url: "postgres://test:" + password + "@127.0.0.1:1/x?sslmode=disable&connect_timeout=5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := Connect(t.Context(), tt.url)
			if err == nil {
				pool.Close()
				t.Fatal("Connect berhasil, seharusnya error")
			}
			if strings.Contains(err.Error(), password) {
				t.Errorf("pesan error membocorkan password: %v", err)
			}
			// Terlihat dengan go test -v, supaya pesannya bisa dibaca.
			t.Logf("pesan error: %v", err)
		})
	}
}

// TestWithTxCommit: baris yang disimpan di dalam WithTx bisa dibaca di
// transaksi yang sama, dan tetap ada setelah COMMIT.
func TestWithTxCommit(t *testing.T) {
	pool := connectForTest(t)
	ctx := t.Context()
	createPracticeTable(t, pool)

	want := practiceContent{Process: "IQC", Point: "cek label supplier"}
	var got practiceContent

	err := WithTx(ctx, pool, func(tx pgx.Tx) error {
		var id int64
		err := tx.QueryRow(ctx,
			"INSERT INTO practice_rows (content) VALUES ($1) RETURNING id", want,
		).Scan(&id)
		if err != nil {
			return err
		}
		// Baris yang baru disimpan sudah bisa dibaca di transaksi yang sama.
		return tx.QueryRow(ctx,
			"SELECT content FROM practice_rows WHERE id = $1", id,
		).Scan(&got)
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}
	if got != want {
		t.Errorf("isi yang terbaca %+v, seharusnya %+v", got, want)
	}

	// Setelah COMMIT, baris bisa dibaca di luar transaksi.
	if n := countRowsWithPoint(t, pool, want.Point); n != 1 {
		t.Errorf("jumlah baris setelah COMMIT %d, seharusnya 1", n)
	}
}

// TestWithTxRollback: bila satu langkah di dalam WithTx gagal, semua
// perubahan di transaksi itu dibatalkan (ROLLBACK).
func TestWithTxRollback(t *testing.T) {
	pool := connectForTest(t)
	ctx := t.Context()
	createPracticeTable(t, pool)

	errSengaja := errors.New("gagal yang disengaja")
	point := "baris yang harus batal"

	err := WithTx(ctx, pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			"INSERT INTO practice_rows (content) VALUES ($1)",
			practiceContent{Process: "SMT", Point: point},
		)
		if err != nil {
			return err
		}
		// Langkah berikutnya gagal, jadi seluruh transaksi harus batal.
		return errSengaja
	})
	if !errors.Is(err, errSengaja) {
		t.Fatalf("error %v, seharusnya %v", err, errSengaja)
	}
	if n := countRowsWithPoint(t, pool, point); n != 0 {
		t.Errorf("jumlah baris setelah ROLLBACK %d, seharusnya 0", n)
	}
}
