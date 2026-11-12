package template

// Versi template: draft lalu terbit (M3.4, R-TPL-05).
//
// Alur satu template:
//
//	versi 1 draft  --terbitkan-->  versi 1 terbit (tidak bisa diubah lagi)
//	                                    |
//	                                    +--versi baru-->  versi 2 draft (salinan versi 1)
//
// Aturannya sama untuk format template (hak A-2) dan process template (hak
// E-6, D-64):
//   - hanya versi draft yang bisa diterbitkan;
//   - versi baru hanya dibuat bila versi terakhir sudah terbit, jadi satu
//     template paling banyak punya satu draft;
//   - versi baru menyalin isi versi terakhir dengan ID yang sama, supaya
//     dokumen yang menunjuk field atau poin itu tetap tersambung (R-TPL-02,
//     R-TPL-04).
//
// Database menjaga hal yang sama lewat trigger dan indeks unik (migrasi
// 00016), jadi aturan ini tetap berlaku bila layanan dilewati. Propagasi
// versi baru ke dokumen belum ada (M5.8, R-TPL-06).

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// statusConflict adalah penolakan karena status versi tidak cocok dengan aksinya.
func statusConflict(message string) error {
	return &problem.ConflictError{Problems: []problem.Problem{{Field: "status", Code: problem.NotAllowed, Message: message}}}
}

const (
	alreadyPublished = "This version is already published."
	draftExists      = "Publish the draft version before making a new version."
)

// utcTime mengubah waktu yang boleh kosong ke UTC.
func utcTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}

// PublishFormatVersion menerbitkan satu versi draft format template (A-2).
// D-143 (keputusan E-87): penyusun draft boleh menerbitkannya sendiri.
// Sesudah itu versi tersebut tidak bisa diubah.
//
// Versi bermetode AP membawa tabel AP, jadi diterbitkan pemegang V-3 dan
// bukan oleh orang yang terakhir mengubahnya (D-139). Versi lain tetap
// diterbitkan admin, termasuk oleh penyusunnya sendiri (D-143).
func (s *Service) PublishFormatVersion(ctx context.Context, actor access.Actor, versionID string) (FormatVersion, error) {
	// Hak yang dibutuhkan bergantung pada metode risiko versi itu. Versi yang
	// tidak ada diperlakukan seperti versi biasa, supaya jawabannya sama.
	needed := access.ManageMasterData
	if db.IsUUID(versionID) {
		if row, err := s.q.GetFormatVersion(ctx, versionID); err == nil && row.RiskMethod == RiskAP {
			needed = access.ApproveForms
		}
	}
	if err := s.authz.Authorize(ctx, actor, needed, access.Resource{}); err != nil {
		return FormatVersion{}, err
	}
	if !db.IsUUID(versionID) {
		return FormatVersion{}, problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Kunci yang sama dengan editVersion: perubahan yang sedang berjalan
		// selesai dulu, baru versinya terbit.
		if err := q.LockFormatVersion(ctx, versionID); err != nil {
			return err
		}
		v, err := getVersion(ctx, q, versionID)
		if err != nil {
			return err
		}
		if v.Status != StatusDraft {
			return statusConflict(alreadyPublished)
		}
		if v.RiskMethod == RiskAP && v.ChangedByPersonID != "" && v.ChangedByPersonID == actor.PersonID {
			return errSamePerson()
		}
		if _, err := q.PublishFormatVersion(ctx, versionID); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionFormatVersionPublished, TargetType: audit.TargetFormatTemplate, TargetID: v.TemplateID,
			Details: map[string]string{"version": strconv.Itoa(v.Version)},
		})
	})
	if err := result(err, "menerbitkan versi format template"); err != nil {
		return FormatVersion{}, err
	}
	return getVersion(ctx, s.q, versionID)
}

// NewFormatVersion membuat versi draft berikutnya dari versi terbit terakhir
// (A-2). Ditolak bila template itu masih punya versi draft.
func (s *Service) NewFormatVersion(ctx context.Context, actor access.Actor, templateID string) (FormatVersion, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageMasterData, access.Resource{}); err != nil {
		return FormatVersion{}, err
	}
	if !db.IsUUID(templateID) {
		return FormatVersion{}, problem.ErrNotFound
	}
	var newID string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Template dikunci supaya dua permintaan versi baru berjalan bergantian.
		if err := q.LockFormatTemplate(ctx, templateID); err != nil {
			return err
		}
		ft, err := getTemplate(ctx, q, templateID)
		if err != nil {
			return err
		}
		last := ft.Versions[len(ft.Versions)-1]
		if last.Status != StatusPublished {
			return statusConflict(draftExists)
		}
		source, err := getVersion(ctx, q, last.ID)
		if err != nil {
			return err
		}
		newID, err = q.CreateFormatVersion(ctx, db.CreateFormatVersionParams{
			TemplateID: templateID, Version: int32(last.Version + 1), RiskMethod: source.RiskMethod,
			HighlightRpnAbove: optionalSmall(source.HighlightRPNAbove), HighlightSeverityAtLeast: optionalSmall(source.HighlightSeverityAtLeast),
			ApRules: apRulesJSON(source.APRules),
		})
		if err != nil {
			return err
		}
		if err := q.CopyFormatVersionFields(ctx, db.CopyFormatVersionFieldsParams{TargetID: newID, SourceID: last.ID}); err != nil {
			return err
		}
		if err := q.CopyFormatRatingCriteria(ctx, db.CopyFormatRatingCriteriaParams{TargetID: newID, SourceID: last.ID}); err != nil {
			return err
		}
		if err := q.CopyFormatSymbols(ctx, db.CopyFormatSymbolsParams{TargetID: newID, SourceID: last.ID}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionFormatVersionCreated, TargetType: audit.TargetFormatTemplate, TargetID: templateID,
			Details: map[string]string{"version": strconv.Itoa(last.Version + 1), "copied_from": strconv.Itoa(last.Version)},
		})
	})
	if err := result(err, "membuat versi format template"); err != nil {
		return FormatVersion{}, err
	}
	return getVersion(ctx, s.q, newID)
}

// PublishProcessVersion menerbitkan satu versi draft process template (E-6).
func (s *ProcessService) PublishProcessVersion(ctx context.Context, actor access.Actor, versionID string) (ProcessVersion, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return ProcessVersion{}, err
	}
	if !db.IsUUID(versionID) {
		return ProcessVersion{}, problem.ErrNotFound
	}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockProcessVersion(ctx, versionID); err != nil {
			return err
		}
		v, err := getProcessVersion(ctx, q, versionID)
		if err != nil {
			return err
		}
		if v.Status != StatusDraft {
			return statusConflict(alreadyPublished)
		}
		if _, err := q.PublishProcessVersion(ctx, versionID); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionProcessVersionPublished, TargetType: audit.TargetProcessTemplate, TargetID: v.TemplateID,
			Details: map[string]string{"version": strconv.Itoa(v.Version)},
		})
	})
	if err := result(err, "menerbitkan versi process template"); err != nil {
		return ProcessVersion{}, err
	}
	return getProcessVersion(ctx, s.q, versionID)
}

// NewProcessVersion membuat versi draft berikutnya dari versi terbit
// terakhir (E-6): sub-proses, poin, dan varian disalin dengan ID yang sama.
func (s *ProcessService) NewProcessVersion(ctx context.Context, actor access.Actor, templateID string) (ProcessVersion, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return ProcessVersion{}, err
	}
	if !db.IsUUID(templateID) {
		return ProcessVersion{}, problem.ErrNotFound
	}
	var newID string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		// Tabel process_templates tidak dikunci, karena role aplikasi tidak
		// punya UPDATE di sana (E-06). Dua permintaan yang bersamaan dijaga
		// indeks unik "satu draft per template": yang kedua ditolak database
		// dan diterjemahkan result() menjadi penolakan yang sama.
		pt, err := getProcessTemplate(ctx, q, templateID)
		if err != nil {
			return err
		}
		last := pt.Versions[len(pt.Versions)-1]
		if last.Status != StatusPublished {
			return statusConflict(draftExists)
		}
		newID, err = q.CreateProcessVersion(ctx, db.CreateProcessVersionParams{
			TemplateID: templateID, ProcessID: pt.Process.ID, Version: int32(last.Version + 1),
		})
		if err != nil {
			return err
		}
		// Urutan penting: poin menunjuk sub-proses, dan varian menunjuk poin.
		if err := q.CopyProcessVersionSubprocesses(ctx, db.CopyProcessVersionSubprocessesParams{TargetID: newID, SourceID: last.ID}); err != nil {
			return err
		}
		if err := q.CopyProcessVersionPoints(ctx, db.CopyProcessVersionPointsParams{TargetID: newID, SourceID: last.ID}); err != nil {
			return err
		}
		if err := q.CopyProcessVersionVariants(ctx, db.CopyProcessVersionVariantsParams{TargetID: newID, SourceID: last.ID}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionProcessVersionCreated, TargetType: audit.TargetProcessTemplate, TargetID: templateID,
			Details: map[string]string{"version": strconv.Itoa(last.Version + 1), "copied_from": strconv.Itoa(last.Version)},
		})
	})
	if err := result(err, "membuat versi process template"); err != nil {
		return ProcessVersion{}, err
	}
	return getProcessVersion(ctx, s.q, newID)
}
