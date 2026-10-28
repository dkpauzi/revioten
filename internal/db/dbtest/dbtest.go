// Package dbtest menyediakan PostgreSQL sungguhan untuk test paket lain,
// lewat testcontainers-go. Paket ini hanya dipakai oleh file _test.go,
// sehingga tidak ikut masuk ke binary program.
//
// Cara pakai di satu paket:
//
//	func TestMain(m *testing.M) { os.Exit(dbtest.Main(m)) }
//
//	func TestSesuatu(t *testing.T) {
//		pool := dbtest.NewPool(t) // database baru yang kosong, skema terbaru
//		...
//	}
//
// Pool dari NewPool login sebagai role aplikasi (db.AppRole), dengan hak yang
// sama dengan server (E-06). Test yang lupa diberi GRANT di migrasi langsung
// gagal di sini, bukan baru ketahuan saat server berjalan.
package dbtest

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/dkpauzi/revioten/internal/db"
)

// templateDB adalah database yang sudah dimigrasi sekali. Setiap test mendapat
// salinannya lewat CREATE DATABASE ... TEMPLATE, yang jauh lebih cepat daripada
// menjalankan migrasi berulang kali.
const templateDB = "revioten_template"

// appPassword adalah password role aplikasi di container test.
const appPassword = "PasswordAplikasiUji2026"

var (
	// serverURL adalah alamat container, diisi Main. Kosong bila -short.
	serverURL string
	// counter membuat nama database test yang berbeda-beda.
	counter atomic.Int64
)

// Main menyalakan satu container PostgreSQL 18.6, menyiapkan database template
// yang sudah dimigrasi, menjalankan semua test di paket itu, lalu membuang
// container. Nilai kembaliannya diteruskan ke os.Exit.
func Main(m *testing.M) int {
	flag.Parse()
	if testing.Short() {
		// -short: test database dilewati, test lain tetap berjalan.
		return m.Run()
	}

	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:18.6",
		postgres.WithDatabase(templateDB),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	// Container dibuang setelah semua test selesai, juga bila ada kegagalan.
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			fmt.Fprintln(os.Stderr, "gagal membuang container test:", err)
		}
	}()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal menyalakan PostgreSQL untuk test:", err)
		fmt.Fprintln(os.Stderr, "Pastikan Docker berjalan, atau pakai `go test -short ./...` untuk melewati test database.")
		return 1
	}

	serverURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal membaca alamat database test:", err)
		return 1
	}
	if err := migrateTemplate(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "gagal menyiapkan database template:", err)
		return 1
	}
	return m.Run()
}

// migrateTemplate menjalankan migrasi sekali di database template sebagai
// pemilik, memasang password role aplikasi, lalu menutup semua koneksinya.
// CREATE DATABASE ... TEMPLATE menolak template yang masih punya koneksi terbuka.
func migrateTemplate(ctx context.Context) error {
	pool, err := db.Connect(ctx, ownerURL(templateDB))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	// Role berlaku untuk seluruh server PostgreSQL di container, jadi
	// password ini cukup dipasang sekali untuk semua database test.
	return db.SetAppRolePassword(ctx, pool, appPassword)
}

// NewPool membuat database baru dari template khusus untuk satu test, lalu
// membuka pool ke database itu sebagai role aplikasi. Pool ditutup otomatis
// setelah test selesai. Test dilewati bila dijalankan dengan -short.
func NewPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	app, _ := NewPools(t)
	return app
}

// NewPools seperti NewPool, tapi juga memberi pool pemilik database (tanpa
// batas hak role aplikasi). Pool pemilik dipakai test yang meniru admin
// database, misalnya percobaan mengubah audit log (M1.3).
func NewPools(t *testing.T) (app, owner *pgxpool.Pool) {
	t.Helper()
	if testing.Short() {
		t.Skip("test database dilewati karena -short")
	}
	ctx := t.Context()

	name := fmt.Sprintf("test_%d", counter.Add(1))
	// CREATE DATABASE dijalankan dari database bawaan "postgres".
	admin, err := pgx.Connect(ctx, ownerURL("postgres"))
	if err != nil {
		t.Fatalf("menghubungi database postgres: %v", err)
	}
	defer admin.Close(ctx)
	// Nama database tidak bisa dikirim sebagai parameter $1, jadi nama
	// diamankan dengan Sanitize (tanda kutip ganda) sebelum disusun ke SQL.
	sql := "CREATE DATABASE " + pgx.Identifier{name}.Sanitize() +
		" TEMPLATE " + pgx.Identifier{templateDB}.Sanitize()
	if _, err := admin.Exec(ctx, sql); err != nil {
		t.Fatalf("membuat database %s: %v", name, err)
	}

	app, err = db.Connect(ctx, appURL(name))
	if err != nil {
		t.Fatalf("Connect ke database %s sebagai %s: %v", name, db.AppRole, err)
	}
	t.Cleanup(app.Close)
	owner, err = db.Connect(ctx, ownerURL(name))
	if err != nil {
		t.Fatalf("Connect ke database %s sebagai pemilik: %v", name, err)
	}
	t.Cleanup(owner.Close)
	return app, owner
}

// ownerURL mengganti nama database di alamat container. User-nya pemilik
// database (superuser di container test).
func ownerURL(name string) string {
	u, err := url.Parse(serverURL)
	if err != nil {
		// Alamat dibuat testcontainers sendiri, jadi ini hanya terjadi bila ada bug.
		panic(fmt.Sprintf("alamat container tidak bisa dibaca: %v", err))
	}
	u.Path = "/" + name
	return u.String()
}

// appURL seperti ownerURL, tapi login sebagai role aplikasi.
func appURL(name string) string {
	u, _ := url.Parse(ownerURL(name))
	u.User = url.UserPassword(db.AppRole, appPassword)
	return u.String()
}
