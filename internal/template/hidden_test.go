package template

import (
	"errors"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
)

// D-218: format template dan form set bisa disembunyikan. Yang tersembunyi
// tidak ditawarkan untuk pilihan baru, tetapi dokumen yang sudah memakainya
// tetap. Untuk saat ini set AIAG-VDA beserta PFMEA dan CP-nya disembunyikan,
// dan format template diberi nama baru (v1, v2).

// TestRenamedAndHiddenForms: nama baru dan isi awal saklar hidden (migrasi 00054).
func TestRenamedAndHiddenForms(t *testing.T) {
	f := newFormSetFixture(t)
	list, err := f.s.ListFormatTemplates(t.Context(), f.author)
	if err != nil {
		t.Fatal(err)
	}
	hidden := map[string]bool{}
	for _, ft := range list {
		hidden[ft.Name] = ft.Hidden
	}
	want := map[string]bool{"PFD v1": false, "PFMEA v1": false, "Control Plan v1": false, "PFMEA v2": true, "Control Plan v2": true}
	for name, h := range want {
		got, ok := hidden[name]
		if !ok || got != h {
			t.Errorf("%s: ada %v, hidden %v; ingin hidden %v", name, ok, got, h)
		}
	}
	if old, vda := f.set(t, "aiag_4th"), f.set(t, "aiag_vda"); old.Hidden || !vda.Hidden {
		t.Errorf("hidden form set: AIAG 4th %v, AIAG-VDA %v", old.Hidden, vda.Hidden)
	}
}

// TestHideFormatTemplate: hanya E-11, dan tercatat di audit log.
func TestHideFormatTemplate(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	vda := f.template(t, "PFMEA v2")
	if _, err := f.s.SetFormatTemplateHidden(ctx, f.author, vda.ID, false); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("tanpa E-11: %v", err)
	}
	got, err := f.s.SetFormatTemplateHidden(ctx, f.master, vda.ID, false)
	if err != nil || got.Hidden {
		t.Fatalf("tampilkan: %+v, %v", got.Hidden, err)
	}
	if action, details := lastAudit(t, f); action != "format_template.hidden_changed" || details != `{"hidden":"true → false"}` {
		t.Errorf("entri audit: %s %s", action, details)
	}
	// Nilai yang sama tidak menulis audit lagi.
	if _, err := f.s.SetFormatTemplateHidden(ctx, f.master, vda.ID, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.s.SetFormatTemplateHidden(ctx, f.master, vda.ID, true); !got.Hidden {
		t.Error("sembunyikan lagi gagal")
	}
	if _, err := f.s.SetFormatTemplateHidden(ctx, f.master, "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", true); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("template tidak ada: %v", err)
	}
}

// TestHideFormSet: set bawaan dan set yang dipilih customer tidak bisa
// disembunyikan; set tersembunyi tidak bisa dipilih customer.
func TestHideFormSet(t *testing.T) {
	f := newFormSetFixture(t)
	ctx := t.Context()
	old, vda := f.set(t, "aiag_4th"), f.set(t, "aiag_vda")
	if _, err := f.s.SetFormSetHidden(ctx, f.author, vda.ID, false); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("tanpa E-11: %v", err)
	}
	if _, err := f.s.SetFormSetHidden(ctx, f.master, old.ID, true); !slices.Equal(problemCodes(t, err), []string{"hidden:not_allowed"}) {
		t.Errorf("set bawaan disembunyikan: %v", err)
	}
	if err := f.s.SetCustomerFormSet(ctx, f.master, f.customer, vda.ID); !slices.Equal(problemCodes(t, err), []string{"form_set_id:not_allowed"}) {
		t.Errorf("customer memilih set tersembunyi: %v", err)
	}
	got, err := f.s.SetFormSetHidden(ctx, f.master, vda.ID, false)
	if err != nil || got.Hidden {
		t.Fatalf("tampilkan: %v, %v", got.Hidden, err)
	}
	if action, details := lastAudit(t, f.fixture); action != "form_set.hidden_changed" || details != `{"hidden":"true → false"}` {
		t.Errorf("entri audit: %s %s", action, details)
	}
	if err := f.s.SetCustomerFormSet(ctx, f.master, f.customer, vda.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.SetFormSetHidden(ctx, f.master, vda.ID, true); !slices.Equal(problemCodes(t, err), []string{"hidden:not_allowed"}) {
		t.Errorf("set yang dipilih customer disembunyikan: %v", err)
	}
	if err := f.s.SetCustomerFormSet(ctx, f.master, f.customer, ""); err != nil {
		t.Fatal(err)
	}
	if got, err := f.s.SetFormSetHidden(ctx, f.master, vda.ID, true); err != nil || !got.Hidden {
		t.Errorf("sembunyikan lagi: %v, %v", got.Hidden, err)
	}
}
