package template

import (
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
)

// Form set langkah B dan C (D-133, D-135, D-140): isi awal form PFMEA
// AIAG-VDA 2019 dan Control Plan AIAG 2024, kolom AP yang dihitung, dan
// field nomor dokumen yang diisi dari register.

// keysOf menulis key field satu area menurut urutan.
func keysOf(v FormatVersion, area string) []string {
	var out []string
	for _, f := range v.Fields {
		if f.Area == area {
			out = append(out, f.Key)
		}
	}
	return out
}

// byKey mencari field menurut key.
func byKey(t *testing.T, v FormatVersion, key string) Field {
	t.Helper()
	for _, f := range v.Fields {
		if f.Key == key {
			return f
		}
	}
	t.Fatalf("field dengan key %q tidak ada", key)
	return Field{}
}

func optionKeys(f Field) []string {
	var out []string
	for _, o := range f.Options {
		out = append(out, o.Key)
	}
	return out
}

// TestSeededPFMEAVDA: form PFMEA AIAG-VDA 2019 persis seperti tabel di prompt pemilik.
func TestSeededPFMEAVDA(t *testing.T) {
	f := newFixture(t)
	ft := f.template(t, "PFMEA AIAG-VDA 2019")
	v := f.version(t, "PFMEA AIAG-VDA 2019")
	if ft.DocumentTypeKey != "pfmea" || ft.Methodology != MethodAIAGVDA || v.RiskMethod != RiskAP || v.Status != StatusDraft {
		t.Errorf("template: %s %s %s %s", ft.DocumentTypeKey, ft.Methodology, v.RiskMethod, v.Status)
	}
	if !slices.Equal(v.APRules, DefaultAPRules()) {
		t.Errorf("tabel AP: %d aturan, ingin tabel bawaan", len(v.APRules))
	}
	wantHeader := []string{
		"part_name", "company_name", "manufacturing_location", "customer_name", "model_year_program", "part_no",
		"pfmea_start_date", "pfmea_revision_date", "cross_functional_team", "pfmea_id_number", "process_responsibility", "confidentiality_level",
	}
	if got := keysOf(v, AreaHeader); !slices.Equal(got, wantHeader) {
		t.Errorf("field header:\n%v\ningin\n%v", got, wantHeader)
	}
	wantRow := []string{
		"change_history", "process_item", "process_step", "work_element",
		"function_process_item", "function_process_step", "function_work_element",
		"failure_effect", "severity", "failure_mode", "failure_cause",
		"prevention_control", "occurrence", "detection_control", "detection", "ap",
		"prevention_action", "detection_action", "responsible_person", "target_completion_date", "action_status",
		"action_taken", "completion_date", "severity_after", "occurrence_after", "detection_after",
		"special_characteristic", "ap_after", "note",
	}
	if got := keysOf(v, AreaRow); !slices.Equal(got, wantRow) {
		t.Errorf("kolom:\n%v\ningin\n%v", got, wantRow)
	}

	kinds := map[string]string{
		"pfmea_id_number": KindDocumentNumber, "pfmea_start_date": KindDate, "work_element": KindSelect, "process_step": KindProcess, "process_item": KindText,
		"severity": KindRating, "occurrence": KindRating, "detection": KindRating, "ap": KindAP,
		"severity_after": KindRating, "ap_after": KindAP, "special_characteristic": KindSymbol,
		"target_completion_date": KindDate, "action_status": KindSelect, "failure_mode": KindText,
	}
	for key, kind := range kinds {
		if got := byKey(t, v, key).Kind; got != kind {
			t.Errorf("jenis %s: %s, ingin %s", key, got, kind)
		}
	}
	if s, a := byKey(t, v, "severity"), byKey(t, v, "severity_after"); s.RatingSet != 1 || a.RatingSet != 2 || a.RatingScale != ScaleSeverity {
		t.Errorf("set severity: %d dan %d", s.RatingSet, a.RatingSet)
	}
	if a, b := byKey(t, v, "ap"), byKey(t, v, "ap_after"); a.RatingSet != 1 || b.RatingSet != 2 {
		t.Errorf("set AP: %d dan %d", a.RatingSet, b.RatingSet)
	}
	if got := optionKeys(byKey(t, v, "work_element")); !slices.Equal(got, []string{"man", "machine", "material", "method", "measurement", "milieu"}) {
		t.Errorf("pilihan work element: %v", got)
	}
	if got := optionKeys(byKey(t, v, "action_status")); !slices.Equal(got, []string{"open", "completed", "discarded"}) {
		t.Errorf("pilihan status tindakan: %v", got)
	}
	groups := map[string]string{
		"change_history": "", "process_item": "STRUCTURE ANALYSIS (STEP 2)", "work_element": "STRUCTURE ANALYSIS (STEP 2)",
		"function_process_item": "FUNCTION ANALYSIS (STEP 3)", "failure_cause": "FAILURE ANALYSIS (STEP 4)",
		"ap": "RISK ANALYSIS (STEP 5)", "prevention_action": "OPTIMIZATION (STEP 6)", "note": "OPTIMIZATION (STEP 6)",
	}
	for key, group := range groups {
		if got := byKey(t, v, key).Group; got != group {
			t.Errorf("grup %s: %q, ingin %q", key, got, group)
		}
	}
	if got := byKey(t, v, "process_item").Label; got != "1. Process Item" {
		t.Errorf("label process_item: %q", got)
	}
	if len(v.Symbols) != 4 || len(v.Ratings) != 30 {
		t.Errorf("simbol %d, kriteria skala %d", len(v.Symbols), len(v.Ratings))
	}
}

// TestSeededCP2024: form Control Plan AIAG 2024, tanpa kolom Process Flow
// dan dengan empat pilihan fase (D-135).
func TestSeededCP2024(t *testing.T) {
	f := newFixture(t)
	ft := f.template(t, "Control Plan AIAG 2024")
	v := f.version(t, "Control Plan AIAG 2024")
	if ft.DocumentTypeKey != "cp" || ft.Methodology != MethodAIAGCP2024 || v.RiskMethod != RiskNone {
		t.Errorf("template: %s %s %s", ft.DocumentTypeKey, ft.Methodology, v.RiskMethod)
	}
	wantHeader := []string{
		"phase", "control_plan_no", "key_contact", "date_orig", "date_rev", "customer_name", "core_team",
		"customer_engineering_approval", "part_number", "organization_plant_approval", "customer_quality_approval",
		"part_name", "other_approval_1", "other_approval_2",
	}
	if got := keysOf(v, AreaHeader); !slices.Equal(got, wantHeader) {
		t.Errorf("field header:\n%v\ningin\n%v", got, wantHeader)
	}
	wantRow := []string{
		"process_no", "process_description", "owner", "machine_tool", "characteristic_no", "product_characteristic",
		"process_characteristic", "special_char_class", "specification_tolerance", "evaluation_technique",
		"sample_size", "sample_frequency", "control_method", "reaction_plan",
	}
	if got := keysOf(v, AreaRow); !slices.Equal(got, wantRow) {
		t.Errorf("kolom:\n%v\ningin\n%v", got, wantRow)
	}
	if got := optionKeys(byKey(t, v, "phase")); !slices.Equal(got, []string{"prototype", "pre_launch", "safe_launch", "production"}) {
		t.Errorf("pilihan fase: %v", got)
	}
	kinds := map[string]string{
		"control_plan_no": KindDocumentNumber, "process_no": KindOperationNumber, "process_description": KindProcess,
		"special_char_class": KindSymbol, "date_orig": KindDate, "phase": KindSelect,
	}
	for key, kind := range kinds {
		if got := byKey(t, v, key).Kind; got != kind {
			t.Errorf("jenis %s: %s, ingin %s", key, got, kind)
		}
	}
	for _, fd := range v.Fields {
		if fd.Kind == KindProcessFlow {
			t.Errorf("CP 2024 punya kolom Process Flow (%s)", fd.Label)
		}
	}
	if g := byKey(t, v, "sample_size").Group; g != "Methods" {
		t.Errorf("grup sample_size: %q", g)
	}
}

// TestAPInContent: kolom AP dihitung dari S, O, D dengan set yang sama, tidak
// bisa diketik, dan baris AP H tanpa tindakan disorot (R-TPL-11).
func TestAPInContent(t *testing.T) {
	f := newFixture(t)
	v := f.version(t, "PFMEA AIAG-VDA 2019")
	id := func(key string) string { return byKey(t, v, key).ID }
	content := map[string]any{id("severity"): 8, id("occurrence"): 6, id("detection"): 4, id("failure_mode"): "Wrong part"}

	if got := AP(v, content); got[id("ap")] != "H" || len(got) != 1 {
		t.Errorf("AP: %v, ingin hanya AP set 1 = H", got)
	}
	if got := Highlights(v, content); !slices.Contains(got, id("ap")) {
		t.Errorf("AP H tanpa tindakan tidak disorot: %v", got)
	}
	content[id("detection_action")] = "Add barcode scan"
	if got := Highlights(v, content); slices.Contains(got, id("ap")) {
		t.Errorf("AP H dengan tindakan masih disorot: %v", got)
	}

	// Sesudah tindakan: O turun menjadi 2, AP sesudah = L (S8, O2, D4).
	content[id("severity_after")], content[id("occurrence_after")], content[id("detection_after")] = 8, 2, 4
	if got := AP(v, content); got[id("ap")] != "H" || got[id("ap_after")] != "L" {
		t.Errorf("AP sesudah tindakan: %v", got)
	}

	// AP tidak diketik, dan tidak pernah dianggap kosong walaupun wajib.
	if c := CheckContent(v, AreaRow, content); !c.Valid() {
		t.Errorf("isi sah ditolak: %+v", c.Problems)
	}
	content[id("ap")] = "L"
	if c := CheckContent(v, AreaRow, content); len(c.Problems) != 1 || c.Problems[0].Field != id("ap") || c.Problems[0].Code != "not_allowed" {
		t.Errorf("AP yang diketik: %+v", c.Problems)
	}
	// Nomor dokumen di header juga tidak diketik.
	header := map[string]any{id("pfmea_id_number"): "SF-AB-001"}
	if c := CheckContent(v, AreaHeader, header); len(c.Problems) != 1 || c.Problems[0].Code != "not_allowed" {
		t.Errorf("nomor dokumen yang diketik: %+v", c.Problems)
	}
}

// TestAPFieldRules: kolom AP hanya untuk metode AP, satu per set; nomor
// dokumen hanya di header. Key dan grup ikut ke salinan dan ke versi baru.
func TestAPFieldRules(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	rpn := f.version(t, "PFMEA AIAG 4th")
	vda := f.version(t, "PFMEA AIAG-VDA 2019")

	if _, err := f.s.AddField(ctx, f.admin, rpn.ID, NewField{Area: AreaRow, Kind: KindAP, Label: "AP", RatingSet: 1}); !slices.Equal(problemCodes(t, err), []string{"kind:not_allowed"}) {
		t.Errorf("kolom AP di template RPN: %v", err)
	}
	if _, err := f.s.AddField(ctx, f.admin, vda.ID, NewField{Area: AreaRow, Kind: KindAP, Label: "AP lagi", RatingSet: 1}); !slices.Equal(problemCodes(t, err), []string{"rating_set:taken"}) {
		t.Errorf("kolom AP kedua untuk set 1: %v", err)
	}
	if _, err := f.s.AddField(ctx, f.admin, vda.ID, NewField{Area: AreaRow, Kind: KindDocumentNumber, Label: "Doc no"}); !slices.Equal(problemCodes(t, err), []string{"kind:not_allowed"}) {
		t.Errorf("nomor dokumen sebagai kolom: %v", err)
	}
	if _, err := f.s.AddField(ctx, f.admin, vda.ID, NewField{Area: AreaHeader, Kind: KindDocumentNumber, Label: "Doc no"}); !slices.Equal(problemCodes(t, err), []string{"kind:taken"}) {
		t.Errorf("nomor dokumen kedua di header: %v", err)
	}

	// Salinan template dan versi baru membawa key dan grup.
	ft := f.template(t, "PFMEA AIAG-VDA 2019")
	copied, err := f.s.CreateFormatTemplate(ctx, f.admin, NewTemplate{Name: "PFMEA VDA salinan", DocumentTypeID: ft.DocumentTypeID, CopyFromVersionID: vda.ID})
	if err != nil {
		t.Fatal(err)
	}
	cv, err := f.s.GetFormatVersion(ctx, f.author, copied.Versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(keysOf(cv, AreaRow), keysOf(vda, AreaRow)) || byKey(t, cv, "ap").Group != "RISK ANALYSIS (STEP 5)" {
		t.Errorf("salinan kehilangan key atau grup")
	}
	approver := newActor(t, f.pool, "ppap.supervisor", access.AccountWork, access.RoleFormApprover)
	if _, err := f.s.PublishFormatVersion(ctx, approver, vda.ID); err != nil {
		t.Fatal(err)
	}
	v2, err := f.s.NewFormatVersion(ctx, f.admin, ft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(keysOf(v2, AreaRow), keysOf(vda, AreaRow)) || byKey(t, v2, "ap").Group != "RISK ANALYSIS (STEP 5)" {
		t.Errorf("versi baru kehilangan key atau grup")
	}
}
