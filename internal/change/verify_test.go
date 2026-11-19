package change

import (
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// M5.3 · Verifikasi atasan (R-CHG-02, D-51, D-54): catatan wajib, pemohon
// tidak memverifikasi permintaannya sendiri, dan hanya pemegang jabatan
// verifikator customer itu yang bisa memverifikasi.

// hold mencatat orang aktor sebagai pemegang jabatan; groupID kosong untuk
// jabatan tanpa Project Group.
func hold(t *testing.T, pool *pgxpool.Pool, actor access.Actor, positionKey, groupID string) {
	t.Helper()
	var group *string
	if groupID != "" {
		group = &groupID
	}
	_, err := pool.Exec(t.Context(), `INSERT INTO position_holders (position_id, per_project_group, person_id, project_group_id)
		SELECT id, per_project_group, $2, $3 FROM positions WHERE key = $1`, positionKey, actor.PersonID, group)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVerifyChange(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	submit := func(by access.Actor) Change {
		t.Helper()
		c, err := s.Create(ctx, by, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, Reason: "Claim", DocumentIDs: []string{set["cp"].ID}})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}

	// Project Engineer GR1 memegang Penyusun dan Verifikator untuk Project
	// Group sendiri (D-71). Project Leader GR1 juga.
	engineer := doctest.NewActor(t, e.Pool, "pe.gr1", "work", doctest.All(access.RoleAuthor), access.RoleGrant{Role: access.RoleVerifier, Scope: access.ScopeOwnProjectGroup})
	hold(t, e.Pool, engineer, "project_engineer", e.GroupID)
	leader := doctest.NewActor(t, e.Pool, "pl.gr1", "work", access.RoleGrant{Role: access.RoleVerifier, Scope: access.ScopeOwnProjectGroup})
	hold(t, e.Pool, leader, "project_leader", e.GroupID)
	// Pemegang V-1 tanpa jabatan verifikator, dan Project Engineer group lain.
	var otherGroup string
	if err := e.Pool.QueryRow(ctx, `INSERT INTO project_groups (code, name) VALUES ('GR9', 'Project Group 9') RETURNING id`).Scan(&otherGroup); err != nil {
		t.Fatal(err)
	}
	stranger := doctest.NewActor(t, e.Pool, "v1.tanpa.jabatan", "work", doctest.All(access.RoleVerifier))
	// Project Engineer GR9 yang dibatasi ke Project Group sendiri (D-71). Tanpa
	// batas (D-206) ia boleh membantu GR1; itu diuji di internal/signature.
	otherEngineer := doctest.NewActor(t, e.Pool, "pe.gr9", "work", access.RoleGrant{Role: access.RoleVerifier, Scope: access.ScopeOwnProjectGroup})
	hold(t, e.Pool, otherEngineer, "project_engineer", otherGroup)
	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	hold(t, e.Pool, controller, "apqp_ppap_controller", "")

	c := submit(engineer)
	for name, x := range map[string]struct {
		actor access.Actor
		note  string
		want  string
	}{
		"tanpa catatan":   {leader, "  ", "note:required"},
		"pemohon sendiri": {engineer, "OK", "verifier:not_allowed"},
		"tanpa jabatan":   {stranger, "OK", "verifier:not_allowed"},
	} {
		_, err := s.Verify(ctx, x.actor, c.ID, true, x.note)
		if got := problemCodes(t, err); !slices.Contains(got, x.want) {
			t.Errorf("%s: %v", name, got)
		}
	}
	if _, err := s.Verify(ctx, otherEngineer, c.ID, true, "OK"); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Project Engineer group lain yang dibatasi: %v", err)
	}
	if _, err := s.Verify(ctx, e.Author, c.ID, true, "OK"); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("penyusun tanpa V-1: %v", err)
	}

	// Project Leader menerima; yang dicatat orangnya, waktunya, dan catatannya.
	c, err := s.Verify(ctx, leader, c.ID, true, "Supporting 8D attached")
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusAccepted || c.Verification == nil || c.Verification.VerifiedBy != "Orang pl.gr1" || c.Verification.Note != "Supporting 8D attached" {
		t.Errorf("diterima: %s %+v", c.Status, c.Verification)
	}
	// Sesudah diverifikasi: tidak bisa diverifikasi lagi atau diubah.
	if _, err := s.Verify(ctx, controller, c.ID, false, "Again"); !slices.Contains(problemCodes(t, err), "status:not_allowed") {
		t.Errorf("verifikasi ulang: %v", err)
	}
	if _, err := s.AddDocument(ctx, engineer, c.ID, set["pfd"].ID); !slices.Contains(problemCodes(t, err), "status:not_allowed") {
		t.Errorf("ubah sesudah diterima: %v", err)
	}

	// APQP/PPAP Controller menolak perubahan customer mana pun.
	c2 := submit(e.Author)
	c2, err = s.Verify(ctx, controller, c2.ID, false, "No supporting document")
	if err != nil || c2.Status != StatusRejected || c2.Verification.VerifiedBy != "Orang apqp.controller" {
		t.Errorf("ditolak: %+v, %v", c2, err)
	}

	// Database juga menolak pemohon sebagai verifikator (R-SEC-11).
	c3 := submit(engineer)
	_, err = e.Pool.Exec(ctx, `UPDATE changes SET status = 'accepted', verified_by_person_id = requested_by_person_id,
		verified_by_account_id = requested_by_account_id, verified_at = now(), verification_note = 'x' WHERE id = $1`, c3.ID)
	if err == nil {
		t.Error("database menerima pemohon sebagai verifikator")
	}
}
