package template

import (
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/problem"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// newActor membuat orang dan akun langsung lewat query, dengan peran tertentu.
func newActor(t *testing.T, pool *pgxpool.Pool, login, kind string, roles ...string) access.Actor {
	t.Helper()
	ctx := t.Context()
	q := db.New(pool)
	person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: "Orang " + login})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{PersonID: person.ID, Kind: kind, Login: login, PasswordHash: "x"})
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range roles {
		if err := q.AddAccountRole(ctx, db.AddAccountRoleParams{AccountID: acc.ID, RoleID: role, Scope: access.ScopeAll}); err != nil {
			t.Fatal(err)
		}
	}
	return access.Actor{Type: access.ActorHuman, AccountID: acc.ID, AccountKind: kind, Login: login, PersonID: person.ID}
}

// fixture: database baru, akun PPAP Engineer (E-11 mengelola format template,
// UC-20, D-209), dan akun Penyusun yang hanya boleh melihat.
type fixture struct {
	s      *Service
	pool   *pgxpool.Pool
	master access.Actor
	author access.Actor
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.NewPool(t)
	return fixture{
		s: NewService(pool, access.NewChecker(pool)), pool: pool,
		master: newActor(t, pool, "ppap.master", access.AccountWork, access.RolePPAPEngineer),
		author: newActor(t, pool, "penyusun.uji", access.AccountWork, access.RoleAuthor),
	}
}

// template mencari format template menurut nama.
func (f fixture) template(t *testing.T, name string) FormatTemplate {
	t.Helper()
	list, err := f.s.ListFormatTemplates(t.Context(), f.author)
	if err != nil {
		t.Fatal(err)
	}
	for _, ft := range list {
		if ft.Name == name {
			return ft
		}
	}
	t.Fatalf("format template %q tidak ada", name)
	return FormatTemplate{}
}

// version mengambil versi pertama format template itu.
func (f fixture) version(t *testing.T, name string) FormatVersion {
	t.Helper()
	ft := f.template(t, name)
	v, err := f.s.GetFormatVersion(t.Context(), f.author, ft.Versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// field mencari field menurut label.
func field(t *testing.T, v FormatVersion, label string) Field {
	t.Helper()
	for _, fd := range v.Fields {
		if fd.Label == label {
			return fd
		}
	}
	t.Fatalf("field %q tidak ada di versi %s", label, v.ID)
	return Field{}
}

// labels menulis label field satu area menurut urutan.
func labels(v FormatVersion, area string) []string {
	var out []string
	for _, fd := range v.Fields {
		if fd.Area == area {
			out = append(out, fd.Label)
		}
	}
	return out
}

// problemCodes mengambil "field:kode" dari penolakan input.
func problemCodes(t *testing.T, err error) []string {
	t.Helper()
	var problems []problem.Problem
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case errors.As(err, &invalid):
		problems = invalid.Problems
	case errors.As(err, &conflict):
		problems = conflict.Problems
	default:
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	var out []string
	for _, p := range problems {
		out = append(out, p.Field+":"+p.Code)
	}
	return out
}

// TestSeededTemplates: isi awal M3.1 dari kolom form yang dipakai sekarang
// (F-09 sampai F-11): PFD, PFMEA v1 dengan RPN, dan CP, masing-masing
// versi 1 berstatus draft.
func TestSeededTemplates(t *testing.T) {
	f := newFixture(t)
	list, err := f.s.ListFormatTemplates(t.Context(), f.author)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, ft := range list {
		names = append(names, ft.Name+":"+ft.DocumentTypeKey+":"+ft.Methodology)
		if len(ft.Versions) != 1 || ft.Versions[0].Version != 1 || ft.Versions[0].Status != StatusDraft {
			t.Errorf("%s: versi %+v", ft.Name, ft.Versions)
		}
	}
	// Sejak migrasi 00018 ada dua form baru: PFMEA v2 dan Control Plan v2 (D-133).
	// Sejak migrasi 00054 namanya v1 dan v2 (D-218), diurutkan menurut nama.
	want := []string{"PFD v1:pfd:", "PFMEA v1:pfmea:aiag_4th", "PFMEA v2:pfmea:aiag_vda", "Control Plan v1:cp:", "Control Plan v2:cp:aiag_cp_2024"}
	if !slices.Equal(names, want) {
		t.Errorf("format template = %v, ingin %v", names, want)
	}

	v := f.version(t, "PFMEA v1")
	if v.RiskMethod != RiskRPN || v.HighlightRPNAbove != 100 || v.HighlightSeverityAtLeast != 8 {
		t.Errorf("PFMEA: metode %s, sorot %d dan %d", v.RiskMethod, v.HighlightRPNAbove, v.HighlightSeverityAtLeast)
	}
	if got := labels(v, AreaHeader); !slices.Equal(got, []string{"Process or Line", "Process Responsibility", "Key Date", "Core Team"}) {
		t.Errorf("header PFMEA: %v", got)
	}
	// Kolom mengikuti form Excel asli (D-124, migrasi 00014). Nomor operasi dan
	// nama proses terisi otomatis dari baris dan tetap tampil (D-123).
	wantPFMEA := []string{"Process Item No.", "Process Step / Function", "Requirement", "Potential Failure Mode",
		"Potential Effect(s) of Failure", "Severity", "Class", "Potential Cause(s) / Mechanism(s) of Failure",
		"Current Process Controls Prevention", "Occurrence", "Current Process Controls Detection", "Detection", "RPN",
		"Recommended Action", "Responsibility & Target Completion Date", "Actions Taken & Completion Date",
		"Result Severity", "Result Occurrence", "Result Detection", "Result RPN"}
	if got := labels(v, AreaRow); !slices.Equal(got, wantPFMEA) {
		t.Errorf("kolom PFMEA: %v", got)
	}
	if no, step := field(t, v, "Process Item No."), field(t, v, "Process Step / Function"); no.Kind != KindOperationNumber || step.Kind != KindProcess {
		t.Errorf("kolom proses PFMEA: %+v dan %+v", no, step)
	}
	if s := field(t, v, "Result Severity"); s.Kind != KindRating || s.RatingScale != ScaleSeverity || s.RatingSet != 2 {
		t.Errorf("Result Severity: %+v", s)
	}
	if len(v.Ratings) != 30 || v.Ratings[0] != (Rating{Scale: ScaleSeverity, Value: 1}) || v.Ratings[29] != (Rating{Scale: ScaleDetection, Value: 10}) {
		t.Errorf("tabel skala PFMEA: %d baris, %+v", len(v.Ratings), v.Ratings)
	}
	// Hanya Revision yang dipasang otomatis (R-REV-09, migrasi 00013).
	if len(v.Symbols) != 4 || v.Symbols[0] != (Symbol{Key: "critical_process", Name: "Critical Process"}) ||
		v.Symbols[1] != (Symbol{Key: "revision", Name: "Revision", Automatic: true}) || v.Symbols[2].Automatic || v.Symbols[3].Automatic {
		t.Errorf("simbol PFMEA: %+v", v.Symbols)
	}

	cp := f.version(t, "Control Plan v1")
	wantCP := []string{"Process Flow", "No", "Process Name / Operation Description",
		"Machine, Equipment, Jigs, Fixture, Tools for Mfg", "Product Characteristics", "Product Specification / Tolerance",
		"Process Characteristics", "Process Specification / Tolerance", "Special Characteristics", "Equipment", "Method",
		"Frequency", "PIC", "Relevant Document", "Pokayoke", "Reaction Plan"}
	if got := labels(cp, AreaRow); cp.RiskMethod != RiskNone || len(cp.Ratings) != 0 || !slices.Equal(got, wantCP) {
		t.Errorf("CP: metode %s, %d tabel skala, kolom %v", cp.RiskMethod, len(cp.Ratings), got)
	}
	// Process Flow dibentuk dari PFD dan selalu tercetak (D-122, R-DOC-09).
	if flow := field(t, cp, "Process Flow"); flow.Kind != KindProcessFlow {
		t.Errorf("Process Flow CP: %+v", flow)
	}
	if special := field(t, cp, "Special Characteristics"); special.Kind != KindSymbol {
		t.Errorf("Special Characteristics: %+v", special)
	}
	// Tidak ada kolom isi awal yang disembunyikan.
	for _, version := range []FormatVersion{v, cp} {
		for _, fl := range version.Fields {
			if fl.Hidden {
				t.Errorf("%s tersembunyi", fl.Label)
			}
		}
	}
	if !cp.Symbols[1].Automatic {
		t.Errorf("simbol Revision di CP harus otomatis: %+v", cp.Symbols)
	}
	pfd := f.version(t, "PFD v1")
	if got := labels(pfd, AreaRow); !slices.Equal(got, []string{"No", "Process", "Flow Symbol", "NG Disposition"}) {
		t.Errorf("kolom PFD: %v", got)
	}
	if flow := field(t, pfd, "Flow Symbol"); flow.Kind != KindSelect || len(flow.Options) != 6 || flow.Options[1] != (Option{Key: "alternate_process", Label: "Alternate Process"}) {
		t.Errorf("Flow Symbol: %+v", flow)
	}
}

// TestRenameKeepsData adalah kriteria selesai M3.1: label kolom bisa diganti
// tanpa merusak data. Isi baris memakai ID field, jadi isi yang sah tetap sah
// setelah label diganti, kolom disembunyikan, atau kolom baru ditambah
// (R-TPL-02).
func TestRenameKeepsData(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	v := f.version(t, "PFMEA v1")
	mode, sev, occ, det := field(t, v, "Potential Failure Mode"), field(t, v, "Severity"), field(t, v, "Occurrence"), field(t, v, "Detection")
	req, class := field(t, v, "Requirement"), field(t, v, "Class")
	content := map[string]any{
		mode.ID: "Solder bridge", sev.ID: 9.0, occ.ID: 3.0, det.ID: 4.0,
		req.ID: "IPC-A-610 class 2", class.ID: "critical_process",
	}
	if c := CheckContent(v, AreaRow, content); !c.Valid() {
		t.Fatalf("isi awal tidak sah: %+v", c)
	}
	if got := RPN(v, content)[field(t, v, "RPN").ID]; got != 108 {
		t.Errorf("RPN %d, ingin 108", got)
	}

	// Ganti label: ID tetap, isi tetap sah.
	v, err := f.s.UpdateField(ctx, f.master, v.ID, mode.ID, FieldUpdate{Label: "Failure Mode"})
	if err != nil {
		t.Fatalf("ganti label: %v", err)
	}
	if renamed := field(t, v, "Failure Mode"); renamed.ID != mode.ID {
		t.Errorf("ID berubah setelah ganti label: %s menjadi %s", mode.ID, renamed.ID)
	}
	// Sembunyikan kolom: datanya tetap sah, tidak dihapus.
	v, err = f.s.UpdateField(ctx, f.master, v.ID, req.ID, FieldUpdate{Label: req.Label, Hidden: new(true)})
	if err != nil {
		t.Fatalf("sembunyikan kolom: %v", err)
	}
	// Kolom wajib baru: isi lama tetap sah, tapi ditandai belum lengkap.
	v, err = f.s.AddField(ctx, f.master, v.ID, NewField{Area: AreaRow, Kind: KindText, Label: "Customer Requirement Ref", Required: true})
	if err != nil {
		t.Fatalf("tambah kolom: %v", err)
	}
	added := field(t, v, "Customer Requirement Ref")
	c := CheckContent(v, AreaRow, content)
	if !c.Valid() || !slices.Equal(c.Missing, []string{added.ID}) {
		t.Errorf("isi lama setelah label diganti, kolom disembunyikan, dan kolom wajib ditambah: %+v", c)
	}
	// Perubahan tanpa hidden (D-111) tidak memunculkan kembali kolom itu.
	v, err = f.s.UpdateField(ctx, f.master, v.ID, req.ID, FieldUpdate{Label: "Requirements (old)"})
	if err != nil {
		t.Fatal(err)
	}
	if added.Position != 21 || !field(t, v, "Requirements (old)").Hidden {
		t.Errorf("kolom baru di urutan %d; Requirements tersembunyi %v", added.Position, field(t, v, "Requirements (old)").Hidden)
	}
}

// TestFieldRules: isian field yang salah ditolak per field.
func TestFieldRules(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	pfmeaV := f.version(t, "PFMEA v1")
	cp := f.version(t, "Control Plan v1")
	mode := field(t, pfmeaV, "Potential Failure Mode")
	pfd := f.version(t, "PFD v1")
	flow := field(t, pfd, "Flow Symbol")

	adds := []struct {
		name string
		v    FormatVersion
		in   NewField
		want []string
	}{
		{"semua kosong", pfmeaV, NewField{}, []string{"area:required", "kind:required", "label:required"}},
		{"jenis tidak dikenal", pfmeaV, NewField{Area: AreaRow, Kind: "image", Label: "Foto"}, []string{"kind:invalid_format"}},
		{"rating di header", pfmeaV, NewField{Area: AreaHeader, Kind: KindRating, Label: "S", RatingScale: ScaleSeverity, RatingSet: 1}, []string{"kind:not_allowed"}},
		{"rating tanpa skala", pfmeaV, NewField{Area: AreaRow, Kind: KindRating, Label: "S baru", RatingSet: 1}, []string{"rating_scale:required"}},
		{"rating ganda", pfmeaV, NewField{Area: AreaRow, Kind: KindRating, Label: "S baru", RatingScale: ScaleSeverity, RatingSet: 1}, []string{"rating_scale:taken"}},
		{"RPN di CP", cp, NewField{Area: AreaRow, Kind: KindRPN, Label: "RPN", RatingSet: 1}, []string{"kind:not_allowed"}},
		// Kolom terisi otomatis hanya kolom baris, satu per jenis (D-122, D-123).
		{"proses di header", pfmeaV, NewField{Area: AreaHeader, Kind: KindProcess, Label: "Proses"}, []string{"kind:not_allowed"}},
		{"kolom proses ganda", pfmeaV, NewField{Area: AreaRow, Kind: KindProcess, Label: "Process 2"}, []string{"kind:taken"}},
		{"process flow ganda", cp, NewField{Area: AreaRow, Kind: KindProcessFlow, Label: "Flow 2"}, []string{"kind:taken"}},
		{"pilihan kosong", pfmeaV, NewField{Area: AreaRow, Kind: KindSelect, Label: "Phase"}, []string{"options:required"}},
		{"pilihan kembar", pfmeaV, NewField{Area: AreaRow, Kind: KindSelect, Label: "Phase", Options: []Option{{Label: "Production"}, {Label: "production"}}}, []string{"options:taken"}},
	}
	for _, tt := range adds {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.AddField(ctx, f.master, tt.v.ID, tt.in)
			if got := problemCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("AddField = %v, ingin %v", got, tt.want)
			}
		})
	}
	if _, err := f.s.AddField(ctx, f.master, pfmeaV.ID, NewField{Area: AreaRow, Kind: KindText, Label: "potential failure mode"}); !slices.Equal(problemCodes(t, err), []string{"label:taken"}) {
		t.Errorf("label kembar (beda huruf besar kecil): %v", err)
	}
	// Template kosong boleh menambah kolom terisi otomatis, misalnya format CP
	// customer yang dibuat dari nol.
	empty, err := f.s.CreateFormatTemplate(ctx, f.master, NewTemplate{Name: "CP kosong", DocumentTypeID: f.template(t, "Control Plan v1").DocumentTypeID, RiskMethod: RiskNone})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := f.s.AddField(ctx, f.master, empty.Versions[0].ID, NewField{Area: AreaRow, Kind: KindProcessFlow, Label: "Process Flow"})
	if err != nil || len(ev.Fields) != 1 || ev.Fields[0].Kind != KindProcessFlow {
		t.Errorf("tambah Process Flow: %+v, %v", ev.Fields, err)
	}

	updates := []struct {
		name  string
		v     FormatVersion
		field Field
		in    FieldUpdate
		want  []string
	}{
		{"label kosong", pfmeaV, mode, FieldUpdate{}, []string{"label:required"}},
		{"tersembunyi dan wajib", pfmeaV, mode, FieldUpdate{Label: "Mode", Hidden: new(true), Required: new(true)}, []string{"required:not_allowed"}},
		{"urutan di luar", pfmeaV, mode, FieldUpdate{Label: "Mode", Position: new(21)}, []string{"position:invalid_format"}},
		{"pilihan dihapus", pfd, flow, FieldUpdate{Label: "Flow Symbol", Options: flow.Options[:5]}, []string{"options:not_allowed"}},
		{"pilihan untuk teks", pfmeaV, mode, FieldUpdate{Label: "Mode", Options: []Option{{Label: "A"}}}, []string{"options:not_allowed"}},
	}
	for _, tt := range updates {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.UpdateField(ctx, f.master, tt.v.ID, tt.field.ID, tt.in)
			if got := problemCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("UpdateField = %v, ingin %v", got, tt.want)
			}
		})
	}

	// Pilihan baru mendapat key dari labelnya; label lama boleh diganti.
	options := append(slices.Clone(flow.Options), Option{Label: "Inspection"})
	options[0].Label = "Operation"
	v, err := f.s.UpdateField(ctx, f.master, pfd.ID, flow.ID, FieldUpdate{Label: "Flow Symbol", Options: options})
	if err != nil {
		t.Fatal(err)
	}
	got := field(t, v, "Flow Symbol").Options
	if len(got) != 7 || got[0] != (Option{Key: "process", Label: "Operation"}) || got[6] != (Option{Key: "inspection", Label: "Inspection"}) {
		t.Errorf("pilihan setelah diubah: %+v", got)
	}
}

// TestMoveField: memindahkan kolom menggeser kolom lain, dan urutan tetap
// bersambung.
func TestMoveField(t *testing.T) {
	f := newFixture(t)
	v := f.version(t, "Control Plan v1")
	spec := field(t, v, "Product Specification / Tolerance")
	v, err := f.s.UpdateField(t.Context(), f.master, v.ID, spec.ID, FieldUpdate{Label: spec.Label, Position: new(2)})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Process Flow", "Product Specification / Tolerance", "No", "Process Name / Operation Description",
		"Machine, Equipment, Jigs, Fixture, Tools for Mfg"}
	if got := labels(v, AreaRow)[:5]; !slices.Equal(got, want) {
		t.Errorf("urutan setelah dipindah ke 2: %v", got)
	}
	v, err = f.s.UpdateField(t.Context(), f.master, v.ID, spec.ID, FieldUpdate{Label: spec.Label, Position: new(16)})
	if err != nil {
		t.Fatal(err)
	}
	got := labels(v, AreaRow)
	if got[15] != "Product Specification / Tolerance" || got[1] != "No" {
		t.Errorf("urutan setelah dipindah ke akhir: %v", got)
	}
	for i, fd := range v.Fields {
		if fd.Area == AreaRow && fd.Position != i-1 { // dua field header di depan
			t.Errorf("urutan %s = %d", fd.Label, fd.Position)
		}
	}
}

// TestCreateTemplate: format template baru dari nol (misalnya SQAM satu
// customer dengan AP) atau salinan template lain dengan ID field baru.
func TestCreateTemplate(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	customers := customer.NewService(f.pool, access.NewChecker(f.pool))
	group, err := customers.CreateProjectGroup(ctx, f.master, customer.ProjectGroupInput{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	kt, err := customers.Create(ctx, f.master, customer.NewCustomer{Code: "KT", Name: "Customer KT", ProjectGroupID: group.ID})
	if err != nil {
		t.Fatal(err)
	}
	pfmeaT := f.template(t, "PFMEA v1")
	cpT := f.template(t, "Control Plan v1")

	sqam, err := f.s.CreateFormatTemplate(ctx, f.master, NewTemplate{
		Name: "PFMEA SQAM KT", DocumentTypeID: pfmeaT.DocumentTypeID, Methodology: MethodSQAM, CustomerID: kt.ID, RiskMethod: RiskAP,
	})
	if err != nil {
		t.Fatalf("template baru: %v", err)
	}
	if sqam.CustomerCode != "KT" || sqam.Methodology != MethodSQAM || len(sqam.Versions) != 1 || sqam.Versions[0].RiskMethod != RiskAP {
		t.Errorf("template SQAM: %+v", sqam)
	}
	v, err := f.s.GetFormatVersion(ctx, f.author, sqam.Versions[0].ID)
	if err != nil || len(v.Fields) != 0 || len(v.Ratings) != 30 || len(v.Symbols) != 0 {
		t.Errorf("versi baru: %d field, %d tabel skala, %d simbol, %v", len(v.Fields), len(v.Ratings), len(v.Symbols), err)
	}

	copied, err := f.s.CreateFormatTemplate(ctx, f.master, NewTemplate{
		Name: "PFMEA v1 copy", DocumentTypeID: pfmeaT.DocumentTypeID, Methodology: MethodAIAG4th, CopyFromVersionID: pfmeaT.Versions[0].ID,
	})
	if err != nil {
		t.Fatalf("salinan: %v", err)
	}
	cv, err := f.s.GetFormatVersion(ctx, f.author, copied.Versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	source := f.version(t, "PFMEA v1")
	if !slices.Equal(labels(cv, AreaRow), labels(source, AreaRow)) || cv.RiskMethod != RiskRPN || cv.HighlightRPNAbove != 100 || len(cv.Ratings) != 30 || len(cv.Symbols) != 4 {
		t.Errorf("salinan: %+v", cv)
	}
	if cv.Fields[0].ID == source.Fields[0].ID {
		t.Errorf("salinan memakai ID field template sumber")
	}
	if !cv.Symbols[1].Automatic || cv.Symbols[0].Automatic {
		t.Errorf("salinan harus membawa tanda simbol otomatis: %+v", cv.Symbols)
	}

	tests := []struct {
		name string
		in   NewTemplate
		want []string
	}{
		{"semua kosong", NewTemplate{}, []string{"name:required", "document_type_id:required", "risk_method:required"}},
		{"nama dipakai", NewTemplate{Name: "pfd V1", DocumentTypeID: pfmeaT.DocumentTypeID, RiskMethod: RiskRPN}, []string{"name:taken"}},
		{"metodologi tidak dikenal", NewTemplate{Name: "X", DocumentTypeID: pfmeaT.DocumentTypeID, Methodology: "iso", RiskMethod: RiskRPN}, []string{"methodology:invalid_format"}},
		{"salinan jenis lain", NewTemplate{Name: "X", DocumentTypeID: cpT.DocumentTypeID, CopyFromVersionID: pfmeaT.Versions[0].ID}, []string{"copy_from_version_id:not_allowed"}},
		{"customer tidak ada", NewTemplate{Name: "X", DocumentTypeID: cpT.DocumentTypeID, CustomerID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", RiskMethod: RiskNone}, []string{"customer_id:invalid_format"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.CreateFormatTemplate(ctx, f.master, tt.in)
			if got := problemCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("CreateFormatTemplate = %v, ingin %v", got, tt.want)
			}
		})
	}

	renamed, err := f.s.RenameFormatTemplate(ctx, f.master, copied.ID, "PFMEA customer KT")
	if err != nil || renamed.Name != "PFMEA customer KT" {
		t.Errorf("ganti nama: %+v, %v", renamed, err)
	}
}

// TestRatingsSymbolsAndHighlights: teks kriteria, simbol, dan aturan sorot
// diubah per versi draft.
func TestRatingsSymbolsAndHighlights(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	v := f.version(t, "PFMEA v1")

	v, err := f.s.SetRatingCriteria(ctx, f.master, v.ID, []Rating{
		{Scale: ScaleSeverity, Value: 10, Criteria: "Hazardous without warning (contoh)"},
		{Scale: ScaleDetection, Value: 1, Criteria: "  Almost certain  "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if v.Ratings[9].Criteria != "Hazardous without warning (contoh)" || v.Ratings[20].Criteria != "Almost certain" {
		t.Errorf("kriteria: %+v dan %+v", v.Ratings[9], v.Ratings[20])
	}
	if _, err := f.s.SetRatingCriteria(ctx, f.master, v.ID, []Rating{{Scale: "x", Value: 11}}); !slices.Equal(problemCodes(t, err), []string{"ratings:invalid_format"}) {
		t.Errorf("skala salah: %v", err)
	}
	cp := f.version(t, "Control Plan v1")
	if _, err := f.s.SetRatingCriteria(ctx, f.master, cp.ID, []Rating{{Scale: ScaleSeverity, Value: 1, Criteria: "x"}}); !slices.Equal(problemCodes(t, err), []string{"ratings:not_allowed"}) {
		t.Errorf("tabel skala untuk CP tanpa metode risiko: %v", err)
	}

	symbols := append(slices.Clone(v.Symbols), Symbol{Name: "Safety"})
	symbols[1].Name = "Revision Mark"
	v, err = f.s.SaveSymbols(ctx, f.master, v.ID, symbols)
	if err != nil {
		t.Fatal(err)
	}
	// Ganti nama tidak mengubah tanda otomatis, dan simbol baru tidak pernah otomatis.
	if len(v.Symbols) != 5 || v.Symbols[1] != (Symbol{Key: "revision", Name: "Revision Mark", Automatic: true}) || v.Symbols[4] != (Symbol{Key: "safety", Name: "Safety"}) {
		t.Errorf("simbol: %+v", v.Symbols)
	}
	symbols = slices.Clone(v.Symbols)
	symbols[4].Automatic = true
	if v, err = f.s.SaveSymbols(ctx, f.master, v.ID, symbols); err != nil || v.Symbols[4].Automatic {
		t.Errorf("tanda otomatis dari isian diabaikan: %+v, %v", v.Symbols, err)
	}
	if _, err := f.s.SaveSymbols(ctx, f.master, v.ID, v.Symbols[1:]); !slices.Equal(problemCodes(t, err), []string{"symbols:not_allowed"}) {
		t.Errorf("simbol dihapus: %v", err)
	}

	v, err = f.s.SetHighlights(ctx, f.master, v.ID, HighlightSettings{RPNAbove: new(120), SeverityAtLeast: new(0)})
	if err != nil || v.HighlightRPNAbove != 120 || v.HighlightSeverityAtLeast != 0 {
		t.Errorf("aturan sorot: %d, %d, %v", v.HighlightRPNAbove, v.HighlightSeverityAtLeast, err)
	}
	if _, err := f.s.SetHighlights(ctx, f.master, cp.ID, HighlightSettings{RPNAbove: new(100)}); !slices.Equal(problemCodes(t, err), []string{"highlight_rpn_above:not_allowed"}) {
		t.Errorf("aturan sorot untuk CP: %v", err)
	}
}

// TestPermissionsAndAudit: semua akun boleh melihat (L-2, D-103), hanya admin
// (A-2) yang mengubah, dan setiap perubahan tercatat di audit log.
func TestPermissionsAndAudit(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	v := f.version(t, "PFD v1")
	ft := f.template(t, "PFD v1")
	if _, err := f.s.AddField(ctx, f.author, v.ID, NewField{Area: AreaRow, Kind: KindText, Label: "Note"}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun menambah kolom: %v", err)
	}
	if _, err := f.s.RenameFormatTemplate(ctx, f.author, ft.ID, "PFD baru"); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun mengganti nama: %v", err)
	}
	if _, err := f.s.GetFormatVersion(ctx, f.author, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("versi tidak ada: %v", err)
	}

	if _, err := f.s.AddField(ctx, f.master, v.ID, NewField{Area: AreaRow, Kind: KindText, Label: "Note"}); err != nil {
		t.Fatal(err)
	}
	var action, details string
	err := f.pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details)
	if err != nil || action != "format_template.field_added" || details != `{"area":"row","kind":"text","label":"Note","version":"1"}` {
		t.Errorf("entri audit: %s %s, %v", action, details, err)
	}
}

// TestDatabaseGuards: role aplikasi tidak bisa menghapus field atau mengubah
// jenisnya, jadi kolom hanya bisa disembunyikan dan data lama tetap terbaca
// (R-TPL-02).
func TestDatabaseGuards(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	for _, stmt := range []string{
		"DELETE FROM format_version_fields WHERE false",
		"DELETE FROM format_fields WHERE false",
		"UPDATE format_fields SET kind = kind WHERE false",
		"UPDATE format_template_versions SET risk_method = risk_method WHERE false",
		// Tanda simbol otomatis hanya diatur migrasi (R-REV-09).
		"UPDATE format_symbols SET automatic = automatic WHERE false",
		"DELETE FROM format_templates WHERE false",
	} {
		_, err := f.pool.Exec(ctx, stmt)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("%s: %v, ingin permission denied (42501)", stmt, err)
		}
	}
}
