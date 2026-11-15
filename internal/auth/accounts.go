package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// AccountInfo adalah data satu akun untuk layar admin (UC-02, A-1).
type AccountInfo struct {
	ID                 string
	Kind               string // KindWork atau KindAdmin
	Login              string
	PersonID           string
	PersonName         string
	EmployeeNumber     string // kosong bila tidak diisi
	Email              string // kosong bila tidak diisi (D-194)
	Roles              []access.RoleGrant
	MustChangePassword bool
	LockedUntil        *time.Time
	Disabled           bool
	CreatedAt          time.Time
}

// NewAccount adalah input untuk membuat akun. Isi PersonID untuk orang yang
// sudah ada (misalnya akun admin untuk PPAP Staff yang sudah punya akun
// kerja, D-72), atau PersonName untuk orang baru.
type NewAccount struct {
	PersonID       string
	PersonName     string
	EmployeeNumber string
	Kind           string
	Login          string
	Roles          []access.RoleGrant // untuk akun kerja; akun admin otomatis memegang Admin
}

// ListAccounts mengembalikan semua akun beserta orang dan perannya (A-1).
func (s *Service) ListAccounts(ctx context.Context, actor access.Actor) ([]AccountInfo, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return nil, err
	}
	rows, err := s.q.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca akun: %w", err)
	}
	roles, err := s.q.ListAccountRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca peran akun: %w", err)
	}
	list := make([]AccountInfo, 0, len(rows))
	for _, r := range rows {
		info := AccountInfo{
			ID: r.ID, Kind: r.Kind, Login: r.Login, PersonID: r.PersonID, PersonName: r.PersonName,
			EmployeeNumber: deref(r.EmployeeNumber), Email: deref(r.Email), MustChangePassword: r.MustChangePassword,
			LockedUntil: r.LockedUntil, Disabled: r.DisabledAt != nil, CreatedAt: r.CreatedAt.UTC(),
			Roles: []access.RoleGrant{},
		}
		for _, ar := range roles {
			if ar.AccountID == r.ID {
				info.Roles = append(info.Roles, access.RoleGrant{Role: ar.RoleID, Scope: ar.Scope})
			}
		}
		list = append(list, info)
	}
	return list, nil
}

// GetAccount mengembalikan satu akun beserta orang dan perannya (A-1).
func (s *Service) GetAccount(ctx context.Context, actor access.Actor, accountID string) (AccountInfo, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return AccountInfo{}, err
	}
	return s.accountInfo(ctx, accountID)
}

// CreateAccount membuat akun dengan password sementara acak (A-1). Password
// sementara dikembalikan sekali untuk disampaikan admin ke pemiliknya, dan
// pemiliknya wajib menggantinya saat login pertama (E-08).
func (s *Service) CreateAccount(ctx context.Context, actor access.Actor, in NewAccount) (AccountInfo, string, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return AccountInfo{}, "", err
	}
	in.PersonID = strings.TrimSpace(in.PersonID)
	in.PersonName = strings.TrimSpace(in.PersonName)
	in.EmployeeNumber = strings.TrimSpace(in.EmployeeNumber)
	in.Login = normalizeLogin(in.Login)

	problems, err := s.checkNewAccount(ctx, in)
	if err != nil {
		return AccountInfo{}, "", err
	}
	if len(problems) > 0 {
		return AccountInfo{}, "", &problem.ValidationError{Problems: problems}
	}

	temporary := newTemporaryPassword()
	hash, err := hashPassword(temporary)
	if err != nil {
		return AccountInfo{}, "", err
	}
	grants := in.Roles
	if in.Kind == KindAdmin {
		// Akun admin selalu memegang peran Admin, dan hanya itu (D-72).
		grants = []access.RoleGrant{{Role: access.RoleAdmin, Scope: access.ScopeAll}}
	}

	var accountID string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		personID := in.PersonID
		if personID == "" {
			person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: in.PersonName, EmployeeNumber: nonEmpty(in.EmployeeNumber)})
			if err != nil {
				return err
			}
			personID = person.ID
		}
		acc, err := q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: personID, Kind: in.Kind, Login: in.Login, PasswordHash: hash, MustChangePassword: true,
		})
		if err != nil {
			return err
		}
		accountID = acc.ID
		if err := access.ReplaceRoles(ctx, q, acc.ID, grants); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionAccountCreated, TargetType: audit.TargetAccount, TargetID: acc.ID,
			Details: map[string]string{"login": acc.Login, "kind": acc.Kind, "roles": describeGrants(grants)},
		})
	})
	if conflict := accountConflict(err, in.Kind); conflict != nil {
		return AccountInfo{}, "", conflict
	}
	if err != nil {
		return AccountInfo{}, "", fmt.Errorf("membuat akun: %w", err)
	}

	info, err := s.accountInfo(ctx, accountID)
	if err != nil {
		return AccountInfo{}, "", err
	}
	return info, temporary, nil
}

// checkNewAccount memeriksa input akun baru dan mengumpulkan semua masalahnya.
func (s *Service) checkNewAccount(ctx context.Context, in NewAccount) ([]problem.Problem, error) {
	var problems []problem.Problem
	switch {
	case in.PersonID == "" && in.PersonName == "":
		problems = append(problems, problem.Problem{Field: "person_name", Code: problem.Required,
			Message: "Enter the name of the person, or choose an existing person."})
	case in.PersonID != "" && in.PersonName != "":
		problems = append(problems, problem.Problem{Field: "person_id", Code: problem.NotAllowed,
			Message: "Choose an existing person or enter a new name, not both."})
	case in.PersonName != "":
		if p, ok := checkPersonName("person_name", in.PersonName); !ok {
			problems = append(problems, p)
		}
		if p, ok := checkEmployeeNumber(in.EmployeeNumber); !ok {
			problems = append(problems, p)
		}
	default:
		exists, err := s.personExists(ctx, in.PersonID)
		if err != nil {
			return nil, err
		}
		if !exists {
			problems = append(problems, problem.Problem{Field: "person_id", Code: problem.InvalidFormat,
				Message: "This person does not exist."})
		}
	}

	if p, ok := checkLogin(in.Login); !ok {
		problems = append(problems, p)
	}
	if in.Kind != KindWork && in.Kind != KindAdmin {
		problems = append(problems, problem.Problem{Field: "kind", Code: problem.InvalidFormat,
			Message: "Choose a work account or an admin account."})
	} else {
		problems = append(problems, access.ValidateRoles(in.Kind, in.Roles)...)
	}
	return problems, nil
}

// accountConflict menerjemahkan pelanggaran batasan UNIQUE saat membuat akun.
func accountConflict(err error, kind string) error {
	taken := func(field, message string) error {
		return &problem.ConflictError{Problems: []problem.Problem{{Field: field, Code: problem.Taken, Message: message}}}
	}
	switch {
	case db.IsUniqueViolation(err, "accounts_login_key"):
		return taken("login", "This login name is already used.")
	case db.IsUniqueViolation(err, "accounts_person_kind_key"):
		// E-02: satu orang paling banyak punya satu akun kerja dan satu akun admin.
		return taken("kind", "This person already has a "+kind+" account.")
	case db.IsUniqueViolation(err, "people_employee_number_key"):
		return taken("employee_number", "This employee number is already used by another person.")
	}
	return nil
}

// ResetPassword memasang password sementara baru untuk akun lain (A-1, E-08).
// Semua sesi akun itu berakhir, kuncinya dibuka, dan pemiliknya wajib
// mengganti password saat login berikutnya.
func (s *Service) ResetPassword(ctx context.Context, actor access.Actor, accountID string) (string, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return "", err
	}
	target, err := s.accountInfo(ctx, accountID)
	if err != nil {
		return "", err
	}
	// ID dari database yang dibandingkan, bukan teks dari klien: UUID dalam
	// huruf besar menunjuk akun yang sama.
	if target.ID == actor.AccountID {
		return "", notAllowedOnOwnAccount("Use the change password page for your own account.")
	}
	accountID = target.ID

	temporary := newTemporaryPassword()
	hash, err := hashPassword(temporary)
	if err != nil {
		return "", err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.ResetAccountPassword(ctx, db.ResetAccountPasswordParams{ID: accountID, PasswordHash: hash}); err != nil {
			return err
		}
		if err := q.DeleteAccountSessions(ctx, accountID); err != nil {
			return err
		}
		// Password sementara tidak pernah masuk audit log (E-18).
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionPasswordReset, TargetType: audit.TargetAccount, TargetID: accountID,
		})
	})
	if err != nil {
		return "", fmt.Errorf("mereset password: %w", err)
	}
	return temporary, nil
}

// DisableAccount menonaktifkan akun lain dan mengakhiri semua sesinya (A-1,
// ASVS 7.4.2). Akun admin aktif terakhir tidak bisa dinonaktifkan (D-73).
// E-26 (D-105): akun tidak pernah dihapus, hanya dinonaktifkan, supaya audit
// log dan riwayat dokumen tetap menunjuk akun dan orangnya (D-74).
func (s *Service) DisableAccount(ctx context.Context, actor access.Actor, accountID string) (AccountInfo, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return AccountInfo{}, err
	}
	target, err := s.accountInfo(ctx, accountID)
	if err != nil {
		return AccountInfo{}, err
	}
	if target.ID == actor.AccountID {
		return AccountInfo{}, notAllowedOnOwnAccount("You cannot disable your own account.")
	}
	accountID = target.ID

	now := s.now()
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Kunci yang sama dengan pembuatan admin pertama: dua admin yang saling
		// menonaktifkan pada saat bersamaan diproses satu per satu, jadi hitungan
		// admin aktif di bawah selalu benar.
		if err := q.LockAdminBootstrap(ctx, adminBootstrapLock); err != nil {
			return err
		}
		target, err := q.GetAccountInfo(ctx, accountID)
		if err != nil {
			return err
		}
		if target.DisabledAt != nil {
			return nil // sudah nonaktif: tidak ada yang berubah, jadi tidak dicatat
		}
		if target.Kind == KindAdmin {
			enabled, err := q.CountEnabledAdmins(ctx)
			if err != nil {
				return err
			}
			if s.afterAdminCount != nil {
				s.afterAdminCount()
			}
			if enabled <= 1 {
				return &problem.ValidationError{Problems: []problem.Problem{{Field: "account_id", Code: problem.NotAllowed,
					Message: "The last active admin account cannot be disabled."}}}
			}
		}
		if err := q.DisableAccount(ctx, db.DisableAccountParams{ID: accountID, DisabledAt: &now}); err != nil {
			return err
		}
		if err := q.DeleteAccountSessions(ctx, accountID); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionAccountDisabled, TargetType: audit.TargetAccount, TargetID: accountID,
		})
	})
	var invalid *problem.ValidationError
	if errors.As(err, &invalid) {
		return AccountInfo{}, err
	}
	if err != nil {
		return AccountInfo{}, fmt.Errorf("menonaktifkan akun: %w", err)
	}
	return s.accountInfo(ctx, accountID)
}

// EnableAccount mengaktifkan lagi akun, termasuk membuka kuncinya (A-1).
func (s *Service) EnableAccount(ctx context.Context, actor access.Actor, accountID string) (AccountInfo, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return AccountInfo{}, err
	}
	target, err := s.accountInfo(ctx, accountID)
	if err != nil {
		return AccountInfo{}, err
	}
	accountID = target.ID
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.EnableAccount(ctx, accountID); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionAccountEnabled, TargetType: audit.TargetAccount, TargetID: accountID,
		})
	})
	if err != nil {
		return AccountInfo{}, fmt.Errorf("mengaktifkan akun: %w", err)
	}
	return s.accountInfo(ctx, accountID)
}

// SetRoles mengganti seluruh peran sebuah akun kerja (A-1). Peran akun admin
// tidak bisa diubah, jadi akun admin terakhir juga tidak bisa kehilangan peran
// Admin (D-73). Peran sebelum dan sesudah dicatat di audit log (R-SEC-10).
func (s *Service) SetRoles(ctx context.Context, actor access.Actor, accountID string, grants []access.RoleGrant) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
		return err
	}
	target, err := s.accountInfo(ctx, accountID)
	if err != nil {
		return err
	}
	if target.Kind == KindAdmin {
		return &problem.ValidationError{Problems: []problem.Problem{{
			Field: "roles", Code: problem.NotAllowed, Message: "The roles of an admin account cannot be changed.",
		}}}
	}
	if problems := access.ValidateRoles(target.Kind, grants); len(problems) > 0 {
		return &problem.ValidationError{Problems: problems}
	}

	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Baris akun dikunci dulu, supaya peran "sebelum" yang dicatat tidak
		// berubah oleh admin lain yang mengubah akun yang sama pada saat itu.
		if err := q.LockAccount(ctx, target.ID); err != nil {
			return err
		}
		before, err := rolesOf(ctx, q, target.ID)
		if err != nil {
			return err
		}
		if err := access.ReplaceRoles(ctx, q, target.ID, grants); err != nil {
			return err
		}
		// Peran sesudah dibaca ulang, supaya urutannya sama dengan katalog.
		after, err := rolesOf(ctx, q, target.ID)
		if err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionRolesChanged, TargetType: audit.TargetAccount, TargetID: target.ID,
			Details: map[string]string{"before": describeGrants(before), "after": describeGrants(after)},
		})
	})
	if err != nil {
		return fmt.Errorf("menyimpan peran: %w", err)
	}
	return nil
}

// rolesOf membaca peran akun, urut seperti katalog peran.
func rolesOf(ctx context.Context, q *db.Queries, accountID string) ([]access.RoleGrant, error) {
	rows, err := q.ListRolesOfAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("membaca peran akun: %w", err)
	}
	grants := make([]access.RoleGrant, 0, len(rows))
	for _, r := range rows {
		grants = append(grants, access.RoleGrant{Role: r.RoleID, Scope: r.Scope})
	}
	return grants, nil
}

// describeGrants menulis peran seperti "author(own_project_group), verifier(all)".
func describeGrants(grants []access.RoleGrant) string {
	parts := make([]string, 0, len(grants))
	for _, g := range grants {
		parts = append(parts, g.Role+"("+g.Scope+")")
	}
	return strings.Join(parts, ", ")
}

// accountInfo membaca satu akun. ID yang tidak ada (atau bukan UUID) memberi
// problem.ErrNotFound.
func (s *Service) accountInfo(ctx context.Context, accountID string) (AccountInfo, error) {
	if !db.IsUUID(accountID) {
		return AccountInfo{}, problem.ErrNotFound
	}
	r, err := s.q.GetAccountInfo(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountInfo{}, problem.ErrNotFound
	}
	if err != nil {
		return AccountInfo{}, fmt.Errorf("membaca akun: %w", err)
	}
	roles, err := rolesOf(ctx, s.q, r.ID)
	if err != nil {
		return AccountInfo{}, err
	}
	return AccountInfo{
		ID: r.ID, Kind: r.Kind, Login: r.Login, PersonID: r.PersonID, PersonName: r.PersonName,
		EmployeeNumber: deref(r.EmployeeNumber), Email: deref(r.Email), MustChangePassword: r.MustChangePassword,
		LockedUntil: r.LockedUntil, Disabled: r.DisabledAt != nil, CreatedAt: r.CreatedAt.UTC(),
		Roles: roles,
	}, nil
}

// personExists melaporkan apakah orang dengan ID itu ada.
func (s *Service) personExists(ctx context.Context, personID string) (bool, error) {
	if !db.IsUUID(personID) {
		return false, nil
	}
	_, err := s.q.GetPerson(ctx, personID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("membaca orang: %w", err)
	}
	return true, nil
}

// checkEmployeeNumber memeriksa nomor karyawan, yang boleh kosong.
// E-02 (D-79): paling banyak 50 karakter, satu baris.
func checkEmployeeNumber(number string) (problem.Problem, bool) {
	switch {
	case utf8.RuneCountInString(number) > 50:
		return problem.Problem{Field: "employee_number", Code: problem.TooLong, Message: "Use at most 50 characters."}, false
	case strings.ContainsFunc(number, unicode.IsControl):
		return problem.Problem{Field: "employee_number", Code: problem.InvalidFormat, Message: "Use one line without control characters."}, false
	}
	return problem.Problem{}, true
}

// notAllowedOnOwnAccount menolak operasi admin pada akunnya sendiri.
func notAllowedOnOwnAccount(message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: "account_id", Code: problem.NotAllowed, Message: message}}}
}

// newTemporaryPassword membuat password sementara: 26 karakter acak (huruf A-Z
// dan angka 2-7, kira-kira 130 bit) dari crypto/rand. NIST SP 800-63B-4
// membolehkan password yang dibuat acak oleh sistem; pemiliknya tetap wajib
// menggantinya saat login pertama.
func newTemporaryPassword() string {
	return rand.Text()
}

// deref mengembalikan isi pointer teks, atau "" bila nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// nonEmpty mengembalikan pointer ke s, atau nil bila s kosong (menjadi NULL).
func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
