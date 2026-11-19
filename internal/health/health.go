// Package health menyusun dasbor admin (fase 4, D-205): kesehatan web,
// bukan isi dokumen. Akun admin mengurus web, jadi dasbor ini menjawab
// "apakah web berjalan sehat, dan di mana pekerjaan tertahan?":
// tanda tangan digital yang tertahan, error server, laporan masalah dari
// pengguna, jumlah dokumen yang diproses per kelompok pemicu, email, akun,
// backup, dan rantai audit log.
//
// Semua operasi butuh hak A-3 (kelola pengaturan sistem), kecuali mengirim
// laporan masalah, yang boleh dilakukan setiap akun yang login.
package health

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/backup"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/metrics"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Kelompok dokumen yang diproses: kelompok laporan kinerja (D-197), ditambah
// revisi awal dan dokumen lama.
const (
	CategoryInitial  = "initial"
	CategoryMigrated = "migrated"
)

// Categories adalah urutan kelompok di layar.
var Categories = append(slices.Clone(metrics.Categories), CategoryInitial, CategoryMigrated)

// Status laporan masalah.
const (
	ProblemNew      = "new"
	ProblemChecking = "checking"
	ProblemDone     = "done"
)

// topN adalah jumlah baris setiap daftar di dasbor; detailnya di halaman sendiri.
const topN = 5

// Service menyusun dasbor admin.
type Service struct {
	pool      *pgxpool.Pool
	q         *db.Queries
	authz     access.Authorizer
	backupDir string
	now       func() time.Time
}

// NewService membuat Service. backupDir adalah BACKUP_DIR; kosong berarti
// backup belum diatur.
func NewService(pool *pgxpool.Pool, authz access.Authorizer, backupDir string) *Service {
	return &Service{pool: pool, q: db.New(pool), authz: authz, backupDir: backupDir, now: time.Now}
}

// WaitingSignature adalah satu tanda tangan digital yang tertahan.
type WaitingSignature struct {
	RequestID    string
	DocumentID   string
	Number       string
	Revision     int
	CustomerCode string
	Slot         string
	Signer       string
	SentBy       string
	SentAt       time.Time
	WaitingSince time.Time
	Hours        float64 // lama menunggu sampai sekarang
}

// ErrorReport adalah satu error server.
type ErrorReport struct {
	ID         string
	OccurredAt time.Time
	RequestID  string
	Method     string
	Path       string
	Message    string
	Login      string
}

// ProblemReport adalah satu laporan masalah dari pengguna.
type ProblemReport struct {
	ID            string
	CreatedAt     time.Time
	ReportedBy    string
	Login         string
	Page          string
	Description   string
	LastRequestID string
	Status        string
	HandledBy     string
	HandledAt     *time.Time
	AdminNote     string
}

// Processed adalah jumlah revisi terbit dalam satu periode per kelompok.
type Processed struct {
	From, To time.Time
	Total    int
	Counts   map[string]int
}

// Backup adalah backup terakhir yang ditemukan di BACKUP_DIR.
type Backup struct {
	Configured bool
	Found      bool
	CreatedAt  time.Time
	File       string
	Size       int64
}

// Overview adalah isi dasbor admin.
type Overview struct {
	SignaturesWaiting int
	TopSignatures     []WaitingSignature
	Errors24h         int
	Errors7d          int
	LatestErrors      []ErrorReport
	ProblemsOpen      int
	LatestProblems    []ProblemReport
	Week              Processed
	Month             Processed
	EmailFailed       int
	EmailPending      int
	AccountsLocked    int
	AccountsDisabled  int
	FailedLogins24h   int
	Backup            Backup
	AuditValid        bool
	AuditEntries      int64
}

func (s *Service) authorize(ctx context.Context, actor access.Actor) error {
	return s.authz.Authorize(ctx, actor, access.ManageSettings, access.Resource{})
}

// Overview menyusun dasbor (A-3).
func (s *Service) Overview(ctx context.Context, actor access.Actor) (Overview, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return Overview{}, err
	}
	now := s.now()
	var o Overview
	waiting, err := s.waiting(ctx, now)
	if err != nil {
		return Overview{}, err
	}
	o.SignaturesWaiting = len(waiting)
	o.TopSignatures = waiting[:min(topN, len(waiting))]

	for _, x := range []struct {
		since time.Duration
		out   *int
	}{{24 * time.Hour, &o.Errors24h}, {7 * 24 * time.Hour, &o.Errors7d}} {
		n, err := s.q.CountErrorReportsSince(ctx, now.Add(-x.since))
		if err != nil {
			return Overview{}, fmt.Errorf("menghitung error: %w", err)
		}
		*x.out = int(n)
	}
	if o.LatestErrors, err = s.errors(ctx, topN); err != nil {
		return Overview{}, err
	}
	open, err := s.q.CountOpenProblemReports(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("menghitung laporan masalah: %w", err)
	}
	o.ProblemsOpen = int(open)
	if o.LatestProblems, err = s.problems(ctx, topN); err != nil {
		return Overview{}, err
	}
	for _, x := range []struct {
		kind string
		out  *Processed
	}{{metrics.PeriodWeek, &o.Week}, {metrics.PeriodMonth, &o.Month}} {
		period, err := metrics.PeriodOf(x.kind, now)
		if err != nil {
			return Overview{}, err
		}
		if *x.out, err = s.processed(ctx, period.From, period.To); err != nil {
			return Overview{}, err
		}
	}
	outbox, err := s.q.OutboxHealth(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("membaca kotak keluar: %w", err)
	}
	o.EmailFailed, o.EmailPending = int(outbox.Failed), int(outbox.Pending)
	accounts, err := s.q.AccountHealth(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("membaca akun: %w", err)
	}
	o.AccountsLocked, o.AccountsDisabled, o.FailedLogins24h = int(accounts.Locked), int(accounts.Disabled), int(accounts.FailedLogins)
	o.Backup = latestBackup(s.backupDir)
	chain, err := audit.VerifyChain(ctx, s.q)
	if err != nil {
		return Overview{}, err
	}
	o.AuditValid, o.AuditEntries = chain.Valid, chain.Entries
	return o, nil
}

// waiting membaca semua tanda tangan yang tertahan, yang paling lama dulu.
func (s *Service) waiting(ctx context.Context, now time.Time) ([]WaitingSignature, error) {
	// Permintaan tanda tangan menyangkut draft, yang disaring RLS untuk akun
	// tanpa L-4, termasuk akun admin. Dasbor ini sudah lolos A-3 dan hanya
	// memuat nomor, slot, dan nama, bukan isi dokumen, jadi dibaca tanpa
	// saringan (AsSystem).
	rows, err := s.q.WaitingSignatureSlots(db.AsSystem(ctx))
	if err != nil {
		return nil, fmt.Errorf("membaca tanda tangan tertahan: %w", err)
	}
	out := make([]WaitingSignature, 0, len(rows))
	for _, r := range rows {
		out = append(out, WaitingSignature{
			RequestID: r.ID, DocumentID: r.DocumentID, Number: r.Number, Revision: int(r.Revision), CustomerCode: r.CustomerCode,
			Slot: r.PrintTitle, Signer: r.PersonName, SentBy: r.SentBy, SentAt: r.SentAt.UTC(), WaitingSince: r.WaitingSince.UTC(),
			Hours: now.Sub(r.WaitingSince).Hours(),
		})
	}
	return out, nil
}

// WaitingSignatures memberi semua tanda tangan yang tertahan (A-3).
func (s *Service) WaitingSignatures(ctx context.Context, actor access.Actor) ([]WaitingSignature, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return nil, err
	}
	return s.waiting(ctx, s.now())
}

func (s *Service) errors(ctx context.Context, limit int) ([]ErrorReport, error) {
	rows, err := s.q.ListErrorReports(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("membaca error: %w", err)
	}
	out := make([]ErrorReport, 0, len(rows))
	for _, r := range rows {
		out = append(out, ErrorReport{ID: r.ID, OccurredAt: r.OccurredAt.UTC(), RequestID: r.RequestID, Method: r.Method, Path: r.Path, Message: r.Message, Login: deref(r.Login)})
	}
	return out, nil
}

// ErrorReports memberi 200 error server terakhir (A-3).
func (s *Service) ErrorReports(ctx context.Context, actor access.Actor) ([]ErrorReport, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return nil, err
	}
	return s.errors(ctx, 200)
}

func (s *Service) problems(ctx context.Context, limit int) ([]ProblemReport, error) {
	rows, err := s.q.ListProblemReports(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("membaca laporan masalah: %w", err)
	}
	out := make([]ProblemReport, 0, len(rows))
	for _, r := range rows {
		p := ProblemReport{
			ID: r.ID, CreatedAt: r.CreatedAt.UTC(), ReportedBy: r.ReportedBy, Login: r.Login, Page: r.Page, Description: r.Description,
			LastRequestID: deref(r.LastRequestID), Status: r.Status, HandledBy: deref(r.HandledBy), AdminNote: deref(r.AdminNote),
		}
		if r.HandledAt != nil {
			at := r.HandledAt.UTC()
			p.HandledAt = &at
		}
		out = append(out, p)
	}
	return out, nil
}

// ProblemReports memberi 200 laporan masalah terakhir, yang belum selesai dulu (A-3).
func (s *Service) ProblemReports(ctx context.Context, actor access.Actor) ([]ProblemReport, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return nil, err
	}
	return s.problems(ctx, 200)
}

// ReportProblem mencatat laporan masalah dari pengguna yang login. Setiap
// akun boleh melapor (seperti mengurus akunnya sendiri, D-104).
func (s *Service) ReportProblem(ctx context.Context, actor access.Actor, page, description, lastRequestID string) (string, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageOwnAccount, access.Resource{AccountID: actor.AccountID}); err != nil {
		return "", err
	}
	description = strings.TrimSpace(description)
	page = strings.TrimSpace(page)
	lastRequestID = strings.TrimSpace(lastRequestID)
	switch {
	case description == "":
		return "", invalid("description", problem.Required, "Describe what went wrong.")
	case utf8.RuneCountInString(description) > 2000:
		return "", invalid("description", problem.TooLong, "The description can be at most 2000 characters.")
	case utf8.RuneCountInString(page) > 500:
		page = string([]rune(page)[:500])
	}
	if utf8.RuneCountInString(lastRequestID) > 100 {
		lastRequestID = ""
	}
	id, err := s.q.AddProblemReport(ctx, db.AddProblemReportParams{
		PersonID: actor.PersonID, AccountID: actor.AccountID, Page: page, Description: description, LastRequestID: optional(lastRequestID),
	})
	if err != nil {
		return "", fmt.Errorf("mencatat laporan masalah: %w", err)
	}
	return id, nil
}

// UpdateProblem mengubah status dan catatan satu laporan masalah (A-3).
func (s *Service) UpdateProblem(ctx context.Context, actor access.Actor, id, status, note string) error {
	if err := s.authorize(ctx, actor); err != nil {
		return err
	}
	if status != ProblemNew && status != ProblemChecking && status != ProblemDone {
		return invalid("status", problem.InvalidFormat, "Choose new, checking, or done.")
	}
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > 2000 {
		return invalid("note", problem.TooLong, "The note can be at most 2000 characters.")
	}
	if !db.IsUUID(id) {
		return problem.ErrNotFound
	}
	n, err := s.q.UpdateProblemReport(ctx, db.UpdateProblemReportParams{ID: id, Status: status, AdminNote: optional(note), HandledByPersonID: &actor.PersonID})
	if err != nil {
		return fmt.Errorf("mengubah laporan masalah: %w", err)
	}
	if n == 0 {
		return problem.ErrNotFound
	}
	return nil
}

// processed menghitung revisi terbit dalam rentang per kelompok.
func (s *Service) processed(ctx context.Context, from, to time.Time) (Processed, error) {
	// Revisi terbit tidak disaring RLS (L-1); AsSystem tetap dipakai supaya
	// hitungan tidak bergantung pada hak akun admin.
	rows, err := s.q.ProcessedRevisions(db.AsSystem(ctx), db.ProcessedRevisionsParams{FromAt: &from, ToAt: &to})
	if err != nil {
		return Processed{}, fmt.Errorf("menghitung dokumen yang diproses: %w", err)
	}
	p := Processed{From: from, To: to, Counts: map[string]int{}}
	for _, c := range Categories {
		p.Counts[c] = 0
	}
	for _, r := range rows {
		key := CategoryInitial
		switch {
		case r.Migrated:
			key = CategoryMigrated
		case r.ByChange:
			key = metrics.Category(r.ReportKey, r.Mechanism)
		}
		p.Counts[key]++
		p.Total++
	}
	return p, nil
}

// latestBackup membaca manifest backup terbaru di dir. Nama manifest memuat
// waktu, jadi urutan nama sama dengan urutan waktu.
func latestBackup(dir string) Backup {
	if dir == "" {
		return Backup{}
	}
	b := Backup{Configured: true}
	files, err := filepath.Glob(filepath.Join(dir, "revioten-*.manifest.json"))
	if err != nil || len(files) == 0 {
		return b
	}
	slices.Sort(files)
	// ReadManifest menerima nama file backup, lalu membaca manifest di sampingnya.
	m, err := backup.ReadManifest(strings.TrimSuffix(files[len(files)-1], ".manifest.json") + ".dump.age")
	if err != nil {
		return b
	}
	b.Found, b.CreatedAt, b.File, b.Size = true, m.CreatedAt.UTC(), m.File, m.Size
	return b
}

// RecordError mencatat satu error server (HTTP 500) tanpa pemeriksaan izin:
// dipanggil server, bukan pengguna. Pesan dipotong 2000 karakter. Kegagalan
// mencatat hanya dikembalikan untuk ditulis ke log.
func (s *Service) RecordError(ctx context.Context, requestID, method, path, accountID, message string) error {
	if utf8.RuneCountInString(message) > 2000 {
		message = string([]rune(message)[:2000])
	}
	if utf8.RuneCountInString(path) > 500 {
		path = string([]rune(path)[:500])
	}
	var account *string
	if db.IsUUID(accountID) {
		account = &accountID
	}
	err := s.q.AddErrorReport(ctx, db.AddErrorReportParams{RequestID: requestID, Method: method, Path: path, AccountID: account, Message: message})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("mencatat error server: %w", err)
	}
	return nil
}

func invalid(field, code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: field, Code: code, Message: message}}}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
