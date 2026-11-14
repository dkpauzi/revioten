package document

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/template"
)

// M4.7 · Salin komposisi model lain (R-CMP-05): dokumen model baru dimulai
// dari baris dokumen model serupa, lalu menjadi dokumen mandiri.

// newSetFor membuat set PFD, PFMEA, dan CP untuk satu model customer AB.
func (f fixture) newSetFor(t *testing.T, key, modelID string) map[string]Document {
	t.Helper()
	ctx := t.Context()
	reg, err := f.numbers.RegisterSet(ctx, f.author, numbering.NewSet{CustomerID: f.custAB, IdempotencyKey: key + "-" + t.Name()})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Document{}
	for _, n := range reg.Numbers {
		if n.Status != "reserved" {
			continue
		}
		d, err := f.s.Create(ctx, f.author, NewDocument{NumberID: n.ID, ModelIDs: []string{modelID}})
		if err != nil {
			t.Fatal(err)
		}
		out[d.DocumentType.Key] = d
	}
	return out
}

func TestCopyFrom(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tpl := f.newIQC(t)
	source := f.newSetFor(t, "sumber", f.k125)
	target := f.newSetFor(t, "tujuan", f.k200)
	id := func(text string) string { return tpl.points[text] }

	// Model K-125: dirakit dari IQC, satu poin dicoret, satu baris diketik
	// sendiri, lalu diurutkan dan diberi nomor operasi.
	src, err := f.s.Compose(ctx, f.author, source["cp"].ID, Compose{
		VersionID: tpl.version.ID,
		Include:   []string{id("Wrong part received"), id("Check part number against PO"), id("Check warpage")},
		Exclude:   []Exclusion{{PointID: id("Check HIC"), Reason: "No HIC"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	product := fieldID(t, f.cpFormat, template.AreaRow, "Product Characteristics")
	if src, err = f.s.AddRow(ctx, f.author, src.ID, NewRow{OperationNumber: "99", Content: map[string]any{product: "Typed by hand"}}); err != nil {
		t.Fatal(err)
	}
	if src, err = f.s.Reorder(ctx, f.author, src.ID, Order{RowIDs: []string{src.Rows[2].ID, src.Rows[0].ID, src.Rows[1].ID}, Renumber: true}); err != nil {
		t.Fatal(err)
	}

	cp := target["cp"]
	if _, err := f.s.CopyFrom(ctx, f.viewer, cp.ID, src.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer menyalin: %v", err)
	}
	got, err := f.s.CopyFrom(ctx, f.author, cp.ID, src.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Baris CP tersalin dengan urutan, proses, nomor operasi, isi, dan tautan
	// template yang sama, tetapi dengan ID baru.
	if len(got.Rows) != len(src.Rows) {
		t.Fatalf("jumlah baris: %d, ingin %d", len(got.Rows), len(src.Rows))
	}
	for i, r := range got.Rows {
		s := src.Rows[i]
		if r.ID == s.ID || r.Position != s.Position || r.ProcessID != s.ProcessID || r.OperationNumber != s.OperationNumber ||
			r.OperationManual != s.OperationManual || r.TemplatePointID != s.TemplatePointID || r.Content[product] != s.Content[product] {
			t.Errorf("baris %d: %+v, sumber %+v", i+1, r, s)
		}
	}
	// Dokumen lain di set ikut tersalin (PFMEA dari PFMEA).
	if pfmea := f.get(t, target["pfmea"].ID); len(pfmea.Rows) != 1 || pfmea.Rows[0].TemplatePointID != id("Wrong part received") {
		t.Errorf("PFMEA tujuan: %+v", pfmea.Rows)
	}
	// Poin yang dikecualikan ikut tersalin beserta alasannya.
	view, err := f.s.Composition(ctx, f.author, cp.ID, tpl.version.ID)
	if err != nil {
		t.Fatal(err)
	}
	if st := states(view); st["Check HIC"] != StateExcluded || st["Check warpage"] != StateIncluded || st["Check MSL label"] != StateUndecided {
		t.Errorf("keadaan poin di salinan: %v", st)
	}

	// Salinan mandiri: mengubah salinan tidak mengubah sumbernya.
	row := got.Rows[0]
	if _, err := f.s.UpdateRow(ctx, f.author, cp.ID, row.ID, RowUpdate{ProcessID: row.ProcessID, OperationNumber: row.OperationNumber, Content: map[string]any{product: "Changed in the copy"}, Version: row.Version}); err != nil {
		t.Fatal(err)
	}
	if again := f.get(t, src.ID); again.Rows[0].Content[product] == "Changed in the copy" || len(again.Rows) != 3 {
		t.Errorf("sumber ikut berubah: %+v", again.Rows[0])
	}
	var action, details string
	if err := f.pool.QueryRow(ctx, "SELECT action, details FROM audit_log WHERE action = 'document.copied' ORDER BY seq DESC LIMIT 1").Scan(&action, &details); err != nil {
		t.Fatal(err)
	}
	if details != `{"rows":"4","source":"`+src.Number+`"}` {
		t.Errorf("audit: %s", details)
	}

	// Menyalin hanya ke dokumen yang masih kosong, dari dokumen lain yang
	// jenis dan customer-nya sama.
	for name, c := range map[string]struct{ target, source, want string }{
		"sudah berisi":   {cp.ID, src.ID, "rows:not_allowed"},
		"diri sendiri":   {src.ID, src.ID, "source_document_id:not_allowed"},
		"jenis berbeda":  {target["pfd"].ID, src.ID, "source_document_id:not_allowed"},
		"sumber tak ada": {target["pfd"].ID, "01900000-0000-7000-8000-000000000000", "source_document_id:invalid_format"},
	} {
		_, err := f.s.CopyFrom(ctx, f.author, c.target, c.source)
		if got := codes(t, err); !slices.Contains(got, c.want) {
			t.Errorf("%s: %v", name, got)
		}
	}
}

// TestListDocumentModels (M4.7): daftar dokumen menampilkan modelnya, supaya
// penyusun bisa memilih dokumen model serupa.
func TestListDocumentModels(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	if _, err := f.s.Create(ctx, f.author, NewDocument{NumberID: f.cpNumber.ID, ModelIDs: []string{f.k200, f.k125}}); err != nil {
		t.Fatal(err)
	}
	list, err := f.s.List(ctx, f.author, f.custAB)
	if err != nil || len(list) != 1 || !slices.Equal(list[0].Models, []string{"K-125", "K-200"}) || list[0].DocumentType.Key != "cp" {
		t.Errorf("daftar: %+v, %v", list, err)
	}
}
