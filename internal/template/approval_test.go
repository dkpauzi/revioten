package template

import (
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
)

// D-139, diperbarui 12 November 2026: aturan persetujuan bisa diatur admin
// di Settings. "all" (bawaan) berarti semua penyetuju form harus setuju
// sebelum versi terbit; "any" berarti satu persetujuan cukup. Penyusun draft
// tidak pernah dihitung sebagai penyetuju.

// names menulis nama orang dari daftar penyetuju.
func names(list []Approver) []string {
	var out []string
	for _, a := range list {
		out = append(out, a.PersonName)
	}
	return out
}

// publishedSheets menerbitkan ketiga format set AIAG 4th dan mengembalikan versi 1-nya.
func publishedSheets(t *testing.T, f formSetFixture) FormSetVersion {
	t.Helper()
	v1 := f.set(t, "aiag_4th").Versions[0]
	for _, id := range []string{v1.PFD.FormatVersionID, v1.PFMEA.FormatVersionID, v1.CP.FormatVersionID} {
		if _, err := f.s.PublishFormatVersion(t.Context(), f.admin, id); err != nil {
			t.Fatal(err)
		}
	}
	return v1
}

// TestApprovalAll: dengan aturan "all", versi baru terbit setelah semua
// penyetuju form setuju. Persetujuan dicatat per orang dan tidak bisa diulang.
func TestApprovalAll(t *testing.T) {
	f := newFormSetFixture(t)
	ctx := t.Context()
	second := newActor(t, f.pool, "project.leader", access.AccountWork, access.RoleFormApprover)
	if mode, err := f.s.q.GetFormApprovalMode(ctx); err != nil || mode != ApprovalAll {
		t.Fatalf("aturan bawaan: %q, %v; ingin all", mode, err)
	}
	v1 := publishedSheets(t, f)

	fs, err := f.s.PublishFormSetVersion(ctx, f.approver, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	v := fs.Versions[0]
	if v.Status != StatusDraft || !slices.Equal(names(v.Approvals), []string{"Orang ppap.supervisor"}) || !slices.Equal(names(v.PendingApprovers), []string{"Orang project.leader"}) {
		t.Fatalf("sesudah satu persetujuan: %s, setuju %v, menunggu %v", v.Status, names(v.Approvals), names(v.PendingApprovers))
	}
	if action, details := lastAudit(t, f.fixture); action != "form_set.version_approved" || details != `{"pending":"1","version":"1"}` {
		t.Errorf("entri audit: %s %s", action, details)
	}
	if _, err := f.s.PublishFormSetVersion(ctx, f.approver, v1.ID); !slices.Equal(problemCodes(t, err), []string{"approver:taken"}) {
		t.Errorf("menyetujui dua kali: %v", err)
	}

	fs, err = f.s.PublishFormSetVersion(ctx, second, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v = fs.Versions[0]; v.Status != StatusPublished || len(v.PendingApprovers) != 0 || len(v.Approvals) != 2 {
		t.Errorf("sesudah semua setuju: %s, setuju %v, menunggu %v", v.Status, names(v.Approvals), names(v.PendingApprovers))
	}

	// Akun nonaktif tidak ditunggu.
	if _, err := f.pool.Exec(ctx, "UPDATE accounts SET disabled_at = now() WHERE id = $1", second.AccountID); err != nil {
		t.Fatal(err)
	}
	vda := f.version(t, "PFMEA AIAG-VDA 2019")
	if v, err := f.s.PublishFormatVersion(ctx, f.approver, vda.ID); err != nil || v.Status != StatusPublished {
		t.Errorf("format AP dengan satu penyetuju aktif: %s, %v", v.Status, err)
	}
}

// TestApprovalAny: dengan aturan "any", satu persetujuan cukup.
func TestApprovalAny(t *testing.T) {
	f := newFormSetFixture(t)
	ctx := t.Context()
	newActor(t, f.pool, "project.leader", access.AccountWork, access.RoleFormApprover)
	if err := f.s.q.SetFormApprovalMode(ctx, ApprovalAny); err != nil {
		t.Fatal(err)
	}
	v1 := publishedSheets(t, f)
	fs, err := f.s.PublishFormSetVersion(ctx, f.approver, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v := fs.Versions[0]; v.Status != StatusPublished || !slices.Equal(names(v.Approvals), []string{"Orang ppap.supervisor"}) {
		t.Errorf("aturan any: %s, setuju %v", v.Status, names(v.Approvals))
	}
}

// TestApprovalPreparerNotCounted: penyusun draft tidak masuk daftar tunggu,
// juga bila ia memegang hak penyetuju.
func TestApprovalPreparerNotCounted(t *testing.T) {
	f := newFormSetFixture(t)
	ctx := t.Context()
	vda := f.version(t, "PFMEA AIAG-VDA 2019")
	changed := DefaultAPRules()
	changed[0].Priority = "M"
	// Penyetuju form sendiri tidak punya A-2, jadi admin yang mengubah; lalu
	// penyetuju menerbitkan. Penyusun (orang admin) tidak ada di daftar tunggu.
	if _, err := f.s.SetAPRules(ctx, f.admin, vda.ID, changed); err != nil {
		t.Fatal(err)
	}
	v, err := f.s.GetFormatVersion(ctx, f.author, vda.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names(v.PendingApprovers), []string{"Orang ppap.supervisor"}) {
		t.Errorf("menunggu: %v", names(v.PendingApprovers))
	}
	if v, err = f.s.PublishFormatVersion(ctx, f.approver, vda.ID); err != nil || v.Status != StatusPublished {
		t.Errorf("terbit: %s, %v", v.Status, err)
	}
}
