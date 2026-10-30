package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
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
	return VerifyChain(ctx, s.q)
}

// VerifyChain memeriksa seluruh rantai di database q, tanpa pemeriksaan izin.
// Hanya untuk perintah di komputer server, misalnya restore (M1.7), yang
// berjalan tanpa akun, sama seperti create-admin (E-07). Layar dan API selalu
// lewat Service.Verify.
func VerifyChain(ctx context.Context, q *db.Queries) (Verification, error) {
	rows, err := q.ListAuditChain(ctx)
	if err != nil {
		return Verification{}, fmt.Errorf("membaca rantai audit: %w", err)
	}
	chain := make([]entryFields, 0, len(rows))
	for _, r := range rows {
		chain = append(chain, fieldsOf(r))
	}
	return verifyChain(chain), nil
}

// ChainHead mengembalikan ujung rantai: nomor urut dan hash entri terakhir.
// Untuk rantai yang masih kosong hasilnya 0 dan "". Dipakai backup (M1.7)
// untuk mencatat ujung rantai di luar database (R-SEC-03, D-119).
func ChainHead(ctx context.Context, q *db.Queries) (int64, string, error) {
	head, err := q.GetAuditChainHead(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("membaca ujung rantai audit: %w", err)
	}
	return head.Seq, head.Hash, nil
}

// HashAt mengembalikan hash entri bernomor seq, atau "" bila entri itu tidak ada.
func HashAt(ctx context.Context, q *db.Queries, seq int64) (string, error) {
	hash, err := q.GetAuditHash(ctx, seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("membaca entri audit %d: %w", seq, err)
	}
	return hash, nil
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
