// Package inbox menyusun daftar "Waiting for you" di Home (D-202): pekerjaan
// yang menunggu orang yang login, menurut haknya. Daftar ini hanya bacaan;
// setiap tugas menautkan ke halaman tempat pekerjaannya dilakukan, dan
// halaman itu tetap memeriksa hak sendiri.
package inbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/review"
)

// Jenis tugas.
const (
	KindVerify        = "verify"         // permintaan perubahan menunggu verifikasi (V-1)
	KindRelease       = "release"        // scan bertanda tangan menunggu rilis (E-7)
	KindReview        = "review"         // set dokumen jatuh tempo review berkala (E-3)
	KindConfirmReview = "confirm_review" // catatan review menunggu konfirmasi (V-1)
	KindYokoten       = "yokoten"        // target yokoten belum diputuskan (E-4)
	KindSign          = "sign"           // dokumen menunggu tanda tangan digital orang ini (S-1, D-198)
)

// Task adalah satu pekerjaan yang menunggu.
type Task struct {
	Kind   string
	Title  string // teks dari data (nomor, model, alasan); label jenisnya diterjemahkan layar
	Detail string
	Link   string
	// Days adalah hari lewat jatuh tempo (positif) untuk review; 0 untuk lainnya.
	Days int
	At   time.Time
}

// maxPerKind membatasi jumlah tugas sejenis, supaya Home tetap pendek.
const maxPerKind = 5

// Service menyusun daftar tugas.
type Service struct {
	q     *db.Queries
	authz access.Authorizer
	now   func() time.Time
}

// NewService membuat Service.
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{q: db.New(pool), authz: authz, now: time.Now}
}

// Tasks memberi pekerjaan yang menunggu actor. Tugas yang tidak boleh
// dikerjakan actor tidak ikut. Total adalah jumlah sebelum dibatasi.
func (s *Service) Tasks(ctx context.Context, actor access.Actor) ([]Task, int, error) {
	allowed := map[string]bool{}
	can := func(perm access.Permission, customerID string) bool {
		key := string(perm) + "/" + customerID
		ok, seen := allowed[key]
		if !seen {
			ok = s.authz.Authorize(ctx, actor, perm, access.Resource{CustomerID: customerID}) == nil
			allowed[key] = ok
		}
		return ok
	}
	var out []Task
	total := 0
	add := func(t Task, count *int) {
		total++
		if *count < maxPerKind {
			out = append(out, t)
			*count++
		}
	}

	// Tanda tangan digital yang menunggu orang ini: paling atas, karena
	// orang berikutnya di rute menunggunya (D-49).
	signs, err := s.q.InboxSignaturesWaiting(ctx, actor.PersonID)
	if err != nil {
		return nil, 0, fmt.Errorf("membaca tanda tangan: %w", err)
	}
	signCount := 0
	for _, r := range signs {
		add(Task{Kind: KindSign, Title: fmt.Sprintf("%s rev %02d", r.Number, r.Revision),
			Detail: fmt.Sprintf("%s · %s", strings.ToUpper(r.CustomerCode), r.PrintTitle), Link: "/signatures/sign?id=" + r.ID, At: r.SentAt}, &signCount)
	}

	// Review berkala yang jatuh tempo: yang paling mendesak dulu.
	sets, err := review.Compute(ctx, s.q, review.Filter{}, s.now())
	if err != nil {
		return nil, 0, err
	}
	n := 0
	for _, set := range sets.Sets {
		if !can(access.ProposeChanges, set.CustomerID) || set.OpenReviewID != "" {
			continue
		}
		models := strings.Join(set.Models, ", ")
		add(Task{Kind: KindReview, Title: fmt.Sprintf("%s · %s", strings.ToUpper(set.CustomerCode), models),
			Detail: set.Documents[0].Number, Link: "/periodic-reviews", Days: -set.DaysLeft, At: set.DueOn}, &n)
	}

	changes, err := s.q.InboxChangesToVerify(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("membaca perubahan: %w", err)
	}
	n = 0
	for _, c := range changes {
		if c.RequestedByPersonID != nil && *c.RequestedByPersonID == actor.PersonID {
			continue // pemohon tidak memverifikasi sendiri (R-SEC-11)
		}
		if !can(access.VerifyChanges, c.CustomerID) {
			continue
		}
		ref := "CHG"
		if c.Seq != nil {
			ref = fmt.Sprintf("CHG-%04d", *c.Seq)
		}
		add(Task{Kind: KindVerify, Title: fmt.Sprintf("%s · %s", ref, c.Reason), Detail: fmt.Sprintf("%s · %s", strings.ToUpper(c.CustomerCode), c.RequestedBy),
			Link: "/changes/view?id=" + c.ID, At: c.CreatedAt}, &n)
	}

	reviews, err := s.q.ListPeriodicReviews(ctx, db.ListPeriodicReviewsParams{OpenOnly: true})
	if err != nil {
		return nil, 0, fmt.Errorf("membaca catatan review: %w", err)
	}
	n = 0
	for _, r := range reviews {
		if r.RecordedByPersonID == actor.PersonID || !can(access.VerifyChanges, r.CustomerID) {
			continue
		}
		add(Task{Kind: KindConfirmReview, Title: fmt.Sprintf("%s · %04d", strings.ToUpper(r.CustomerCode), r.Sequence),
			Detail: r.RecordedBy, Link: "/periodic-reviews", At: r.RecordedAt}, &n)
	}

	scans, err := s.q.InboxScansReady(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("membaca scan: %w", err)
	}
	n = 0
	for _, r := range scans {
		if !can(access.RecordRelease, r.CustomerID) {
			continue
		}
		at := time.Time{}
		if r.ScanUploadedAt != nil {
			at = *r.ScanUploadedAt
		}
		add(Task{Kind: KindRelease, Title: fmt.Sprintf("%s rev %02d", r.Number, r.Revision), Link: "/documents/view?id=" + r.DocumentID, At: at}, &n)
	}
	// Draft yang lengkap ditandatangani digital juga menunggu rilis (D-198).
	signed, err := s.q.InboxSignaturesCompleted(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("membaca tanda tangan lengkap: %w", err)
	}
	for _, r := range signed {
		if !can(access.RecordRelease, r.CustomerID) {
			continue
		}
		at := time.Time{}
		if r.ClosedAt != nil {
			at = *r.ClosedAt
		}
		add(Task{Kind: KindRelease, Title: fmt.Sprintf("%s rev %02d", r.Number, r.Revision), Link: "/documents/view?id=" + r.DocumentID, At: at}, &n)
	}

	yokotens, err := s.q.InboxYokotenPending(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("membaca yokoten: %w", err)
	}
	n = 0
	for _, y := range yokotens {
		if !can(access.WorkRevisions, y.CustomerID) {
			continue
		}
		add(Task{Kind: KindYokoten, Title: y.Reason, Detail: fmt.Sprintf("%d/%d", y.Decided, y.Targets), Link: "/yokoten?id=" + y.ID}, &n)
	}
	if out == nil {
		out = []Task{}
	}
	return out, total, nil
}
