package audit

import (
	"context"
	"errors"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/problem"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// newActor membuat orang dan akun langsung lewat query, lalu memberi peran.
func newActor(t *testing.T, pool *pgxpool.Pool, login, kind string, roles ...string) access.Actor {
	t.Helper()
	ctx := t.Context()
	q := db.New(pool)
	person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: "Orang " + login})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{
		PersonID: person.ID, Kind: kind, Login: login, PasswordHash: "tidak-dipakai",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range roles {
		err := q.AddAccountRole(ctx, db.AddAccountRoleParams{AccountID: acc.ID, RoleID: role, Scope: access.ScopeAll})
		if err != nil {
			t.Fatal(err)
		}
	}
	return access.Actor{
		Type: access.ActorHuman, AccountID: acc.ID, AccountKind: kind, Login: login,
		PersonID: person.ID, PersonName: person.Name,
	}
}

// record menulis satu entri di transaksinya sendiri, seperti layanan domain.
func record(ctx context.Context, pool *pgxpool.Pool, actor access.Actor, e Event) error {
	return db.WithTx(ctx, pool, func(tx pgx.Tx) error {
		return Record(ctx, db.New(pool).WithTx(tx), actor, e)
	})
}

// chainOf membaca seluruh rantai dari database.
func chainOf(t *testing.T, pool *pgxpool.Pool) []entryFields {
	t.Helper()
	rows, err := db.New(pool).ListAuditChain(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	chain := make([]entryFields, 0, len(rows))
	for _, r := range rows {
		chain = append(chain, fieldsOf(r))
	}
	return chain
}

// TestRecord: setiap entri mencatat jenis aktor, akun dan orangnya, request ID,
// dan alamat IP (R-API-03), lalu tersambung ke entri sebelumnya.
func TestRecord(t *testing.T) {
	pool := dbtest.NewPool(t)
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	ctx := WithRequest(t.Context(), Request{ID: "UJI-REQ-1", IP: "192.0.2.10"})

	events := []struct {
		actor access.Actor
		event Event
	}{
		{admin, Event{Action: ActionCustomerCreated, TargetType: TargetCustomer,
			TargetID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489", Details: map[string]string{"code": "AB", "name": "Example Customer"}}},
		{access.Actor{Type: access.ActorSystem}, Event{Action: ActionAccountCreated, TargetType: TargetAccount,
			TargetID: admin.AccountID, Details: map[string]string{"login": "admin.uji", "via": "create-admin"}}},
		// Rincian kosong ditulis {} (bukan null).
		{access.Actor{Type: access.ActorAnonymous}, Event{Action: ActionLoginFailed, TargetType: TargetAccount,
			TargetID: admin.AccountID}},
	}
	for _, e := range events {
		if err := record(ctx, pool, e.actor, e.event); err != nil {
			t.Fatalf("Record %s: %v", e.event.Action, err)
		}
	}

	chain := chainOf(t, pool)
	if len(chain) != 3 {
		t.Fatalf("%d entri, ingin 3", len(chain))
	}
	first := chain[0]
	if first.Seq != "1" || first.PrevHash != genesisHash {
		t.Errorf("entri pertama: seq %s prev_hash %s", first.Seq, first.PrevHash)
	}
	if first.ActorType != "human" || first.ActorAccountID != admin.AccountID || first.ActorPersonID != admin.PersonID {
		t.Errorf("aktor entri pertama: %+v", first)
	}
	if first.RequestID != "UJI-REQ-1" || first.SourceIP != "192.0.2.10" {
		t.Errorf("request ID dan IP: %q %q", first.RequestID, first.SourceIP)
	}
	if first.Details != `{"code":"AB","name":"Example Customer"}` {
		t.Errorf("rincian: %s", first.Details)
	}
	if chain[1].ActorType != "system" || chain[1].ActorAccountID != "" || chain[1].ActorPersonID != "" {
		t.Errorf("aktor sistem: %+v", chain[1])
	}
	if chain[2].ActorType != "anonymous" || chain[2].Details != "{}" {
		t.Errorf("aktor anonim: %+v", chain[2])
	}
	if v := verifyChain(chain); !v.Valid || v.Entries != 3 {
		t.Errorf("rantai: %+v", v)
	}
}

// TestRecordRollsBackWithChange: entri audit ikut batal bila transaksi
// perubahan datanya batal, jadi log tidak pernah mencatat perubahan yang tidak terjadi.
func TestRecordRollsBackWithChange(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	failure := errors.New("perubahan gagal")
	err := db.WithTx(ctx, pool, func(tx pgx.Tx) error {
		if err := Record(ctx, db.New(pool).WithTx(tx), access.Actor{Type: access.ActorSystem},
			Event{Action: ActionCustomerCreated}); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("WithTx: %v", err)
	}
	if chain := chainOf(t, pool); len(chain) != 0 {
		t.Errorf("%d entri tersimpan, ingin 0", len(chain))
	}
}

// TestRecordConcurrent: 20 transaksi yang menulis bersamaan tetap menghasilkan
// satu rantai utuh dengan nomor 1 sampai 20 tanpa lubang (E-06).
func TestRecordConcurrent(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	const writers = 20

	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for range writers {
		wg.Go(func() {
			errs <- record(ctx, pool, access.Actor{Type: access.ActorSystem}, Event{Action: ActionCustomerCreated})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("Record paralel: %v", err)
		}
	}

	chain := chainOf(t, pool)
	var seqs []string
	for _, f := range chain {
		seqs = append(seqs, f.Seq)
	}
	var want []string
	for i := int64(1); i <= writers; i++ {
		want = append(want, seqText(i))
	}
	if !slices.Equal(seqs, want) {
		t.Errorf("nomor urut %v, ingin 1 sampai %d", seqs, writers)
	}
	if v := verifyChain(chain); !v.Valid {
		t.Errorf("rantai rusak: %+v", v)
	}
}

// TestVerifyDetectsTampering meniru admin database yang mematikan trigger
// penolak lalu mengubah isi satu entri. Pemeriksaan rantai menemukannya (R-SEC-03).
func TestVerifyDetectsTampering(t *testing.T) {
	app, owner := dbtest.NewPools(t)
	ctx := t.Context()
	admin := newActor(t, app, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	s := NewService(app, access.NewChecker(app))
	for i := range 5 {
		err := record(ctx, app, admin, Event{Action: ActionCustomerCreated, Details: map[string]string{"nomor": seqText(int64(i + 1))}})
		if err != nil {
			t.Fatal(err)
		}
	}
	before, err := s.Verify(ctx, admin)
	if err != nil || !before.Valid || before.Entries != 5 {
		t.Fatalf("sebelum diubah: %+v, %v", before, err)
	}

	tamper := []string{
		"ALTER TABLE audit_log DISABLE TRIGGER audit_log_append_only",
		`UPDATE audit_log SET details = '{"nomor":"palsu"}' WHERE seq = 3`,
		"ALTER TABLE audit_log ENABLE TRIGGER audit_log_append_only",
	}
	for _, sql := range tamper {
		if _, err := owner.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	after, err := s.Verify(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	if after.Valid || after.BrokenSeq != 3 || after.Reason != ReasonHashMismatch {
		t.Errorf("setelah diubah: %+v, ingin rusak di entri 3 (%s)", after, ReasonHashMismatch)
	}
}

// TestVerifyNeedsExternalAnchor menunjukkan batas rantai tanpa jangkar luar:
// entri terakhir yang dihapus tidak memutus rantai. Yang menemukannya adalah
// hash terakhir yang dicatat di luar database (usulan R-SEC-03, belum dibangun).
func TestVerifyNeedsExternalAnchor(t *testing.T) {
	app, owner := dbtest.NewPools(t)
	ctx := t.Context()
	admin := newActor(t, app, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	s := NewService(app, access.NewChecker(app))
	for range 3 {
		if err := record(ctx, app, admin, Event{Action: ActionCustomerCreated}); err != nil {
			t.Fatal(err)
		}
	}
	noted, err := s.Verify(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}

	for _, sql := range []string{
		"ALTER TABLE audit_log DISABLE TRIGGER audit_log_append_only",
		"DELETE FROM audit_log WHERE seq = 3",
		"ALTER TABLE audit_log ENABLE TRIGGER audit_log_append_only",
	} {
		if _, err := owner.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	after, err := s.Verify(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Valid {
		t.Errorf("rantai tanpa entri terakhir dilaporkan rusak: %+v", after)
	}
	if after.LastSeq == noted.LastSeq || after.LastHash == noted.LastHash {
		t.Errorf("hash terakhir tidak berubah setelah entri terakhir dihapus: %+v", after)
	}
}

// TestServiceRequiresViewAuditLog: hanya pemegang A-4 (akun admin, D-75) yang
// bisa membaca dan memeriksa audit log.
func TestServiceRequiresViewAuditLog(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	s := NewService(pool, access.NewChecker(pool))
	worker := newActor(t, pool, "penyusun.uji", access.AccountWork, access.RoleAuthor, access.RoleDocumentControl)
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)

	if _, err := s.List(ctx, worker, 0, 10); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("List oleh akun kerja: %v, ingin ErrForbidden", err)
	}
	if _, err := s.Verify(ctx, worker); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Verify oleh akun kerja: %v, ingin ErrForbidden", err)
	}
	if _, err := s.List(ctx, admin, 0, 10); err != nil {
		t.Errorf("List oleh admin: %v", err)
	}
}

// TestList: entri terbaru dulu, dibaca per halaman dengan before, beserta login
// dan nama orang aktor (D-74) serta nama tampilan sasaran.
func TestList(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := WithRequest(t.Context(), Request{ID: "UJI-REQ-2", IP: "192.0.2.20"})
	s := NewService(pool, access.NewChecker(pool))
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	worker := newActor(t, pool, "penyusun.uji", access.AccountWork, access.RoleAuthor)

	for range 4 {
		if err := record(ctx, pool, access.Actor{Type: access.ActorSystem}, Event{Action: ActionLogout}); err != nil {
			t.Fatal(err)
		}
	}
	err := record(ctx, pool, admin, Event{Action: ActionAccountDisabled, TargetType: TargetAccount, TargetID: worker.AccountID})
	if err != nil {
		t.Fatal(err)
	}

	page1, err := s.List(ctx, admin, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 || page1[0].Seq != 5 || page1[1].Seq != 4 {
		t.Fatalf("halaman pertama: %+v", page1)
	}
	newest := page1[0]
	if newest.ActorLogin != "admin.uji" || newest.ActorPersonName != "Orang admin.uji" || newest.ActorType != "human" {
		t.Errorf("aktor: %+v", newest)
	}
	if newest.TargetLabel != "penyusun.uji" || newest.RequestID != "UJI-REQ-2" || newest.SourceIP != "192.0.2.20" {
		t.Errorf("sasaran dan asal permintaan: %+v", newest)
	}
	if newest.Details == nil || len(newest.Details) != 0 {
		t.Errorf("rincian kosong seharusnya peta kosong: %#v", newest.Details)
	}

	page2, err := s.List(ctx, admin, page1[1].Seq, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 3 || page2[0].Seq != 3 || page2[2].Seq != 1 {
		t.Errorf("halaman kedua: %+v", page2)
	}

	// Batas halaman di kontrak: limit 1 sampai 200, before tidak negatif.
	for _, tt := range []struct {
		before int64
		limit  int
		field  string
	}{{0, 0, "limit"}, {0, 201, "limit"}, {-1, 10, "before"}} {
		_, err := s.List(ctx, admin, tt.before, tt.limit)
		var invalid *problem.ValidationError
		if !errors.As(err, &invalid) || invalid.Problems[0].Field != tt.field {
			t.Errorf("before %d limit %d: %v, ingin penolakan field %s", tt.before, tt.limit, err, tt.field)
		}
	}
}
