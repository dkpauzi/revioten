// Package notify mengirim email notifikasi (D-194).
//
// Cara kerjanya dua langkah:
//
//  1. Pendengar audit (Listener) dipanggil setiap kali satu peristiwa dicatat
//     di audit log, di transaksi yang sama. Untuk peristiwa yang punya
//     notifikasi, ia mencari penerimanya dan menulis email ke kotak keluar
//     (tabel notification_outbox). Bila transaksinya batal, emailnya ikut batal.
//  2. Pekerja (Worker) di latar belakang mengambil email dari kotak keluar dan
//     mengirimnya lewat Mailer: Microsoft Graph atau SMTP Microsoft 365, yang
//     diatur IT lewat pengaturan. Gagal kirim dicoba lagi beberapa kali.
//
// Email tidak pernah memuat password, password sementara, atau token.
package notify

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
)

// Jenis peristiwa yang punya email (sama dengan tabel notification_events).
const (
	EventChangeSubmitted     = "change_submitted"
	EventChangeVerified      = "change_verified"
	EventReleaseReady        = "release_ready"
	EventRevisionReleased    = "revision_released"
	EventFormApprovalWaiting = "form_approval_waiting"
	EventYokotenCreated      = "yokoten_created"
	EventMigrationWaiting    = "migration_waiting"
	EventTokenExpiring       = "token_expiring"
	EventTest                = "test"
)

// Message adalah satu email yang akan ditulis ke kotak keluar.
type message struct {
	event   string
	subject string
	body    string
}

// Notifier menyusun email dari peristiwa audit.
type Notifier struct {
	// baseURL adalah alamat aplikasi untuk tautan di email, misalnya
	// http://192.168.1.10:8080. Kosong berarti tanpa tautan.
	baseURL string
	// strict membuat sasaran yang tidak ditemukan menjadi error. Dipakai test,
	// supaya peristiwa yang salah dipetakan langsung ketahuan.
	strict bool
}

// NewNotifier membuat Notifier. baseURL dipakai untuk tautan di email.
func NewNotifier(baseURL string) *Notifier {
	return &Notifier{baseURL: strings.TrimRight(baseURL, "/")}
}

// Strict memberi salinan Notifier yang tidak memaafkan sasaran yang tidak
// ditemukan. Hanya untuk test.
func (n *Notifier) Strict() *Notifier {
	c := *n
	c.strict = true
	return &c
}

// Listener memberi fungsi untuk audit.SetListener.
func (n *Notifier) Listener() audit.Listener {
	return n.handle
}

func (n *Notifier) link(path string) string {
	if n.baseURL == "" {
		return ""
	}
	return "\n\n" + n.baseURL + path
}

func changeRef(seq *int64) string {
	if seq == nil {
		return "CHG"
	}
	return fmt.Sprintf("CHG-%04d", *seq)
}

func text(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// handle memetakan satu peristiwa audit ke email. Peristiwa lain diabaikan.
func (n *Notifier) handle(ctx context.Context, q *db.Queries, actor access.Actor, e audit.Event) error {
	err := n.dispatch(ctx, q, actor, e)
	// Sasaran yang tidak ditemukan tidak boleh menggagalkan aksi orangnya:
	// emailnya saja yang tidak ditulis. Error database lain diteruskan.
	if errors.Is(err, pgx.ErrNoRows) && !n.strict {
		return nil
	}
	return err
}

// dispatch memilih penyusun email untuk satu peristiwa.
func (n *Notifier) dispatch(ctx context.Context, q *db.Queries, actor access.Actor, e audit.Event) error {
	switch e.Action {
	case audit.ActionChangeCreated, audit.ActionChangeProposed:
		return n.changeSubmitted(ctx, q, actor, e.TargetID)
	case audit.ActionChangeAccepted, audit.ActionChangeRejected:
		return n.changeVerified(ctx, q, actor, e.TargetID, e.Action == audit.ActionChangeAccepted)
	case audit.ActionDocumentScanUploaded:
		return n.releaseReady(ctx, q, actor, e.TargetID)
	case audit.ActionDocumentReleased, audit.ActionDocumentMigrationVerified:
		revision, _ := strconv.Atoi(e.Details["revision"])
		return n.revisionReleased(ctx, q, actor, e.TargetID, revision)
	case audit.ActionFormSetVersionApproved:
		return n.formApproval(ctx, q, actor, true, e.TargetID)
	case audit.ActionFormatVersionApproved:
		return n.formApproval(ctx, q, actor, false, e.TargetID)
	case audit.ActionYokotenCreated:
		// Sasaran peristiwa ini perubahan sumber; ID yokoten ada di rincian.
		return n.yokotenCreated(ctx, q, actor, e.Details["yokoten"])
	case audit.ActionDocumentMigrated:
		return n.migrationWaiting(ctx, q, actor, e.TargetID)
	}
	return nil
}

// enqueue menulis satu email untuk setiap penerima, tanpa orang yang memicu
// peristiwa itu sendiri, dan hanya bila jenis peristiwanya menyala.
func enqueue(ctx context.Context, q *db.Queries, actor access.Actor, m message, people []string) error {
	on, err := q.NotificationEventEnabled(ctx, m.event)
	if err != nil {
		return fmt.Errorf("membaca saklar notifikasi: %w", err)
	}
	if !on {
		return nil
	}
	slices.Sort(people)
	for _, p := range slices.Compact(people) {
		if p == "" || p == actor.PersonID {
			continue
		}
		if err := q.EnqueueNotification(ctx, db.EnqueueNotificationParams{Event: m.event, PersonID: p, Subject: m.subject, Body: m.body}); err != nil {
			return fmt.Errorf("menulis kotak keluar: %w", err)
		}
	}
	return nil
}

func (n *Notifier) changeSubmitted(ctx context.Context, q *db.Queries, actor access.Actor, id string) error {
	c, err := q.ChangeForNotice(ctx, id)
	if err != nil {
		return err
	}
	people, err := q.ChangeVerifiers(ctx, c.CustomerID)
	if err != nil {
		return err
	}
	ref := changeRef(c.Seq)
	m := message{
		event:   EventChangeSubmitted,
		subject: fmt.Sprintf("[%s] %s waits for your verification", c.CustomerCode, ref),
		body: fmt.Sprintf("%s was submitted by %s and waits for verification.\n%s menunggu verifikasi Anda.\n\nCustomer: %s\nDocuments: %s\nReason: %s",
			ref, c.RequestedBy, ref, c.CustomerCode, c.Documents, c.Reason) + n.link("/changes/view?id="+c.ID),
	}
	return enqueue(ctx, q, actor, m, people)
}

func (n *Notifier) changeVerified(ctx context.Context, q *db.Queries, actor access.Actor, id string, accepted bool) error {
	c, err := q.ChangeForNotice(ctx, id)
	if err != nil {
		return err
	}
	if c.RequestedByPersonID == nil {
		return nil // usulan dari aplikasi lain: aplikasi itu membaca statusnya lewat API
	}
	ref := changeRef(c.Seq)
	result, hasil := "rejected", "ditolak"
	next := "Read the note, then submit a new change if needed."
	if accepted {
		result, hasil = "accepted", "diterima"
		next = "The drafts of the affected documents are open for editing."
	}
	m := message{
		event:   EventChangeVerified,
		subject: fmt.Sprintf("[%s] %s is %s", c.CustomerCode, ref, result),
		body: fmt.Sprintf("%s is %s by %s.\n%s %s.\n\nNote: %s\nDocuments: %s\n\n%s",
			ref, result, actor.PersonName, ref, hasil, text(c.VerificationNote), c.Documents, next) + n.link("/changes/view?id="+c.ID),
	}
	return enqueue(ctx, q, actor, m, []string{*c.RequestedByPersonID})
}

func (n *Notifier) releaseReady(ctx context.Context, q *db.Queries, actor access.Actor, id string) error {
	d, err := q.DocumentForNotice(ctx, id)
	if err != nil {
		return err
	}
	people, err := q.PeopleWithPermission(ctx, db.PeopleWithPermissionParams{Permission: string(access.RecordRelease), CustomerID: &d.CustomerID})
	if err != nil {
		return err
	}
	m := message{
		event:   EventReleaseReady,
		subject: fmt.Sprintf("[%s] %s revision %d is signed and ready to release", d.CustomerCode, d.Number, d.Revision),
		body: fmt.Sprintf("The signed scan of %s revision %d was uploaded. Record the release when it is complete.\nScan bertanda tangan %s revisi %d sudah diunggah; catat rilisnya bila sudah lengkap.",
			d.Number, d.Revision, d.Number, d.Revision) + n.link("/documents/view?id="+d.ID),
	}
	return enqueue(ctx, q, actor, m, people)
}

func (n *Notifier) revisionReleased(ctx context.Context, q *db.Queries, actor access.Actor, id string, revision int) error {
	d, err := q.DocumentForNotice(ctx, id)
	if err != nil {
		return err
	}
	people, err := q.RequestersOfRevision(ctx, db.RequestersOfRevisionParams{DocumentID: id, Revision: int16(revision)})
	if err != nil {
		return err
	}
	m := message{
		event:   EventRevisionReleased,
		subject: fmt.Sprintf("[%s] %s revision %d is released", d.CustomerCode, d.Number, revision),
		body: fmt.Sprintf("%s revision %d is released and is now in force.\n%s revisi %d sudah terbit dan berlaku.",
			d.Number, revision, d.Number, revision) + n.link("/documents/view?id="+d.ID),
	}
	return enqueue(ctx, q, actor, m, people)
}

func (n *Notifier) formApproval(ctx context.Context, q *db.Queries, actor access.Actor, formSet bool, targetID string) error {
	var versionID, name, preparer string
	var version int32
	if formSet {
		v, err := q.DraftFormSetVersion(ctx, targetID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		versionID, name, version, preparer = v.ID, v.Name, int32(v.Version), text(v.PreparedByPersonID)
	} else {
		v, err := q.DraftFormatVersion(ctx, targetID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		versionID, name, version, preparer = v.ID, v.Name, int32(v.Version), text(v.ChangedByPersonID)
	}
	var preparerID *string
	if preparer != "" {
		preparerID = &preparer
	}
	people, err := q.PendingFormApprovers(ctx, db.PendingFormApproversParams{VersionID: versionID, PreparerID: preparerID})
	if err != nil {
		return err
	}
	path := "/format-templates"
	if formSet {
		path = "/form-sets"
	}
	m := message{
		event:   EventFormApprovalWaiting,
		subject: fmt.Sprintf("%s version %d waits for your approval", name, version),
		body: fmt.Sprintf("%s approved %s version %d. It is published when the remaining form approvers approve it.\n%s versi %d menunggu persetujuan Anda.",
			actor.PersonName, name, version, name, version) + n.link(path),
	}
	return enqueue(ctx, q, actor, m, people)
}

func (n *Notifier) yokotenCreated(ctx context.Context, q *db.Queries, actor access.Actor, id string) error {
	y, err := q.YokotenForNotice(ctx, id)
	if err != nil {
		return err
	}
	var people []string
	for _, c := range y.CustomerIds {
		list, err := q.GroupLeadersOfCustomer(ctx, c)
		if err != nil {
			return err
		}
		people = append(people, list...)
	}
	ref := fmt.Sprintf("YKT-%04d", derefSeq(y.Seq))
	m := message{
		event:   EventYokotenCreated,
		subject: fmt.Sprintf("%s: decide for your models", ref),
		body: fmt.Sprintf("%s spreads %s to %d models. Decide Apply, Adapt, or Not applicable for the models of your customers.\n%s menyebarkan %s; putuskan untuk model customer Anda.\n\nReason: %s",
			ref, changeRef(y.SourceSeq), y.Targets, ref, changeRef(y.SourceSeq), y.Reason) + n.link("/yokoten?id="+y.ID),
	}
	return enqueue(ctx, q, actor, m, people)
}

func derefSeq(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func (n *Notifier) migrationWaiting(ctx context.Context, q *db.Queries, actor access.Actor, id string) error {
	d, err := q.DocumentForNotice(ctx, id)
	if err != nil {
		return err
	}
	people, err := q.PeopleWithPermission(ctx, db.PeopleWithPermissionParams{Permission: string(access.MigrateDocuments), CustomerID: &d.CustomerID})
	if err != nil {
		return err
	}
	m := message{
		event:   EventMigrationWaiting,
		subject: fmt.Sprintf("[%s] Legacy document %s waits for verification", d.CustomerCode, d.Number),
		body: fmt.Sprintf("%s entered the legacy document %s at revision %d. Another migration officer compares it with the signed hard copy and verifies it.\nDokumen lama %s menunggu verifikasi terhadap hardcopy.",
			actor.PersonName, d.Number, d.Revision, d.Number) + n.link("/documents/view?id="+d.ID),
	}
	return enqueue(ctx, q, actor, m, people)
}
