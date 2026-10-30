package numbering

import (
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/problem"
)

// numberTexts mengambil teks nomor dari daftar, urut seperti daftarnya.
func numberTexts(list []DocumentNumber) []string {
	var texts []string
	for _, n := range list {
		texts = append(texts, n.Number)
	}
	return texts
}

// TestListNumbers: register bisa dicari dan disaring per customer, jenis
// dokumen, status, dan potongan teks nomor, terbaru dulu, per halaman (M2.3,
// UC-12). Semua akun bisa melihatnya (L-2).
func TestListNumbers(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	ab := f.newCustomer(t, "AB")
	for _, key := range []string{"k1", "k2", "k3"} {
		f.register(t, f.author, f.kt, key)
	}
	f.register(t, f.author, ab, "k4")
	if _, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyPFD].ID, IdempotencyKey: "k5"}); err != nil {
		t.Fatal(err)
	}
	viewer := newActor(t, f.pool, "viewer.uji", access.AccountWork)

	tests := []struct {
		name   string
		filter ListFilter
		want   []string
	}{
		{"semua, terbaru dulu", ListFilter{}, []string{"SH-KT-001", "SC-AB-001", "SC-KT-003", "SC-KT-002", "SC-KT-001"}},
		{"satu customer", ListFilter{CustomerID: ab.ID}, []string{"SC-AB-001"}},
		{"satu jenis", ListFilter{DocumentTypeID: f.types[KeyPFD].ID}, []string{"SH-KT-001"}},
		{"potongan teks tanpa membedakan huruf", ListFilter{Query: "sc-kt-00"}, []string{"SC-KT-003", "SC-KT-002", "SC-KT-001"}},
		{"tanda persen dibaca apa adanya", ListFilter{Query: "%"}, nil},
		{"status", ListFilter{Status: StatusVoid}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, _, err := f.s.ListNumbers(ctx, viewer, tt.filter)
			if err != nil {
				t.Fatalf("ListNumbers: %v", err)
			}
			if got := numberTexts(list); !slices.Equal(got, tt.want) {
				t.Errorf("hasil %v, ingin %v", got, tt.want)
			}
		})
	}

	// Per halaman: dua nomor, lalu dua berikutnya mulai dari ID terakhir.
	page1, next, err := f.s.ListNumbers(ctx, viewer, ListFilter{Limit: 2})
	if err != nil || next == "" || !slices.Equal(numberTexts(page1), []string{"SH-KT-001", "SC-AB-001"}) {
		t.Fatalf("halaman 1: %v, next %q, %v", numberTexts(page1), next, err)
	}
	page2, _, err := f.s.ListNumbers(ctx, viewer, ListFilter{Limit: 2, Before: next})
	if err != nil || !slices.Equal(numberTexts(page2), []string{"SC-KT-003", "SC-KT-002"}) {
		t.Errorf("halaman 2: %v, %v", numberTexts(page2), err)
	}

	_, _, err = f.s.ListNumbers(ctx, viewer, ListFilter{CustomerID: "bukan-uuid", Status: "hilang", Limit: 501})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"customer_id:invalid_format", "status:invalid_format", "limit:invalid_format"}) {
		t.Errorf("saringan salah: %v", got)
	}
}

// TestVoidAndObsolete: dokumen kontrol membatalkan nomor yang belum rilis
// dengan alasan, dan menandai nomor rilis sebagai Obsolete (M2.3, UC-13).
// Perubahan lain ditolak, dan setiap perubahan tercatat di audit log.
func TestVoidAndObsolete(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	control := newActor(t, f.pool, "dokumen.kontrol", access.AccountWork, access.RoleDocumentControl)
	reg, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: "k1"})
	if err != nil {
		t.Fatal(err)
	}
	id := reg.Numbers[0].ID

	if _, err := f.s.Void(ctx, f.author, id, "salah daftar"); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun membatalkan nomor: %v, ingin ErrForbidden", err)
	}
	if _, err := f.s.Void(ctx, control, id, " "); !slices.Equal(problemCodes(t, err), []string{"reason:required"}) {
		t.Errorf("Void tanpa alasan: %v", err)
	}
	if _, err := f.s.MakeObsolete(ctx, control, id, ""); !slices.Equal(problemCodes(t, err), []string{"status:not_allowed"}) {
		t.Errorf("Obsolete untuk nomor Reserved: %v", err)
	}
	voided, err := f.s.Void(ctx, control, id, " Salah pilih customer ")
	if err != nil || voided.Status != StatusVoid || voided.StatusReason != "Salah pilih customer" || voided.StatusChangedAt == nil {
		t.Fatalf("Void: %+v, %v", voided, err)
	}
	if _, err := f.s.Void(ctx, control, id, "lagi"); !slices.Equal(problemCodes(t, err), []string{"status:not_allowed"}) {
		t.Errorf("Void dua kali: %v", err)
	}
	var action, details string
	if err := f.pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details); err != nil {
		t.Fatal(err)
	}
	if action != audit.ActionDocumentNumberVoided || details != `{"number":"SC-KT-001","reason":"Salah pilih customer","status":"reserved → void"}` {
		t.Errorf("entri audit %s %s", action, details)
	}

	// Rilis belum ada di Tahap 2, jadi status Released dipasang langsung.
	released := f.register(t, f.author, f.kt, "k2")
	if _, err := f.pool.Exec(ctx, "UPDATE document_numbers SET status = 'released' WHERE number = $1", released); err != nil {
		t.Fatal(err)
	}
	list, _, err := f.s.ListNumbers(ctx, control, ListFilter{Status: StatusReleased})
	if err != nil || len(list) != 1 {
		t.Fatalf("nomor rilis: %v, %v", list, err)
	}
	obsolete, err := f.s.MakeObsolete(ctx, control, list[0].ID, "")
	if err != nil || obsolete.Status != StatusObsolete || obsolete.StatusReason != "" {
		t.Errorf("Obsolete: %+v, %v", obsolete, err)
	}
	if _, err := f.s.Void(ctx, control, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", "x"); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("nomor yang tidak ada: %v", err)
	}
}

// TestNumbersNeverDeletedOrReused adalah kriteria selesai M2.3: nomor tidak
// pernah dihapus atau dipakai ulang (R-NUM-04). Database menolak DELETE dan
// perubahan teks nomor dari role aplikasi, dan nomor yang dibatalkan tidak
// pernah diberikan lagi.
func TestNumbersNeverDeletedOrReused(t *testing.T) {
	f := newNumberFixture(t)
	ctx := t.Context()
	control := newActor(t, f.pool, "dokumen.kontrol", access.AccountWork, access.RoleDocumentControl)
	reg, err := f.s.Register(ctx, f.author, NewNumber{CustomerID: f.kt.ID, DocumentTypeID: f.types[KeyCP].ID, IdempotencyKey: "k1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Void(ctx, control, reg.Numbers[0].ID, "Tidak jadi dipakai"); err != nil {
		t.Fatal(err)
	}

	for name, sql := range map[string]string{
		"hapus nomor":      "DELETE FROM document_numbers WHERE false",
		"ganti teks nomor": "UPDATE document_numbers SET number = number WHERE false",
		"hapus penghitung": "DELETE FROM number_counters WHERE false",
	} {
		_, err := f.pool.Exec(ctx, sql)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("%s oleh role aplikasi: %v, ingin permission denied (42501)", name, err)
		}
	}

	if got := f.register(t, f.author, f.kt, "k2"); got != "SC-KT-002" {
		t.Errorf("nomor setelah SC-KT-001 dibatalkan: %q, ingin SC-KT-002", got)
	}
}
