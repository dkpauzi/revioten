package auth

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
)

// I1 (D-224, E-143 opsi b): admin mengimpor akun dari berkas Excel. Semua akun
// hasil import memakai password sementara password123, hanya saat Migration
// mode menyala, wajib diganti saat login pertama, dan kedaluwarsa 7 hari.

// accountsBook membuat berkas Excel dengan judul kolom template dan baris isi.
func accountsBook(t *testing.T, rows ...[]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	all := append([][]string{accountImportHeaders}, rows...)
	for y, row := range all {
		for x, text := range row {
			cell, _ := excelize.CoordinatesToCellName(x+1, y+1)
			if err := f.SetCellStr("Sheet1", cell, text); err != nil {
				t.Fatal(err)
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func setMigrationMode(t *testing.T, s *Service, on bool) {
	t.Helper()
	if err := s.q.SetMigrationMode(t.Context(), on); err != nil {
		t.Fatal(err)
	}
}

func rowStatuses(r AccountImport) []string {
	var out []string
	for _, row := range r.Rows {
		out = append(out, row.Status)
	}
	return out
}

func TestImportAccounts(t *testing.T) {
	s, clock := newServiceWithAdmin(t)
	ctx := t.Context()
	admin, _, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	// Satu akun sudah ada: dilewati, tidak diubah.
	if _, _, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Sudah Ada", Kind: KindWork, Login: "sudah.ada"}); err != nil {
		t.Fatal(err)
	}

	book := accountsBook(t,
		[]string{"Rina Pratiwi", "K-101", "Rina.Pratiwi", "rina@example.co.id", "PPAP Engineer", ""},
		[]string{"Budi Santoso", "", "budi.santoso", "", "Project Leader, PPAP Engineer", "All customers"},
		[]string{"Dewi Lestari", "K-103", "dewi.lestari", "", "Project Leader", ""},
		[]string{"Sudah Ada", "", "sudah.ada", "", "PPAP Engineer", ""},
		[]string{"", "", "", "", "", ""}, // baris kosong diabaikan
		[]string{"Tanpa Login", "", "", "", "PPAP Engineer", ""},
		[]string{"Peran Salah", "", "peran.salah", "", "Penyihir", ""},
		[]string{"Email Salah", "", "email.salah", "bukan-email", "", ""},
		[]string{"Rina Kembar", "", "rina.pratiwi", "", "", ""},
	)

	// Hanya admin (A-1).
	ppap, _, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "PPAP Uji", Kind: KindWork, Login: "ppap.uji",
		Roles: []access.RoleGrant{{Role: access.RolePPAPEngineer, Scope: access.ScopeAll}}})
	if err != nil {
		t.Fatal(err)
	}
	ppapActor := access.Actor{Type: access.ActorHuman, AccountID: ppap.ID, AccountKind: KindWork, PersonID: ppap.PersonID}
	if _, err := s.ImportAccounts(ctx, ppapActor, book, false); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("PPAP Engineer mengimpor akun: %v", err)
	}

	// Migration mode mati: Preview boleh (tidak menyimpan), Import ditolak.
	if r, err := s.ImportAccounts(ctx, admin, book, false); err != nil || r.Committed || r.New != 3 {
		t.Errorf("Preview tanpa Migration mode: %+v, %v", r, err)
	}
	if _, err := s.ImportAccounts(ctx, admin, book, true); !slices.Equal(problemCodes(t, err), []string{"migration_mode:not_allowed"}) {
		t.Errorf("Import tanpa Migration mode: %v", err)
	}
	setMigrationMode(t, s, true)

	// Bukan berkas Excel.
	if _, err := s.ImportAccounts(ctx, admin, []byte("bukan excel"), false); !slices.Equal(problemCodes(t, err), []string{"file:invalid_format"}) {
		t.Errorf("bukan Excel: %v", err)
	}
	// Judul kolom yang tidak dikenal.
	f := excelize.NewFile()
	_ = f.SetCellStr("Sheet1", "A1", "Nama")
	var wrong bytes.Buffer
	_ = f.Write(&wrong)
	if _, err := s.ImportAccounts(ctx, admin, wrong.Bytes(), false); !slices.Equal(problemCodes(t, err), []string{"file:invalid_format"}) {
		t.Errorf("judul kolom salah: %v", err)
	}

	// Preview tidak menyimpan apa pun.
	preview, err := s.ImportAccounts(ctx, admin, book, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{ImportNew, ImportNew, ImportNew, ImportExisting, ImportInvalid, ImportInvalid, ImportInvalid, ImportInvalid}
	if !slices.Equal(rowStatuses(preview), want) || preview.Committed || preview.New != 3 || preview.Existing != 1 || preview.Invalid != 4 {
		t.Fatalf("preview: %v, %+v", rowStatuses(preview), preview)
	}
	if preview.Rows[0].Row != 2 || preview.Rows[4].Row != 7 {
		t.Errorf("nomor baris Excel: %d, %d", preview.Rows[0].Row, preview.Rows[4].Row)
	}
	if _, err := s.q.GetAccountByLogin(ctx, "rina.pratiwi"); err == nil {
		t.Error("Preview menyimpan akun")
	}

	// Import: baris yang benar disimpan, yang lain dilaporkan.
	clock.Advance(time.Minute)
	done, err := s.ImportAccounts(ctx, admin, book, true)
	if err != nil {
		t.Fatal(err)
	}
	if !done.Committed || done.New != 3 || !done.ExpiresAt.Equal(clock.Now().Add(ImportPasswordLifetime)) {
		t.Fatalf("import: %+v", done)
	}
	accounts, err := s.ListAccounts(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	byLogin := map[string]AccountInfo{}
	for _, a := range accounts {
		byLogin[a.Login] = a
	}
	rina, budi, dewi := byLogin["rina.pratiwi"], byLogin["budi.santoso"], byLogin["dewi.lestari"]
	if rina.EmployeeNumber != "K-101" || rina.Email != "rina@example.co.id" || !rina.MustChangePassword ||
		!slices.Equal(rina.Roles, []access.RoleGrant{{Role: access.RolePPAPEngineer, Scope: access.ScopeAll}}) {
		t.Errorf("Rina: %+v", rina)
	}
	if !slices.Equal(budi.Roles, []access.RoleGrant{{Role: access.RolePPAPEngineer, Scope: access.ScopeAll}, {Role: access.RoleProjectLeader, Scope: access.ScopeAll}}) {
		t.Errorf("Budi (All customers): %+v", budi.Roles)
	}
	if !slices.Equal(dewi.Roles, []access.RoleGrant{{Role: access.RoleProjectLeader, Scope: access.ScopeOwnProjectGroup}}) {
		t.Errorf("Dewi (Project Group sendiri): %+v", dewi.Roles)
	}
	if rina.TemporaryPasswordExpiresAt == nil {
		t.Error("akun hasil import tanpa batas password sementara")
	}

	// Import ulang berkas yang sama: semua sudah ada.
	again, err := s.ImportAccounts(ctx, admin, book, true)
	if err != nil || again.New != 0 || again.Existing != 4 {
		t.Errorf("import ulang: %+v, %v", again, err)
	}

	// Password sementara tidak pernah masuk audit log.
	var leaked bool
	if err := s.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM audit_log WHERE details::text LIKE '%"+ImportPassword+"%')").Scan(&leaked); err != nil || leaked {
		t.Errorf("password di audit log: %v, %v", leaked, err)
	}

	// Login pertama dengan password123, lalu wajib ganti.
	actor, _, err := s.Login(ctx, "rina.pratiwi", ImportPassword)
	if err != nil || !actor.MustChangePassword {
		t.Fatalf("login pertama: %+v, %v", actor, err)
	}
	if _, err := s.ChangePassword(ctx, actor, ImportPassword, "kopi pahit di pagi hari"); err != nil {
		t.Fatal(err)
	}
	if info, _ := s.GetAccount(ctx, admin, rina.ID); info.TemporaryPasswordExpiresAt != nil {
		t.Error("batas password sementara masih ada sesudah ganti password")
	}

	// Lewat 7 hari: password sementara ditolak seperti password salah.
	clock.Advance(ImportPasswordLifetime)
	if _, _, err := s.Login(ctx, "budi.santoso", ImportPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("password sementara kedaluwarsa: %v", err)
	}
	// Admin mereset: password sementara acak tanpa batas 7 hari (seperti biasa).
	temporary, err := s.ResetPassword(ctx, admin, budi.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Login(ctx, "budi.santoso", temporary); err != nil {
		t.Errorf("login sesudah reset: %v", err)
	}
	// Password pilihan sendiri tidak pernah kedaluwarsa.
	if _, _, err := s.Login(ctx, "rina.pratiwi", "kopi pahit di pagi hari"); err != nil {
		t.Errorf("login dengan password sendiri: %v", err)
	}
}

func TestAccountImportTemplate(t *testing.T) {
	s, admin := newAdminActor(t)
	data, err := s.AccountImportTemplate(t.Context(), admin)
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil || len(rows) < 1 || !slices.Equal(rows[0], accountImportHeaders) {
		t.Fatalf("judul kolom template: %v, %v", rows, err)
	}
	// Sheet daftar peran memuat nama peran aktif.
	roles, _ := f.GetRows("Roles")
	var text []string
	for _, r := range roles {
		text = append(text, strings.Join(r, " "))
	}
	if !strings.Contains(strings.Join(text, "\n"), "PPAP Engineer") {
		t.Errorf("daftar peran: %v", text)
	}
}
