package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Entry adalah satu entri audit untuk dibaca admin.
type Entry struct {
	Seq        int64
	OccurredAt time.Time
	ActorType  string
	// Akun dan orang pemegangnya saat itu (D-74). Kosong untuk aktor sistem dan anonim.
	ActorAccountID  string
	ActorLogin      string
	ActorPersonID   string
	ActorPersonName string
	RequestID       string
	SourceIP        string
	Action          string
	TargetType      string
	TargetID        string
	// TargetLabel adalah nama tampilan sasaran, misalnya login akun atau kode customer.
	TargetLabel string
	Details     map[string]string
	PrevHash    string
	Hash        string
}

// Service membaca dan memeriksa audit log. Keduanya butuh hak A-4, yang hanya
// dimiliki akun admin (D-72, D-75).
type Service struct {
	q     *db.Queries
	authz access.Authorizer
}

// NewService membuat Service yang memakai pool database.
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{q: db.New(pool), authz: authz}
}

// maxListEntries adalah jumlah entri paling banyak dalam satu halaman.
const maxListEntries = 200

// List mengembalikan paling banyak limit entri (1 sampai 200) yang nomornya
// lebih kecil dari before, terbaru dulu. before 0 berarti mulai dari entri
// terbaru. Halaman berikutnya diminta dengan before = nomor entri terakhir di
// halaman ini.
func (s *Service) List(ctx context.Context, actor access.Actor, before int64, limit int) ([]Entry, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewAuditLog, access.Resource{}); err != nil {
		return nil, err
	}
	var problems []problem.Problem
	if before < 0 {
		problems = append(problems, problem.Problem{Field: "before", Code: problem.InvalidFormat, Message: "Use a positive entry number."})
	}
	if limit < 1 || limit > maxListEntries {
		problems = append(problems, problem.Problem{Field: "limit", Code: problem.InvalidFormat, Message: "Use 1 to 200."})
	}
	if len(problems) > 0 {
		return nil, &problem.ValidationError{Problems: problems}
	}
	if before == 0 {
		before = math.MaxInt64
	}
	rows, err := s.q.ListAuditEntries(ctx, db.ListAuditEntriesParams{Before: before, MaxEntries: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("membaca audit log: %w", err)
	}
	entries := make([]Entry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, Entry{
			Seq: r.Seq, OccurredAt: r.OccurredAt.UTC(), ActorType: r.ActorType,
			ActorAccountID: text(r.ActorAccountID), ActorLogin: text(r.ActorLogin),
			ActorPersonID: text(r.ActorPersonID), ActorPersonName: text(r.ActorPersonName),
			RequestID: text(r.RequestID), SourceIP: text(r.SourceIp),
			Action: r.Action, TargetType: text(r.TargetType), TargetID: text(r.TargetID), TargetLabel: r.TargetLabel,
			Details: detailsOf(r.Details), PrevHash: r.PrevHash, Hash: r.Hash,
		})
	}
	return entries, nil
}

// Verify memeriksa seluruh rantai dari entri pertama.
func (s *Service) Verify(ctx context.Context, actor access.Actor) (Verification, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewAuditLog, access.Resource{}); err != nil {
		return Verification{}, err
	}
	rows, err := s.q.ListAuditChain(ctx)
	if err != nil {
		return Verification{}, fmt.Errorf("membaca rantai audit: %w", err)
	}
	chain := make([]entryFields, 0, len(rows))
	for _, r := range rows {
		chain = append(chain, fieldsOf(r))
	}
	return verifyChain(chain), nil
}

// detailsOf membaca rincian JSON. Rincian yang bukan JSON (hanya mungkin bila
// tabel diubah di luar aplikasi) tetap ditampilkan apa adanya dengan kunci raw.
func detailsOf(text string) map[string]string {
	details := map[string]string{}
	if err := json.Unmarshal([]byte(text), &details); err != nil {
		return map[string]string{"raw": text}
	}
	return details
}
