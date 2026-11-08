package template

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/problem"
)

// processFixture: database baru, Pengelola template (E-6, D-64), akun admin
// dan Penyusun yang hanya melihat, serta id jenis dokumen PFD, PFMEA, dan CP.
type processFixture struct {
	s          *ProcessService
	q          *db.Queries
	maintainer access.Actor
	admin      access.Actor
	author     access.Actor
	types      map[string]string // key jenis dokumen ke id
}

func newProcessFixture(t *testing.T) processFixture {
	t.Helper()
	pool := dbtest.NewPool(t)
	f := processFixture{
		s: NewProcessService(pool, access.NewChecker(pool)), q: db.New(pool),
		maintainer: newActor(t, pool, "ppap.uji", access.AccountWork, access.RoleTemplateMaintainer),
		admin:      newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin),
		author:     newActor(t, pool, "penyusun.uji", access.AccountWork, access.RoleAuthor),
		types:      map[string]string{},
	}
	rows, err := pool.Query(t.Context(), "SELECT key, id FROM document_types WHERE key IS NOT NULL")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, id string
		if err := rows.Scan(&key, &id); err != nil {
			t.Fatal(err)
		}
		f.types[key] = id
	}
	return f
}

// process membuat proses master langsung lewat query, seperti M1.5.
func (f processFixture) process(t *testing.T, name string) string {
	t.Helper()
	id, err := f.q.CreateProcess(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.q.AddProcessName(t.Context(), db.AddProcessNameParams{ProcessID: id, Kind: "display", Name: name}); err != nil {
		t.Fatal(err)
	}
	return id
}

// newTemplate membuat proses dan process template-nya, lalu mengembalikan versi 1.
func (f processFixture) newTemplate(t *testing.T, processName string, subprocesses ...string) ProcessVersion {
	t.Helper()
	pt, err := f.s.CreateProcessTemplate(t.Context(), f.maintainer, f.process(t, processName))
	if err != nil {
		t.Fatal(err)
	}
	v, err := f.s.GetProcessVersion(t.Context(), f.maintainer, pt.Versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range subprocesses {
		if v, err = f.s.AddSubprocess(t.Context(), f.maintainer, v.ID, name); err != nil {
			t.Fatal(err)
		}
	}
	return v
}

// addPoint menambah satu poin dan mengembalikan versi sesudahnya.
func (f processFixture) addPoint(t *testing.T, v ProcessVersion, subprocess, documentType, text string) ProcessVersion {
	t.Helper()
	sub := subprocessNamed(t, v, subprocess)
	v, err := f.s.AddPoint(t.Context(), f.maintainer, v.ID, NewPoint{SubprocessID: sub.ID, DocumentTypeID: f.types[documentType], Text: text})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func subprocessNamed(t *testing.T, v ProcessVersion, name string) Subprocess {
	t.Helper()
	for _, s := range v.Subprocesses {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("sub-proses %q tidak ada", name)
	return Subprocess{}
}

func subprocessNames(v ProcessVersion) []string {
	var names []string
	for _, s := range v.Subprocesses {
		names = append(names, s.Name)
	}
	return names
}

// pointTexts menulis poin satu sub-proses sebagai "jenis:teks".
func pointTexts(s Subprocess) []string {
	var out []string
	for _, p := range s.Points {
		out = append(out, p.DocumentType.Key+":"+p.Text)
	}
	return out
}

// TestCleanItemUpdate: nama atau teks wajib satu baris, urutan dalam batas,
// dan isian yang tidak dikirim berarti tetap (D-111).
func TestCleanItemUpdate(t *testing.T) {
	old := itemChange{Text: "IQC MSD", Position: 2, Hidden: true}
	got, list := cleanItemUpdate("name", "  IQC Moisture  ", maxSubprocessName, nil, nil, old, 5)
	if err := problems(list...); err != nil || got != (itemChange{Text: "IQC Moisture", Position: 2, Hidden: true}) {
		t.Errorf("tanpa urutan dan tersembunyi: %+v, %v", got, err)
	}
	position, hidden := 5, false
	got, list = cleanItemUpdate("name", "IQC MSD", maxSubprocessName, &position, &hidden, old, 5)
	if err := problems(list...); err != nil || got != (itemChange{Text: "IQC MSD", Position: 5, Hidden: false}) {
		t.Errorf("dengan urutan dan tersembunyi: %+v, %v", got, err)
	}
	for _, c := range []struct {
		text     string
		position int
		want     string
	}{
		{"", 1, "text:required"},
		{strings.Repeat("a", maxPointText+1), 1, "text:too_long"},
		{"dua\nbaris", 1, "text:invalid_format"},
		{"Cek label", 0, "position:invalid_format"},
		{"Cek label", 4, "position:invalid_format"},
	} {
		_, list := cleanItemUpdate("text", c.text, maxPointText, &c.position, nil, old, 3)
		var codes []string
		for _, p := range list {
			if p != nil {
				codes = append(codes, p.Field+":"+p.Code)
			}
		}
		if !slices.Equal(codes, []string{c.want}) {
			t.Errorf("%q di urutan %d: %v, ingin %s", c.text, c.position, codes, c.want)
		}
	}
}

// TestMoveRange: butir di antara urutan lama dan baru bergeser satu tempat.
func TestMoveRange(t *testing.T) {
	if first, last, delta := moveRange(5, 1); first != 1 || last != 4 || delta != 1 {
		t.Errorf("5 ke 1: %d..%d %+d", first, last, delta)
	}
	if first, last, delta := moveRange(1, 3); first != 2 || last != 3 || delta != -1 {
		t.Errorf("1 ke 3: %d..%d %+d", first, last, delta)
	}
}

// TestIQCExample adalah kriteria selesai M3.2: contoh IQC dengan lima
// sub-proses tersimpan, beserta poinnya, dan terbaca lagi utuh. Nama proses
// diambil dari master lewat ID, jadi mengganti nama proses ikut tampil (D-06).
func TestIQCExample(t *testing.T) {
	f := newProcessFixture(t)
	ctx := t.Context()
	five := []string{"IQC General", "IQC MSD", "IQC PCB", "IQC Wireharness", "IQC Connector"}
	v := f.newTemplate(t, "IQC", five...)
	v = f.addPoint(t, v, "IQC MSD", "cp", "Check MSL label and humidity indicator card")
	v = f.addPoint(t, v, "IQC MSD", "pfmea", "Floor life exceeded before use")
	v = f.addPoint(t, v, "IQC MSD", "pfd", "Incoming inspection of MSD")
	v = f.addPoint(t, v, "IQC MSD", "pfmea", "Damaged moisture barrier bag")
	f.addPoint(t, v, "IQC PCB", "pfmea", "Warped PCB")

	// Dibaca lagi oleh akun lain, yang hanya punya hak melihat (L-2).
	list, err := f.s.ListProcessTemplates(ctx, f.author)
	if err != nil || len(list) != 1 || list[0].Process.Name != "IQC" || len(list[0].Versions) != 1 ||
		list[0].Versions[0].Version != 1 || list[0].Versions[0].Status != StatusDraft {
		t.Fatalf("daftar template: %+v, %v", list, err)
	}
	got, err := f.s.GetProcessVersion(ctx, f.author, list[0].Versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(subprocessNames(got), five) {
		t.Errorf("sub-proses: %v", subprocessNames(got))
	}
	for i, s := range got.Subprocesses {
		if s.ID == "" || s.Position != i+1 || s.Hidden {
			t.Errorf("sub-proses %d: %+v", i+1, s)
		}
	}
	// Poin urut menurut jenis dokumen (PFD, PFMEA, CP), lalu urutan tambahnya.
	msd := subprocessNamed(t, got, "IQC MSD")
	want := []string{"pfd:Incoming inspection of MSD", "pfmea:Floor life exceeded before use",
		"pfmea:Damaged moisture barrier bag", "cp:Check MSL label and humidity indicator card"}
	if !slices.Equal(pointTexts(msd), want) {
		t.Errorf("poin IQC MSD: %v", pointTexts(msd))
	}
	if msd.Points[2].Position != 2 || msd.Points[3].Position != 1 || msd.Points[0].DocumentType.Name != "PFD" {
		t.Errorf("urutan dan jenis poin: %+v", msd.Points)
	}
	if general := subprocessNamed(t, got, "IQC General"); general.Points == nil || len(general.Points) != 0 {
		t.Errorf("sub-proses tanpa poin harus daftar kosong: %#v", general.Points)
	}

	if err := f.q.RenameProcess(ctx, db.RenameProcessParams{ProcessID: got.Process.ID, Name: "Incoming Quality Control"}); err != nil {
		t.Fatal(err)
	}
	if again, err := f.s.GetProcessVersion(ctx, f.author, got.ID); err != nil || again.Process.Name != "Incoming Quality Control" {
		t.Errorf("nama proses sesudah diganti: %q, %v", again.Process.Name, err)
	}
}

// TestRenameKeepsIDs: mengganti nama sub-proses dan teks poin, atau
// menyembunyikannya, tidak mengubah ID-nya. Perubahan tanpa hidden tidak
// memunculkan kembali yang tersembunyi (D-111).
func TestRenameKeepsIDs(t *testing.T) {
	f := newProcessFixture(t)
	ctx := t.Context()
	v := f.newTemplate(t, "IQC", "IQC General", "IQC MSD")
	v = f.addPoint(t, v, "IQC MSD", "pfmea", "Floor life exceeded")
	msd := subprocessNamed(t, v, "IQC MSD")
	point := msd.Points[0]

	yes := true
	v, err := f.s.UpdateSubprocess(ctx, f.maintainer, v.ID, msd.ID, SubprocessUpdate{Name: "IQC Moisture Sensitive", Hidden: &yes})
	if err != nil {
		t.Fatal(err)
	}
	v, err = f.s.UpdatePoint(ctx, f.maintainer, v.ID, point.ID, PointUpdate{Text: "Floor life exceeded before reflow", Hidden: &yes})
	if err != nil {
		t.Fatal(err)
	}
	// Ganti nama lagi tanpa mengirim hidden: tetap tersembunyi.
	v, err = f.s.UpdateSubprocess(ctx, f.maintainer, v.ID, msd.ID, SubprocessUpdate{Name: "IQC MSD"})
	if err != nil {
		t.Fatal(err)
	}
	v, err = f.s.UpdatePoint(ctx, f.maintainer, v.ID, point.ID, PointUpdate{Text: "Floor life exceeded"})
	if err != nil {
		t.Fatal(err)
	}
	got := v.Subprocesses[1]
	if got.ID != msd.ID || got.Name != "IQC MSD" || !got.Hidden || got.Position != 2 {
		t.Errorf("sub-proses: %+v", got)
	}
	if p := got.Points[0]; p.ID != point.ID || p.Text != "Floor life exceeded" || !p.Hidden || p.Position != 1 {
		t.Errorf("poin: %+v", p)
	}
	// Nama yang sama dengan huruf berbeda tetap milik sub-proses yang sama.
	if _, err := f.s.UpdateSubprocess(ctx, f.maintainer, v.ID, msd.ID, SubprocessUpdate{Name: "iqc msd"}); err != nil {
		t.Errorf("ganti huruf besar dan kecil: %v", err)
	}
}

// TestProcessTemplateRules: isian yang salah ditolak per field.
func TestProcessTemplateRules(t *testing.T) {
	f := newProcessFixture(t)
	ctx := t.Context()
	v := f.newTemplate(t, "IQC", "IQC General", "IQC MSD")
	general := subprocessNamed(t, v, "IQC General")

	check := func(name string, err error, want ...string) {
		t.Helper()
		if got := problemCodes(t, err); !slices.Equal(got, want) {
			t.Errorf("%s: %v, ingin %v", name, got, want)
		}
	}
	_, err := f.s.CreateProcessTemplate(ctx, f.maintainer, "")
	check("tanpa proses", err, "process_id:required")
	_, err = f.s.CreateProcessTemplate(ctx, f.maintainer, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489")
	check("proses tidak ada", err, "process_id:invalid_format")
	_, err = f.s.CreateProcessTemplate(ctx, f.maintainer, v.Process.ID)
	check("template kedua untuk proses yang sama (D-129)", err, "process_id:taken")

	_, err = f.s.AddSubprocess(ctx, f.maintainer, v.ID, " ")
	check("nama kosong", err, "name:required")
	_, err = f.s.AddSubprocess(ctx, f.maintainer, v.ID, "iqc msd")
	check("nama kembar", err, "name:taken")
	_, err = f.s.UpdateSubprocess(ctx, f.maintainer, v.ID, general.ID, SubprocessUpdate{Name: "IQC MSD"})
	check("ganti ke nama kembar", err, "name:taken")

	other, err := f.q.CreateDocumentType(ctx, db.CreateDocumentTypeParams{Name: "Work Instruction", Letter: "W"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.AddPoint(ctx, f.maintainer, v.ID, NewPoint{SubprocessID: general.ID, DocumentTypeID: other, Text: "Check"})
	check("jenis dokumen tanpa poin", err, "document_type_id:not_allowed")
	_, err = f.s.AddPoint(ctx, f.maintainer, v.ID, NewPoint{SubprocessID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", DocumentTypeID: f.types["cp"]})
	check("sub-proses asing dan teks kosong", err, "text:required", "subprocess_id:invalid_format")

	// Sub-proses milik template lain tidak bisa dipakai di versi ini.
	pcb := f.newTemplate(t, "SMT", "SMT Printing")
	_, err = f.s.AddPoint(ctx, f.maintainer, v.ID, NewPoint{SubprocessID: pcb.Subprocesses[0].ID, DocumentTypeID: f.types["cp"], Text: "Check"})
	check("sub-proses template lain", err, "subprocess_id:invalid_format")

	if _, err := f.s.UpdateSubprocess(ctx, f.maintainer, v.ID, pcb.Subprocesses[0].ID, SubprocessUpdate{Name: "X"}); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("ubah sub-proses template lain: %v", err)
	}
	if _, err := f.s.UpdatePoint(ctx, f.maintainer, v.ID, general.ID, PointUpdate{Text: "X"}); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("ubah poin yang tidak ada: %v", err)
	}
	if _, err := f.s.AddSubprocess(ctx, f.maintainer, "bukan-id", "X"); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("versi yang tidak ada: %v", err)
	}

	// Paling banyak 50 sub-proses per versi (ASUMSI E-73).
	for i := len(v.Subprocesses); i < maxSubprocesses; i++ {
		if _, err := f.s.AddSubprocess(ctx, f.maintainer, v.ID, "Variant "+strings.Repeat("x", i)); err != nil {
			t.Fatal(err)
		}
	}
	_, err = f.s.AddSubprocess(ctx, f.maintainer, v.ID, "One more")
	check("sub-proses ke-51", err, "name:not_allowed")
}

// TestMoveSubprocessAndPoint: urutan berubah, butir lain bergeser, dan
// daftar poin jenis dokumen lain tidak ikut bergeser.
func TestMoveSubprocessAndPoint(t *testing.T) {
	f := newProcessFixture(t)
	ctx := t.Context()
	v := f.newTemplate(t, "IQC", "IQC General", "IQC MSD", "IQC PCB")
	v = f.addPoint(t, v, "IQC MSD", "pfmea", "A")
	v = f.addPoint(t, v, "IQC MSD", "pfmea", "B")
	v = f.addPoint(t, v, "IQC MSD", "pfmea", "C")
	v = f.addPoint(t, v, "IQC MSD", "cp", "Z")

	three := 3
	pcb := subprocessNamed(t, v, "IQC PCB")
	one := 1
	v, err := f.s.UpdateSubprocess(ctx, f.maintainer, v.ID, pcb.ID, SubprocessUpdate{Name: pcb.Name, Position: &one})
	if err != nil {
		t.Fatal(err)
	}
	if got := subprocessNames(v); !slices.Equal(got, []string{"IQC PCB", "IQC General", "IQC MSD"}) {
		t.Errorf("sesudah PCB ke 1: %v", got)
	}

	a := subprocessNamed(t, v, "IQC MSD").Points[0]
	v, err = f.s.UpdatePoint(ctx, f.maintainer, v.ID, a.ID, PointUpdate{Text: a.Text, Position: &three})
	if err != nil {
		t.Fatal(err)
	}
	msd := subprocessNamed(t, v, "IQC MSD")
	if got := pointTexts(msd); !slices.Equal(got, []string{"pfmea:B", "pfmea:C", "pfmea:A", "cp:Z"}) {
		t.Errorf("sesudah A ke 3: %v", got)
	}
	if z := msd.Points[3]; z.Position != 1 {
		t.Errorf("poin CP ikut bergeser: %+v", z)
	}
	far := math.MaxInt16
	_, err = f.s.UpdatePoint(ctx, f.maintainer, v.ID, a.ID, PointUpdate{Text: a.Text, Position: &far})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"position:invalid_format"}) {
		t.Errorf("urutan di luar batas: %v", got)
	}
}

// TestProcessTemplatePermissionsAndAudit: hanya pemegang E-6 (Pengelola
// template, D-64) yang mengubah; semua akun melihat (L-2). Setiap perubahan
// tercatat dengan sasaran process template, dan audit log menampilkan nama
// prosesnya.
func TestProcessTemplatePermissionsAndAudit(t *testing.T) {
	f := newProcessFixture(t)
	ctx := t.Context()
	processID := f.process(t, "IQC")
	for _, actor := range []access.Actor{f.author, f.admin} {
		if _, err := f.s.CreateProcessTemplate(ctx, actor, processID); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s membuat template: %v", actor.Login, err)
		}
	}
	pt, err := f.s.CreateProcessTemplate(ctx, f.maintainer, processID)
	if err != nil {
		t.Fatal(err)
	}
	versionID := pt.Versions[0].ID
	for _, actor := range []access.Actor{f.author, f.admin} {
		if _, err := f.s.AddSubprocess(ctx, actor, versionID, "IQC MSD"); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s menambah sub-proses: %v", actor.Login, err)
		}
		if _, err := f.s.GetProcessVersion(ctx, actor, versionID); err != nil {
			t.Errorf("%s melihat versi: %v", actor.Login, err)
		}
	}

	v, err := f.s.AddSubprocess(ctx, f.maintainer, versionID, "IQC MSD")
	if err != nil {
		t.Fatal(err)
	}
	f.addPoint(t, v, "IQC MSD", "cp", "Check MSL label")
	entries, err := f.q.ListAuditEntries(ctx, db.ListAuditEntriesParams{Before: math.MaxInt64, MaxEntries: 3})
	if err != nil {
		t.Fatal(err)
	}
	want := []struct{ action, details string }{
		{"process_template.point_added", `{"document_type":"CP","position":"1","subprocess":"IQC MSD","text":"Check MSL label","version":"1"}`},
		{"process_template.subprocess_added", `{"name":"IQC MSD","position":"1","version":"1"}`},
		{"process_template.created", `{"process":"IQC","version":"1"}`},
	}
	for i, e := range entries {
		if e.Action != want[i].action || e.Details != want[i].details || e.TargetLabel != "IQC" || *e.TargetID != pt.ID {
			t.Errorf("entri %d: %s %s %q", i, e.Action, e.Details, e.TargetLabel)
		}
	}
}

// TestProcessTemplateDatabaseGuards: role aplikasi tidak bisa menghapus atau
// mengubah identitas, dan foreign key gabungan menolak sub-proses milik
// proses lain.
func TestProcessTemplateDatabaseGuards(t *testing.T) {
	f := newProcessFixture(t)
	for _, stmt := range []string{
		"DELETE FROM process_template_points WHERE false",
		"DELETE FROM template_points WHERE false",
		"DELETE FROM process_template_subprocesses WHERE false",
		"DELETE FROM subprocesses WHERE false",
		"DELETE FROM process_template_versions WHERE false",
		"DELETE FROM process_templates WHERE false",
		"UPDATE template_points SET subprocess_id = subprocess_id WHERE false",
		"UPDATE subprocesses SET process_id = process_id WHERE false",
		"UPDATE process_template_points SET document_type_id = document_type_id WHERE false",
		"UPDATE process_template_versions SET version = version WHERE false",
	} {
		err := f.exec(t, stmt)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("%s: %v, ingin permission denied (42501)", stmt, err)
		}
	}

	iqc := f.newTemplate(t, "IQC")
	smt := f.newTemplate(t, "SMT", "SMT Printing")
	// Sub-proses SMT dimasukkan ke versi IQC: ditolak foreign key, apa pun
	// process_id yang ditulis.
	for _, processID := range []string{iqc.Process.ID, smt.Process.ID} {
		err := f.exec(t, "INSERT INTO process_template_subprocesses (version_id, process_id, subprocess_id, name, position) VALUES ($1, $2, $3, 'X', 1)",
			iqc.ID, processID, smt.Subprocesses[0].ID)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
			t.Errorf("sub-proses SMT di versi IQC (process_id %s): %v, ingin foreign key (23503)", processID, err)
		}
	}
}

// exec menjalankan satu perintah SQL sebagai role aplikasi.
func (f processFixture) exec(t *testing.T, sql string, args ...any) error {
	t.Helper()
	_, err := f.s.pool.Exec(t.Context(), sql, args...)
	return err
}
