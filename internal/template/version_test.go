package template

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
)

// M3.4 · Versi template (R-TPL-05): versi berjalan dari draft ke terbit
// (published). Versi terbit tidak bisa diubah; perubahan berikutnya dibuat di
// versi baru yang menyalin isi versi terbit terakhir, dengan ID field,
// sub-proses, dan poin yang sama. Propagasi ke dokumen belum ada (M5.8).

// publishedCode adalah SQLSTATE dari trigger penolak versi terbit (migrasi 00016).
const publishedCode = "55000"

// lastAudit membaca entri audit terakhir.
func lastAudit(t *testing.T, f fixture) (action, details string) {
	t.Helper()
	err := f.pool.QueryRow(t.Context(), "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details)
	if err != nil {
		t.Fatal(err)
	}
	return action, details
}

// TestPublishFormatVersion: hanya admin (A-2) yang menerbitkan; sesudah
// terbit, versi itu menolak setiap perubahan dan tidak bisa diterbitkan lagi.
func TestPublishFormatVersion(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	draft := f.version(t, "PFMEA AIAG 4th")
	if draft.Status != StatusDraft || draft.PublishedAt != nil {
		t.Fatalf("versi awal: status %s, terbit %v", draft.Status, draft.PublishedAt)
	}

	if _, err := f.s.PublishFormatVersion(ctx, f.author, draft.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun menerbitkan: %v", err)
	}
	v, err := f.s.PublishFormatVersion(ctx, f.master, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != StatusPublished || v.PublishedAt == nil {
		t.Errorf("sesudah terbit: status %s, terbit %v", v.Status, v.PublishedAt)
	}
	if action, details := lastAudit(t, f); action != "format_template.version_published" || details != `{"version":"1"}` {
		t.Errorf("entri audit: %s %s", action, details)
	}
	summary := f.template(t, "PFMEA AIAG 4th").Versions[0]
	if summary.Status != StatusPublished || summary.PublishedAt == nil {
		t.Errorf("ringkasan versi: %+v", summary)
	}

	// Versi terbit tidak bisa diubah lewat layanan.
	if _, err := f.s.AddField(ctx, f.master, v.ID, NewField{Area: AreaRow, Kind: KindText, Label: "Note"}); !slices.Equal(problemCodes(t, err), []string{"status:not_allowed"}) {
		t.Errorf("menambah kolom di versi terbit: %v", err)
	}
	if _, err := f.s.SetHighlights(ctx, f.master, v.ID, HighlightSettings{RPNAbove: new(80)}); !slices.Equal(problemCodes(t, err), []string{"status:not_allowed"}) {
		t.Errorf("mengubah aturan sorot di versi terbit: %v", err)
	}
	if _, err := f.s.PublishFormatVersion(ctx, f.master, v.ID); !slices.Equal(problemCodes(t, err), []string{"status:not_allowed"}) {
		t.Errorf("menerbitkan dua kali: %v", err)
	}
}

// TestNewFormatVersion: versi baru hanya dibuat setelah versi terakhir
// terbit. Isinya salinan versi itu dengan ID field yang sama, dan mengubah
// versi baru tidak mengubah versi terbit.
func TestNewFormatVersion(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	ft := f.template(t, "PFMEA AIAG 4th")
	v1 := f.version(t, "PFMEA AIAG 4th")

	// Masih ada draft: versi baru ditolak.
	if _, err := f.s.NewFormatVersion(ctx, f.master, ft.ID); !slices.Equal(problemCodes(t, err), []string{"status:not_allowed"}) {
		t.Errorf("versi baru saat masih ada draft: %v", err)
	}
	if _, err := f.s.PublishFormatVersion(ctx, f.master, v1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.NewFormatVersion(ctx, f.author, ft.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun membuat versi baru: %v", err)
	}

	v2, err := f.s.NewFormatVersion(ctx, f.master, ft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Version != 2 || v2.Status != StatusDraft || v2.PublishedAt != nil || v2.ID == v1.ID {
		t.Fatalf("versi baru: %d %s %v", v2.Version, v2.Status, v2.PublishedAt)
	}
	if action, details := lastAudit(t, f); action != "format_template.version_created" || details != `{"copied_from":"1","version":"2"}` {
		t.Errorf("entri audit: %s %s", action, details)
	}
	// Isi sama: field dengan ID yang sama, tabel skala, simbol, dan aturan sorot.
	if !slices.EqualFunc(v1.Fields, v2.Fields, func(a, b Field) bool {
		return a.ID == b.ID && a.Label == b.Label && a.Position == b.Position && a.Kind == b.Kind && a.Hidden == b.Hidden
	}) {
		t.Errorf("field versi 2 berbeda dari versi 1")
	}
	if !slices.Equal(v1.Ratings, v2.Ratings) || !slices.Equal(v1.Symbols, v2.Symbols) {
		t.Errorf("tabel skala atau simbol versi 2 berbeda dari versi 1")
	}
	if v2.RiskMethod != v1.RiskMethod || v2.HighlightRPNAbove != v1.HighlightRPNAbove || v2.HighlightSeverityAtLeast != v1.HighlightSeverityAtLeast {
		t.Errorf("pengaturan versi 2 berbeda dari versi 1")
	}

	// Mengubah versi 2 tidak mengubah versi 1 (kriteria selesai M3.4).
	target := field(t, v2, "Potential Failure Mode")
	if _, err := f.s.UpdateField(ctx, f.master, v2.ID, target.ID, FieldUpdate{Label: "Failure Mode"}); err != nil {
		t.Fatal(err)
	}
	again, err := f.s.GetFormatVersion(ctx, f.author, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := field(t, again, "Potential Failure Mode"); got.ID != target.ID {
		t.Errorf("label versi 1 ikut berubah")
	}

	// Hanya satu draft per template.
	if _, err := f.s.NewFormatVersion(ctx, f.master, ft.ID); !slices.Equal(problemCodes(t, err), []string{"status:not_allowed"}) {
		t.Errorf("versi baru kedua: %v", err)
	}
	if got := f.template(t, "PFMEA AIAG 4th").Versions; len(got) != 2 || got[1].Version != 2 {
		t.Errorf("ringkasan versi: %+v", got)
	}
}

// expectCode menjalankan satu perintah SQL dan memeriksa SQLSTATE penolakannya.
func expectCode(t *testing.T, exec func(sql string, args ...any) error, code, sql string, args ...any) {
	t.Helper()
	err := exec(sql, args...)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		t.Errorf("%s: %v, ingin SQLSTATE %s", sql, err, code)
	}
}

// TestPublishedFormatGuards: database sendiri menolak perubahan pada versi
// terbit, juga untuk pemilik database yang melewati layanan (seperti audit
// log, R-SEC-03). Versi draft tetap bisa diubah.
func TestPublishedFormatGuards(t *testing.T) {
	app, owner := dbtest.NewPools(t)
	f := fixture{s: NewService(app, access.NewChecker(app)), pool: app}
	f.master = newActor(t, app, "ppap.master", access.AccountWork, access.RolePPAPEngineer)
	f.author = newActor(t, app, "penyusun.uji", access.AccountWork, access.RoleAuthor)
	ctx := t.Context()
	ft := f.template(t, "PFMEA AIAG 4th")
	v1 := f.version(t, "PFMEA AIAG 4th")
	asOwner := func(sql string, args ...any) error {
		_, err := owner.Exec(ctx, sql, args...)
		return err
	}

	// Selama draft, pemilik database masih boleh mengubah (dipakai migrasi).
	if err := asOwner("UPDATE format_version_fields SET label = label WHERE version_id = $1", v1.ID); err != nil {
		t.Fatalf("mengubah draft: %v", err)
	}
	if _, err := f.s.PublishFormatVersion(ctx, f.master, v1.ID); err != nil {
		t.Fatal(err)
	}

	fieldID := v1.Fields[0].ID
	for _, stmt := range []string{
		"UPDATE format_version_fields SET label = 'X' WHERE version_id = $1",
		"DELETE FROM format_version_fields WHERE version_id = $1",
		"UPDATE format_rating_criteria SET criteria = 'X' WHERE version_id = $1",
		"UPDATE format_symbols SET name = name || ' X' WHERE version_id = $1",
		"INSERT INTO format_symbols (version_id, key, name, position) VALUES ($1, 'baru', 'Baru', 99)",
		"UPDATE format_template_versions SET status = 'draft', published_at = NULL WHERE id = $1",
		"UPDATE format_template_versions SET highlight_rpn_above = 50 WHERE id = $1",
		"DELETE FROM format_template_versions WHERE id = $1",
	} {
		expectCode(t, asOwner, publishedCode, stmt, v1.ID)
	}
	expectCode(t, asOwner, publishedCode,
		"INSERT INTO format_version_fields (version_id, field_id, template_id, area, label, position) VALUES ($1, $2, $3, 'row', 'Baru', 99)",
		v1.ID, fieldID, ft.ID)
	expectCode(t, asOwner, publishedCode, "TRUNCATE format_rating_criteria")

	// Status terbit selalu punya waktu terbit, dan sebaliknya.
	v2, err := f.s.NewFormatVersion(ctx, f.master, ft.ID)
	if err != nil {
		t.Fatal(err)
	}
	expectCode(t, asOwner, "23514", "UPDATE format_template_versions SET status = 'published' WHERE id = $1", v2.ID)
	// Draft kedua untuk template yang sama ditolak indeks unik.
	expectCode(t, asOwner, "23505",
		"INSERT INTO format_template_versions (template_id, version, risk_method) VALUES ($1, 3, 'rpn')", ft.ID)
}

// TestPublishProcessVersion: hanya Pengelola template (E-6) yang menerbitkan
// process template; versi terbit menolak perubahan sub-proses, poin, dan varian.
func TestPublishProcessVersion(t *testing.T) {
	f := newProcessFixture(t)
	ctx := t.Context()
	draft := f.newTemplate(t, "IQC", "IQC General")
	draft = f.addPoint(t, draft, "IQC General", "cp", "Check label")
	point := draft.Subprocesses[0].Points[0]

	for _, actor := range []access.Actor{f.author, f.admin} {
		if _, err := f.s.PublishProcessVersion(ctx, actor, draft.ID); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s menerbitkan: %v", actor.Login, err)
		}
	}
	v, err := f.s.PublishProcessVersion(ctx, f.maintainer, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != StatusPublished || v.PublishedAt == nil {
		t.Errorf("sesudah terbit: status %s, terbit %v", v.Status, v.PublishedAt)
	}
	var action, details string
	if err := f.s.pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details); err != nil {
		t.Fatal(err)
	}
	if action != "process_template.version_published" || details != `{"version":"1"}` {
		t.Errorf("entri audit: %s %s", action, details)
	}

	notAllowed := []string{"status:not_allowed"}
	if _, err := f.s.AddSubprocess(ctx, f.maintainer, v.ID, "IQC MSD"); !slices.Equal(problemCodes(t, err), notAllowed) {
		t.Errorf("menambah sub-proses di versi terbit: %v", err)
	}
	if _, err := f.s.UpdatePoint(ctx, f.maintainer, v.ID, point.ID, PointUpdate{Text: "Check label lagi"}); !slices.Equal(problemCodes(t, err), notAllowed) {
		t.Errorf("mengubah poin di versi terbit: %v", err)
	}
	if _, err := f.s.PublishProcessVersion(ctx, f.maintainer, v.ID); !slices.Equal(problemCodes(t, err), notAllowed) {
		t.Errorf("menerbitkan dua kali: %v", err)
	}
}

// TestNewProcessVersion: versi baru menyalin sub-proses, poin, dan varian
// dengan ID yang sama (R-TPL-04), dan perubahannya tidak menyentuh versi terbit.
func TestNewProcessVersion(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()
	v1, laser := f.laserTemplate(t)
	v1, err := f.s.AddVariant(ctx, f.maintainer, v1.ID, laser.ID, NewVariant{
		Scope: ScopeInput{Level: ScopeModel, TargetID: f.modelX}, Text: "Laser type: fiber 20 W",
	})
	if err != nil {
		t.Fatal(err)
	}

	notAllowed := []string{"status:not_allowed"}
	if _, err := f.s.NewProcessVersion(ctx, f.maintainer, v1.TemplateID); !slices.Equal(problemCodes(t, err), notAllowed) {
		t.Errorf("versi baru saat masih ada draft: %v", err)
	}
	if _, err := f.s.PublishProcessVersion(ctx, f.maintainer, v1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.NewProcessVersion(ctx, f.author, v1.TemplateID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun membuat versi baru: %v", err)
	}

	v2, err := f.s.NewProcessVersion(ctx, f.maintainer, v1.TemplateID)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Version != 2 || v2.Status != StatusDraft || v2.PublishedAt != nil {
		t.Fatalf("versi baru: %d %s %v", v2.Version, v2.Status, v2.PublishedAt)
	}
	if len(v2.Subprocesses) != len(v1.Subprocesses) {
		t.Fatalf("versi 2 punya %d sub-proses, ingin %d", len(v2.Subprocesses), len(v1.Subprocesses))
	}
	for i, s1 := range v1.Subprocesses {
		s2 := v2.Subprocesses[i]
		if s1.ID != s2.ID || s1.Name != s2.Name || s1.Position != s2.Position || len(s1.Points) != len(s2.Points) {
			t.Fatalf("sub-proses %d berbeda: %+v dan %+v", i, s1, s2)
		}
		for j, p1 := range s1.Points {
			p2 := s2.Points[j]
			if p1.ID != p2.ID || p1.Text != p2.Text || p1.Position != p2.Position || p1.Scope != p2.Scope {
				t.Errorf("poin %d.%d berbeda", i, j)
			}
			if !reflect.DeepEqual(p1.Variants, p2.Variants) || !reflect.DeepEqual(p1.Content, p2.Content) {
				t.Errorf("varian poin %d.%d berbeda: %+v dan %+v", i, j, p1.Variants, p2.Variants)
			}
		}
	}

	// Ubah teks poin di versi 2: versi 1 tetap, dan pratinjau versi 1 tetap sama.
	if _, err := f.s.UpdatePoint(ctx, f.maintainer, v2.ID, laser.ID, PointUpdate{Text: "Laser type sesuai WI"}); err != nil {
		t.Fatal(err)
	}
	old, err := f.s.GetProcessVersion(ctx, f.author, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, p, ok := findPoint(old, laser.ID); !ok || p.Text != laser.Text {
		t.Errorf("teks poin versi 1 ikut berubah: %q", p.Text)
	}
	if _, err := f.s.NewProcessVersion(ctx, f.maintainer, v1.TemplateID); !slices.Equal(problemCodes(t, err), notAllowed) {
		t.Errorf("versi baru kedua: %v", err)
	}
}

// TestPublishedProcessGuards: database menolak perubahan pada process
// template yang sudah terbit, juga untuk pemilik database.
func TestPublishedProcessGuards(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()
	v1, laser := f.laserTemplate(t)
	v1, err := f.s.AddVariant(ctx, f.maintainer, v1.ID, laser.ID, NewVariant{
		Scope: ScopeInput{Level: ScopeModel, TargetID: f.modelX}, Text: "Laser type: fiber 20 W",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.PublishProcessVersion(ctx, f.maintainer, v1.ID); err != nil {
		t.Fatal(err)
	}
	// Role aplikasi punya UPDATE untuk kolom ini, jadi yang menolak adalah trigger.
	asApp := func(sql string, args ...any) error { return f.exec(t, sql, args...) }
	for _, stmt := range []string{
		"UPDATE process_template_subprocesses SET name = 'X' WHERE version_id = $1",
		"UPDATE process_template_points SET text = 'X' WHERE version_id = $1",
		"UPDATE process_template_variants SET text = 'X' WHERE version_id = $1",
		"UPDATE process_template_versions SET status = 'draft' WHERE id = $1",
	} {
		expectCode(t, asApp, publishedCode, stmt, v1.ID)
	}
	// Draft kedua ditolak indeks unik.
	if _, err := f.s.NewProcessVersion(ctx, f.maintainer, v1.TemplateID); err != nil {
		t.Fatal(err)
	}
	expectCode(t, asApp, "23505",
		"INSERT INTO process_template_versions (template_id, process_id, version) VALUES ($1, $2, 3)", v1.TemplateID, v1.Process.ID)
}
