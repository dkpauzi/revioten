package db

// Subjek row-level security (M8.1, R-SEC-02, RK-14).
//
// Pemeriksaan izin di aplikasi (internal/access) adalah pagar pertama. Row-level
// security (RLS) di PostgreSQL adalah pagar kedua: database sendiri menyaring
// baris dokumen menurut akun yang sedang dilayani, jadi query yang lupa
// memeriksa izin tetap tidak membocorkan customer lain.
//
// Database perlu tahu akun mana yang sedang dilayani. Caranya: server menaruh
// ID akun di context (WithSubject), dan setiap kali satu koneksi diambil dari
// pool, ID itu ditulis ke variabel sesi app.account_id (prepareConn). Kebijakan
// RLS di migrasi 00042 membaca variabel itu.
//
// Tanpa subjek (perintah CLI, migrasi, job, dan test layanan), variabel itu
// kosong dan RLS tidak menyaring: di jalur itu hanya pemeriksaan izin yang
// berlaku (ASUMSI E-118).

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type subjectKey struct{}

// WithSubject menandai ctx sebagai permintaan milik akun accountID. Semua
// query yang memakai ctx itu disaring RLS untuk akun tersebut.
func WithSubject(ctx context.Context, accountID string) context.Context {
	return context.WithValue(ctx, subjectKey{}, accountID)
}

// AsSystem melepas subjek dari ctx, sehingga RLS tidak menyaring. Hanya untuk
// operasi yang memang lintas customer dan sudah lolos pemeriksaan izin,
// misalnya mencari target yokoten di Project Group lain (M5.9).
func AsSystem(ctx context.Context) context.Context {
	return context.WithValue(ctx, subjectKey{}, "")
}

// Subject memberi ID akun subjek ctx, atau "" bila tidak ada.
func Subject(ctx context.Context) string {
	id, _ := ctx.Value(subjectKey{}).(string)
	return id
}

// subjectData adalah kunci catatan subjek terakhir di satu koneksi.
const subjectData = "revioten.subject"

// prepareConn dipanggil pool setiap kali koneksi akan dipinjamkan. Ia menulis
// subjek ctx ke variabel sesi app.account_id. Nilai terakhir dicatat di
// koneksi, supaya perintah SET hanya dikirim bila subjeknya berganti.
func prepareConn(ctx context.Context, conn *pgx.Conn) (bool, error) {
	want := Subject(ctx)
	data := conn.PgConn().CustomData()
	if last, ok := data[subjectData].(string); ok && last == want {
		return true, nil
	}
	if _, err := conn.Exec(ctx, "SELECT set_config('app.account_id', $1, false)", want); err != nil {
		// Koneksi dibuang: lebih baik gagal daripada memakai subjek lama.
		return false, err
	}
	data[subjectData] = want
	return true, nil
}
