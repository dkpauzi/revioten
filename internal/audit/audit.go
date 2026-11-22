// Package audit menulis dan membaca audit log tahan manipulasi (M1.3,
// R-SEC-03, E-06).
//
// Setiap perubahan data menulis satu entri lewat Record, di transaksi yang sama
// dengan perubahannya. Data dan catatannya selalu tersimpan bersama atau batal
// bersama.
//
// E-28 (D-107): yang dicatat adalah perubahan data, termasuk sesi (login dan
// logout) dan hitungan gagal login. Membaca data tidak dicatat, dan
// pembersihan sesi yang sudah habis waktunya juga tidak. Entri membentuk satu rantai hash untuk seluruh sistem: hash setiap
// entri dihitung dari isinya dan hash entri sebelumnya (hash.go).
//
// Database menolak UPDATE, DELETE, dan TRUNCATE di tabel audit_log (role
// aplikasi tanpa hak itu, ditambah trigger penolak). Admin database yang
// mematikan trigger tetap bisa mengubah isi tabel, tapi rantainya putus, dan
// Verify menemukannya.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
)

// Kode peristiwa. Teks layarnya ada di katalog teks web.
const (
	ActionLogin                  = "account.login"
	ActionLoginFailed            = "account.login_failed"
	ActionLogout                 = "account.logout"
	ActionPasswordChanged        = "account.password_changed"
	ActionPasswordChangeFailed   = "account.password_change_failed"
	ActionPasswordConfirmFailed  = "account.password_confirm_failed" // D-198: password salah saat menandatangani
	ActionAccountCreated         = "account.created"
	ActionRolesChanged           = "account.roles_changed"
	ActionRolePermissionsChanged = "role.permissions_changed" // M8.1
	ActionRoleCreated            = "role.created"             // D-199
	ActionRoleUpdated            = "role.updated"             // D-199: nama atau cakupan
	ActionRoleDeactivated        = "role.deactivated"         // D-199
	ActionRoleActivated          = "role.activated"           // D-199
	// Fase 3: tanda tangan digital (D-198).
	ActionSignatureSent      = "signature.sent"
	ActionSignatureSigned    = "signature.signed"
	ActionSignatureCompleted = "signature.completed"
	ActionSignatureRejected  = "signature.rejected"
	ActionSignatureWithdrawn = "signature.withdrawn"
	ActionSignatureModeSet   = "customer.signature_mode_changed"
	// D-207: Form Control Submission.
	ActionSubmissionChanged        = "document.submission_changed"
	ActionSubmissionScanUploaded   = "document.submission_scan_uploaded"
	ActionSubmissionDefaultChanged = "customer.submission_default_changed"
	// Tahap 9: API integrasi.
	ActionServiceAccountCreated  = "service_account.created"
	ActionServiceAccountDisabled = "service_account.disabled"
	ActionServiceTokenIssued     = "service_account.token_issued"
	ActionServiceTokenRevoked    = "service_account.token_revoked"
	ActionServiceTokenRefused    = "service_account.token_refused"
	ActionChangeProposed         = "change.proposed"
	ActionChangeProposalApplied  = "change.proposal_applied"
	ActionExportRequested        = "export.requested"
	// D-194: email notifikasi.
	ActionNotificationEventChanged = "notification.event_changed"
	ActionNotificationTestSent     = "notification.test_sent"
	ActionPersonEmailChanged       = "account.email_changed"
	ActionPasswordReset            = "account.password_reset"
	ActionAccountDisabled          = "account.disabled"
	ActionAccountEnabled           = "account.enabled"
	ActionCustomerCreated          = "customer.created"
	ActionCustomerUpdated          = "customer.updated"
	ActionCustomerCodeChanged      = "customer.code_changed"
	ActionProjectGroupCreated      = "project_group.created"
	ActionProjectGroupUpdated      = "project_group.updated"
	ActionModelGroupCreated        = "model_group.created"
	ActionModelGroupUpdated        = "model_group.updated"
	ActionModelCreated             = "model.created"
	ActionModelUpdated             = "model.updated"
	ActionPartNumberCreated        = "part_number.created"
	ActionPartNumberUpdated        = "part_number.updated"
	ActionProcessCreated           = "process.created"
	ActionProcessUpdated           = "process.updated"
	ActionNumberSettingsUpdated    = "number_settings.updated"
	ActionDocumentTypeCreated      = "document_type.created"
	ActionDocumentTypeUpdated      = "document_type.updated"
	ActionSpecialCodeCreated       = "special_code.created"
	ActionSpecialCodeUpdated       = "special_code.updated"
	ActionChangeTypeUpdated        = "change_type.updated"
	ActionChangeSourceCreated      = "change_source.created"
	ActionChangeSourceUpdated      = "change_source.updated"
	ActionFourMCategoryUpdated     = "four_m_category.updated"
	ActionPositionUpdated          = "position.updated"
	ActionPositionHolderAdded      = "position.holder_added"
	ActionPositionHolderRemoved    = "position.holder_removed"
	ActionApprovalRouteCreated     = "approval_route.created"
	ActionApprovalRouteUpdated     = "approval_route.updated"
	ActionCustomerRouteChanged     = "customer.default_route_changed"
	// Form set (langkah D, D-133, D-139).
	ActionCustomerFormSetChanged  = "customer.default_form_set_changed"
	ActionFormSetCreated          = "form_set.created"
	ActionFormSetChanged          = "form_set.changed"
	ActionFormSetVersionCreated   = "form_set.version_created"
	ActionFormSetVersionPublished = "form_set.version_published"
	ActionFormSetVersionApproved  = "form_set.version_approved"
	ActionFormatVersionApproved   = "format_template.version_approved"
	// Dokumen (M4.1).
	ActionDocumentCreated             = "document.created"
	ActionDocumentHeaderChanged       = "document.header_changed"
	ActionDocumentRowAdded            = "document.row_added"
	ActionDocumentRowsPasted          = "document.rows_pasted"           // M4.4
	ActionDocumentRowDetached         = "document.row_detached"          // M4.8
	ActionPFMEACPChecked              = "document.pfmea_cp_checked"      // M4.9
	ActionPFMEACPUnchecked            = "document.pfmea_cp_unchecked"    // M4.9
	ActionDocumentSectionAdded        = "document.section_added"         // D-155
	ActionDocumentSectionChanged      = "document.section_changed"       // D-155
	ActionDocumentSectionRemoved      = "document.section_removed"       // D-155
	ActionChangeCreated               = "change.created"                 // M5.1
	ActionChangeDocumentAdded         = "change.document_added"          // M5.1
	ActionChangeDocumentRemoved       = "change.document_removed"        // M5.1
	ActionChangeAttachmentAdded       = "change.attachment_added"        // M5.2
	ActionChangeAttachmentRemoved     = "change.attachment_removed"      // M5.2
	ActionChangeAccepted              = "change.accepted"                // M5.3
	ActionChangeRejected              = "change.rejected"                // M5.3
	ActionDocumentDraftOpened         = "document.draft_opened"          // M5.4
	ActionDocumentReleased            = "document.released"              // M5.5
	ActionDocumentMigrated            = "document.migrated"              // M7.1
	ActionDocumentHistoryChanged      = "document.history_changed"       // M7.2
	ActionDocumentRowsLinked          = "document.rows_linked"           // M7.4
	ActionDocumentMigrationVerified   = "document.migration_verified"    // M7.5
	ActionChangeImpactsSaved          = "change.impacts_saved"           // M5.7
	ActionDocumentTemplateModeChanged = "document.template_mode_changed" // M5.8
	ActionChangeTemplateApplied       = "change.template_applied"        // M5.8
	ActionYokotenCreated              = "yokoten.created"                // M5.9
	ActionYokotenDecided              = "yokoten.decided"                // M5.9
	ActionDocumentPrinted             = "document.printed"               // M6.6
	ActionDocumentScanUploaded        = "document.scan_uploaded"         // M6.7
	ActionPrintLayoutChanged          = "print_layout.changed"           // M6.1
	ActionCustomerLayoutChanged       = "customer.layout_changed"        // M6.1
	ActionDocumentCopied              = "document.copied"                // M4.7
	ActionDocumentComposed            = "document.composed"              // M4.5
	ActionRefDocCreated               = "referenced_document.created"    // D-130
	ActionRefDocUpdated               = "referenced_document.updated"    // D-130
	ActionRefDocLinked                = "referenced_document.linked"     // D-130
	ActionRefDocUnlinked              = "referenced_document.unlinked"   // D-130
	ActionDocumentRefsFilled          = "document.references_filled"     // D-130
	ActionDocumentRowsReordered       = "document.rows_reordered"        // M4.6
	// Review berkala tanpa perubahan (D-196). Sasarannya catatan review.
	ActionPeriodicReviewRecorded  = "periodic_review.recorded"
	ActionPeriodicReviewConfirmed = "periodic_review.confirmed"
	// Fase 1 desain (D-200, D-202).
	ActionModelEOLChanged        = "model.eol_changed"
	ActionDocumentMarkedReady    = "document.marked_ready"
	ActionDocumentRowChanged     = "document.row_changed"
	ActionDocumentRowRemoved     = "document.row_removed"
	ActionChangeTypeRouteChanged = "change_type.route_changed"
	ActionSystemSettingsUpdated  = "system_settings.updated"
	// Register nomor dokumen (Tahap 2).
	ActionDocumentNumberRegistered = "document_number.registered"
	ActionDocumentNumberImported   = "document_number.imported"
	ActionDocumentNumberVoided     = "document_number.voided"
	ActionDocumentNumberObsoleted  = "document_number.obsoleted"
	// Format template (Tahap 3). Sasarannya selalu template; versinya ada di rincian.
	ActionFormatTemplateCreated = "format_template.created"
	ActionFormatTemplateRenamed = "format_template.renamed"
	// D-218: format template dan form set disembunyikan atau ditampilkan.
	ActionFormatTemplateHiddenChanged = "format_template.hidden_changed"
	ActionFormSetHiddenChanged        = "form_set.hidden_changed"
	ActionFormatSettingsChanged       = "format_template.settings_changed"
	ActionFormatFieldAdded            = "format_template.field_added"
	ActionFormatFieldChanged          = "format_template.field_changed"
	ActionFormatRatingsChanged        = "format_template.ratings_changed"
	ActionFormatSymbolsChanged        = "format_template.symbols_changed"
	// M3.4: versi terbit dan versi baru (R-TPL-05).
	ActionFormatVersionPublished = "format_template.version_published"
	ActionFormatVersionCreated   = "format_template.version_created"
	// Tabel AP (form set langkah A, D-134).
	ActionFormatAPRulesChanged = "format_template.ap_rules_changed"
	// Process template (M3.2). Sasarannya selalu template; versinya ada di rincian.
	ActionProcessTemplateCreated = "process_template.created"
	ActionSubprocessAdded        = "process_template.subprocess_added"
	ActionSubprocessChanged      = "process_template.subprocess_changed"
	ActionTemplatePointAdded     = "process_template.point_added"
	ActionTemplatePointChanged   = "process_template.point_changed"
	// Varian poin untuk cakupan yang lebih sempit (M3.3, D-129).
	ActionTemplateVariantAdded   = "process_template.variant_added"
	ActionTemplateVariantChanged = "process_template.variant_changed"
	// M3.4: versi terbit dan versi baru (R-TPL-05).
	ActionProcessVersionPublished = "process_template.version_published"
	ActionProcessVersionCreated   = "process_template.version_created"
)

// Jenis sasaran entri.
const (
	TargetAccount         = "account"
	TargetCustomer        = "customer"
	TargetProjectGroup    = "project_group"
	TargetModelGroup      = "model_group"
	TargetModel           = "model"
	TargetPartNumber      = "part_number"
	TargetProcess         = "process"
	TargetNumberSettings  = "number_settings"
	TargetDocumentType    = "document_type"
	TargetSpecialCode     = "special_code"
	TargetChangeType      = "change_type"
	TargetChangeSource    = "change_source"
	TargetFourMCategory   = "four_m_category"
	TargetPosition        = "position"
	TargetApprovalRoute   = "approval_route"
	TargetSystemSettings  = "system_settings"
	TargetDocumentNumber  = "document_number"
	TargetFormatTemplate  = "format_template"
	TargetProcessTemplate = "process_template"
	TargetFormSet         = "form_set"
	TargetReferencedDoc   = "referenced_document" // D-130
	TargetChange          = "change"              // M5.1
	TargetDocument        = "document"
)

// Change menulis perubahan satu field untuk rincian entri: "lama → baru".
// Nilai kosong ditulis "-".
func Change(before, after string) string {
	dash := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	return dash(before) + " → " + dash(after)
}

// Event adalah satu perubahan data yang dicatat.
type Event struct {
	Action     string
	TargetType string // kosong bila tidak ada sasaran
	TargetID   string
	// Details tidak pernah berisi password, token, atau kunci (E-18).
	Details map[string]string
}

// Request adalah asal sebuah permintaan HTTP: request ID dan alamat IP
// pengirim (R-API-03). Server menaruhnya di context; Record membacanya.
type Request struct {
	ID string
	IP string
}

type requestKey struct{}

// WithRequest menaruh asal permintaan di context.
func WithRequest(ctx context.Context, r Request) context.Context {
	return context.WithValue(ctx, requestKey{}, r)
}

// RequestFrom mengambil asal permintaan dari context. Perintah CLI tidak punya
// permintaan HTTP, jadi hasilnya kosong.
func RequestFrom(ctx context.Context) Request {
	r, _ := ctx.Value(requestKey{}).(Request)
	return r
}

// chainLock adalah nomor kunci advisory untuk ujung rantai. Nomornya bebas,
// asal tidak dipakai kunci lain (create-admin memakai 11001).
const chainLock int64 = 11002

// now bisa diganti di test.
var now = time.Now

// Record menulis satu entri audit. q harus terikat ke transaksi yang membuat
// perubahan datanya (q.WithTx), dan Record dipanggil sebagai langkah terakhir
// transaksi itu: kunci rantai dipegang sampai COMMIT, jadi transaksi lain yang
// ingin menulis audit menunggu sesingkat mungkin.
func Record(ctx context.Context, q *db.Queries, actor access.Actor, e Event) error {
	// Satu rantai untuk seluruh sistem (E-06): penulis audit antre di sini,
	// supaya tidak ada dua entri yang menyambung ke entri sebelumnya yang sama.
	if err := q.LockAuditChain(ctx, chainLock); err != nil {
		return fmt.Errorf("mengunci rantai audit: %w", err)
	}
	prevSeq, prevHash := int64(0), genesisHash
	head, err := q.GetAuditChainHead(ctx)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// Entri pertama: menyambung ke genesisHash.
	case err != nil:
		return fmt.Errorf("membaca ujung rantai audit: %w", err)
	default:
		prevSeq, prevHash = head.Seq, head.Hash
	}

	details := e.Details
	if details == nil {
		details = map[string]string{}
	}
	// json.Marshal mengurutkan kunci map, jadi teks yang sama selalu dihasilkan.
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("menyusun rincian audit: %w", err)
	}

	// Hanya aktor manusia yang punya akun dan orang.
	var accountID, personID string
	if actor.Type == access.ActorHuman {
		accountID, personID = actor.AccountID, actor.PersonID
	}
	req := RequestFrom(ctx)
	// Ketelitian disamakan dengan timestamptz (mikrodetik), supaya waktu yang
	// dibaca ulang dari database memberi teks yang sama saat diperiksa.
	at := now().UTC().Truncate(time.Microsecond)
	seq := prevSeq + 1

	f := entryFields{
		PrevHash: prevHash, Seq: seqText(seq), OccurredAt: timeText(at),
		ActorType: string(actor.Type), ActorAccountID: accountID, ActorPersonID: personID,
		RequestID: req.ID, SourceIP: req.IP,
		Action: e.Action, TargetType: e.TargetType, TargetID: e.TargetID,
		Details: string(detailsJSON),
	}
	err = q.InsertAuditEntry(ctx, db.InsertAuditEntryParams{
		Seq: seq, OccurredAt: at, ActorType: f.ActorType,
		ActorAccountID: nullable(f.ActorAccountID), ActorPersonID: nullable(f.ActorPersonID),
		RequestID: nullable(f.RequestID), SourceIp: nullable(f.SourceIP),
		Action: f.Action, TargetType: nullable(f.TargetType), TargetID: nullable(f.TargetID),
		Details: f.Details, PrevHash: f.PrevHash, Hash: f.hash(),
	})
	if err != nil {
		return fmt.Errorf("menulis entri audit %s: %w", e.Action, err)
	}
	// Pendengar (misalnya notifikasi email, D-194) berjalan di transaksi yang
	// sama, jadi email tidak pernah tertulis untuk aksi yang batal.
	if l := listener.Load(); l != nil {
		if err := (*l)(ctx, q, actor, e); err != nil {
			return fmt.Errorf("pendengar audit %s: %w", e.Action, err)
		}
	}
	return nil
}

// Listener dipanggil sesudah setiap entri audit tertulis, dengan queries
// transaksi yang sama.
type Listener func(ctx context.Context, q *db.Queries, actor access.Actor, e Event) error

var listener atomic.Pointer[Listener]

// SetListener memasang pendengar untuk seluruh program; nil melepasnya.
// Dipasang sekali saat server mulai (notifikasi email, D-194).
func SetListener(l Listener) {
	if l == nil {
		listener.Store(nil)
		return
	}
	listener.Store(&l)
}

// nullable mengubah teks kosong menjadi NULL di database.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// text mengubah NULL dari database menjadi teks kosong.
func text(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
