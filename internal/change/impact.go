package change

// Daftar periksa dampak (M5.7, R-CHG-05). Untuk setiap perubahan dicatat
// dokumen terkait mana yang terdampak (PFD, PFMEA, CP, WI, standar inspeksi),
// keputusannya, dan alasannya. Tidak setiap perubahan PFMEA butuh revisi CP,
// dan daftar ini mencatat keputusan itu, terdampak maupun tidak.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Jenis dokumen terkait di daftar periksa.
var impactKinds = []string{"pfd", "pfmea", "cp", "wi", "inspection_standard", "other"}

// Keputusan per dokumen terkait.
const (
	ImpactAffected    = "affected"
	ImpactNotAffected = "not_affected"
)

const (
	maxImpacts         = 50
	maxImpactReference = 200
	maxImpactReason    = 500
)

// Impact adalah satu baris daftar periksa dampak.
type Impact struct {
	ID                string
	Kind              string
	DocumentReference string // nomor atau nama dokumen; "" bila tidak diisi
	Decision          string
	Reason            string
	UpdatedBy         string
	UpdatedAt         time.Time
}

// ImpactInput adalah isian satu baris daftar periksa.
type ImpactInput struct {
	Kind              string
	DocumentReference string
	Decision          string
	Reason            string
}

// SaveImpacts menyimpan daftar periksa dampak satu perubahan secara utuh (E-3).
// Daftar bisa diisi selama perubahan diajukan atau sudah diterima; perubahan
// yang ditolak tidak diubah lagi (ASUMSI E-104).
func (s *RequestService) SaveImpacts(ctx context.Context, actor access.Actor, id string, items []ImpactInput) (Change, error) {
	clean, err := checkImpacts(items)
	if err != nil {
		return Change{}, err
	}
	return s.update(ctx, actor, id, []string{StatusSubmitted, StatusAccepted}, func(q *db.Queries, c Change) (string, map[string]string, error) {
		if err := q.ClearChangeImpacts(ctx, c.ID); err != nil {
			return "", nil, err
		}
		affected := 0
		for i, x := range clean {
			if err := q.AddChangeImpact(ctx, db.AddChangeImpactParams{
				ChangeID: c.ID, Position: int32(i + 1), Kind: x.Kind, DocumentReference: optional(x.DocumentReference),
				Decision: x.Decision, Reason: optional(x.Reason), UpdatedByPersonID: actor.PersonID,
			}); err != nil {
				return "", nil, err
			}
			if x.Decision == ImpactAffected {
				affected++
			}
		}
		return audit.ActionChangeImpactsSaved, map[string]string{"items": strconv.Itoa(len(clean)), "affected": strconv.Itoa(affected)}, nil
	})
}

// checkImpacts memeriksa dan merapikan isian daftar periksa.
func checkImpacts(items []ImpactInput) ([]ImpactInput, error) {
	if len(items) > maxImpacts {
		return nil, problems(invalid("items", problem.TooLong, fmt.Sprintf("A checklist can have at most %d lines.", maxImpacts)))
	}
	out := make([]ImpactInput, 0, len(items))
	for i, x := range items {
		field := fmt.Sprintf("items.%d", i)
		x.Kind, x.Decision = strings.TrimSpace(x.Kind), strings.TrimSpace(x.Decision)
		x.DocumentReference, x.Reason = strings.TrimSpace(x.DocumentReference), strings.TrimSpace(x.Reason)
		switch {
		case !contains(impactKinds, x.Kind):
			return nil, problems(invalid(field+".kind", problem.InvalidFormat, "Choose the kind of document."))
		case x.Decision != ImpactAffected && x.Decision != ImpactNotAffected:
			return nil, problems(invalid(field+".decision", problem.Required, "Choose affected or not affected."))
		case x.Decision == ImpactNotAffected && x.Reason == "":
			return nil, problems(invalid(field+".reason", problem.Required, "Write why the document is not affected."))
		case utf8.RuneCountInString(x.DocumentReference) > maxImpactReference:
			return nil, problems(invalid(field+".document_reference", problem.TooLong, "A document number can be at most 200 characters."))
		case utf8.RuneCountInString(x.Reason) > maxImpactReason:
			return nil, problems(invalid(field+".reason", problem.TooLong, "A reason can be at most 500 characters."))
		}
		out = append(out, x)
	}
	return out, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// impacts membaca daftar periksa dampak satu perubahan.
func impacts(ctx context.Context, q *db.Queries, id string) ([]Impact, error) {
	rows, err := q.ListChangeImpacts(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Impact, 0, len(rows))
	for _, r := range rows {
		out = append(out, Impact{
			ID: r.ID, Kind: r.Kind, DocumentReference: deref(r.DocumentReference), Decision: r.Decision, Reason: deref(r.Reason),
			UpdatedBy: r.UpdatedBy, UpdatedAt: r.UpdatedAt.UTC(),
		})
	}
	return out, nil
}
