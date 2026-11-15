package integration

// Usulan perubahan dari luar (M9.4) dan ekspor asinkron (M9.5).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Proposal adalah satu usulan perubahan terhadap satu dokumen terbit.
type Proposal struct {
	DocumentID        string
	BaseRevision      int    // revisi terbit yang dibaca aplikasi pemanggil (R-API-11)
	SourceApplication string // nama aplikasi pemanggil
	OnBehalfOf        string // orang yang disebut pemanggil; hanya dicatat (R-API-09)
	ChangeType        string // content, layout, atau administrative; kosong berarti content
	Reason            string
	Reference         string
	Operations        []document.PatchOp
}

// ProposedChange adalah hasil satu usulan: perubahan yang menunggu verifikasi orang.
type ProposedChange struct {
	ID               string
	Ref              string // misalnya CHG-0007
	Status           string // submitted, accepted, atau rejected
	CreatedAt        time.Time
	VerifiedAt       *time.Time
	VerificationNote string
}

// changeRef memberi nomor rujukan perubahan, sama bentuknya dengan internal/change.
func changeRef(seq *int64) string {
	if seq == nil {
		return ""
	}
	return fmt.Sprintf("CHG-%04d", *seq)
}

// Propose membuat perubahan berisi operasi patch terhadap revisi dasar (scope
// changes:propose). Perubahan itu menunggu verifikasi orang seperti perubahan
// lain; isinya baru masuk ke draft saat diterima. key adalah Idempotency-Key:
// permintaan yang diulang dengan kunci dan isi yang sama mengembalikan hasil
// pertama (replayed true), dan tidak membuat perubahan kedua (R-API-10).
func (s *Service) Propose(ctx context.Context, c Client, key string, in Proposal) (ProposedChange, bool, error) {
	if err := c.require(ScopeChangesPropose); err != nil {
		return ProposedChange{}, false, err
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 200 {
		return ProposedChange{}, false, invalid("Idempotency-Key", problem.Required, "Send an Idempotency-Key header of at most 200 characters with every write.")
	}
	var list []problem.Problem
	text := func(field, value string, required bool, max int) string {
		clean, p := problem.Text(field, value, required, max)
		if p != nil {
			list = append(list, *p)
		}
		return clean
	}
	app := text("source_application", in.SourceApplication, true, 100)
	behalf := text("on_behalf_of", in.OnBehalfOf, false, 200)
	reason := text("reason", in.Reason, true, 2000)
	reference := text("reference", in.Reference, false, 200)
	changeType := in.ChangeType
	if changeType == "" {
		changeType = "content"
	}
	if len(list) > 0 {
		return ProposedChange{}, false, &problem.ValidationError{Problems: list}
	}
	body, err := json.Marshal(in)
	if err != nil {
		return ProposedChange{}, false, err
	}
	sum := sha256.Sum256(body)
	requestHash := hex.EncodeToString(sum[:])

	var out ProposedChange
	replayed := false
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Kunci idempoten diperiksa dulu. Dokumen dikunci sesudahnya, jadi dua
		// permintaan kembar yang datang bersamaan tetap bergantian.
		if !db.IsUUID(in.DocumentID) {
			return problem.ErrNotFound
		}
		if err := q.LockDocument(ctx, in.DocumentID); err != nil {
			return err
		}
		prev, err := q.GetIntegrationRequest(ctx, db.GetIntegrationRequestParams{ServiceAccountID: c.ServiceAccountID, IdempotencyKey: key})
		switch {
		case err == nil:
			if prev.RequestHash != requestHash || prev.ChangeID == nil {
				return &problem.ConflictError{Problems: []problem.Problem{{Field: "Idempotency-Key", Code: problem.Taken, Message: "This Idempotency-Key was used for a different request."}}}
			}
			replayed = true
			out, err = s.changeStatus(ctx, q, c, *prev.ChangeID)
			return err
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}

		r, snap, err := s.releasedRevision(ctx, q, c, in.DocumentID, nil)
		if err != nil {
			return err
		}
		// Revisi dasar harus revisi yang berlaku sekarang; sistem tidak pernah
		// menggabung perubahan luar secara diam-diam (R-API-11).
		if int(r.Revision) != in.BaseRevision {
			return &problem.ConflictError{Problems: []problem.Problem{{Field: "base_revision", Code: problem.Changed,
				Message: fmt.Sprintf("The revision in force is %d. Read it again and send the proposal against it.", r.Revision)}}}
		}
		if err := document.CheckPatchAgainstSnapshot(ctx, q, snap, in.Operations); err != nil {
			return err
		}
		typeID, err := q.ChangeTypeByKey(ctx, changeType)
		if errors.Is(err, pgx.ErrNoRows) {
			return invalid("change_type", problem.InvalidFormat, "Use content, layout, or administrative.")
		}
		if err != nil {
			return err
		}
		var ref *string
		if reference != "" {
			ref = &reference
		}
		created, err := q.CreateIntegrationChange(ctx, db.CreateIntegrationChangeParams{
			CustomerID: r.CustomerID, ChangeTypeID: typeID, Reason: reason, SourceReference: ref, RequestedByServiceAccountID: &c.ServiceAccountID,
		})
		if err != nil {
			return err
		}
		if err := q.AddChangeDocument(ctx, db.AddChangeDocumentParams{ChangeID: created.ID, DocumentID: in.DocumentID}); err != nil {
			return err
		}
		ops, err := json.Marshal(in.Operations)
		if err != nil {
			return err
		}
		var onBehalf *string
		if behalf != "" {
			onBehalf = &behalf
		}
		if err := q.CreateChangeProposal(ctx, db.CreateChangeProposalParams{
			ChangeID: created.ID, DocumentID: in.DocumentID, BaseRevision: r.Revision, SourceApplication: app, OnBehalfOf: onBehalf, Operations: ops,
		}); err != nil {
			return err
		}
		if err := q.AddIntegrationRequest(ctx, db.AddIntegrationRequestParams{ServiceAccountID: c.ServiceAccountID, IdempotencyKey: key, RequestHash: requestHash, ChangeID: &created.ID}); err != nil {
			return err
		}
		details := c.details(map[string]string{"ref": changeRef(created.Seq), "document": r.Number, "base_revision": strconv.Itoa(in.BaseRevision),
			"operations": strconv.Itoa(len(in.Operations)), "source_application": app})
		if behalf != "" {
			details["on_behalf_of"] = behalf // pernyataan pemanggil, bukan bukti
		}
		if err := audit.Record(ctx, q, c.actor(), audit.Event{Action: audit.ActionChangeProposed, TargetType: audit.TargetChange, TargetID: created.ID, Details: details}); err != nil {
			return err
		}
		out, err = s.changeStatus(ctx, q, c, created.ID)
		return err
	})
	if err != nil {
		return ProposedChange{}, false, pass(err, "membuat usulan perubahan")
	}
	return out, replayed, nil
}

func (s *Service) changeStatus(ctx context.Context, q *db.Queries, c Client, id string) (ProposedChange, error) {
	if !db.IsUUID(id) {
		return ProposedChange{}, problem.ErrNotFound
	}
	r, err := q.GetIntegrationChangeStatus(ctx, db.GetIntegrationChangeStatusParams{ID: id, RequestedByServiceAccountID: &c.ServiceAccountID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ProposedChange{}, problem.ErrNotFound
	}
	if err != nil {
		return ProposedChange{}, err
	}
	out := ProposedChange{ID: r.ID, Ref: changeRef(r.Seq), Status: r.Status, CreatedAt: r.CreatedAt.UTC(), VerifiedAt: r.VerifiedAt}
	if r.VerificationNote != nil {
		out.VerificationNote = *r.VerificationNote
	}
	return out, nil
}

// ChangeStatus memberi status satu usulan milik service account ini (scope
// changes:read). Usulan aplikasi lain dijawab "tidak ada".
func (s *Service) ChangeStatus(ctx context.Context, c Client, id string) (ProposedChange, error) {
	if err := c.require(ScopeChangesRead); err != nil {
		return ProposedChange{}, err
	}
	return s.changeStatus(ctx, s.q, c, id)
}

// Job adalah satu job ekspor (R-API-14).
type Job struct {
	ID         string
	DocumentID string
	Revision   int
	Status     string // queued, running, done, atau failed
	FileName   string
	SizeBytes  int64
	Error      string
	CreatedAt  time.Time
	FinishedAt *time.Time
}

// CreateExport mengantrekan ekspor Excel satu revisi terbit (scope
// exports:create). Pekerjaannya dijalankan di latar belakang, supaya ekspor
// besar tidak menahan permintaan dan tidak memperlambat layar pengguna.
func (s *Service) CreateExport(ctx context.Context, c Client, documentID string, revision *int) (Job, error) {
	if err := c.require(ScopeExportsCreate); err != nil {
		return Job{}, err
	}
	r, _, err := s.releasedRevision(ctx, s.q, c, documentID, revision)
	if err != nil {
		return Job{}, err
	}
	var job Job
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		created, err := q.CreateExportJob(ctx, db.CreateExportJobParams{ServiceAccountID: c.ServiceAccountID, DocumentID: r.ID, Revision: r.Revision})
		if err != nil {
			return err
		}
		job = Job{ID: created.ID, DocumentID: r.ID, Revision: int(r.Revision), Status: "queued", CreatedAt: created.CreatedAt.UTC()}
		return audit.Record(ctx, q, c.actor(), audit.Event{Action: audit.ActionExportRequested, TargetType: audit.TargetDocument, TargetID: r.ID,
			Details: c.details(map[string]string{"number": r.Number, "revision": strconv.Itoa(int(r.Revision))})})
	})
	if err != nil {
		return Job{}, pass(err, "mengantrekan ekspor")
	}
	return job, nil
}

// GetExport memberi keadaan satu job milik service account ini.
func (s *Service) GetExport(ctx context.Context, c Client, id string) (Job, error) {
	if err := c.require(ScopeExportsCreate); err != nil {
		return Job{}, err
	}
	job, _, err := s.exportJob(ctx, c, id)
	return job, err
}

func (s *Service) exportJob(ctx context.Context, c Client, id string) (Job, string, error) {
	if !db.IsUUID(id) {
		return Job{}, "", problem.ErrNotFound
	}
	r, err := s.q.GetExportJob(ctx, db.GetExportJobParams{ID: id, ServiceAccountID: c.ServiceAccountID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, "", problem.ErrNotFound
	}
	if err != nil {
		return Job{}, "", err
	}
	job := Job{ID: r.ID, DocumentID: r.DocumentID, Revision: int(r.Revision), Status: r.Status, CreatedAt: r.CreatedAt.UTC(), FinishedAt: r.FinishedAt}
	if r.FileName != nil {
		job.FileName = *r.FileName
	}
	if r.SizeBytes != nil {
		job.SizeBytes = *r.SizeBytes
	}
	if r.Error != nil {
		job.Error = *r.Error
	}
	sum := ""
	if r.FileSha256 != nil {
		sum = *r.FileSha256
	}
	return job, sum, nil
}

// OpenExport membuka berkas hasil satu job yang sudah selesai.
func (s *Service) OpenExport(ctx context.Context, c Client, id string) (Job, *os.File, error) {
	if err := c.require(ScopeExportsCreate); err != nil {
		return Job{}, nil, err
	}
	job, sum, err := s.exportJob(ctx, c, id)
	if err != nil {
		return Job{}, nil, err
	}
	if job.Status != "done" || s.store == nil {
		return Job{}, nil, problem.ErrNotFound
	}
	f, err := s.store.Open(sum)
	if err != nil {
		return Job{}, nil, problem.ErrNotFound
	}
	return job, f, nil
}

// ProcessNextExport mengerjakan satu job yang antre. Hasil false berarti
// antrean kosong. Kegagalan satu job dicatat di job itu, bukan dikembalikan.
func (s *Service) ProcessNextExport(ctx context.Context) (bool, error) {
	if s.printer == nil || s.store == nil {
		return false, errors.New("ekspor belum dipasang")
	}
	job, err := s.q.ClaimExportJob(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("mengambil job ekspor: %w", err)
	}
	file, err := s.printer.ExportReleased(ctx, job.DocumentID, int(job.Revision))
	if err == nil {
		var sum string
		var size int64
		if sum, size, err = s.store.Put(strings.NewReader(string(file.Body)), int64(len(file.Body))+1); err == nil {
			return true, s.q.FinishExportJob(ctx, db.FinishExportJobParams{ID: job.ID, FileName: &file.Name, FileSha256: &sum, SizeBytes: &size})
		}
	}
	// Pesan untuk pemanggil sengaja umum; rinciannya untuk log server (R-API-13).
	message := "The export could not be made."
	if failErr := s.q.FailExportJob(ctx, db.FailExportJobParams{ID: job.ID, Error: &message}); failErr != nil {
		return true, failErr
	}
	return true, fmt.Errorf("job ekspor %s gagal: %w", job.ID, err)
}

// RunExports mengerjakan antrean ekspor sampai ctx selesai. Dipanggil sekali
// saat server mulai, sebagai goroutine. report menerima kegagalan untuk log.
func (s *Service) RunExports(ctx context.Context, every time.Duration, report func(error)) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		for {
			worked, err := s.ProcessNextExport(ctx)
			if err != nil && report != nil {
				report(err)
			}
			if !worked || ctx.Err() != nil {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
