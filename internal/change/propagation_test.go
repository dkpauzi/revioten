package change

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/template"
)

// M5.8 · Propagasi template (R-TPL-06 sampai R-TPL-09): versi template baru
// menghasilkan perubahan, dan saat perubahan diterima isi barunya masuk ke
// draft dokumen yang mengikuti. Baris yang diubah lokal dibiarkan (keep),
// dokumen yang dikunci versinya hanya ikut dengan cakupan all.
func TestPropagation(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	maintainer := doctest.NewActor(t, e.Pool, "ppap.engineer", "work", doctest.All(access.RoleTemplateMaintainer))
	svc := template.NewProcessService(e.Pool, e.Checker)

	// Process template IQC versi 1 dengan tiga poin CP.
	process, err := e.Q.CreateProcess(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Q.AddProcessName(ctx, db.AddProcessNameParams{ProcessID: process, Kind: "display", Name: "IQC"}); err != nil {
		t.Fatal(err)
	}
	tpl, err := svc.CreateProcessTemplate(ctx, maintainer, process)
	if err != nil {
		t.Fatal(err)
	}
	var cpType string
	if err := e.Pool.QueryRow(ctx, "SELECT id FROM document_types WHERE key = 'cp'").Scan(&cpType); err != nil {
		t.Fatal(err)
	}
	v1 := tpl.Versions[0].ID
	v, err := svc.AddSubprocess(ctx, maintainer, v1, "IQC General")
	if err != nil {
		t.Fatal(err)
	}
	sub := v.Subprocesses[0].ID
	for _, p := range []template.NewPoint{
		{SubprocessID: sub, DocumentTypeID: cpType, Text: "Check part number", Content: map[string]any{"product_characteristic": "Part number", "control_method": "Visual"}},
		{SubprocessID: sub, DocumentTypeID: cpType, Text: "Check MSL label", Content: map[string]any{"product_characteristic": "MSL label"}},
		{SubprocessID: sub, DocumentTypeID: cpType, Text: "Check HIC", Content: map[string]any{"product_characteristic": "HIC"}},
	} {
		if v, err = svc.AddPoint(ctx, maintainer, v1, p); err != nil {
			t.Fatal(err)
		}
	}
	points := map[string]string{}
	for _, p := range v.Subprocesses[0].Points {
		points[p.Text] = p.ID
	}
	if _, err := svc.PublishProcessVersion(ctx, maintainer, v1); err != nil {
		t.Fatal(err)
	}
	all := []string{points["Check part number"], points["Check MSL label"], points["Check HIC"]}
	compose := func(key, customerID, modelID string) document.Document {
		t.Helper()
		cp := e.NewSet(t, key, customerID, modelID)["cp"]
		d, err := e.Docs.Compose(ctx, e.Author, cp.ID, document.Compose{VersionID: v1, Include: all})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	follow := compose("ab1", e.CustomerAB, e.ModelK125)
	local := compose("ab2", e.CustomerAB, e.ModelK200)
	pinned := compose("cd", e.CustomerCD, e.ModelB1)
	method := doctest.FieldID(t, e.CPFormat, "Method")
	row := local.Rows[0]
	content := map[string]any{}
	for k, x := range row.Content {
		content[k] = x
	}
	content[method] = "Scanner"
	if _, err := e.Docs.UpdateRow(ctx, e.Author, local.ID, row.ID, document.RowUpdate{ProcessID: row.ProcessID, Content: content, Version: row.Version}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Docs.SetTemplateMode(ctx, e.Author, pinned.ID, tpl.ID, document.ModePinned, ""); !slices.Contains(problemCodes(t, err), "reason:required") {
		t.Errorf("pinned tanpa alasan: %v", err)
	}
	if _, err := e.Docs.SetTemplateMode(ctx, e.Author, pinned.ID, tpl.ID, document.ModePinned, "Customer CD approved this version only"); err != nil {
		t.Fatal(err)
	}

	// Versi 2: metode poin pertama berubah, poin HIC disembunyikan.
	v2v, err := svc.NewProcessVersion(ctx, maintainer, tpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	v2 := v2v.ID
	changed := map[string]any{"product_characteristic": "Part number", "control_method": "SPI"}
	if _, err := svc.UpdatePoint(ctx, maintainer, v2, points["Check part number"], template.PointUpdate{Text: "Check part number", Content: &changed}); err != nil {
		t.Fatal(err)
	}
	hidden := true
	if _, err := svc.UpdatePoint(ctx, maintainer, v2, points["Check HIC"], template.PointUpdate{Text: "Check HIC", Hidden: &hidden}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Preview(ctx, maintainer, v2, ScopeFollowing); !slices.Contains(problemCodes(t, err), "version:not_allowed") {
		t.Errorf("pratinjau versi draft: %v", err)
	}
	if _, err := svc.PublishProcessVersion(ctx, maintainer, v2); err != nil {
		t.Fatal(err)
	}

	// Pratinjau: dokumen yang mengikuti, baris yang berubah, dan konflik lokal.
	if _, err := s.Preview(ctx, e.Author, v2, ScopeFollowing); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun tanpa E-6: %v", err)
	}
	items, err := s.Preview(ctx, maintainer, v2, ScopeFollowing)
	if err != nil {
		t.Fatal(err)
	}
	byDoc := map[string]PreviewItem{}
	for _, it := range items {
		byDoc[it.DocumentID] = it
	}
	if it := byDoc[follow.ID]; !it.Included || it.Updates != 1 || it.Conflicts != 0 || it.Removed != 1 {
		t.Errorf("mengikuti: %+v", it)
	}
	if it := byDoc[local.ID]; !it.Included || it.Updates != 0 || it.Conflicts != 1 || it.Removed != 1 {
		t.Errorf("berubah lokal: %+v", it)
	}
	if it := byDoc[pinned.ID]; it.Included || it.Skipped != "pinned" {
		t.Errorf("dikunci: %+v", it)
	}
	if items, _ = s.Preview(ctx, maintainer, v2, ScopeAll); !slices.ContainsFunc(items, func(it PreviewItem) bool { return it.DocumentID == pinned.ID && it.Included }) {
		t.Error("cakupan all tidak mengikutkan dokumen yang dikunci")
	}

	// Cakupan all wajib rujukan SQAM dan tanggal berlaku.
	_, err = s.Propagate(ctx, maintainer, Propagation{VersionID: v2, Scope: ScopeAll, ConflictPolicy: ConflictKeep, ChangeTypeID: m.content, Reason: "SQAM rev 5"})
	if got := problemCodes(t, err); !slices.Contains(got, "source_reference:required") || !slices.Contains(got, "effective_date:required") {
		t.Errorf("cakupan all: %v", got)
	}
	changes, err := s.Propagate(ctx, maintainer, Propagation{VersionID: v2, Scope: ScopeFollowing, ConflictPolicy: ConflictKeep, ChangeTypeID: m.content, Reason: "IQC uses SPI for part number"})
	if err != nil {
		t.Fatal(err)
	}
	// Satu perubahan untuk customer AB yang menunjuk dua dokumen; CD dikunci.
	if len(changes) != 1 || changes[0].Mechanism != MechanismTemplate || changes[0].CustomerID != e.CustomerAB || len(changes[0].Documents) != 2 ||
		changes[0].Status != StatusSubmitted || changes[0].Template == nil || changes[0].Template.VersionID != v2 {
		t.Fatalf("perubahan: %+v", changes)
	}
	// Belum diterima: isi dokumen belum berubah (R-TPL-06).
	if d, _ := e.Docs.Get(ctx, e.Author, follow.ID); d.Rows[0].Content[method] != "Visual" {
		t.Errorf("isi berubah sebelum diterima: %v", d.Rows[0].Content[method])
	}
	// Propagasi yang sama tidak membuat perubahan dobel.
	if _, err := s.Propagate(ctx, maintainer, Propagation{VersionID: v2, Scope: ScopeFollowing, ConflictPolicy: ConflictKeep, ChangeTypeID: m.content, Reason: "Again"}); !slices.Contains(problemCodes(t, err), "version:not_allowed") {
		t.Errorf("propagasi dobel: %v", err)
	}

	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	hold(t, e.Pool, controller, "apqp_ppap_controller", "")
	c, err := s.Verify(ctx, controller, changes[0].ID, true, "OK for IQC")
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]AffectedDocument{}
	for _, d := range c.Documents {
		result[d.ID] = d
	}
	if r := result[follow.ID]; r.RowsUpdated == nil || *r.RowsUpdated != 1 || *r.RowsConflict != 0 || *r.RowsRemoved != 1 {
		t.Errorf("hasil mengikuti: %+v", r)
	}
	if r := result[local.ID]; r.RowsUpdated == nil || *r.RowsUpdated != 0 || *r.RowsConflict != 1 {
		t.Errorf("hasil lokal: %+v", r)
	}
	d, err := e.Docs.Get(ctx, e.Author, follow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Rows[0].Content[method] != "SPI" || d.Rows[0].TemplateVersionID != v2 || d.Rows[2].TemplateVersionID != v1 {
		t.Errorf("draft mengikuti: %v, %s, %s", d.Rows[0].Content[method], d.Rows[0].TemplateVersionID, d.Rows[2].TemplateVersionID)
	}
	if d, _ = e.Docs.Get(ctx, e.Author, local.ID); d.Rows[0].Content[method] != "Scanner" {
		t.Errorf("baris lokal ditimpa: %v", d.Rows[0].Content[method])
	}
	if d, _ = e.Docs.Get(ctx, e.Author, pinned.ID); d.Rows[0].Content[method] != "Visual" {
		t.Errorf("dokumen dikunci berubah: %v", d.Rows[0].Content[method])
	}

	// Cakupan all dengan overwrite: dokumen dikunci ikut.
	date := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	changes, err = s.Propagate(ctx, maintainer, Propagation{VersionID: v2, Scope: ScopeAll, ConflictPolicy: ConflictOverwrite, ChangeTypeID: m.content,
		Reason: "SQAM rev 5 applies to every active document", SourceReference: "SQAM-CD rev 5", EffectiveDate: &date})
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].CustomerID != e.CustomerCD || changes[0].Template.EffectiveDate == nil || !changes[0].Template.EffectiveDate.Equal(date) {
		t.Fatalf("cakupan all: %+v", changes)
	}
	if _, err := s.Verify(ctx, controller, changes[0].ID, true, "SQAM"); err != nil {
		t.Fatal(err)
	}
	if d, _ = e.Docs.Get(ctx, e.Author, pinned.ID); d.Rows[0].Content[method] != "SPI" {
		t.Errorf("dokumen dikunci sesudah cakupan all: %v", d.Rows[0].Content[method])
	}
}
