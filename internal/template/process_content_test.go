package template

import (
	"reflect"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/db/dbtest"
)

// TestPointContent (D-146): poin dan varian menyimpan isi per kolom format,
// dengan key tetap kolom itu sebagai kuncinya. Versi baru menyalin isinya.
func TestPointContent(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()
	v, laser := f.laserTemplate(t)

	content := map[string]any{"failure_mode": " Wrong laser power ", "severity": 7, "note": "  "}
	text := laser.Text
	v, err := f.s.UpdatePoint(ctx, f.maintainer, v.ID, laser.ID, PointUpdate{Text: text, Content: &content})
	if err != nil {
		t.Fatal(err)
	}
	_, point, _ := findPoint(v, laser.ID)
	// Teks dirapikan, isian kosong dibuang, dan rating disimpan sebagai angka.
	want := map[string]any{"failure_mode": "Wrong laser power", "severity": float64(7)}
	if !reflect.DeepEqual(point.Content, want) {
		t.Errorf("isi poin: %#v", point.Content)
	}

	// Content nil berarti isi tetap (D-111).
	v, err = f.s.UpdatePoint(ctx, f.maintainer, v.ID, laser.ID, PointUpdate{Text: text + " (rev)"})
	if err != nil {
		t.Fatal(err)
	}
	if _, point, _ = findPoint(v, laser.ID); !reflect.DeepEqual(point.Content, want) {
		t.Errorf("isi poin sesudah ganti teks saja: %#v", point.Content)
	}

	// Kunci dan nilai yang salah ditolak.
	for name, bad := range map[string]map[string]any{
		"kunci":     {"Failure Mode": "x"},
		"rating":    {"severity": 11},
		"jenis":     {"failure_mode": true},
		"pecahan":   {"severity": 2.5},
		"daftar":    {"failure_mode": []any{"a"}},
		"kepanjang": {"failure_mode": string(make([]byte, MaxTextLength+1))},
	} {
		if _, err := f.s.UpdatePoint(ctx, f.maintainer, v.ID, laser.ID, PointUpdate{Text: text, Content: &bad}); !slices.Equal(problemCodes(t, err), []string{"content:invalid_format"}) {
			t.Errorf("%s: %v", name, err)
		}
	}

	// Poin baru dan varian juga membawa isi.
	sub := v.Subprocesses[0]
	v, err = f.s.AddPoint(ctx, f.maintainer, v.ID, NewPoint{
		SubprocessID: sub.ID, DocumentTypeID: laser.DocumentType.ID, Text: "Marking depth",
		Content: map[string]any{"product_characteristic": "Marking depth"},
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err = f.s.AddVariant(ctx, f.maintainer, v.ID, laser.ID, NewVariant{
		Scope: ScopeInput{Level: ScopeModel, TargetID: f.modelX}, Text: "Laser type: fiber 20 W",
		Content: map[string]any{"failure_mode": "Wrong fiber power"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, point, _ = findPoint(v, laser.ID)
	if len(point.Variants) != 1 || point.Variants[0].Content["failure_mode"] != "Wrong fiber power" {
		t.Fatalf("isi varian: %+v", point.Variants)
	}
	changed := map[string]any{"failure_mode": "Fiber power too low"}
	v, err = f.s.UpdateVariant(ctx, f.maintainer, v.ID, point.Variants[0].ID, VariantUpdate{Text: point.Variants[0].Text, Content: &changed})
	if err != nil {
		t.Fatal(err)
	}

	// Pratinjau untuk model X memakai isi varian; model lain memakai isi poin.
	resolved, err := f.s.ResolvePoints(ctx, f.author, v.ID, f.modelX)
	if err != nil {
		t.Fatal(err)
	}
	if got := resolved.Subprocesses[0].Points[0].Content["failure_mode"]; got != "Fiber power too low" {
		t.Errorf("isi untuk model X: %v", got)
	}

	// Versi baru menyalin isi poin dan varian.
	if _, err := f.s.PublishProcessVersion(ctx, f.maintainer, v.ID); err != nil {
		t.Fatal(err)
	}
	v2, err := f.s.NewProcessVersion(ctx, f.maintainer, v.TemplateID)
	if err != nil {
		t.Fatal(err)
	}
	_, point, _ = findPoint(v2, laser.ID)
	if !reflect.DeepEqual(point.Content, want) || point.Variants[0].Content["failure_mode"] != "Fiber power too low" {
		t.Errorf("isi di versi 2: %#v, %+v", point.Content, point.Variants)
	}
}

// TestSeedFieldKeys (D-146): kolom form AIAG 4th punya key yang sama dengan
// kolom sejenis di form AIAG-VDA, supaya satu poin melayani kedua form set.
func TestSeedFieldKeys(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	keys := func(template string) []string {
		rows, err := pool.Query(ctx, `SELECT f.key FROM format_fields f JOIN format_templates t ON t.id = f.template_id
			WHERE t.name = $1 AND f.area = 'row' AND f.key IS NOT NULL ORDER BY f.key`, template)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var k string
			if err := rows.Scan(&k); err != nil {
				t.Fatal(err)
			}
			out = append(out, k)
		}
		return out
	}
	for template, wantKeys := range map[string][]string{
		"PFMEA AIAG 4th": {"failure_mode", "failure_effect", "failure_cause", "prevention_control", "detection_control", "severity", "occurrence", "detection"},
		"CP":             {"product_characteristic", "process_characteristic", "control_method", "reaction_plan", "sample_frequency"},
		"PFD":            {"flow_symbol"},
	} {
		got := keys(template)
		for _, k := range wantKeys {
			if !slices.Contains(got, k) {
				t.Errorf("%s: key %s tidak ada di %v", template, k, got)
			}
		}
	}
}
