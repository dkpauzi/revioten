package auth

// Import akun dari berkas Excel (I1, D-224, E-143 opsi b). Admin mengunduh
// template, mengisinya di Excel, lalu mengunggahnya: Preview melaporkan
// setiap baris tanpa menyimpan, Import menyimpan baris yang benar. Login yang
// sudah ada dilewati, jadi berkas yang sama aman diunggah lagi, misalnya
// sesudah database direset.
//
// Semua akun hasil import memakai password sementara bersama password123.
// Supaya password yang mudah ditebak ini tidak terbawa ke pemakaian
// sungguhan, Import hanya jalan saat Migration mode menyala (Preview boleh
// kapan saja), setiap akun
// wajib mengganti password saat login pertama, dan password sementaranya
// kedaluwarsa 7 hari bila tidak dipakai. Daftar password terlarang (D-132)
// tetap berlaku untuk password yang dipilih pemiliknya.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Password sementara akun hasil import dan masa berlakunya (D-224).
const (
	ImportPassword         = "password123"
	ImportPasswordLifetime = 7 * 24 * time.Hour
)

// Batas berkas import.
const (
	MaxAccountImportBytes = 5 << 20 // 5 MB
	MaxAccountImportRows  = 500
)

// Hasil satu baris import.
const (
	ImportNew      = "new"      // akun baru (dibuat bila Import)
	ImportExisting = "existing" // login sudah ada, tidak diubah
	ImportInvalid  = "invalid"  // ada isian yang salah; lihat Problems
)

// scopeAllText adalah isian kolom Scope untuk cakupan semua customer (D-206).
// Teks layar Inggris dan Indonesia keduanya diterima.
var scopeAllText = []string{"all customers", "semua customer"}

// accountImportHeaders adalah judul kolom template, urut seperti di berkas.
var accountImportHeaders = []string{"Name", "Employee number", "Login", "Email", "Roles", "Scope"}

// AccountImportRow adalah satu baris berkas beserta hasilnya.
type AccountImportRow struct {
	Row            int // nomor baris di Excel, mulai dari 2
	Name           string
	EmployeeNumber string
	Login          string
	Email          string
	RolesText      string // isi kolom Roles apa adanya
	ScopeText      string // isi kolom Scope apa adanya
	Roles          []access.RoleGrant
	Status         string
	Problems       []problem.Problem
}

// AccountImport adalah laporan satu Preview atau Import.
type AccountImport struct {
	Rows      []AccountImportRow
	New       int
	Existing  int
	Invalid   int
	Committed bool      // true bila akun baru sudah disimpan
	ExpiresAt time.Time // batas password sementara akun baru (hanya bila Committed)
}

// AccountImportTemplate membuat berkas Excel template (A-1): sheet Accounts
// dengan judul kolom, sheet Roles berisi peran yang bisa diberikan, dan sheet
// Help berisi keterangan setiap kolom. Template tidak memuat baris contoh,
// supaya tidak ada akun contoh yang ikut terimpor.
func (s *Service) AccountImportTemplate(ctx context.Context, actor access.Actor) ([]byte, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return nil, err
	}
	catalog, err := access.LoadRoles(ctx, s.q)
	if err != nil {
		return nil, err
	}
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", "Accounts"); err != nil {
		return nil, err
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	for i, h := range accountImportHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellStr("Accounts", cell, h); err != nil {
			return nil, err
		}
	}
	_ = f.SetCellStyle("Accounts", "A1", "F1", bold)
	_ = f.SetColWidth("Accounts", "A", "A", 28)
	_ = f.SetColWidth("Accounts", "B", "D", 22)
	_ = f.SetColWidth("Accounts", "E", "E", 36)
	_ = f.SetColWidth("Accounts", "F", "F", 18)

	// Daftar peran yang bisa diberikan, supaya penulisan namanya tepat.
	if _, err := f.NewSheet("Roles"); err != nil {
		return nil, err
	}
	_ = f.SetSheetRow("Roles", "A1", &[]any{"Role (English)", "Peran (Indonesia)", "Can be limited to the own project group"})
	_ = f.SetCellStyle("Roles", "A1", "C1", bold)
	row := 2
	for _, r := range catalog {
		if !r.Active || r.ID == access.RoleAdmin || r.ID == access.RoleViewer {
			continue
		}
		scope := "No"
		if r.GroupScope {
			scope = "Yes"
		}
		cell, _ := excelize.CoordinatesToCellName(1, row)
		_ = f.SetSheetRow("Roles", cell, &[]any{r.NameEN, r.NameID, scope})
		row++
	}
	_ = f.SetColWidth("Roles", "A", "C", 30)

	if _, err := f.NewSheet("Help"); err != nil {
		return nil, err
	}
	help := [][]any{
		{"Column", "What to fill in"},
		{"Name", "Required. The full name of the person."},
		{"Employee number", "Optional. Unique for each person."},
		{"Login", "Required. Letters, digits, dots, hyphens, and underscores. A login that already exists is skipped."},
		{"Email", "Optional. For notification emails."},
		{"Roles", "Work roles from the Roles sheet, separated by commas. Empty means view only."},
		{"Scope", "All customers, or empty. Empty limits Project Leader to the own project group."},
		{"Password", "Every imported account gets the temporary password " + ImportPassword + ". It must be changed at the first log-in and expires after 7 days."},
	}
	for i, line := range help {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		_ = f.SetSheetRow("Help", cell, &line)
	}
	_ = f.SetCellStyle("Help", "A1", "B1", bold)
	_ = f.SetColWidth("Help", "A", "A", 18)
	_ = f.SetColWidth("Help", "B", "B", 100)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("menulis template akun: %w", err)
	}
	return buf.Bytes(), nil
}

// ImportAccounts membaca berkas Excel akun (A-1). commit false memberi laporan
// saja (Preview); commit true juga menyimpan akun baru dalam satu transaksi.
func (s *Service) ImportAccounts(ctx context.Context, actor access.Actor, data []byte, commit bool) (AccountImport, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return AccountImport{}, err
	}
	// Preview tidak menyimpan apa pun, jadi boleh saat Migration mode mati;
	// hanya Import yang butuh Migration mode (keputusan pemilik 25 November 2026).
	if commit {
		on, err := s.q.GetMigrationMode(ctx)
		if err != nil {
			return AccountImport{}, fmt.Errorf("membaca mode migrasi: %w", err)
		}
		if !on {
			return AccountImport{}, fileProblem("migration_mode", problem.NotAllowed,
				"Migration mode is off. Switch it on in Settings before importing accounts.")
		}
	}
	rows, err := readAccountRows(data)
	if err != nil {
		return AccountImport{}, err
	}
	catalog, err := access.LoadRoles(ctx, s.q)
	if err != nil {
		return AccountImport{}, err
	}

	report := AccountImport{Rows: rows}
	seenLogin := map[string]int{}
	seenNumber := map[string]int{}
	for i := range report.Rows {
		r := &report.Rows[i]
		if err := s.checkImportRow(ctx, r, catalog, seenLogin, seenNumber); err != nil {
			return AccountImport{}, err
		}
		switch r.Status {
		case ImportNew:
			report.New++
		case ImportExisting:
			report.Existing++
		default:
			report.Invalid++
		}
	}
	if !commit || report.New == 0 {
		report.Committed = commit
		return report, nil
	}

	// Hash dihitung sebelum transaksi: argon2id sengaja lambat. Setiap akun
	// mendapat salt sendiri, walaupun passwordnya sama.
	hashes := make([]string, len(report.Rows))
	for i, r := range report.Rows {
		if r.Status != ImportNew {
			continue
		}
		if hashes[i], err = hashPassword(ImportPassword); err != nil {
			return AccountImport{}, err
		}
	}
	expires := s.now().Add(ImportPasswordLifetime)
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		for i, r := range report.Rows {
			if r.Status != ImportNew {
				continue
			}
			person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: r.Name, EmployeeNumber: nonEmpty(r.EmployeeNumber)})
			if err != nil {
				return err
			}
			if r.Email != "" {
				if err := q.SetPersonEmail(ctx, db.SetPersonEmailParams{ID: person.ID, Email: &r.Email}); err != nil {
					return err
				}
			}
			acc, err := q.CreateAccount(ctx, db.CreateAccountParams{
				PersonID: person.ID, Kind: KindWork, Login: r.Login, PasswordHash: hashes[i], MustChangePassword: true,
			})
			if err != nil {
				return err
			}
			if err := q.SetTemporaryPasswordExpiry(ctx, db.SetTemporaryPasswordExpiryParams{ID: acc.ID, TemporaryPasswordExpiresAt: &expires}); err != nil {
				return err
			}
			if err := access.ReplaceRoles(ctx, q, acc.ID, r.Roles); err != nil {
				return err
			}
			// Password sementara tidak pernah masuk audit log (E-18).
			if err := audit.Record(ctx, q, actor, audit.Event{
				Action: audit.ActionAccountCreated, TargetType: audit.TargetAccount, TargetID: acc.ID,
				Details: map[string]string{"login": acc.Login, "kind": acc.Kind, "roles": describeGrants(r.Roles), "source": "import"},
			}); err != nil {
				return err
			}
		}
		return nil
	})
	// Akun dengan login atau nomor karyawan yang sama bisa dibuat orang lain
	// di antara pemeriksaan dan transaksi; laporannya sama dengan membuat akun.
	if conflict := accountConflict(err, KindWork); conflict != nil {
		return AccountImport{}, conflict
	}
	if err != nil {
		return AccountImport{}, fmt.Errorf("mengimpor akun: %w", err)
	}
	report.Committed = true
	report.ExpiresAt = expires.UTC()
	return report, nil
}

// checkImportRow memeriksa satu baris dan mengisi Status, Roles, dan Problems.
func (s *Service) checkImportRow(ctx context.Context, r *AccountImportRow, catalog []access.Role, seenLogin, seenNumber map[string]int) error {
	add := func(p problem.Problem) { r.Problems = append(r.Problems, p) }
	r.Login = normalizeLogin(r.Login)
	if p, ok := checkLogin(r.Login); !ok {
		add(p)
	} else if first, dup := seenLogin[r.Login]; dup {
		add(problem.Problem{Field: "login", Code: problem.Taken, Message: fmt.Sprintf("This login is already in row %d.", first)})
	} else {
		seenLogin[r.Login] = r.Row
		if _, err := s.q.GetAccountByLogin(ctx, r.Login); err == nil {
			// Login yang sudah ada dilewati tanpa memeriksa kolom lain.
			r.Status = ImportExisting
			return nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("membaca akun: %w", err)
		}
	}
	if p, ok := checkPersonName("name", r.Name); !ok {
		add(p)
	}
	if r.EmployeeNumber != "" {
		if p, ok := checkEmployeeNumber(r.EmployeeNumber); !ok {
			add(p)
		} else if first, dup := seenNumber[r.EmployeeNumber]; dup {
			add(problem.Problem{Field: "employee_number", Code: problem.Taken, Message: fmt.Sprintf("This employee number is already in row %d.", first)})
		} else {
			seenNumber[r.EmployeeNumber] = r.Row
			taken, err := s.q.EmployeeNumberTaken(ctx, &r.EmployeeNumber)
			if err != nil {
				return fmt.Errorf("membaca nomor karyawan: %w", err)
			}
			if taken {
				add(problem.Problem{Field: "employee_number", Code: problem.Taken, Message: "This employee number is already used by another person."})
			}
		}
	}
	if email, p := problem.Email("email", r.Email); p != nil {
		add(*p)
	} else if email != nil {
		r.Email = *email
	}
	r.Problems = append(r.Problems, s.importRoles(r, catalog)...)
	if len(r.Problems) > 0 {
		r.Status = ImportInvalid
	} else {
		r.Status = ImportNew
	}
	return nil
}

// importRoles membaca kolom Roles dan Scope menjadi peran akun. Nama peran
// boleh ditulis dalam bahasa Inggris atau Indonesia, besar kecil huruf bebas.
func (s *Service) importRoles(r *AccountImportRow, catalog []access.Role) []problem.Problem {
	scopeText := strings.ToLower(strings.TrimSpace(r.ScopeText))
	all := slices.Contains(scopeAllText, scopeText)
	if scopeText != "" && !all {
		return []problem.Problem{{Field: "scope", Code: problem.InvalidFormat, Message: "Write All customers, or leave the scope empty."}}
	}
	r.Roles = []access.RoleGrant{}
	for _, name := range strings.Split(r.RolesText, ",") {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		i := slices.IndexFunc(catalog, func(c access.Role) bool {
			return strings.ToLower(c.NameEN) == name || strings.ToLower(c.NameID) == name || c.ID == name
		})
		if i < 0 {
			return []problem.Problem{{Field: "roles", Code: problem.InvalidFormat, Message: "Unknown role: " + strings.TrimSpace(name) + ". Use a name from the Roles sheet."}}
		}
		scope := access.ScopeAll
		if catalog[i].GroupScope && !all {
			scope = access.ScopeOwnProjectGroup
		}
		r.Roles = append(r.Roles, access.RoleGrant{Role: catalog[i].ID, Scope: scope})
	}
	// Urut seperti katalog, seperti peran yang dibaca dari database.
	slices.SortStableFunc(r.Roles, func(a, b access.RoleGrant) int {
		return slices.IndexFunc(catalog, func(c access.Role) bool { return c.ID == a.Role }) -
			slices.IndexFunc(catalog, func(c access.Role) bool { return c.ID == b.Role })
	})
	return access.ValidateRoles(KindWork, r.Roles, catalog)
}

// readAccountRows membaca sheet pertama berkas Excel. Baris pertama adalah
// judul kolom; urutan kolom bebas, asal judulnya seperti template.
func readAccountRows(data []byte) ([]AccountImportRow, error) {
	if len(data) == 0 {
		return nil, fileProblem("file", problem.Required, "Choose the Excel file.")
	}
	if len(data) > MaxAccountImportBytes {
		return nil, fileProblem("file", problem.TooLong, "The Excel file can be at most 5 MB.")
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fileProblem("file", problem.InvalidFormat, "Upload an Excel workbook (.xlsx). Save an old .xls file as .xlsx first.")
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return nil, fileProblem("file", problem.InvalidFormat, "The Excel file has no sheet.")
	}
	lines, err := book.GetRows(sheets[0])
	if err != nil {
		return nil, fileProblem("file", problem.InvalidFormat, "The first sheet cannot be read.")
	}
	column := map[string]int{}
	if len(lines) > 0 {
		for i, h := range lines[0] {
			column[strings.ToLower(strings.TrimSpace(h))] = i
		}
	}
	for _, need := range []string{"name", "login"} {
		if _, ok := column[need]; !ok {
			return nil, fileProblem("file", problem.InvalidFormat,
				"The first row of the first sheet must have the columns "+strings.Join(accountImportHeaders, ", ")+". Download the template.")
		}
	}
	cell := func(line []string, name string) string {
		i, ok := column[strings.ToLower(name)]
		if !ok || i >= len(line) {
			return ""
		}
		return strings.TrimSpace(line[i])
	}
	var rows []AccountImportRow
	for y, line := range lines[1:] {
		r := AccountImportRow{
			Row: y + 2, Name: cell(line, "Name"), EmployeeNumber: cell(line, "Employee number"), Login: cell(line, "Login"),
			Email: cell(line, "Email"), RolesText: cell(line, "Roles"), ScopeText: cell(line, "Scope"),
		}
		if r.Name == "" && r.EmployeeNumber == "" && r.Login == "" && r.Email == "" && r.RolesText == "" && r.ScopeText == "" {
			continue // baris kosong
		}
		rows = append(rows, r)
	}
	if len(rows) == 0 {
		return nil, fileProblem("file", problem.Required, "The Excel file has no account rows.")
	}
	if len(rows) > MaxAccountImportRows {
		return nil, fileProblem("file", problem.TooLong, fmt.Sprintf("Import at most %d accounts at a time.", MaxAccountImportRows))
	}
	return rows, nil
}

func fileProblem(field, code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: field, Code: code, Message: message}}}
}
