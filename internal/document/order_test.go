package document

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
)

// M4.6 · Urutan dan nomor operasi (R-CMP-01): baris diurutkan ulang, nomor
// operasi diisi otomatis per langkah proses, dan nomor yang diketik tetap.

// operations memberi "posisi:nomor operasi" setiap baris, untuk dibandingkan.
func operations(d Document) []string {
	var out []string
	for _, r := range d.Rows {
		out = append(out, r.OperationNumber)
	}
	return out
}

func TestReorder(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	second, err := f.q.CreateProcess(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.q.AddProcessName(ctx, db.AddProcessNameParams{ProcessID: second, Kind: "display", Name: "Reflow"}); err != nil {
		t.Fatal(err)
	}
	d, err := f.s.Create(ctx, f.author, NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k125}})
	if err != nil {
		t.Fatal(err)
	}
	// A dan B: SMT Printing. C: Reflow. D: tanpa proses. E: SMT Printing lagi.
	for _, p := range []string{f.process, f.process, second, "", f.process} {
		if d, err = f.s.AddRow(ctx, f.author, d.ID, NewRow{ProcessID: p}); err != nil {
			t.Fatal(err)
		}
	}
	a, b, c, dd, e := d.Rows[0].ID, d.Rows[1].ID, d.Rows[2].ID, d.Rows[3].ID, d.Rows[4].ID

	if _, err := f.s.Reorder(ctx, f.viewer, d.ID, Order{RowIDs: []string{a, b, c, dd, e}}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer mengurutkan: %v", err)
	}
	// Reflow pindah ke depan; nomor operasi diisi per langkah: baris berurutan
	// dengan proses yang sama mendapat satu nomor, kelipatan 10.
	got, err := f.s.Reorder(ctx, f.author, d.ID, Order{RowIDs: []string{c, a, b, dd, e}, Renumber: true})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for i, r := range got.Rows {
		ids = append(ids, r.ID)
		if r.Position != i+1 {
			t.Errorf("posisi baris %d: %d", i+1, r.Position)
		}
	}
	if !slices.Equal(ids, []string{c, a, b, dd, e}) || !slices.Equal(operations(got), []string{"10", "20", "20", "30", "40"}) {
		t.Errorf("sesudah urut: %v, nomor %v", ids, operations(got))
	}
	// Nomor yang berubah menaikkan versi baris, supaya layar lama tidak menimpanya (M4.3).
	if got.Rows[0].Version != 2 || got.Rows[0].OperationManual {
		t.Errorf("baris pertama: %+v", got.Rows[0])
	}

	// Nomor yang diketik penyusun ditandai manual dan tidak ditimpa.
	last := got.Rows[4]
	if got, err = f.s.UpdateRow(ctx, f.author, d.ID, e, RowUpdate{ProcessID: last.ProcessID, OperationNumber: "45", Version: last.Version}); err != nil {
		t.Fatal(err)
	}
	if !got.Rows[4].OperationManual {
		t.Errorf("nomor yang diketik belum manual: %+v", got.Rows[4])
	}
	// Diurutkan tanpa penomoran: nomor tidak berubah.
	if got, err = f.s.Reorder(ctx, f.author, d.ID, Order{RowIDs: []string{a, b, c, dd, e}}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(operations(got), []string{"20", "20", "10", "30", "45"}) {
		t.Errorf("urut tanpa penomoran: %v", operations(got))
	}
	if got, err = f.s.Reorder(ctx, f.author, d.ID, Order{RowIDs: []string{a, b, c, dd, e}, Renumber: true}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(operations(got), []string{"10", "10", "20", "30", "45"}) {
		t.Errorf("penomoran dengan nomor manual: %v", operations(got))
	}
	// Nomor manual yang dikosongkan kembali otomatis.
	last = got.Rows[4]
	if got, err = f.s.UpdateRow(ctx, f.author, d.ID, e, RowUpdate{ProcessID: last.ProcessID, Version: last.Version}); err != nil {
		t.Fatal(err)
	}
	if got, err = f.s.Reorder(ctx, f.author, d.ID, Order{RowIDs: []string{a, b, c, dd, e}, Renumber: true}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(operations(got), []string{"10", "10", "20", "30", "40"}) || got.Rows[4].OperationManual {
		t.Errorf("sesudah nomor manual dikosongkan: %v", operations(got))
	}

	// Daftar harus memuat setiap baris dokumen tepat satu kali.
	for name, rows := range map[string][]string{
		"kurang":  {a, b, c, dd},
		"kembar":  {a, a, c, dd, e},
		"asing":   {a, b, c, dd, "01900000-0000-7000-8000-000000000000"},
		"kosong":  nil,
		"bukanid": {a, b, c, dd, "x"},
	} {
		if _, err := f.s.Reorder(ctx, f.author, d.ID, Order{RowIDs: rows}); !slices.Contains(codes(t, err), "row_ids:invalid_format") {
			t.Errorf("%s: %v", name, err)
		}
	}
	var action, details string
	if err := f.pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details); err != nil {
		t.Fatal(err)
	}
	if action != "document.rows_reordered" || details != `{"count":"5","renumbered":"true"}` {
		t.Errorf("audit: %s %s", action, details)
	}
}

// TestTypedOperationIsManual: nomor operasi yang diketik atau ditempel saat
// baris dibuat juga dianggap manual.
func TestTypedOperationIsManual(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	d, err := f.s.Create(ctx, f.author, NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k125}})
	if err != nil {
		t.Fatal(err)
	}
	if d, err = f.s.AddRows(ctx, f.author, d.ID, []NewRow{{OperationNumber: "5"}, {}}); err != nil {
		t.Fatal(err)
	}
	if !d.Rows[0].OperationManual || d.Rows[1].OperationManual {
		t.Errorf("tanda manual: %+v", d.Rows)
	}
	d, err = f.s.Reorder(ctx, f.author, d.ID, Order{RowIDs: []string{d.Rows[0].ID, d.Rows[1].ID}, Renumber: true})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(operations(d), []string{"5", "20"}) {
		t.Errorf("nomor: %v", operations(d))
	}
}
