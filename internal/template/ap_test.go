package template

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
)

// Mesin AP (form set langkah A, D-134, D-140): Action Priority dihitung dari
// tabel yang disimpan sebagai data di versi format template, mengikuti tabel
// PFMEA AIAG & VDA FMEA Handbook 2019.

// sourceTable adalah bentuk file dari paket desain pemilik.
type sourceTable struct {
	DetectionBands [][2]int `json:"detection_bands"`
	Table          []struct {
		S  [2]int   `json:"s"`
		O  [2]int   `json:"o"`
		AP []string `json:"ap_by_detection_band"`
	} `json:"table"`
}

// TestDefaultAPRulesMatchSource: tabel bawaan di kode sama dengan file dari
// paket desain pemilik, untuk semua 1.000 kombinasi S, O, D.
func TestDefaultAPRulesMatchSource(t *testing.T) {
	data, err := os.ReadFile("../../docs/design/form-set-aiag-vda/ap-table-aiag-vda-2019.json")
	if err != nil {
		t.Fatal(err)
	}
	var src sourceTable
	if err := json.Unmarshal(data, &src); err != nil {
		t.Fatal(err)
	}
	// want mencari AP langsung di file sumber.
	want := func(s, o, d int) string {
		for _, row := range src.Table {
			if s < row.S[0] || s > row.S[1] || o < row.O[0] || o > row.O[1] {
				continue
			}
			for i, band := range src.DetectionBands {
				if d >= band[0] && d <= band[1] {
					return row.AP[i]
				}
			}
		}
		return ""
	}
	rules := DefaultAPRules()
	if p := checkAPRules(rules); p != nil {
		t.Fatalf("tabel bawaan tidak lengkap: %+v", p)
	}
	count := map[string]int{}
	for s := 1; s <= 10; s++ {
		for o := 1; o <= 10; o++ {
			for d := 1; d <= 10; d++ {
				got, ok := ActionPriority(rules, s, o, d)
				if !ok || got != want(s, o, d) {
					t.Errorf("S%d O%d D%d: AP %q (%v), ingin %q", s, o, d, got, ok, want(s, o, d))
				}
				count[got]++
			}
		}
	}
	if count["H"]+count["M"]+count["L"] != 1000 {
		t.Errorf("jumlah kombinasi: %v", count)
	}
}

// TestActionPriorityExamples: contoh dari prompt pemilik.
func TestActionPriorityExamples(t *testing.T) {
	rules := DefaultAPRules()
	tests := []struct {
		s, o, d int
		want    string
	}{
		{8, 6, 4, "H"}, {7, 3, 7, "M"}, {7, 3, 4, "L"}, {5, 3, 5, "L"}, {8, 2, 4, "L"}, {9, 1, 10, "L"},
	}
	for _, tt := range tests {
		if got, ok := ActionPriority(rules, tt.s, tt.o, tt.d); !ok || got != tt.want {
			t.Errorf("S%d O%d D%d: AP %q (%v), ingin %q", tt.s, tt.o, tt.d, got, ok, tt.want)
		}
	}
	// Nilai di luar 1 sampai 10, atau tabel kosong, tidak punya AP.
	for _, v := range [][3]int{{0, 5, 5}, {5, 11, 5}, {5, 5, -1}} {
		if got, ok := ActionPriority(rules, v[0], v[1], v[2]); ok {
			t.Errorf("%v: AP %q, ingin tidak ada", v, got)
		}
	}
	if _, ok := ActionPriority(nil, 5, 5, 5); ok {
		t.Error("tabel kosong memberi AP")
	}
}

// TestCheckAPRules: tabel harus menutup setiap kombinasi tepat satu kali.
func TestCheckAPRules(t *testing.T) {
	full := DefaultAPRules()
	tests := []struct {
		name  string
		rules []APRule
		want  string // kode masalah; kosong berarti diterima
	}{
		{"tabel bawaan", full, ""},
		{"ada kombinasi tanpa aturan", full[1:], "invalid_format"},
		{"dua aturan tumpang tindih", append(slices.Clone(full), full[0]), "invalid_format"},
		{"nilai di luar 1 sampai 10", append(slices.Clone(full[1:]), APRule{SMin: 9, SMax: 11, OMin: 8, OMax: 10, DMin: 7, DMax: 10, Priority: "H"}), "invalid_format"},
		{"rentang terbalik", append(slices.Clone(full[1:]), APRule{SMin: 10, SMax: 9, OMin: 8, OMax: 10, DMin: 7, DMax: 10, Priority: "H"}), "invalid_format"},
		{"prioritas bukan H, M, atau L", append(slices.Clone(full[1:]), APRule{SMin: 9, SMax: 10, OMin: 8, OMax: 10, DMin: 7, DMax: 10, Priority: "X"}), "invalid_format"},
		{"kosong", nil, "required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := checkAPRules(tt.rules)
			got := ""
			if p != nil {
				got = p.Code
				if p.Field != "ap_rules" {
					t.Errorf("field %q, ingin ap_rules", p.Field)
				}
			}
			if got != tt.want {
				t.Errorf("masalah %q, ingin %q", got, tt.want)
			}
		})
	}
}

// TestAPRulesInVersions: format template dengan metode risiko AP mendapat
// tabel bawaan; tabel hanya diubah admin di versi draft, ikut disalin ke
// versi baru, dan tidak bisa diubah setelah terbit.
func TestAPRulesInVersions(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	pfmea := f.template(t, "PFMEA v1")
	if v := f.version(t, "PFMEA v1"); len(v.APRules) != 0 {
		t.Errorf("template RPN punya %d aturan AP", len(v.APRules))
	}
	created, err := f.s.CreateFormatTemplate(ctx, f.master, NewTemplate{
		Name: "PFMEA AP uji", DocumentTypeID: pfmea.DocumentTypeID, Methodology: MethodAIAGVDA, RiskMethod: RiskAP,
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err := f.s.GetFormatVersion(ctx, f.author, created.Versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(v.APRules, DefaultAPRules()) {
		t.Fatalf("versi AP baru punya %d aturan, ingin tabel bawaan", len(v.APRules))
	}

	// Koreksi satu sel: S 9-10, O 1 menjadi M untuk semua D.
	changed := slices.Clone(v.APRules)
	for i, r := range changed {
		if r.SMin == 9 && r.OMin == 1 && r.OMax == 1 {
			changed[i].Priority = "M"
		}
	}
	if _, err := f.s.SetAPRules(ctx, f.author, v.ID, changed); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun mengubah tabel AP: %v", err)
	}
	if _, err := f.s.SetAPRules(ctx, f.master, v.ID, changed[1:]); !slices.Equal(problemCodes(t, err), []string{"ap_rules:invalid_format"}) {
		t.Errorf("tabel tidak lengkap: %v", err)
	}
	v, err = f.s.SetAPRules(ctx, f.master, v.ID, changed)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := ActionPriority(v.APRules, 9, 1, 10); got != "M" {
		t.Errorf("sesudah koreksi S9 O1 D10: %q, ingin M", got)
	}
	if action, details := lastAudit(t, f); action != "format_template.ap_rules_changed" || details != `{"changed_combinations":"20","version":"1"}` {
		t.Errorf("entri audit: %s %s", action, details)
	}
	// Tabel RPN tidak punya tabel AP.
	if _, err := f.s.SetAPRules(ctx, f.master, f.version(t, "PFMEA v1").ID, changed); !slices.Equal(problemCodes(t, err), []string{"ap_rules:not_allowed"}) {
		t.Errorf("tabel AP di template RPN: %v", err)
	}

	// Hitung lewat layanan: semua akun boleh (L-2).
	if got, err := f.s.ActionPriorityOf(ctx, f.author, v.ID, 8, 6, 4); err != nil || got != "H" {
		t.Errorf("hitung S8 O6 D4: %q, %v", got, err)
	}
	if _, err := f.s.ActionPriorityOf(ctx, f.author, v.ID, 0, 6, 4); !slices.Equal(problemCodes(t, err), []string{"severity:invalid_format"}) {
		t.Errorf("S di luar rentang: %v", err)
	}

	// Terbit, lalu versi baru: tabel ikut disalin, versi terbit menolak perubahan.
	approver := newActor(t, f.pool, "ppap.supervisor", access.AccountWork, access.RoleFormApprover)
	if _, err := f.s.PublishFormatVersion(ctx, approver, v.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.SetAPRules(ctx, f.master, v.ID, DefaultAPRules()); !slices.Equal(problemCodes(t, err), []string{"status:not_allowed"}) {
		t.Errorf("mengubah tabel AP versi terbit: %v", err)
	}
	v2, err := f.s.NewFormatVersion(ctx, f.master, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(v2.APRules, changed) {
		t.Errorf("tabel AP versi 2 tidak sama dengan versi 1")
	}
	// Salinan template juga membawa tabelnya.
	copied, err := f.s.CreateFormatTemplate(ctx, f.master, NewTemplate{Name: "PFMEA AP salinan", DocumentTypeID: pfmea.DocumentTypeID, CopyFromVersionID: v.ID})
	if err != nil {
		t.Fatal(err)
	}
	cv, err := f.s.GetFormatVersion(ctx, f.author, copied.Versions[0].ID)
	if err != nil || !slices.Equal(cv.APRules, changed) {
		t.Errorf("tabel AP salinan: %d aturan, %v", len(cv.APRules), err)
	}
}
