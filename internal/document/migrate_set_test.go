package document

import (
	"errors"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
)

// Menu migrasi (D-215, D-216): satu set dokumen lama masuk sekaligus, dengan
// nomor asli yang diketik di tiga kolom.

func legacySetDoc(key, number string, revision int) LegacySetDocument {
	return LegacySetDocument{TypeKey: key, Number: number, Revision: revision, EffectiveDate: "2025-10-23",
		PreparedBy: "Budi Santoso", ApprovedBy: "Siti Aminah"}
}

func (f fixture) numberTypeID(t *testing.T, key string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(t.Context(), "SELECT id FROM document_types WHERE key = $1", key).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f fixture) registered(t *testing.T, number string) bool {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(t.Context(), "SELECT count(*) FROM document_numbers WHERE number = $1", number).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func numbersOf(docs []Document) []string {
	var out []string
	for _, d := range docs {
		out = append(out, d.Number)
	}
	return out
}

func TestMigrateSet(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	staff := newActor(t, f.pool, "ppap.staff", access.AccountWork, all(access.RoleMigration))
	oneSet := LegacySet{CustomerID: f.custAB, OneSet: true, ModelIDs: []string{f.k125}, Documents: []LegacySetDocument{
		legacySetDoc("pfd", "SH-AB-020", 2), legacySetDoc("pfmea", "", 3), legacySetDoc("cp", "", 4),
	}}

	// Mode migrasi mati, lalu akun tanpa E-10.
	if _, err := f.s.MigrateSet(ctx, staff, oneSet); code(t, err) != "migration_mode:not_allowed" {
		t.Errorf("mode mati: %v", err)
	}
	f.migrationMode(t, true)
	if _, err := f.s.MigrateSet(ctx, f.author, oneSet); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("tanpa E-10: %v", err)
	}

	// Penolakan per kolom. Sesudah penolakan tidak ada nomor yang tertinggal.
	for name, bad := range map[string]func(*LegacySet){
		"documents.pfd.number:invalid_format":     func(l *LegacySet) { l.Documents[0].Number = "SH-AB-020-02" },
		"documents.pfd.number:not_allowed":        func(l *LegacySet) { l.Documents[0].Number = "SH-ZZ-020" },
		"documents.cp.number:not_allowed":         func(l *LegacySet) { l.Documents[2].Number = "SC-AB-021" },
		"documents.pfmea.revision:invalid_format": func(l *LegacySet) { l.Documents[1].Revision = -1 },
		"documents.cp.effective_date:required":    func(l *LegacySet) { l.Documents[2].EffectiveDate = "" },
		"model_ids:required":                      func(l *LegacySet) { l.ModelIDs = nil },
		"documents.pfmea.number:required": func(l *LegacySet) {
			l.OneSet = false
		},
		"documents:required": func(l *LegacySet) { l.Documents = nil },
		"documents.pfd.document_type_key:invalid_format": func(l *LegacySet) {
			l.Documents = append(l.Documents, legacySetDoc("pfd", "SH-AB-022", 1))
		},
	} {
		wrong := oneSet
		wrong.Documents = append([]LegacySetDocument(nil), oneSet.Documents...)
		bad(&wrong)
		if _, err := f.s.MigrateSet(ctx, staff, wrong); code(t, err) != name {
			t.Errorf("%s: %v", name, err)
		}
	}
	if f.registered(t, "SH-AB-020") {
		t.Fatal("penolakan meninggalkan nomor di register")
	}

	// Satu set: cukup nomor PFD; PFMEA dan CP mengikuti nomor berjalan 020.
	docs, err := f.s.MigrateSet(ctx, staff, oneSet)
	if err != nil {
		t.Fatal(err)
	}
	if got := numbersOf(docs); len(got) != 3 || got[0] != "SH-AB-020" || got[1] != "SF-AB-020" || got[2] != "SC-AB-020" {
		t.Fatalf("nomor set: %v", got)
	}
	for i, rev := range []int{2, 3, 4} {
		d := docs[i]
		if d.Revision != rev || d.Origin != OriginMigrated || d.Baseline == nil || d.Baseline.PreparedBy != "Budi Santoso" || len(d.Set) != 3 {
			t.Errorf("dokumen %s: revisi %d, asal %s, set %d", d.Number, d.Revision, d.Origin, len(d.Set))
		}
	}
	if pfdOf(docs[2]) != docs[0].ID {
		t.Error("CP harus menemukan PFD set yang sama")
	}
	// Nomor baru melanjutkan sesudah nomor lama.
	if next, err := f.numbers.NextNumber(ctx, f.author, f.custAB, f.numberTypeID(t, "pfd")); err != nil || next != "SH-AB-021" {
		t.Errorf("nomor PFD berikutnya %q, %v", next, err)
	}
	// Nomor yang sudah punya dokumen ditolak.
	if _, err := f.s.MigrateSet(ctx, staff, oneSet); code(t, err) != "documents.pfd.number:not_allowed" {
		t.Errorf("nomor dipakai lagi: %v", err)
	}

	// Tidak sejajar: ketiga nomor diketik, tetap satu set.
	docs, err = f.s.MigrateSet(ctx, staff, LegacySet{CustomerID: f.custAB, ModelIDs: []string{f.k125}, Documents: []LegacySetDocument{
		legacySetDoc("cp", "SC-AB-005", 4), legacySetDoc("pfmea", "SF-AB-005", 3), legacySetDoc("pfd", "SH-AB-004", 2),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := numbersOf(docs); len(got) != 3 || got[0] != "SH-AB-004" || got[2] != "SC-AB-005" || pfdOf(docs[2]) != docs[0].ID {
		t.Errorf("tidak sejajar: %v", got)
	}

	// Hanya satu dokumen (misalnya CP saja).
	docs, err = f.s.MigrateSet(ctx, staff, LegacySet{CustomerID: f.custAB, ModelIDs: []string{f.k125}, Documents: []LegacySetDocument{
		legacySetDoc("cp", "SC-AB-030", 1),
	}})
	if err != nil || len(docs) != 1 || docs[0].Number != "SC-AB-030" {
		t.Errorf("CP saja: %v, %v", numbersOf(docs), err)
	}

	// Nomor yang sudah terdaftar sebagai set dipakai apa adanya.
	reserved := f.reservedNumbers(t)
	var cpText string
	if err := f.pool.QueryRow(ctx, "SELECT number FROM document_numbers WHERE id = $1", reserved["cp"]).Scan(&cpText); err != nil {
		t.Fatal(err)
	}
	docs, err = f.s.MigrateSet(ctx, staff, LegacySet{CustomerID: f.custAB, OneSet: true, ModelIDs: []string{f.k125}, Documents: []LegacySetDocument{
		legacySetDoc("pfmea", "", 1), legacySetDoc("cp", cpText, 1),
	}})
	if err != nil || len(docs) != 2 || docs[1].Number != cpText {
		t.Fatalf("nomor terdaftar: %v, %v", numbersOf(docs), err)
	}

	// Nomor lama dari set lain dicampur dengan nomor baru: ditolak.
	if _, err := f.s.MigrateSet(ctx, staff, LegacySet{CustomerID: f.custAB, ModelIDs: []string{f.k125}, Documents: []LegacySetDocument{
		legacySetDoc("pfd", "SH-AB-040", 1), legacySetDoc("cp", "SC-AB-030", 1),
	}}); code(t, err) != "documents.cp.number:not_allowed" {
		t.Errorf("campuran set: %v", err)
	}

	// Papan menu migrasi: prasyarat dan status setiap dokumen lama.
	view, err := f.s.MigrationOverview(ctx, staff, f.custAB)
	if err != nil {
		t.Fatal(err)
	}
	if !view.MigrationMode || !view.FormSetPublished || view.Models != 2 || view.Processes != 1 || len(view.Documents) != 9 {
		t.Errorf("papan: %+v", view)
	}
	if first := view.Documents[0]; first.Number != "SH-AB-004" || first.State != LegacyEmpty || first.EnteredBy == "" {
		t.Errorf("baris pertama: %+v", first)
	}
	if _, err := f.s.MigrationOverview(ctx, f.viewer, f.custAB); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("papan tanpa E-10: %v", err)
	}
}
