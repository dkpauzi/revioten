package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/health"
	"github.com/dkpauzi/revioten/internal/notify"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/signature"
)

// Fase 4 · Dasbor admin (D-205).

func TestMain(m *testing.M) {
	audit.SetListener(notify.NewNotifier("http://contoh.test").Strict().Listener())
	os.Exit(dbtest.Main(m))
}

type passwords struct{}

func (passwords) ConfirmPassword(context.Context, access.Actor, string) error { return nil }

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) || len(invalid.Problems) == 0 {
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
}

func TestOverview(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()

	// Backup: satu manifest di folder BACKUP_DIR.
	dir := t.TempDir()
	manifest, _ := json.Marshal(map[string]any{"format": 1, "created_at": "2026-11-19T20:00:00Z", "file": "revioten-20261119T200000Z.dump.age", "size": 1234})
	if err := os.WriteFile(filepath.Join(dir, "revioten-20261119T200000Z.manifest.json"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	svc := health.NewService(e.Pool, e.Checker, dir)

	// Satu tanda tangan tertahan: CP customer AB menunggu Project Leader GR1.
	sender := doctest.NewActor(t, e.Pool, "ppap.engineer", "work", doctest.All(access.RolePPAPEngineer))
	leader := doctest.NewActor(t, e.Pool, "pl.gr1", "work", access.RoleGrant{Role: access.RoleProjectLeader, Scope: access.ScopeOwnProjectGroup})
	npi := doctest.NewActor(t, e.Pool, "npi.manager", "work", doctest.All(access.RoleManager))
	qa := doctest.NewActor(t, e.Pool, "qa.manager", "work", doctest.All(access.RoleManager))
	for person, key := range map[string]string{sender.PersonID: "ppap_engineer", leader.PersonID: "project_leader", npi.PersonID: "npi_manager", qa.PersonID: "qa_qc_manager"} {
		group := any(nil)
		if key == "project_leader" {
			group = e.GroupID
		}
		if _, err := e.Pool.Exec(ctx, `INSERT INTO position_holders (position_id, per_project_group, person_id, project_group_id)
			SELECT id, per_project_group, $1, $3 FROM positions WHERE key = $2`, person, key, group); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.Pool.Exec(ctx, `UPDATE customers SET default_route_id = (SELECT id FROM approval_routes WHERE name = 'Up to QA QC Manager')`); err != nil {
		t.Fatal(err)
	}
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	sigs := signature.NewService(e.Pool, e.Checker, passwords{})
	if _, err := sigs.Send(ctx, sender, set["cp"].ID, signature.SendInput{Signers: map[int]string{1: sender.PersonID, 2: leader.PersonID}}); err != nil {
		t.Fatal(err)
	}
	// Satu revisi terbit: PFD revisi awal.
	recorder := doctest.NewActor(t, e.Pool, "ppap.dua", "work", doctest.All(access.RolePPAPEngineer))
	doctest.AttachScan(t, e.Pool, set["pfd"].ID)
	if _, err := e.Docs.Release(ctx, recorder, set["pfd"].ID); err != nil {
		t.Fatal(err)
	}
	// Error server dan laporan masalah.
	if err := svc.RecordError(ctx, "REQ1", "GET", "/api/v1/numbers", e.Author.AccountID, strings.Repeat("x", 3000)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReportProblem(ctx, e.Viewer, "/numbers", "  ", ""); codeOf(t, err) != "description:required" {
		t.Errorf("laporan kosong: %v", err)
	}
	pid, err := svc.ReportProblem(ctx, e.Viewer, "/numbers", "Tombol Register tidak bereaksi", "REQ1")
	if err != nil {
		t.Fatal(err)
	}

	// Hanya admin (A-3).
	if _, err := svc.Overview(ctx, sender); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja membuka dasbor admin: %v", err)
	}
	o, err := svc.Overview(ctx, e.Admin)
	if err != nil {
		t.Fatal(err)
	}
	if o.SignaturesWaiting != 1 || o.TopSignatures[0].Signer != "Orang pl.gr1" || o.TopSignatures[0].Slot != "Engineer" || o.TopSignatures[0].Number != set["cp"].Number {
		t.Errorf("tanda tangan tertahan: %+v", o.TopSignatures)
	}
	if o.Errors24h != 1 || o.Errors7d != 1 || len([]rune(o.LatestErrors[0].Message)) != 2000 || o.LatestErrors[0].Login != "penyusun.uji" {
		t.Errorf("error: %d %d %+v", o.Errors24h, o.Errors7d, o.LatestErrors)
	}
	if o.ProblemsOpen != 1 || o.LatestProblems[0].ReportedBy != "Orang viewer.uji" || o.LatestProblems[0].LastRequestID != "REQ1" {
		t.Errorf("laporan masalah: %+v", o.LatestProblems)
	}
	if o.Week.Total != 1 || o.Week.Counts[health.CategoryInitial] != 1 || o.Month.Total != 1 {
		t.Errorf("dokumen diproses: %+v %+v", o.Week, o.Month)
	}
	if !o.Backup.Configured || !o.Backup.Found || o.Backup.CreatedAt != time.Date(2026, 11, 19, 20, 0, 0, 0, time.UTC) {
		t.Errorf("backup: %+v", o.Backup)
	}
	if !o.AuditValid || o.AuditEntries == 0 {
		t.Errorf("audit: %v %d", o.AuditValid, o.AuditEntries)
	}

	// Laporan masalah ditangani admin.
	if err := svc.UpdateProblem(ctx, sender, pid, health.ProblemDone, ""); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja menangani laporan: %v", err)
	}
	if err := svc.UpdateProblem(ctx, e.Admin, pid, "selesai", ""); codeOf(t, err) != "status:invalid_format" {
		t.Errorf("status tidak dikenal: %v", err)
	}
	if err := svc.UpdateProblem(ctx, e.Admin, pid, health.ProblemDone, "Sudah diperbaiki"); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ProblemReports(ctx, e.Admin)
	if err != nil || list[0].Status != health.ProblemDone || list[0].HandledBy != "Orang admin.uji" || list[0].AdminNote != "Sudah diperbaiki" {
		t.Errorf("laporan sesudah ditangani: %+v, %v", list, err)
	}
	waiting, err := svc.WaitingSignatures(ctx, e.Admin)
	if err != nil || len(waiting) != 1 {
		t.Errorf("daftar tanda tangan tertahan: %+v, %v", waiting, err)
	}
}
