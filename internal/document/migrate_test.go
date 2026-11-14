package document

import (
	"errors"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// Tahap 7 · Migrasi (M7.1, M7.2, M7.4, M7.5; R-MIG-03, R-MIG-05, R-MIG-06,
// R-REV-01, R-REV-05).

func code(t *testing.T, err error) string {
	t.Helper()
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) || len(invalid.Problems) == 0 {
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
}

func (f fixture) migrationMode(t *testing.T, on bool) {
	t.Helper()
	if _, err := f.pool.Exec(t.Context(), "UPDATE system_settings SET migration_mode = $1", on); err != nil {
		t.Fatal(err)
	}
}

// reservedNumbers mendaftarkan satu set nomor dan memberi ID-nya per jenis dokumen.
func (f fixture) reservedNumbers(t *testing.T) map[string]string {
	t.Helper()
	reg, err := f.numbers.RegisterSet(t.Context(), f.author, numbering.NewSet{CustomerID: f.custAB, IdempotencyKey: "mig-" + t.Name()})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, n := range reg.Numbers {
		if n.Status == "reserved" {
			var key string
			if err := f.pool.QueryRow(t.Context(), "SELECT key FROM document_types WHERE id = $1", n.DocumentTypeID).Scan(&key); err != nil {
				t.Fatal(err)
			}
			out[key] = n.ID
		}
	}
	return out
}

// TestMigrateLegacyDocument: kriteria selesai M7.1, M7.2, dan M7.5. Dokumen
// lama masuk di revisi -04, riwayatnya tersimpan sebagai teks, dan terbit
// sebagai Released (migrated) sesudah dicocokkan orang lain dengan hardcopy.
func TestMigrateLegacyDocument(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	numbers := f.reservedNumbers(t)
	staff := newActor(t, f.pool, "ppap.staff", access.AccountWork, all(access.RoleMigration), all(access.RoleAuthor))
	engineer := newActor(t, f.pool, "ppap.engineer2", access.AccountWork, all(access.RoleMigration))
	in := LegacyDocument{
		NewDocument: NewDocument{NumberID: numbers["cp"], ModelIDs: []string{f.k125}},
		Revision:    4, EffectiveDate: "2025-03-10", PreparedBy: "Budi Santoso", ApprovedBy: "Siti Aminah, Agus Wijaya",
	}

	// Mode migrasi mati: ditolak, juga untuk petugas migrasi.
	if _, err := f.s.Migrate(ctx, staff, in); code(t, err) != "migration_mode:not_allowed" {
		t.Errorf("mode mati: %v", err)
	}
	f.migrationMode(t, true)
	if _, err := f.s.Migrate(ctx, f.author, in); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun tanpa E-10: %v", err)
	}
	for name, bad := range map[string]func(*LegacyDocument){
		"revision:invalid_format":       func(l *LegacyDocument) { l.Revision = -1 },
		"effective_date:invalid_format": func(l *LegacyDocument) { l.EffectiveDate = "10/03/2025" },
		"effective_date:required":       func(l *LegacyDocument) { l.EffectiveDate = "" },
		"prepared_by:required":          func(l *LegacyDocument) { l.PreparedBy = " " },
		"approved_by:required":          func(l *LegacyDocument) { l.ApprovedBy = "" },
		"model_ids:required":            func(l *LegacyDocument) { l.ModelIDs = nil },
	} {
		wrong := in
		bad(&wrong)
		if _, err := f.s.Migrate(ctx, staff, wrong); code(t, err) != name {
			t.Errorf("%s: %v", name, err)
		}
	}

	d, err := f.s.Migrate(ctx, staff, in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Revision != 4 || d.Origin != OriginMigrated || d.Status != StatusDraft || d.Baseline == nil ||
		d.Baseline.EffectiveDate != "2025-03-10" || d.Baseline.PreparedBy != "Budi Santoso" || d.Baseline.VerifiedAt != nil {
		t.Fatalf("baseline: %+v %+v", d, d.Baseline)
	}
	if _, err := f.s.Migrate(ctx, staff, in); code(t, err) != "document_number_id:not_allowed" {
		t.Errorf("nomor dipakai dua kali: %v", err)
	}
	// Nomor yang sudah dipakai tidak ditawarkan lagi.
	list, err := f.s.MigratableNumbers(ctx, staff, f.custAB)
	if err != nil || len(list) != 3 { // dua nomor set ini dan nomor CP milik fixture
		t.Errorf("nomor yang bisa dimigrasi: %d, %v", len(list), err)
	}

	// Riwayat revisi sebagai teks (M7.2).
	history := []HistoryEntry{
		{Revision: "-00", Date: "2023-01-05", Description: "Initial release"},
		{Revision: "-03", Date: "", Description: "Add ICT step"},
	}
	if _, err := f.s.SetHistory(ctx, f.author, d.ID, history); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("riwayat tanpa E-10: %v", err)
	}
	if _, err := f.s.SetHistory(ctx, staff, d.ID, []HistoryEntry{{Revision: "-00", Date: "kemarin", Description: "x"}}); code(t, err) != "entries.0.date:invalid_format" {
		t.Errorf("tanggal riwayat: %v", err)
	}
	if d, err = f.s.SetHistory(ctx, staff, d.ID, history); err != nil || len(d.Baseline.History) != 2 || d.Baseline.History[1].Revision != "-03" {
		t.Fatalf("riwayat: %+v, %v", d.Baseline, err)
	}

	// Isi diketik seperti dokumen biasa (R-CMP-06).
	if _, err := f.s.AddRow(ctx, staff, d.ID, NewRow{Content: map[string]any{}}); err != nil {
		t.Fatal(err)
	}

	// Verifikasi hardcopy (M7.5): bukan oleh orang yang memasukkan, dan hanya saat mode menyala.
	effort := 45
	if _, err := f.s.VerifyMigration(ctx, f.author, d.ID, &effort); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("verifikasi tanpa E-10: %v", err)
	}
	if _, err := f.s.VerifyMigration(ctx, staff, d.ID, &effort); code(t, err) != "migration_verifier:not_allowed" {
		t.Errorf("memverifikasi masukan sendiri: %v", err)
	}
	f.migrationMode(t, false)
	if _, err := f.s.VerifyMigration(ctx, engineer, d.ID, &effort); code(t, err) != "migration_mode:not_allowed" {
		t.Errorf("verifikasi saat mode mati: %v", err)
	}
	f.migrationMode(t, true)
	zero := 0
	if _, err := f.s.VerifyMigration(ctx, engineer, d.ID, &zero); code(t, err) != "effort_minutes:invalid_format" {
		t.Errorf("menit nol: %v", err)
	}
	// Rilis biasa tidak berlaku untuk baseline migrasi (R-MIG-06).
	dcc := newActor(t, f.pool, "dcc.uji", access.AccountWork, all(access.RoleDocumentControl))
	if _, err := f.s.Release(ctx, dcc, d.ID); code(t, err) != "status:not_allowed" {
		t.Errorf("rilis biasa untuk baseline: %v", err)
	}
	// Tanpa scan hardcopy bertanda tangan, verifikasi ditolak (D-173).
	if _, err := f.s.VerifyMigration(ctx, engineer, d.ID, &effort); code(t, err) != "scan:required" {
		t.Errorf("verifikasi tanpa scan: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE document_revisions
		SET scan_file_name = 'hardcopy.pdf', scan_content_type = 'application/pdf', scan_size_bytes = 10,
		    scan_sha256 = repeat('a', 64), scan_uploaded_by_person_id = created_by_person_id, scan_uploaded_at = now()
		WHERE document_id = $1 AND status = 'draft'`, d.ID); err != nil {
		t.Fatal(err)
	}
	d, err = f.s.VerifyMigration(ctx, engineer, d.ID, &effort)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != StatusReleased || d.ReleasedRevision == nil || *d.ReleasedRevision != 4 || !d.ReleasedMigrated ||
		d.Baseline.VerifiedBy != "Orang ppap.engineer2" || d.Baseline.EffortMinutes == nil || *d.Baseline.EffortMinutes != 45 {
		t.Fatalf("sesudah verifikasi: %+v %+v", d, d.Baseline)
	}
	revs, err := f.s.Revisions(ctx, f.author, d.ID)
	if err != nil || len(revs) != 1 || !revs[0].Migrated || revs[0].ReleaseDate != "2025-03-10" || len(revs[0].SHA256) != 64 {
		t.Fatalf("revisi: %+v, %v", revs, err)
	}

	// Baseline yang sudah diverifikasi tidak berubah lagi, juga lewat SQL langsung.
	if _, err := f.s.SetHistory(ctx, staff, d.ID, nil); code(t, err) != "status:not_allowed" {
		t.Errorf("riwayat sesudah verifikasi: %v", err)
	}
	if _, err := f.s.VerifyMigration(ctx, engineer, d.ID, nil); code(t, err) != "status:not_allowed" {
		t.Errorf("verifikasi dua kali: %v", err)
	}
	for _, sql := range []string{
		"UPDATE document_baselines SET effort_minutes = 1 WHERE document_id = $1",
		"DELETE FROM document_history_entries WHERE document_id = $1",
	} {
		if _, err := f.pool.Exec(ctx, sql, d.ID); err == nil {
			t.Errorf("database menerima: %s", sql)
		}
	}
}

// TestLinkRows: kriteria selesai M7.4. Baris yang diketik sendiri mendapat
// usulan poin template lewat kemiripan teks; sesudah dikonfirmasi, dokumen
// itu ikut propagasi versi template berikutnya.
func TestLinkRows(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	tpl := f.newIQC(t)
	cp := f.newSet(t)["cp"]
	format, err := template.LoadFormatVersion(ctx, f.q, cp.FormatVersionID)
	if err != nil {
		t.Fatal(err)
	}
	var characteristic string
	for _, fld := range format.Fields {
		if fld.Area == template.AreaRow && fld.Key == "product_characteristic" {
			characteristic = fld.ID
		}
	}
	add := func(process, text string) string {
		t.Helper()
		d, err := f.s.AddRow(ctx, f.author, cp.ID, NewRow{ProcessID: process, Content: map[string]any{characteristic: text}})
		if err != nil {
			t.Fatal(err)
		}
		return d.Rows[len(d.Rows)-1].ID
	}
	partRow := add(tpl.process, "Part numbers")   // mirip "Part number"
	mslRow := add(tpl.process, "MSL  Label")      // sama dengan "MSL label" selain huruf besar dan spasi
	add(tpl.process, "Torque of the screwdriver") // tidak mirip poin mana pun
	add("", "Part number")                        // tanpa proses: tidak ada template yang dicari

	if _, err := f.s.LinkSuggestions(ctx, f.viewer, cp.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("usulan tanpa hak: %v", err)
	}
	suggestions, err := f.s.LinkSuggestions(ctx, f.author, cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]LinkSuggestion{}
	for _, s := range suggestions {
		got[s.RowID] = s
	}
	if len(suggestions) != 2 || got[partRow].PointID != tpl.points["Check part number against PO"] || got[mslRow].PointID != tpl.points["Check MSL label"] || got[mslRow].Score != 100 {
		t.Fatalf("usulan: %+v", suggestions)
	}

	links := []RowLink{
		{RowID: partRow, PointID: got[partRow].PointID, VersionID: got[partRow].VersionID},
		{RowID: mslRow, PointID: got[mslRow].PointID, VersionID: got[mslRow].VersionID},
	}
	if _, err := f.s.LinkRows(ctx, f.viewer, cp.ID, links); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("menautkan tanpa E-2: %v", err)
	}
	// Dua baris ke poin yang sama ditolak.
	if _, err := f.s.LinkRows(ctx, f.author, cp.ID, []RowLink{links[0], {RowID: mslRow, PointID: links[0].PointID, VersionID: links[0].VersionID}}); code(t, err) != "links.1.point_id:not_allowed" {
		t.Errorf("poin ganda: %v", err)
	}
	d, err := f.s.LinkRows(ctx, f.author, cp.ID, links)
	if err != nil {
		t.Fatal(err)
	}
	linked := 0
	for _, r := range d.Rows {
		if r.TemplatePointID != "" {
			linked++
		}
	}
	if linked != 2 {
		t.Fatalf("baris tertaut: %d", linked)
	}
	if _, err := f.s.LinkRows(ctx, f.author, cp.ID, links[:1]); code(t, err) != "links.0.row_id:not_allowed" {
		t.Errorf("menautkan dua kali: %v", err)
	}
	// Isi yang berbeda dari template menjadi perbedaan lokal (R-MIG-03).
	diffs, err := f.s.Differences(ctx, f.author, cp.ID)
	if err != nil || len(diffs) != 2 {
		t.Fatalf("perbedaan: %+v, %v", diffs, err)
	}
	// Sesudah ditautkan, usulan habis.
	if rest, _ := f.s.LinkSuggestions(ctx, f.author, cp.ID); len(rest) != 0 {
		t.Errorf("usulan tersisa: %+v", rest)
	}

	// Dokumen sekarang ikut propagasi versi template berikutnya.
	v2, err := tpl.svc.NewProcessVersion(ctx, tpl.actor, tpl.version.TemplateID)
	if err != nil {
		t.Fatal(err)
	}
	content := map[string]any{"product_characteristic": "MSL label and date code"}
	if _, err := tpl.svc.UpdatePoint(ctx, tpl.actor, v2.ID, tpl.points["Check MSL label"], template.PointUpdate{Text: "Check MSL label", Content: &content}); err != nil {
		t.Fatal(err)
	}
	if _, err := tpl.svc.PublishProcessVersion(ctx, tpl.actor, v2.ID); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanTemplate(ctx, f.q, cp.ID, v2.ID)
	if err != nil || len(plan.Rows) == 0 {
		t.Fatalf("rencana propagasi: %+v, %v", plan, err)
	}
}

func TestSimilarity(t *testing.T) {
	for _, c := range []struct {
		a, b string
		min  int
		max  int
	}{
		{"Parts Receiving", "Part Receiving", 85, 99},
		{"MSL  Label", "msl label", 100, 100},
		{"Cleaning PCB", "Torque of the screwdriver", 0, 30},
		{"", "abc", 0, 0},
		{"7", "7", 100, 100},
	} {
		if got := similarity(c.a, c.b); got < c.min || got > c.max {
			t.Errorf("similarity(%q, %q) = %d, ingin %d sampai %d", c.a, c.b, got, c.min, c.max)
		}
	}
}
