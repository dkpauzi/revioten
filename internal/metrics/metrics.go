// Package metrics menghitung ukuran untuk dasbor (M7.7, UC-64): persentase
// baris yang mengikuti template, dokumen per status, status yokoten, waktu
// revisi, dan usaha migrasi. Semua angka dihitung dari data saat diminta;
// tidak ada angka yang diketik atau disimpan (brief bagian 15, butir 5).
package metrics

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Metrics adalah isi dasbor, untuk semua customer atau satu customer.
type Metrics struct {
	Rows      Rows
	Documents []DocumentCount // tidak pernah nil
	Yokoten   []YokotenCount  // tidak pernah nil
	Revision  RevisionTime
	Migration Migration
}

// Rows menghitung baris dokumen menurut tautannya ke poin template.
type Rows struct {
	Total    int
	Linked   int // tertaut dan belum diputus: mengikuti template
	Detached int // tautan diputus dengan alasan
	Free     int // diketik atau ditempel sendiri, tanpa tautan
	// LinkedPercent adalah Linked dibagi Total, 0 sampai 100; 0 bila Total 0.
	LinkedPercent float64
}

// DocumentCount adalah jumlah dokumen satu jenis dengan satu status: draft,
// released, atau released_migrated.
type DocumentCount struct {
	TypeKey  string
	TypeName string
	Status   string
	Count    int
}

// YokotenCount adalah jumlah target yokoten dengan satu keputusan: pending,
// apply, adapt, atau not_applicable.
type YokotenCount struct {
	Decision string
	Count    int
}

// RevisionTime mengukur waktu revisi: dari draft dibuka perubahan yang
// diterima sampai rilisnya dicatat. Revisi awal dan baseline migrasi tidak
// dihitung.
type RevisionTime struct {
	Count        int
	AverageHours float64
	MedianHours  float64
}

// Migration menghitung dokumen migrasi dan usahanya (M7.6).
type Migration struct {
	Documents    int // semua dokumen migrasi
	Verified     int // sudah dicocokkan dengan hardcopy
	Measured     int // yang menit usahanya diisi
	TotalMinutes int
	// AverageMinutes adalah TotalMinutes dibagi Measured; 0 bila Measured 0.
	AverageMinutes float64
}

// Service menghitung ukuran.
type Service struct {
	q     *db.Queries
	authz access.Authorizer
}

// NewService membuat Service yang memakai pool database.
func NewService(pool *pgxpool.Pool, authz access.Authorizer) *Service {
	return &Service{q: db.New(pool), authz: authz}
}

// Get menghitung ukuran (L-2). customerID kosong berarti semua customer.
func (s *Service) Get(ctx context.Context, actor access.Actor, customerID string) (Metrics, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return Metrics{}, err
	}
	var filter *string
	if customerID != "" {
		if !db.IsUUID(customerID) {
			return Metrics{}, &problem.ValidationError{Problems: []problem.Problem{{Field: "customer_id", Code: problem.InvalidFormat, Message: "Choose an existing customer."}}}
		}
		filter = &customerID
	}
	out := Metrics{Documents: []DocumentCount{}, Yokoten: []YokotenCount{}}

	rows, err := s.q.MetricRows(ctx, filter)
	if err != nil {
		return Metrics{}, fmt.Errorf("menghitung baris: %w", err)
	}
	out.Rows = Rows{Total: int(rows.Total), Linked: int(rows.Linked), Detached: int(rows.Detached), Free: int(rows.Total - rows.Linked - rows.Detached)}
	if rows.Total > 0 {
		out.Rows.LinkedPercent = 100 * float64(rows.Linked) / float64(rows.Total)
	}

	docs, err := s.q.MetricDocuments(ctx, filter)
	if err != nil {
		return Metrics{}, fmt.Errorf("menghitung dokumen: %w", err)
	}
	for _, d := range docs {
		key := ""
		if d.DocumentTypeKey != nil {
			key = *d.DocumentTypeKey
		}
		out.Documents = append(out.Documents, DocumentCount{TypeKey: key, TypeName: d.DocumentTypeName, Status: d.Status, Count: int(d.Count)})
	}

	yokoten, err := s.q.MetricYokoten(ctx, filter)
	if err != nil {
		return Metrics{}, fmt.Errorf("menghitung yokoten: %w", err)
	}
	for _, y := range yokoten {
		out.Yokoten = append(out.Yokoten, YokotenCount{Decision: y.Decision, Count: int(y.Count)})
	}

	rev, err := s.q.MetricRevisionTime(ctx, filter)
	if err != nil {
		return Metrics{}, fmt.Errorf("menghitung waktu revisi: %w", err)
	}
	out.Revision = RevisionTime{Count: int(rev.Revisions), AverageHours: rev.AverageHours, MedianHours: rev.MedianHours}

	mig, err := s.q.MetricMigration(ctx, filter)
	if err != nil {
		return Metrics{}, fmt.Errorf("menghitung migrasi: %w", err)
	}
	out.Migration = Migration{Documents: int(mig.Documents), Verified: int(mig.Verified), Measured: int(mig.Measured), TotalMinutes: int(mig.TotalMinutes)}
	if mig.Measured > 0 {
		out.Migration.AverageMinutes = float64(mig.TotalMinutes) / float64(mig.Measured)
	}
	return out, nil
}
