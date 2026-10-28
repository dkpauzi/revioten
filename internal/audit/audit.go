// Package audit menulis dan membaca audit log tahan manipulasi (M1.3,
// R-SEC-03, E-06).
//
// Setiap perubahan data menulis satu entri lewat Record, di transaksi yang sama
// dengan perubahannya. Data dan catatannya selalu tersimpan bersama atau batal
// bersama.
//
// ASUMSI E-28: yang dicatat adalah perubahan data, termasuk sesi (login dan
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
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
)

// Kode peristiwa. Teks layarnya ada di katalog teks web.
const (
	ActionLogin                = "account.login"
	ActionLoginFailed          = "account.login_failed"
	ActionLogout               = "account.logout"
	ActionPasswordChanged      = "account.password_changed"
	ActionPasswordChangeFailed = "account.password_change_failed"
	ActionAccountCreated       = "account.created"
	ActionRolesChanged         = "account.roles_changed"
	ActionPasswordReset        = "account.password_reset"
	ActionAccountDisabled      = "account.disabled"
	ActionAccountEnabled       = "account.enabled"
	ActionCustomerCreated      = "customer.created"
	ActionCustomerUpdated      = "customer.updated"
	ActionCustomerCodeChanged  = "customer.code_changed"
	ActionProjectGroupCreated  = "project_group.created"
	ActionProjectGroupUpdated  = "project_group.updated"
	ActionModelGroupCreated    = "model_group.created"
	ActionModelGroupUpdated    = "model_group.updated"
	ActionModelCreated         = "model.created"
	ActionModelUpdated         = "model.updated"
	ActionPartNumberCreated    = "part_number.created"
	ActionPartNumberUpdated    = "part_number.updated"
)

// Jenis sasaran entri.
const (
	TargetAccount      = "account"
	TargetCustomer     = "customer"
	TargetProjectGroup = "project_group"
	TargetModelGroup   = "model_group"
	TargetModel        = "model"
	TargetPartNumber   = "part_number"
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
	return nil
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
