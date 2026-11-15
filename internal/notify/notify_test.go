package notify_test

import (
	"context"

	"errors"
	"github.com/jackc/pgx/v5"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/notify"
	"github.com/dkpauzi/revioten/internal/problem"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// fakeMailer mencatat email yang dikirim, dan bisa dibuat gagal.
type fakeMailer struct {
	mu   sync.Mutex
	sent []notify.Mail
	fail error
}

func (f *fakeMailer) Send(_ context.Context, m notify.Mail) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail != nil {
		return f.fail
	}
	f.sent = append(f.sent, m)
	return nil
}

func code(t *testing.T, err error) string {
	t.Helper()
	var invalid *problem.ValidationError
	if !errors.As(err, &invalid) || len(invalid.Problems) == 0 {
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	return invalid.Problems[0].Field + ":" + invalid.Problems[0].Code
}

// TestNotifications: perubahan yang diajukan menulis email untuk verifikator
// customer itu, keputusan verifikasi untuk pemohon, dan scan bertanda tangan
// untuk dokumen kontrol. Email dikirim pekerja lewat Mailer (D-194).
func TestNotifications(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	audit.SetListener(notify.NewNotifier("http://contoh.test").Strict().Listener())
	t.Cleanup(func() { audit.SetListener(nil) })
	mailer := &fakeMailer{}
	svc := notify.NewService(e.Pool, e.Checker, mailer, notify.Config{Provider: "log"}, nil)
	changes := change.NewRequestService(e.Pool, e.Checker)

	controller := doctest.NewActor(t, e.Pool, "apqp.controller", "work", doctest.All(access.RoleVerifier))
	if _, err := e.Pool.Exec(ctx, `INSERT INTO position_holders (position_id, per_project_group, person_id)
		SELECT id, per_project_group, $1 FROM positions WHERE key = 'apqp_ppap_controller'`, controller.PersonID); err != nil {
		t.Fatal(err)
	}
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))

	// Alamat email: diisi admin, atau oleh orangnya sendiri; alamat salah ditolak.
	if _, err := svc.SetEmail(ctx, controller, controller.AccountID, "bukan email"); code(t, err) != "email:invalid_format" {
		t.Errorf("email salah: %v", err)
	}
	if _, err := svc.SetEmail(ctx, controller, controller.AccountID, " controller@contoh.test "); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetEmail(ctx, controller, e.Author.AccountID, "x@contoh.test"); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja mengubah email orang lain: %v", err)
	}
	for _, a := range []access.Actor{e.Author, dcc} {
		if _, err := svc.SetEmail(ctx, e.Admin, a.AccountID, a.Login+"@contoh.test"); err != nil {
			t.Fatal(err)
		}
	}
	if own, err := svc.OwnEmail(ctx, controller); err != nil || own != "controller@contoh.test" {
		t.Errorf("email sendiri: %q, %v", own, err)
	}

	cp := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)["cp"]
	if _, err := e.Docs.AddRow(ctx, e.Author, cp.ID, document.NewRow{Content: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	doctest.AttachScan(t, e.Pool, cp.ID)
	if _, err := e.Docs.Release(ctx, dcc, cp.ID); err != nil {
		t.Fatal(err)
	}
	var source, changeType string
	if err := e.Pool.QueryRow(ctx, `SELECT id FROM change_sources WHERE name = 'Customer claim'`).Scan(&source); err != nil {
		t.Fatal(err)
	}
	if err := e.Pool.QueryRow(ctx, `SELECT id FROM change_types ORDER BY position LIMIT 1`).Scan(&changeType); err != nil {
		t.Fatal(err)
	}
	c, err := changes.Create(ctx, e.Author, change.NewChange{CustomerID: e.CustomerAB, TriggerID: source, ChangeTypeID: changeType, Reason: "Claim 31", DocumentIDs: []string{cp.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := changes.Verify(ctx, controller, c.ID, true, "OK"); err != nil {
		t.Fatal(err)
	}

	if n, err := svc.SendPending(ctx, 50); err != nil || n == 0 {
		t.Fatalf("mengirim: %d, %v", n, err)
	}
	got := map[string][]string{}
	for _, m := range mailer.sent {
		got[m.To] = append(got[m.To], m.Subject)
	}
	verifier := strings.Join(got["controller@contoh.test"], " | ")
	author := strings.Join(got["penyusun.uji@contoh.test"], " | ")
	if !strings.Contains(verifier, c.Ref+" waits for your verification") {
		t.Errorf("email verifikator: %v", got)
	}
	if !strings.Contains(author, c.Ref+" is accepted") {
		t.Errorf("email pemohon: %v", got)
	}
	// Pemohon tidak mendapat email atas pengajuannya sendiri.
	if strings.Contains(author, "waits for your verification") {
		t.Errorf("pemohon diberi tahu pengajuannya sendiri: %v", got)
	}
	for _, m := range mailer.sent {
		if strings.Contains(m.Body, "verification") && !strings.Contains(m.Body, "http://contoh.test/changes/view?id="+c.ID) {
			t.Errorf("tautan di email: %s", m.Body)
		}
	}

	// Scan bertanda tangan revisi 1: dokumen kontrol diberi tahu, pengunggahnya tidak.
	doctest.AttachScan(t, e.Pool, cp.ID)
	before := len(mailer.sent)
	if err := auditScan(ctx, e, controller, cp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendPending(ctx, 50); err != nil {
		t.Fatal(err)
	}
	if len(mailer.sent) != before+1 || mailer.sent[before].To != "dcc.uji@contoh.test" || !strings.Contains(mailer.sent[before].Subject, "ready to release") {
		t.Errorf("email siap rilis: %+v", mailer.sent[before:])
	}

	// Saklar mati: peristiwa itu tidak menulis email.
	if err := svc.SetEvent(ctx, e.Admin, notify.EventChangeSubmitted, false); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetEvent(ctx, e.Author, notify.EventChangeSubmitted, true); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("akun kerja mengubah saklar: %v", err)
	}
	if _, err := changes.Create(ctx, e.Author, change.NewChange{CustomerID: e.CustomerAB, TriggerID: source, ChangeTypeID: changeType, Reason: "Claim 32", DocumentIDs: []string{cp.ID}}); err != nil {
		t.Fatal(err)
	}
	if n, _ := svc.SendPending(ctx, 50); n != 0 {
		t.Errorf("email ditulis walau saklar mati: %d", n)
	}

	// Penerima tanpa alamat langsung skipped; layar admin menampilkannya.
	if _, err := svc.SetEmail(ctx, e.Admin, dcc.AccountID, ""); err != nil {
		t.Fatal(err)
	}
	if err := auditScan(ctx, e, controller, cp.ID); err != nil {
		t.Fatal(err)
	}
	view, err := svc.Overview(ctx, e.Admin)
	if err != nil {
		t.Fatal(err)
	}
	if view.Provider != "log" || len(view.Events) != 8 || view.Outbox[0].Status != "skipped" || view.Outbox[0].Reason != "no_email" {
		t.Errorf("layar admin: %+v", view.Outbox[0])
	}

	// Gagal kirim dicoba lagi, lalu ditandai gagal sesudah percobaan habis.
	if _, err := svc.SetEmail(ctx, e.Admin, dcc.AccountID, "dcc.uji@contoh.test"); err != nil {
		t.Fatal(err)
	}
	if err := auditScan(ctx, e, controller, cp.ID); err != nil {
		t.Fatal(err)
	}
	mailer.fail = errors.New("server menolak")
	for i := 0; i < notify.MaxAttempts; i++ {
		if _, err := svc.SendPending(ctx, 50); err != nil {
			t.Fatal(err)
		}
		if _, err := e.Pool.Exec(ctx, `UPDATE notification_outbox SET next_attempt_at = now() WHERE status = 'pending'`); err != nil {
			t.Fatal(err)
		}
	}
	var status, reason string
	if err := e.Pool.QueryRow(ctx, `SELECT status, reason FROM notification_outbox ORDER BY created_at DESC LIMIT 1`).Scan(&status, &reason); err != nil || status != "failed" || reason != "server menolak" {
		t.Errorf("sesudah gagal berulang: %s %s %v", status, reason, err)
	}

	// Tanpa pengirim (MAIL_PROVIDER kosong): email ditandai skipped.
	off := notify.NewService(e.Pool, e.Checker, nil, notify.Config{}, nil)
	if err := auditScan(ctx, e, controller, cp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := off.SendPending(ctx, 50); err != nil {
		t.Fatal(err)
	}
	if err := e.Pool.QueryRow(ctx, `SELECT status, reason FROM notification_outbox ORDER BY created_at DESC LIMIT 1`).Scan(&status, &reason); err != nil || status != "skipped" || reason != "mail_not_configured" {
		t.Errorf("tanpa pengirim: %s %s %v", status, reason, err)
	}

	// Email uji ke diri sendiri, dan tidak ada rahasia di kotak keluar.
	if _, err := svc.SetEmail(ctx, e.Admin, e.Admin.AccountID, "admin@contoh.test"); err != nil {
		t.Fatal(err)
	}
	if err := svc.SendTest(ctx, e.Admin); err != nil {
		t.Fatal(err)
	}
	var leaks int
	if err := e.Pool.QueryRow(ctx, `SELECT count(*) FROM notification_outbox WHERE body ILIKE '%password:%' OR body LIKE '%rvt\_%'`).Scan(&leaks); err != nil || leaks != 0 {
		t.Errorf("rahasia di email: %d, %v", leaks, err)
	}
}

// auditScan mencatat peristiwa "scan bertanda tangan diunggah" seperti
// layanan cetak, tanpa berkas sungguhan.
func auditScan(ctx context.Context, e doctest.Env, actor access.Actor, docID string) error {
	return db.WithTx(ctx, e.Pool, func(tx pgx.Tx) error {
		return audit.Record(ctx, db.New(e.Pool).WithTx(tx), actor, audit.Event{Action: audit.ActionDocumentScanUploaded, TargetType: audit.TargetDocument, TargetID: docID})
	})
}
