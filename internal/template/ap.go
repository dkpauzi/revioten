package template

// Mesin AP (form set langkah A, D-134, D-140).
//
// Action Priority (AP) adalah cara AIAG & VDA FMEA Handbook 2019 menentukan
// prioritas tindakan: H (tinggi), M (sedang), atau L (rendah). Berbeda dengan
// RPN yang dihitung dengan perkalian S x O x D, AP dibaca dari tabel: setiap
// kombinasi Severity, Occurrence, dan Detection (masing-masing 1 sampai 10,
// jadi 1.000 kombinasi) punya tepat satu AP.
//
// Tabel itu disimpan sebagai data di versi format template (kolom ap_rules),
// bukan di kode, supaya bisa dikoreksi lewat versi baru. Satu aturan mewakili
// satu "kotak" rentang, misalnya S 9-10, O 4-5, D 7-10 = H. Tabel handbook
// cukup ditulis dengan 84 aturan.

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Nilai AP.
const (
	PriorityHigh   = "H"
	PriorityMedium = "M"
	PriorityLow    = "L"
)

// APRule memberi satu AP untuk satu kotak rentang S, O, dan D. Batas rentang
// ikut dihitung: SMin 9 dan SMax 10 berarti severity 9 dan 10.
type APRule struct {
	SMin, SMax int
	OMin, OMax int
	DMin, DMax int
	Priority   string // H, M, atau L
}

// covers melaporkan apakah aturan ini berlaku untuk nilai S, O, D itu.
func (r APRule) covers(s, o, d int) bool {
	return s >= r.SMin && s <= r.SMax && o >= r.OMin && o <= r.OMax && d >= r.DMin && d <= r.DMax
}

// ActionPriority mencari AP untuk satu kombinasi S, O, D. ok false berarti
// tidak ada aturan yang berlaku: tabel kosong, atau nilainya di luar 1 sampai 10.
// Fungsi ini murni (tanpa database), jadi bisa dipakai ulang oleh editor dan
// migrasi dokumen nanti.
func ActionPriority(rules []APRule, s, o, d int) (priority string, ok bool) {
	for _, r := range rules {
		if r.covers(s, o, d) {
			return r.Priority, true
		}
	}
	return "", false
}

// checkAPRules memeriksa tabel AP: setiap aturan sah, dan setiap dari 1.000
// kombinasi ditutup tepat satu aturan (tidak ada lubang, tidak ada tumpang tindih).
func checkAPRules(rules []APRule) *problem.Problem {
	if len(rules) == 0 {
		return &problem.Problem{Field: "ap_rules", Code: problem.Required, Message: "Enter the AP table."}
	}
	invalid := func(message string) *problem.Problem {
		return &problem.Problem{Field: "ap_rules", Code: problem.InvalidFormat, Message: message}
	}
	inRange := func(lo, hi int) bool { return lo >= 1 && hi <= maxRating && lo <= hi }
	for _, r := range rules {
		if !inRange(r.SMin, r.SMax) || !inRange(r.OMin, r.OMax) || !inRange(r.DMin, r.DMax) {
			return invalid("Every rule needs ranges within 1 to 10.")
		}
		if r.Priority != PriorityHigh && r.Priority != PriorityMedium && r.Priority != PriorityLow {
			return invalid("The action priority of every rule is H, M, or L.")
		}
	}
	for s := 1; s <= maxRating; s++ {
		for o := 1; o <= maxRating; o++ {
			for d := 1; d <= maxRating; d++ {
				matches := 0
				for _, r := range rules {
					if r.covers(s, o, d) {
						matches++
					}
				}
				if matches != 1 {
					return invalid(fmt.Sprintf("The AP table must give exactly one priority for every S, O, D; S %d, O %d, D %d has %d.", s, o, d, matches))
				}
			}
		}
	}
	return nil
}

// apRuleJSON adalah bentuk satu aturan di kolom ap_rules dan di file bawaan.
type apRuleJSON struct {
	S  [2]int `json:"s"`
	O  [2]int `json:"o"`
	D  [2]int `json:"d"`
	AP string `json:"ap"`
}

// apRulesJSON menulis tabel untuk kolom ap_rules; nil bila kosong.
func apRulesJSON(rules []APRule) []byte {
	if len(rules) == 0 {
		return nil
	}
	list := make([]apRuleJSON, 0, len(rules))
	for _, r := range rules {
		list = append(list, apRuleJSON{S: [2]int{r.SMin, r.SMax}, O: [2]int{r.OMin, r.OMax}, D: [2]int{r.DMin, r.DMax}, AP: r.Priority})
	}
	data, err := json.Marshal(list)
	if err != nil {
		panic(fmt.Sprintf("menulis tabel AP: %v", err)) // tidak mungkin: isinya hanya angka dan teks
	}
	return data
}

// parseAPRules membaca kolom ap_rules. Kolom kosong menghasilkan daftar kosong.
func parseAPRules(data []byte) ([]APRule, error) {
	rules := []APRule{}
	if len(data) == 0 {
		return rules, nil
	}
	var list []apRuleJSON
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("membaca tabel AP: %w", err)
	}
	for _, r := range list {
		rules = append(rules, APRule{SMin: r.S[0], SMax: r.S[1], OMin: r.O[0], OMax: r.O[1], DMin: r.D[0], DMax: r.D[1], Priority: r.AP})
	}
	return rules, nil
}

// Tabel PFMEA AIAG & VDA FMEA Handbook 2019, dari paket desain pemilik
// (docs/design/form-set-aiag-vda/ap-table-aiag-vda-2019.json). Test
// TestDefaultAPRulesMatchSource memastikan keduanya sama.
//
//go:embed ap_rules_aiag_vda_2019.json
var defaultAPRulesFile []byte

// DefaultAPRules mengembalikan tabel bawaan untuk versi baru dengan metode
// risiko AP. Setiap pemanggilan memberi salinan baru.
func DefaultAPRules() []APRule {
	rules, err := parseAPRules(defaultAPRulesFile)
	if err != nil {
		panic(err) // file ikut di-embed dan diuji, jadi ini kesalahan build
	}
	return rules
}

// changedCombinations menghitung berapa dari 1.000 kombinasi yang AP-nya berbeda.
func changedCombinations(old, updated []APRule) int {
	n := 0
	for s := 1; s <= maxRating; s++ {
		for o := 1; o <= maxRating; o++ {
			for d := 1; d <= maxRating; d++ {
				a, _ := ActionPriority(old, s, o, d)
				b, _ := ActionPriority(updated, s, o, d)
				if a != b {
					n++
				}
			}
		}
	}
	return n
}

// SetAPRules mengganti tabel AP satu versi draft (E-11). Hanya untuk versi
// dengan metode risiko AP. Siapa yang menyetujui perubahan ini saat versi
// diterbitkan diatur D-139 dan dibangun bersama form set (langkah D).
func (s *Service) SetAPRules(ctx context.Context, actor access.Actor, versionID string, rules []APRule) (FormatVersion, error) {
	return s.editVersion(ctx, actor, versionID, audit.ActionFormatAPRulesChanged, func(q *db.Queries, v FormatVersion) (map[string]string, error) {
		if v.RiskMethod != RiskAP {
			return nil, problems(&problem.Problem{Field: "ap_rules", Code: problem.NotAllowed, Message: "Only a version with the AP risk method has an AP table."})
		}
		if err := problems(checkAPRules(rules)); err != nil {
			return nil, err
		}
		changed := changedCombinations(v.APRules, rules)
		if changed == 0 {
			return nil, nil
		}
		if err := q.SetFormatAPRules(ctx, db.SetFormatAPRulesParams{ID: v.ID, ApRules: apRulesJSON(rules)}); err != nil {
			return nil, err
		}
		return map[string]string{"changed_combinations": strconv.Itoa(changed)}, nil
	})
}

// ActionPriorityOf menghitung AP untuk satu kombinasi S, O, D memakai tabel
// satu versi. Semua akun boleh (L-2), karena ini hanya membaca.
func (s *Service) ActionPriorityOf(ctx context.Context, actor access.Actor, versionID string, severity, occurrence, detection int) (string, error) {
	v, err := s.GetFormatVersion(ctx, actor, versionID)
	if err != nil {
		return "", err
	}
	check := func(field string, value int) *problem.Problem {
		if value < 1 || value > maxRating {
			return &problem.Problem{Field: field, Code: problem.InvalidFormat, Message: "Use a whole number from 1 to 10."}
		}
		return nil
	}
	if err := problems(check("severity", severity), check("occurrence", occurrence), check("detection", detection)); err != nil {
		return "", err
	}
	priority, ok := ActionPriority(v.APRules, severity, occurrence, detection)
	if !ok {
		return "", problems(&problem.Problem{Field: "ap_rules", Code: problem.NotAllowed, Message: "This version has no AP table."})
	}
	return priority, nil
}
