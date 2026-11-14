package change

import (
	"errors"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// TestYokotenAcrossGroups: pemilik target di Project Group lain bisa membuka
// papan status dan memutuskan targetnya sendiri, tetapi tidak melihat target
// customer lain (D-118).
func TestYokotenAcrossGroups(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	hold(t, e.Pool, controller, "apqp_ppap_controller", "")
	// Customer CD pindah ke Project Group GR9 dengan Project Engineer sendiri.
	var gr9 string
	if err := e.Pool.QueryRow(ctx, `INSERT INTO project_groups (code, name) VALUES ('GR9', 'Project Group 9') RETURNING id`).Scan(&gr9); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Pool.Exec(ctx, `UPDATE customers SET project_group_id = $1 WHERE id = $2`, gr9, e.CustomerCD); err != nil {
		t.Fatal(err)
	}
	pe := doctest.NewActor(t, e.Pool, "pe.gr9", "work",
		access.RoleGrant{Role: access.RoleAuthor, Scope: access.ScopeOwnProjectGroup}, access.RoleGrant{Role: access.RoleVerifier, Scope: access.ScopeOwnProjectGroup})
	hold(t, e.Pool, pe, "project_engineer", gr9)

	k125 := e.NewSet(t, "k125", e.CustomerAB, e.ModelK125)["cp"]
	e.NewSet(t, "k200", e.CustomerAB, e.ModelK200)
	e.NewSet(t, "b1", e.CustomerCD, e.ModelB1)
	source, err := s.Create(ctx, e.Author, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, Reason: "AOI", DocumentIDs: []string{k125.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Verify(ctx, controller, source.ID, true, "OK"); err != nil {
		t.Fatal(err)
	}
	y, err := s.CreateYokoten(ctx, e.Author, NewYokoten{SourceChangeID: source.ID, Reason: "Spread", Targets: TargetFilter{ModelIDs: []string{e.ModelK200, e.ModelB1}}})
	if err != nil || len(y.Targets) != 2 {
		t.Fatalf("yokoten: %+v, %v", y.Targets, err)
	}

	// PE GR9 melihat papan, tetapi hanya target customer CD.
	board, err := s.GetYokoten(ctx, pe, y.ID)
	if err != nil {
		t.Fatalf("PE GR9 membuka papan: %v", err)
	}
	if len(board.Targets) != 1 || board.Targets[0].ModelName != "B-1" {
		t.Fatalf("target yang terlihat: %+v", board.Targets)
	}
	if list, _ := s.ListYokotens(ctx, pe); len(list) != 1 {
		t.Errorf("daftar yokoten PE GR9: %d", len(list))
	}
	if board, err = s.Decide(ctx, pe, y.ID, board.Targets[0].ID, DecisionApply, ""); err != nil || board.Targets[0].Progress != ProgressVerification {
		t.Fatalf("PE GR9 memutuskan: %+v, %v", board.Targets, err)
	}
	// Target customer AB tidak bisa diputuskan PE GR9.
	var k200Target string
	for _, x := range y.Targets {
		if x.ModelName == "K-200" {
			k200Target = x.ID
		}
	}
	if _, err := s.Decide(ctx, pe, y.ID, k200Target, DecisionApply, ""); err == nil {
		t.Error("PE GR9 memutuskan target customer AB")
	}
	// Orang tanpa hak di customer mana pun tidak melihat yokoten.
	if _, err := s.GetYokoten(ctx, e.Viewer, y.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer: %v", err)
	}
}
