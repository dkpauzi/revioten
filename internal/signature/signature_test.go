package signature_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/notify"
	"github.com/dkpauzi/revioten/internal/printout"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/signature"
	"github.com/dkpauzi/revioten/internal/storage"
)

// Fase 3 · Tanda tangan digital (D-198, D-203, D-204).

func TestMain(m *testing.M) {
	audit.SetListener(notify.NewNotifier("http://contoh.test").Strict().Listener())
	os.Exit(dbtest.Main(m))
}

// passwords palsu: hanya "benar" yang diterima.
type passwords struct{}

func (passwords) ConfirmPassword(_ context.Context, _ access.Actor, pw string) error {
	if pw == "benar" {
		return nil
	}
	return &problem.ValidationError{Problems: []problem.Problem{{Field: "password", Code: problem.Incorrect, Message: "salah"}}}
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) || len(invalid.Problems) == 0 {
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
}

type world struct {
	doctest.Env
	svc                             *signature.Service
	sender, pe, pl, npi, qa, record access.Actor
	cp                              document.Document
}

// setup: rute "Up to QA QC Manager" untuk customer AB, pemegang jabatan, dan
// satu Control Plan dengan satu baris dari pengirim.
func setup(t *testing.T) world {
	t.Helper()
	e := doctest.Setup(t)
	ctx := t.Context()
	w := world{Env: e, svc: signature.NewService(e.Pool, e.Checker, passwords{})}
	w.sender = doctest.NewActor(t, e.Pool, "ppap.engineer", "work", doctest.All(access.RolePPAPEngineer))
	w.record = doctest.NewActor(t, e.Pool, "ppap.dua", "work", doctest.All(access.RolePPAPEngineer))
	group := access.RoleGrant{Role: access.RoleProjectLeader, Scope: access.ScopeOwnProjectGroup}
	w.pe = doctest.NewActor(t, e.Pool, "pe.gr1", "work", group)
	w.pl = doctest.NewActor(t, e.Pool, "pl.gr1", "work", group)
	w.npi = doctest.NewActor(t, e.Pool, "npi.manager", "work", doctest.All(access.RoleManager))
	w.qa = doctest.NewActor(t, e.Pool, "qa.manager", "work", doctest.All(access.RoleManager))

	hold := func(person, key string, group *string) {
		t.Helper()
		_, err := e.Pool.Exec(ctx, `INSERT INTO position_holders (position_id, per_project_group, person_id, project_group_id)
			SELECT id, per_project_group, $1, $3 FROM positions WHERE key = $2`, person, key, group)
		if err != nil {
			t.Fatal(err)
		}
	}
	hold(w.sender.PersonID, "ppap_engineer", nil)
	hold(w.pe.PersonID, "project_engineer", &e.GroupID)
	hold(w.pl.PersonID, "project_leader", &e.GroupID)
	hold(w.npi.PersonID, "npi_manager", nil)
	hold(w.qa.PersonID, "qa_qc_manager", nil)
	if _, err := e.Pool.Exec(ctx, `UPDATE customers SET default_route_id = (SELECT id FROM approval_routes WHERE name = 'Up to QA QC Manager')`); err != nil {
		t.Fatal(err)
	}
	w.cp = e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)["cp"]
	if _, err := e.Docs.AddRow(ctx, w.sender, w.cp.ID, document.NewRow{Content: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func names(c []signature.Candidate) []string {
	out := []string{}
	for _, x := range c {
		out = append(out, x.Name)
	}
	return out
}

func TestPlan(t *testing.T) {
	w := setup(t)
	p, err := w.svc.Plan(t.Context(), w.sender, w.cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != signature.ModeDigital || p.RouteName != "Up to QA QC Manager" || len(p.Slots) != 4 {
		t.Fatalf("rencana: %+v", p)
	}
	want := [][]string{
		{"Orang pe.gr1", "Orang ppap.engineer"}, // Prepared: pengirim atau Project Leader tingkat Leader
		{"Orang pe.gr1", "Orang pl.gr1"},        // Checked: Leader atau Supervisor
		{"Orang npi.manager"},
		{"Orang qa.manager"},
	}
	for i, slot := range p.Slots {
		got := names(slot.Candidates)
		if len(got) != len(want[i]) {
			t.Errorf("slot %d: %v, ingin %v", slot.Order, got, want[i])
			continue
		}
		for _, n := range want[i] {
			found := false
			for _, g := range got {
				found = found || g == n
			}
			if !found {
				t.Errorf("slot %d: %v, ingin %v", slot.Order, got, want[i])
			}
		}
	}
	// Viewer tanpa E-5 tidak bisa membuka formulir.
	if _, err := w.svc.Plan(t.Context(), w.Viewer, w.cp.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer: %v", err)
	}
}

func TestSendSignRelease(t *testing.T) {
	w := setup(t)
	ctx := t.Context()
	signers := map[int]string{1: w.sender.PersonID, 2: w.pl.PersonID}

	if _, err := w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: signers, Password: "salah"}); codeOf(t, err) != "password:incorrect" {
		t.Errorf("password salah: %v", err)
	}
	// D-203: bila Project Leader tingkat Leader Prepared, Checked harus Supervisor.
	if _, err := w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: map[int]string{1: w.pe.PersonID, 2: w.pe.PersonID}, Password: "benar"}); codeOf(t, err) != "signers.2:not_allowed" {
		t.Errorf("satu orang dua slot: %v", err)
	}
	if _, err := w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: map[int]string{1: w.pe.PersonID, 2: w.sender.PersonID}, Password: "benar"}); codeOf(t, err) != "signers.2:not_allowed" {
		t.Errorf("checked bukan calon: %v", err)
	}
	if _, err := w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: map[int]string{1: w.sender.PersonID}, Password: "benar"}); codeOf(t, err) != "signers.2:required" {
		t.Errorf("checked kosong: %v", err)
	}

	r, err := w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: signers, Note: "Rev 0", Password: "benar"})
	if err != nil {
		t.Fatal(err)
	}
	today := clock.Date(time.Now()).Format(time.DateOnly)
	if r.Status != signature.StatusWaiting || r.ReleaseDate != today || r.NextSlot != 2 || r.Slots[0].SignedAt == nil || len(r.ContentSHA256) != 64 {
		t.Fatalf("terkirim: %+v", r)
	}
	// Draft terkunci: layanan dan database menolak perubahan isi (R-REV-06).
	if _, err := w.Docs.AddRow(ctx, w.sender, w.cp.ID, document.NewRow{Content: map[string]any{}}); codeOf(t, err) != "status:not_allowed" {
		t.Errorf("edit saat menunggu: %v", err)
	}
	if _, err := w.Pool.Exec(ctx, `UPDATE document_rows SET content = '{}' WHERE document_id = $1`, w.cp.ID); err == nil {
		t.Error("database mengizinkan edit draft yang sedang ditandatangani")
	}
	if _, err := w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: signers, Password: "benar"}); codeOf(t, err) != "status:not_allowed" {
		t.Errorf("kirim dua kali: %v", err)
	}
	// Rilis ditolak selama masih menunggu.
	if _, err := w.Docs.Release(ctx, w.record, w.cp.ID); codeOf(t, err) != "signatures:not_allowed" {
		t.Errorf("rilis saat menunggu: %v", err)
	}

	// Berurutan (D-49): QA QC Manager belum gilirannya.
	if _, err := w.svc.Sign(ctx, w.qa, r.ID, "benar"); codeOf(t, err) != "signer:not_allowed" {
		t.Errorf("tidak berurutan: %v", err)
	}
	if _, err := w.svc.Sign(ctx, w.pl, r.ID, "salah"); codeOf(t, err) != "password:incorrect" {
		t.Errorf("password salah saat tanda tangan: %v", err)
	}
	for _, a := range []access.Actor{w.pl, w.npi} {
		if r, err = w.svc.Sign(ctx, a, r.ID, "benar"); err != nil {
			t.Fatal(err)
		}
	}
	if r.Status != signature.StatusWaiting || r.NextSlot != 4 {
		t.Errorf("sesudah NPI: %+v", r)
	}
	// Daftar "menunggu saya" dan "dikirim".
	o, err := w.svc.List(ctx, w.qa)
	if err != nil || len(o.Waiting) != 1 || o.Waiting[0].ID != r.ID {
		t.Errorf("menunggu QA: %+v, %v", o.Waiting, err)
	}
	if r, err = w.svc.Sign(ctx, w.qa, r.ID, "benar"); err != nil {
		t.Fatal(err)
	}
	if r.Status != signature.StatusCompleted || r.NextSlot != 0 {
		t.Fatalf("lengkap: %+v", r)
	}
	o, err = w.svc.List(ctx, w.record)
	if err != nil || len(o.ToRelease) != 1 {
		t.Errorf("menunggu rilis: %+v, %v", o.ToRelease, err)
	}
	if _, err := w.svc.Withdraw(ctx, w.sender, r.ID); codeOf(t, err) != "status:not_allowed" {
		t.Errorf("tarik sesudah lengkap: %v", err)
	}

	// Rilis tanpa scan; pencatat bukan penyusun; tanggal rilis = tanggal dikirim (D-204).
	if _, err := w.Docs.Release(ctx, w.sender, w.cp.ID); codeOf(t, err) != "recorder:not_allowed" {
		t.Errorf("penyusun mencatat rilis: %v", err)
	}
	d, err := w.Docs.Release(ctx, w.record, w.cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	revs, err := w.Docs.Revisions(ctx, w.record, d.ID)
	if err != nil || revs[0].Status != "released" || revs[0].ReleaseDate != today || revs[0].SHA256 != r.ContentSHA256 {
		t.Errorf("revisi terbit: %+v, %v", revs, err)
	}

	// Cetakan menulis "Signed digitally" dan tanggal revisinya tetap tanggal
	// kirim; cetak untuk tanda tangan basah tetap bisa, tanpa keterangan itu (D-198).
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	prints := printout.NewService(w.Pool, w.Checker, w.Docs, store, "")
	text := func(purpose string) string {
		t.Helper()
		f, err := prints.Print(ctx, w.record, w.cp.ID, printout.Request{Purpose: purpose, Format: "xlsx"})
		if err != nil {
			t.Fatalf("cetak %s: %v", purpose, err)
		}
		book, err := excelize.OpenReader(bytes.NewReader(f.Body))
		if err != nil {
			t.Fatal(err)
		}
		var all strings.Builder
		for _, sheet := range book.GetSheetList() {
			rows, _ := book.GetRows(sheet)
			for _, row := range rows {
				all.WriteString(strings.Join(row, "|") + "\n")
			}
		}
		return all.String()
	}
	reference := text(printout.PurposeReference)
	if !strings.Contains(reference, "Signed digitally "+today) || !strings.Contains(reference, "Orang qa.manager") || !strings.Contains(reference, "signed digitally") {
		t.Errorf("cetakan referensi tanpa tanda tangan digital:\n%s", reference)
	}
	wet := text(printout.PurposeSignature)
	if strings.Contains(wet, "Signed digitally") || !strings.Contains(wet, "Orang pl.gr1") {
		t.Errorf("cetak untuk tanda tangan basah:\n%s", wet)
	}
	revs, _ = w.Docs.Revisions(ctx, w.record, d.ID)
	if revs[0].ReleaseDate != today {
		t.Errorf("tanggal revisi berubah karena cetak: %s", revs[0].ReleaseDate)
	}

	// Tanda tangan dan slot tidak bisa diubah atau dihapus, juga lewat SQL.
	for _, sql := range []string{
		`UPDATE signatures SET person_name = 'x'`, `DELETE FROM signatures`,
		`UPDATE signature_slots SET person_name = 'x'`, `UPDATE signature_requests SET status = 'waiting'`,
		`DELETE FROM signature_requests`,
	} {
		if _, err := w.Pool.Exec(ctx, sql); err == nil {
			t.Errorf("database mengizinkan: %s", sql)
		}
	}
}

func TestRejectAndWithdraw(t *testing.T) {
	w := setup(t)
	ctx := t.Context()
	// Project Leader tingkat Leader sebagai Prepared: Checked otomatis Supervisor (satu-satunya calon).
	r, err := w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: map[int]string{1: w.pe.PersonID, 2: w.pl.PersonID}, Password: "benar"})
	if err != nil {
		t.Fatal(err)
	}
	if r.NextSlot != 1 || r.Slots[0].SignedAt != nil || r.Slots[0].PersonName != "Orang pe.gr1" {
		t.Fatalf("prepared oleh Project Engineer: %+v", r)
	}
	if _, err := w.svc.Withdraw(ctx, w.pe, r.ID); codeOf(t, err) != "sender:not_allowed" {
		t.Errorf("tarik oleh bukan pengirim: %v", err)
	}
	if _, err := w.svc.Reject(ctx, w.pl, r.ID, "x", "benar"); codeOf(t, err) != "signer:not_allowed" {
		t.Errorf("tolak bukan giliran: %v", err)
	}
	if _, err := w.svc.Reject(ctx, w.pe, r.ID, " ", "benar"); codeOf(t, err) != "reason:required" {
		t.Errorf("tolak tanpa alasan: %v", err)
	}
	if r, err = w.svc.Reject(ctx, w.pe, r.ID, "Frekuensi op. 20 salah", "benar"); err != nil {
		t.Fatal(err)
	}
	if r.Status != signature.StatusRejected || r.RejectReason != "Frekuensi op. 20 salah" || r.ClosedBy != "Orang pe.gr1" {
		t.Errorf("ditolak: %+v", r)
	}
	// Draft terbuka lagi dan tanggal rilisnya kosong.
	if _, err := w.Docs.AddRow(ctx, w.sender, w.cp.ID, document.NewRow{Content: map[string]any{}}); err != nil {
		t.Errorf("edit sesudah ditolak: %v", err)
	}
	revs, err := w.Docs.Revisions(ctx, w.sender, w.cp.ID)
	if err != nil || revs[0].ReleaseDate != "" {
		t.Errorf("tanggal rilis sesudah ditolak: %+v, %v", revs, err)
	}

	// Kirim lagi, lalu tarik kembali oleh pengirim.
	r, err = w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: map[int]string{1: w.sender.PersonID, 2: w.pe.PersonID}, Password: "benar"})
	if err != nil {
		t.Fatal(err)
	}
	if r, err = w.svc.Withdraw(ctx, w.sender, r.ID); err != nil || r.Status != signature.StatusWithdrawn {
		t.Errorf("tarik kembali: %+v, %v", r, err)
	}
	all, err := w.svc.ForDocument(ctx, w.sender, w.cp.ID)
	if err != nil || len(all) != 2 || all[0].Status != signature.StatusWithdrawn {
		t.Errorf("riwayat permintaan: %+v, %v", all, err)
	}
}

func TestWetOnlyCustomer(t *testing.T) {
	w := setup(t)
	ctx := t.Context()
	if err := w.svc.SetMode(ctx, w.Author, w.CustomerAB, signature.ModeWetOnly); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun tanpa E-11 mengatur cara tanda tangan: %v", err)
	}
	// Admin tidak lagi mengurus master data (D-209).
	if err := w.svc.SetMode(ctx, w.Admin, w.CustomerAB, signature.ModeWetOnly); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("admin mengatur cara tanda tangan: %v", err)
	}
	if err := w.svc.SetMode(ctx, w.sender, w.CustomerAB, "pena"); codeOf(t, err) != "mode:invalid_format" {
		t.Errorf("cara tidak dikenal: %v", err)
	}
	if err := w.svc.SetMode(ctx, w.sender, w.CustomerAB, signature.ModeWetOnly); err != nil {
		t.Fatal(err)
	}
	_, err := w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: map[int]string{1: w.sender.PersonID, 2: w.pl.PersonID}, Password: "benar"})
	if codeOf(t, err) != "mode:not_allowed" {
		t.Errorf("customer wajib basah: %v", err)
	}
}

// D-206: Project Leader Project Group lain (GR5) yang membantu GR1 boleh
// menandatangani dan memverifikasi bila perannya tidak dibatasi; bila
// dibatasi ke Project Group sendiri, tidak.
func TestHelperProjectLeader(t *testing.T) {
	w := setup(t)
	ctx := t.Context()
	var gr5 string
	if err := w.Pool.QueryRow(ctx, `INSERT INTO project_groups (code, name) VALUES ('GR5', 'Project Group 5') RETURNING id`).Scan(&gr5); err != nil {
		t.Fatal(err)
	}
	limited := doctest.NewActor(t, w.Pool, "pl.gr5.terbatas", "work", access.RoleGrant{Role: access.RoleProjectLeader, Scope: access.ScopeOwnProjectGroup})
	helper := doctest.NewActor(t, w.Pool, "pl.gr5.bantu", "work", doctest.All(access.RoleProjectLeader))
	for _, a := range []access.Actor{limited, helper} {
		if _, err := w.Pool.Exec(ctx, `INSERT INTO position_holders (position_id, per_project_group, person_id, project_group_id)
			SELECT id, true, $1, $2 FROM positions WHERE key = 'project_engineer'`, a.PersonID, gr5); err != nil {
			// Project Engineer GR5 hanya satu orang (D-115): yang kedua jadi Project Leader GR5.
			if _, err := w.Pool.Exec(ctx, `INSERT INTO position_holders (position_id, per_project_group, person_id, project_group_id)
				SELECT id, true, $1, $2 FROM positions WHERE key = 'project_leader'`, a.PersonID, gr5); err != nil {
				t.Fatal(err)
			}
		}
	}
	p, err := w.svc.Plan(ctx, w.sender, w.cp.ID)
	if err != nil {
		t.Fatal(err)
	}
	checked := names(p.Slots[1].Candidates)
	has := func(list []string, n string) bool {
		for _, x := range list {
			if x == n {
				return true
			}
		}
		return false
	}
	if !has(checked, "Orang pl.gr5.bantu") || has(checked, "Orang pl.gr5.terbatas") {
		t.Errorf("calon Checked: %v", checked)
	}
	for _, c := range p.Slots[1].Candidates {
		if c.Name == "Orang pl.gr5.bantu" && c.Position != "Project Leader GR5" {
			t.Errorf("label jabatan: %q", c.Position)
		}
	}
	r, err := w.svc.Send(ctx, w.sender, w.cp.ID, signature.SendInput{Signers: map[int]string{1: w.sender.PersonID, 2: helper.PersonID}, Password: "benar"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.svc.Sign(ctx, helper, r.ID, "benar"); err != nil {
		t.Errorf("Project Leader pembantu menandatangani: %v", err)
	}
	// Verifikasi perubahan mengikuti aturan yang sama.
	for login, want := range map[string]bool{"pl.gr5.bantu": true, "pl.gr5.terbatas": false} {
		person := helper.PersonID
		if login == "pl.gr5.terbatas" {
			person = limited.PersonID
		}
		var ok bool
		if err := w.Pool.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM position_holders h JOIN positions p ON p.id = h.position_id WHERE h.person_id = $1)`, person).Scan(&ok); err != nil || !ok {
			t.Fatalf("pemegang %s: %v", login, err)
		}
		got, err := w.Q.IsChangeVerifier(ctx, db.IsChangeVerifierParams{PersonID: person, CustomerID: w.CustomerAB})
		if err != nil || got != want {
			t.Errorf("verifikator %s: %v, ingin %v (%v)", login, got, want, err)
		}
	}
}
