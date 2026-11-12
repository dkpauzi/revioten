package template

// Persetujuan form set dan tabel AP (D-139, diperbarui 12 November 2026).
//
// Siapa yang boleh menyetujui: pemegang hak V-3 (peran Penyetuju form),
// bukan penyusun drafnya. Berapa yang harus setuju diatur admin di Settings:
//
//	all   semua penyetuju harus setuju (bawaan, keputusan pemilik)
//	any   satu persetujuan cukup
//
// Setiap persetujuan dicatat per orang di form_approvals, yang hanya bisa
// ditambah. "Semua penyetuju" dihitung saat persetujuan diberikan dari akun
// kerja aktif yang memegang V-3, jadi penyetuju yang ditambah admin di tengah
// jalan ikut ditunggu, dan akun yang dinonaktifkan tidak lagi ditunggu.

import (
	"context"
	"slices"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Aturan persetujuan (system_settings.form_approval_mode).
const (
	ApprovalAll = "all"
	ApprovalAny = "any"
)

// Jenis sasaran persetujuan.
const (
	approvalFormSetVersion = "form_set_version"
	approvalFormatVersion  = "format_version"
)

// Approver adalah satu penyetuju: yang sudah setuju (ApprovedAt terisi) atau
// yang masih ditunggu.
type Approver struct {
	PersonID   string
	PersonName string
	ApprovedAt *time.Time // nil untuk penyetuju yang masih ditunggu; selalu UTC
}

// approvalState membaca persetujuan satu versi, dan, selama versi itu masih
// draft, penyetuju yang masih ditunggu menurut aturan yang berlaku.
func approvalState(ctx context.Context, q *db.Queries, kind, targetID, preparerPersonID string, draft bool) (approved, pending []Approver, err error) {
	rows, err := q.ListFormApprovals(ctx, db.ListFormApprovalsParams{TargetKind: kind, TargetID: targetID})
	if err != nil {
		return nil, nil, err
	}
	approved = []Approver{}
	for _, r := range rows {
		at := r.ApprovedAt.UTC()
		approved = append(approved, Approver{PersonID: r.PersonID, PersonName: r.PersonName, ApprovedAt: &at})
	}
	pending = []Approver{}
	if !draft {
		return approved, pending, nil
	}
	mode, err := q.GetFormApprovalMode(ctx)
	if err != nil {
		return nil, nil, err
	}
	if mode == ApprovalAny && len(approved) > 0 {
		return approved, pending, nil
	}
	eligible, err := q.ListFormApprovers(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range eligible {
		done := slices.ContainsFunc(approved, func(a Approver) bool { return a.PersonID == e.PersonID })
		if e.PersonID != preparerPersonID && !done {
			pending = append(pending, Approver{PersonID: e.PersonID, PersonName: e.PersonName})
		}
	}
	return approved, pending, nil
}

// approve mencatat persetujuan actor untuk satu versi draft, lalu melaporkan
// apakah syarat terbit sudah terpenuhi, dan berapa penyetuju yang masih
// ditunggu. Pemanggil sudah memeriksa hak V-3, status draft, dan bahwa actor
// bukan penyusunnya.
func approve(ctx context.Context, q *db.Queries, actor access.Actor, kind, targetID, preparerPersonID string) (complete bool, waiting int, err error) {
	before, _, err := approvalState(ctx, q, kind, targetID, preparerPersonID, true)
	if err != nil {
		return false, 0, err
	}
	if slices.ContainsFunc(before, func(a Approver) bool { return a.PersonID == actor.PersonID }) {
		return false, 0, &problem.ConflictError{Problems: []problem.Problem{{
			Field: "approver", Code: problem.Taken, Message: "You already approved this version. It waits for the other approvers.",
		}}}
	}
	err = q.AddFormApproval(ctx, db.AddFormApprovalParams{TargetKind: kind, TargetID: targetID, PersonID: actor.PersonID, AccountID: actor.AccountID})
	if err != nil {
		return false, 0, err
	}
	_, pending, err := approvalState(ctx, q, kind, targetID, preparerPersonID, true)
	if err != nil {
		return false, 0, err
	}
	return len(pending) == 0, len(pending), nil
}
