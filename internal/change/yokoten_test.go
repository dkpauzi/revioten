package change

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// M5.9 · Yokoten (R-CHG-04, D-40, D-118): perbaikan yang diterima disebarkan
// ke model lain; target dipilih lewat customer, group model, atau model, dan
// disaring proses atau teks; pemilik target memutuskan; papan status per model.
func TestYokoten(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	m := readMasters(t, e)
	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	hold(t, e.Pool, controller, "apqp_ppap_controller", "")
	product := doctest.FieldID(t, e.CPFormat, "Product Characteristics")

	// Sumber: perbaikan AOI di CP model K-125, sudah diterima.
	k125 := e.NewSet(t, "k125", e.CustomerAB, e.ModelK125)["cp"]
	k200set := e.NewSet(t, "k200", e.CustomerAB, e.ModelK200)
	k200 := k200set["cp"]
	e.NewSet(t, "b1", e.CustomerCD, e.ModelB1) // CP B-1 tanpa baris AOI
	if _, err := e.Docs.AddRow(ctx, e.Author, k200.ID, document.NewRow{ProcessID: e.ProcessSMT, Content: map[string]any{product: "AOI 100% inspection"}}); err != nil {
		t.Fatal(err)
	}
	source, err := s.Create(ctx, e.Author, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, Reason: "False NG on AOI", DocumentIDs: []string{k125.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateYokoten(ctx, e.Author, NewYokoten{SourceChangeID: source.ID, Reason: "x", Targets: TargetFilter{CustomerID: e.CustomerAB}}); !slices.Contains(problemCodes(t, err), "source_change_id:not_allowed") {
		t.Errorf("sumber belum diterima: %v", err)
	}
	if source, err = s.Verify(ctx, controller, source.ID, true, "OK"); err != nil {
		t.Fatal(err)
	}

	// Group model berisi K-125 dan K-200 (D-40).
	var group string
	if err := e.Pool.QueryRow(ctx, `INSERT INTO model_groups (customer_id, name) VALUES ($1, 'Family K') RETURNING id`, e.CustomerAB).Scan(&group); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Pool.Exec(ctx, `UPDATE models SET model_group_id = $1 WHERE id IN ($2, $3)`, group, e.ModelK125, e.ModelK200); err != nil {
		t.Fatal(err)
	}

	if _, err := s.CreateYokoten(ctx, e.Viewer, NewYokoten{SourceChangeID: source.ID, Reason: "x", Targets: TargetFilter{CustomerID: e.CustomerAB}}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer: %v", err)
	}
	models := func(y Yokoten) []string {
		var out []string
		for _, t := range y.Targets {
			out = append(out, t.ModelName)
		}
		return out
	}
	// Group model: K-125 sudah di perubahan sumber, jadi hanya K-200.
	y, err := s.CreateYokoten(ctx, e.Author, NewYokoten{SourceChangeID: source.ID, Reason: "Spread AOI fix", Targets: TargetFilter{ModelGroupID: group}, DryRun: true})
	if err != nil || !slices.Equal(models(y), []string{"K-200"}) {
		t.Errorf("group model: %v, %v", models(y), err)
	}
	// Saringan teks dan proses: B-1 (customer CD) tidak punya baris AOI.
	y, err = s.CreateYokoten(ctx, e.Author, NewYokoten{SourceChangeID: source.ID, Reason: "Spread", Targets: TargetFilter{ModelIDs: []string{e.ModelK200, e.ModelB1}, Text: "aoi", ProcessID: e.ProcessSMT}, DryRun: true})
	if err != nil || !slices.Equal(models(y), []string{"K-200"}) {
		t.Errorf("saringan teks: %v, %v", models(y), err)
	}
	y, err = s.CreateYokoten(ctx, e.Author, NewYokoten{SourceChangeID: source.ID, Reason: "Spread", Targets: TargetFilter{ModelIDs: []string{e.ModelK200, e.ModelB1}}, DryRun: true})
	if err != nil || !slices.Equal(models(y), []string{"K-200", "B-1"}) {
		t.Errorf("tanpa saringan: %v, %v", models(y), err)
	}
	if list, _ := s.ListYokotens(ctx, e.Author); len(list) != 0 {
		t.Errorf("dry run tersimpan: %d", len(list))
	}

	// Simpan dua target: K-200 dan B-1.
	y, err = s.CreateYokoten(ctx, e.Author, NewYokoten{SourceChangeID: source.ID, Reason: "Spread AOI fix", Targets: TargetFilter{ModelIDs: []string{e.ModelK200, e.ModelB1}}})
	if err != nil {
		t.Fatal(err)
	}
	if y.Ref != "YKT-0001" || y.SourceChange != source.Ref || len(y.Targets) != 2 || y.Targets[0].Progress != ProgressUndecided || len(y.Targets[0].Documents) != 3 || y.Targets[0].Documents[2].Number != k200.Number {
		t.Fatalf("yokoten: %+v", y)
	}
	byModel := map[string]YokotenTarget{}
	for _, t := range y.Targets {
		byModel[t.ModelName] = t
	}

	// Keputusan pemilik target.
	if _, err := s.Decide(ctx, e.Viewer, y.ID, byModel["K-200"].ID, DecisionApply, ""); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer memutuskan: %v", err)
	}
	if _, err := s.Decide(ctx, e.Author, y.ID, byModel["B-1"].ID, DecisionNotApplicable, ""); !slices.Contains(problemCodes(t, err), "reason:required") {
		t.Errorf("tidak berlaku tanpa alasan: %v", err)
	}
	if y, err = s.Decide(ctx, e.Author, y.ID, byModel["B-1"].ID, DecisionNotApplicable, "B-1 has no AOI"); err != nil {
		t.Fatal(err)
	}
	if y, err = s.Decide(ctx, e.Author, y.ID, byModel["K-200"].ID, DecisionApply, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Decide(ctx, e.Author, y.ID, byModel["K-200"].ID, DecisionAdapt, ""); !slices.Contains(problemCodes(t, err), "decision:not_allowed") {
		t.Errorf("putus dua kali: %v", err)
	}
	for _, t2 := range y.Targets {
		byModel[t2.ModelName] = t2
	}
	if byModel["B-1"].Progress != ProgressNotApplicable || byModel["B-1"].Reason != "B-1 has no AOI" || byModel["K-200"].Progress != ProgressVerification || byModel["K-200"].ChangeRef == "" {
		t.Errorf("papan: %+v", y.Targets)
	}
	target, err := s.Get(ctx, e.Author, byModel["K-200"].ChangeID)
	if err != nil || target.Mechanism != MechanismYokoten || len(target.Documents) != 3 || target.ChangeType.ID != m.content {
		t.Fatalf("perubahan target: %+v, %v", target, err)
	}

	// Diterima: draft (masih revisi awal) lalu terbit.
	if _, err := s.Verify(ctx, controller, target.ID, true, "Same AOI"); err != nil {
		t.Fatal(err)
	}
	if y, _ = s.GetYokoten(ctx, e.Author, y.ID); y.Targets[0].Progress != ProgressInDraft {
		t.Errorf("dalam draft: %s", y.Targets[0].Progress)
	}
	recorder := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	for _, key := range []string{"pfd", "pfmea"} {
		if _, err := e.Docs.Release(ctx, recorder, k200set[key].ID); err != nil {
			t.Fatal(err)
		}
	}
	// Satu dokumen masih draft: target masih dalam draft.
	if y, _ = s.GetYokoten(ctx, e.Author, y.ID); y.Targets[0].Progress != ProgressInDraft {
		t.Errorf("sebagian terbit: %s", y.Targets[0].Progress)
	}
	if _, err := e.Docs.Release(ctx, recorder, k200.ID); err != nil {
		t.Fatal(err)
	}
	if y, _ = s.GetYokoten(ctx, e.Author, y.ID); y.Targets[0].Progress != ProgressReleased {
		t.Errorf("terbit: %s", y.Targets[0].Progress)
	}
	if list, _ := s.ListYokotens(ctx, e.Author); len(list) != 1 {
		t.Errorf("daftar yokoten: %d", len(list))
	}
}
