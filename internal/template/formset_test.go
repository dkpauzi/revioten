package template

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
)

// Form set langkah D (D-133, D-139, D-140): satu form set adalah paket versi
// format template untuk PFD, PFMEA, dan Control Plan. Setiap customer punya
// form set bawaan untuk dokumen baru; tanpa pilihan berarti set AIAG 4th.
// Form set dan versi format bermetode AP diterbitkan pemegang hak V-3
// (Penyetuju form), dan bukan oleh orang yang menyusun drafnya.

// formSetFixture menambah akun Penyetuju form dan satu customer.
type formSetFixture struct {
	fixture
	approver access.Actor
	customer string
}

func newFormSetFixture(t *testing.T) formSetFixture {
	t.Helper()
	f := formSetFixture{fixture: newFixture(t)}
	f.approver = newActor(t, f.pool, "ppap.supervisor", access.AccountWork, access.RoleFormApprover)
	q := db.New(f.pool)
	pg, err := q.CreateProjectGroup(t.Context(), db.CreateProjectGroupParams{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := q.CreateCustomer(t.Context(), db.CreateCustomerParams{Code: "AB", Name: "Customer AB", ProjectGroupID: &pg.ID})
	if err != nil {
		t.Fatal(err)
	}
	f.customer = c.ID
	return f
}

// set mencari form set menurut key.
func (f formSetFixture) set(t *testing.T, key string) FormSet {
	t.Helper()
	list, err := f.s.ListFormSets(t.Context(), f.author)
	if err != nil {
		t.Fatal(err)
	}
	for _, fs := range list.Sets {
		if fs.Key == key {
			return fs
		}
	}
	t.Fatalf("form set %q tidak ada", key)
	return FormSet{}
}

// sheetNames menulis nama template ketiga sheet satu versi form set.
func sheetNames(v FormSetVersion) []string {
	return []string{v.PFD.TemplateName, v.PFMEA.TemplateName, v.CP.TemplateName}
}

// TestSeededFormSets: dua set bawaan, PFD dipakai bersama.
func TestSeededFormSets(t *testing.T) {
	f := newFormSetFixture(t)
	list, err := f.s.ListFormSets(t.Context(), f.author)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Sets) != 2 {
		t.Fatalf("%d form set, ingin 2", len(list.Sets))
	}
	old, vda := f.set(t, "aiag_4th"), f.set(t, "aiag_vda")
	if old.Name != "AIAG 4th set" || !old.Default || vda.Name != "AIAG-VDA set" || vda.Default {
		t.Errorf("set: %+v dan %+v", old, vda)
	}
	if got := sheetNames(old.Versions[0]); !slices.Equal(got, []string{"PFD v1", "PFMEA v1", "Control Plan v1"}) {
		t.Errorf("isi set AIAG 4th: %v", got)
	}
	if got := sheetNames(vda.Versions[0]); !slices.Equal(got, []string{"PFD v1", "PFMEA v2", "Control Plan v2"}) {
		t.Errorf("isi set AIAG-VDA: %v", got)
	}
	if old.Versions[0].PFD.FormatVersionID != vda.Versions[0].PFD.FormatVersionID {
		t.Errorf("PFD tidak dipakai bersama oleh kedua set")
	}
	for _, fs := range list.Sets {
		if v := fs.Versions[0]; len(fs.Versions) != 1 || v.Version != 1 || v.Status != StatusDraft {
			t.Errorf("%s: versi %+v", fs.Key, fs.Versions)
		}
	}
	// Customer tanpa pilihan memakai set bawaan (set AIAG 4th).
	if len(list.Defaults) != 1 || list.Defaults[0].CustomerCode != "AB" || list.Defaults[0].FormSetID != "" {
		t.Errorf("bawaan customer: %+v", list.Defaults)
	}
	fs, err := f.s.FormSetFor(t.Context(), f.author, f.customer)
	if err != nil || fs.Key != "aiag_4th" {
		t.Errorf("form set customer tanpa pilihan: %s, %v", fs.Key, err)
	}
}

// TestCustomerDefaultFormSet: admin memilih form set bawaan customer.
func TestCustomerDefaultFormSet(t *testing.T) {
	f := newFormSetFixture(t)
	ctx := t.Context()
	vda := f.set(t, "aiag_vda")
	if err := f.s.SetCustomerFormSet(ctx, f.author, f.customer, vda.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun memilih form set customer: %v", err)
	}
	if err := f.s.SetCustomerFormSet(ctx, f.master, f.customer, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"); !slices.Equal(problemCodes(t, err), []string{"form_set_id:invalid_format"}) {
		t.Errorf("form set tidak ada: %v", err)
	}
	// Set AIAG-VDA tersembunyi sejak D-218; ditampilkan dulu supaya bisa dipilih.
	if _, err := f.s.SetFormSetHidden(ctx, f.master, vda.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := f.s.SetCustomerFormSet(ctx, f.master, f.customer, vda.ID); err != nil {
		t.Fatal(err)
	}
	if action, details := lastAudit(t, f.fixture); action != "customer.default_form_set_changed" || details != `{"form_set":"- → AIAG-VDA set"}` {
		t.Errorf("entri audit: %s %s", action, details)
	}
	if fs, err := f.s.FormSetFor(ctx, f.author, f.customer); err != nil || fs.Key != "aiag_vda" {
		t.Errorf("form set customer: %s, %v", fs.Key, err)
	}
	// Kosong berarti kembali ke set bawaan.
	if err := f.s.SetCustomerFormSet(ctx, f.master, f.customer, ""); err != nil {
		t.Fatal(err)
	}
	if fs, _ := f.s.FormSetFor(ctx, f.author, f.customer); fs.Key != "aiag_4th" {
		t.Errorf("form set customer sesudah dikosongkan: %s", fs.Key)
	}
}

// TestFormSetVersions: admin menyusun, Penyetuju form menerbitkan; versi
// terbit tidak berubah, dan isinya harus versi format yang sudah terbit.
func TestFormSetVersions(t *testing.T) {
	f := newFormSetFixture(t)
	ctx := t.Context()
	old := f.set(t, "aiag_4th")
	v1 := old.Versions[0]
	notAllowed := []string{"status:not_allowed"}

	// Hanya pemegang V-3 yang menerbitkan; admin dan Penyusun tidak.
	for _, actor := range []access.Actor{f.master, f.author} {
		if _, err := f.s.PublishFormSetVersion(ctx, actor, v1.ID); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s menerbitkan form set: %v", actor.Login, err)
		}
	}
	// Versi format di dalamnya masih draft: form set belum bisa terbit.
	if _, err := f.s.PublishFormSetVersion(ctx, f.approver, v1.ID); !slices.Equal(problemCodes(t, err), []string{"sheets:not_allowed"}) {
		t.Errorf("menerbitkan form set berisi format draft: %v", err)
	}
	for _, id := range []string{v1.PFD.FormatVersionID, v1.PFMEA.FormatVersionID, v1.CP.FormatVersionID} {
		if _, err := f.s.PublishFormatVersion(ctx, f.master, id); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := f.s.PublishFormSetVersion(ctx, f.approver, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v := fs.Versions[0]; v.Status != StatusPublished || v.PublishedAt == nil {
		t.Errorf("sesudah terbit: %+v", v)
	}
	if action, details := lastAudit(t, f.fixture); action != "form_set.version_published" || details != `{"version":"1"}` {
		t.Errorf("entri audit: %s %s", action, details)
	}
	if _, err := f.s.PublishFormSetVersion(ctx, f.approver, v1.ID); !slices.Equal(problemCodes(t, err), notAllowed) {
		t.Errorf("menerbitkan dua kali: %v", err)
	}
	sheets := FormSetSheets{PFD: v1.PFD.FormatVersionID, PFMEA: v1.PFMEA.FormatVersionID, CP: v1.CP.FormatVersionID}
	if _, err := f.s.SetFormSetSheets(ctx, f.master, v1.ID, sheets); !slices.Equal(problemCodes(t, err), notAllowed) {
		t.Errorf("mengubah versi form set yang sudah terbit: %v", err)
	}

	// Versi baru menyalin isi versi terbit; admin mengganti sheet CP.
	if _, err := f.s.NewFormSetVersion(ctx, f.author, old.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun membuat versi form set: %v", err)
	}
	fs, err = f.s.NewFormSetVersion(ctx, f.master, old.ID)
	if err != nil {
		t.Fatal(err)
	}
	v2 := fs.Versions[1]
	if v2.Version != 2 || v2.Status != StatusDraft || !slices.Equal(sheetNames(v2), sheetNames(v1)) {
		t.Fatalf("versi 2: %+v", v2)
	}
	if _, err := f.s.NewFormSetVersion(ctx, f.master, old.ID); !slices.Equal(problemCodes(t, err), notAllowed) {
		t.Errorf("versi baru saat masih ada draft: %v", err)
	}
	cp2024 := f.version(t, "Control Plan v2")
	sheets.CP = cp2024.ID
	fs, err = f.s.SetFormSetSheets(ctx, f.master, v2.ID, sheets)
	if err != nil {
		t.Fatal(err)
	}
	if got := sheetNames(fs.Versions[1]); !slices.Equal(got, []string{"PFD v1", "PFMEA v1", "Control Plan v2"}) {
		t.Errorf("isi versi 2: %v", got)
	}
	if got := sheetNames(fs.Versions[0]); !slices.Equal(got, []string{"PFD v1", "PFMEA v1", "Control Plan v1"}) {
		t.Errorf("isi versi 1 ikut berubah: %v", got)
	}
	// Sheet harus berisi format dengan jenis dokumen yang cocok.
	wrong := FormSetSheets{PFD: v1.PFMEA.FormatVersionID, PFMEA: v1.PFMEA.FormatVersionID, CP: v1.CP.FormatVersionID}
	if _, err := f.s.SetFormSetSheets(ctx, f.master, v2.ID, wrong); !slices.Equal(problemCodes(t, err), []string{"pfd_format_version_id:not_allowed"}) {
		t.Errorf("format PFMEA di sheet PFD: %v", err)
	}
}

// TestCreateFormSet: admin membuat form set baru; nama harus unik.
func TestCreateFormSet(t *testing.T) {
	f := newFormSetFixture(t)
	ctx := t.Context()
	v1 := f.set(t, "aiag_4th").Versions[0]
	in := NewFormSet{Name: "  Customer AB set ", Sheets: FormSetSheets{PFD: v1.PFD.FormatVersionID, PFMEA: v1.PFMEA.FormatVersionID, CP: v1.CP.FormatVersionID}}
	if _, err := f.s.CreateFormSet(ctx, f.author, in); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun membuat form set: %v", err)
	}
	fs, err := f.s.CreateFormSet(ctx, f.master, in)
	if err != nil {
		t.Fatal(err)
	}
	if fs.Name != "Customer AB set" || fs.Key != "customer_ab_set" || fs.Default || len(fs.Versions) != 1 || fs.Versions[0].Status != StatusDraft {
		t.Errorf("form set baru: %+v", fs)
	}
	if _, err := f.s.CreateFormSet(ctx, f.master, in); !slices.Equal(problemCodes(t, err), []string{"name:taken"}) {
		t.Errorf("nama kembar: %v", err)
	}
	in.Name = ""
	in.Sheets.CP = ""
	if _, err := f.s.CreateFormSet(ctx, f.master, in); !slices.Equal(problemCodes(t, err), []string{"name:required", "cp_format_version_id:required"}) {
		t.Errorf("isian kosong: %v", err)
	}
}

// TestSeparationOfDuties: penyusun draft tidak menerbitkan form set atau
// tabel AP yang ia susun sendiri, walaupun memegang hak V-3 (D-139). Sejak
// D-209 PPAP Engineer Supervisor memegang E-11 (menyusun) dan V-3 (menerbitkan)
// di satu akun, jadi aturan ini yang menjaga.
func TestSeparationOfDuties(t *testing.T) {
	f := newFormSetFixture(t)
	ctx := t.Context()
	same := newActor(t, f.pool, "ppap.supervisor.uji", access.AccountWork, access.RolePPAPEngineerSupervisor)

	// Tabel AP: Supervisor mengubah, lalu orang yang sama tidak boleh menerbitkan.
	vda := f.version(t, "PFMEA v2")
	changed := DefaultAPRules()
	changed[0].Priority = "M"
	if _, err := f.s.SetAPRules(ctx, same, vda.ID, changed); err != nil {
		t.Fatal(err)
	}
	// PPAP Engineer tanpa V-3 tidak menerbitkan versi bermetode AP.
	if _, err := f.s.PublishFormatVersion(ctx, f.master, vda.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("PPAP Engineer menerbitkan versi AP: %v", err)
	}
	if _, err := f.s.PublishFormatVersion(ctx, same, vda.ID); !slices.Equal(problemCodes(t, err), []string{"approver:not_allowed"}) {
		t.Errorf("penyusun menerbitkan tabel AP sendiri: %v", err)
	}
	if v, err := f.s.PublishFormatVersion(ctx, f.approver, vda.ID); err != nil || v.Status != StatusPublished {
		t.Errorf("Penyetuju form menerbitkan versi AP: %v", err)
	}

	// Form set: Supervisor membuat versi, orang yang sama tidak boleh menerbitkannya.
	v1 := f.set(t, "aiag_4th").Versions[0]
	for _, id := range []string{v1.PFD.FormatVersionID, v1.PFMEA.FormatVersionID, v1.CP.FormatVersionID} {
		if _, err := f.s.PublishFormatVersion(ctx, f.master, id); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := f.s.CreateFormSet(ctx, same, NewFormSet{Name: "Set uji", Sheets: FormSetSheets{PFD: v1.PFD.FormatVersionID, PFMEA: v1.PFMEA.FormatVersionID, CP: v1.CP.FormatVersionID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.PublishFormSetVersion(ctx, same, fs.Versions[0].ID); !slices.Equal(problemCodes(t, err), []string{"approver:not_allowed"}) {
		t.Errorf("penyusun menerbitkan form set sendiri: %v", err)
	}
	if _, err := f.s.PublishFormSetVersion(ctx, f.approver, fs.Versions[0].ID); err != nil {
		t.Errorf("Penyetuju form menerbitkan form set: %v", err)
	}
}
