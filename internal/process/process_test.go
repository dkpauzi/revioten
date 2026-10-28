package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/problem"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

// newActor membuat orang dan akun langsung lewat query, dengan peran tertentu.
func newActor(t *testing.T, pool *pgxpool.Pool, login, kind string, roles ...string) access.Actor {
	t.Helper()
	ctx := t.Context()
	q := db.New(pool)
	person, err := q.CreatePerson(ctx, db.CreatePersonParams{Name: "Orang " + login})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := q.CreateAccount(ctx, db.CreateAccountParams{PersonID: person.ID, Kind: kind, Login: login, PasswordHash: "x"})
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range roles {
		if err := q.AddAccountRole(ctx, db.AddAccountRoleParams{AccountID: acc.ID, RoleID: role, Scope: access.ScopeAll}); err != nil {
			t.Fatal(err)
		}
	}
	return access.Actor{Type: access.ActorHuman, AccountID: acc.ID, AccountKind: kind, Login: login, PersonID: person.ID}
}

// fixture: database baru dan akun PPAP Engineer, pemegang peran Pengelola
// template (E-6, D-64). owner adalah pool pemilik database, untuk test yang
// meniru tabel lain yang menunjuk proses.
type fixture struct {
	s        *Service
	pool     *pgxpool.Pool
	owner    *pgxpool.Pool
	engineer access.Actor
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	pool, owner := dbtest.NewPools(t)
	engineer := newActor(t, pool, "ppap.engineer.uji", access.AccountWork, access.RoleAuthor, access.RoleTemplateMaintainer)
	return fixture{s: NewService(pool, access.NewChecker(pool)), pool: pool, owner: owner, engineer: engineer}
}

// problemCodes mengambil "field:kode" dari penolakan input.
func problemCodes(t *testing.T, err error) []string {
	t.Helper()
	var problems []problem.Problem
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case errors.As(err, &invalid):
		problems = invalid.Problems
	case errors.As(err, &conflict):
		problems = conflict.Problems
	default:
		t.Fatalf("ingin penolakan input, dapat %v", err)
	}
	var codes []string
	for _, p := range problems {
		codes = append(codes, p.Field+":"+p.Code)
	}
	return codes
}

// second mengambil error dari hasil dua nilai, supaya tabel test tetap ringkas.
func second[T any](_ T, err error) error { return err }

// auditOf mengambil semua entri audit untuk satu sasaran, urut dari yang terlama.
func auditOf(t *testing.T, pool *pgxpool.Pool, targetID string) []db.AuditLog {
	t.Helper()
	rows, err := db.New(pool).ListAuditChain(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var list []db.AuditLog
	for _, r := range rows {
		if r.TargetID != nil && *r.TargetID == targetID {
			list = append(list, r)
		}
	}
	return list
}

// TestRenameKeepsReferences: kriteria selesai M1.5. Nama proses bisa diganti
// tanpa memutus rujukan, karena semua rujukan memakai ID tetap (D-06).
func TestRenameKeepsReferences(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	created, err := f.s.Create(ctx, f.engineer, Input{Name: " Parts  Receiving ", Aliases: []string{"Part Receiving"}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Name != "Parts Receiving" || !slices.Equal(created.Aliases, []string{"Part Receiving"}) {
		t.Errorf("proses baru: %+v", created)
	}

	// Tabel dokumen belum ada (Tahap 3). Tabel ini menirunya: barisnya
	// menunjuk proses lewat ID dengan foreign key, seperti document_row (R-DOC-02).
	_, err = f.owner.Exec(ctx, `
		CREATE TABLE test_rows (id int PRIMARY KEY, process_id uuid NOT NULL REFERENCES processes (id));
		INSERT INTO test_rows VALUES (1, '`+created.ID+`')`)
	if err != nil {
		t.Fatal(err)
	}

	renamed, err := f.s.Update(ctx, f.engineer, created.ID, Input{Name: "Receiving", Aliases: created.Aliases})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if renamed.ID != created.ID || renamed.Name != "Receiving" || !slices.Equal(renamed.Aliases, []string{"Part Receiving"}) {
		t.Errorf("setelah ganti nama: %+v", renamed)
	}

	// Rujukan 1: baris yang menunjuk ID kini menampilkan nama baru.
	var shown string
	err = f.owner.QueryRow(ctx, `
		SELECT n.name FROM test_rows r
		JOIN process_names n ON n.process_id = r.process_id AND n.kind = 'display'
		WHERE r.id = 1`).Scan(&shown)
	if err != nil || shown != "Receiving" {
		t.Errorf("baris yang menunjuk proses: %q, %v", shown, err)
	}

	// Rujukan 2: pemanggil yang menyimpan ID mendapat proses yang sama.
	got, err := f.s.Get(ctx, f.engineer, created.ID)
	if err != nil || got.Name != "Receiving" {
		t.Errorf("Get dengan ID lama: %+v, %v", got, err)
	}

	// Rujukan 3: audit log menunjuk ID, jadi entri lama ikut menampilkan nama baru.
	admin := newActor(t, f.pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	entries, err := audit.NewService(f.pool, access.NewChecker(f.pool)).List(ctx, admin, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	var labels []string
	for _, e := range entries {
		if e.TargetID == created.ID {
			labels = append(labels, e.Action+" "+e.TargetLabel)
		}
	}
	if want := []string{"process.updated Receiving", "process.created Receiving"}; !slices.Equal(labels, want) {
		t.Errorf("audit log: %v, ingin %v", labels, want)
	}

	// Proses yang dirujuk tidak bisa dihapus, juga oleh pemilik database.
	_, err = f.owner.Exec(ctx, `DELETE FROM processes WHERE id = $1`, created.ID)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Errorf("hapus proses yang dirujuk: %v, ingin foreign_key_violation", err)
	}
}

// TestNamesAndAliasesUnique: nama tampilan dan alias berbagi satu ruang nama,
// tanpa membedakan huruf besar kecil dan spasi ganda (ASUMSI E-32).
func TestNamesAndAliasesUnique(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	iqc, err := f.s.Create(ctx, f.engineer, Input{Name: "IQC", Aliases: []string{"Incoming QC"}})
	if err != nil {
		t.Fatal(err)
	}
	smt, err := f.s.Create(ctx, f.engineer, Input{Name: "SMT"})
	if err != nil {
		t.Fatal(err)
	}
	if smt.Aliases == nil || len(smt.Aliases) != 0 {
		t.Errorf("proses tanpa alias: %#v, ingin daftar kosong", smt.Aliases)
	}

	tests := []struct {
		name string
		err  error
		want []string
	}{
		{"nama ganda beda huruf", second(f.s.Create(ctx, f.engineer, Input{Name: "iqc"})), []string{"name:taken"}},
		{"nama sama dengan alias proses lain", second(f.s.Create(ctx, f.engineer, Input{Name: "Incoming  qc"})), []string{"name:taken"}},
		{"alias sama dengan nama proses lain", second(f.s.Create(ctx, f.engineer, Input{Name: "Reflow", Aliases: []string{"smt"}})), []string{"aliases:taken"}},
		{"alias ditulis dua kali", second(f.s.Create(ctx, f.engineer, Input{Name: "Reflow", Aliases: []string{"Reflow Oven", "reflow oven"}})), []string{"aliases:taken"}},
		{"alias sama dengan namanya sendiri", second(f.s.Create(ctx, f.engineer, Input{Name: "Reflow", Aliases: []string{"REFLOW"}})), []string{"aliases:taken"}},
		{"ganti nama ke alias proses lain", second(f.s.Update(ctx, f.engineer, smt.ID, Input{Name: "Incoming QC"})), []string{"name:taken"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, tt.err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}

	// Penolakan membatalkan seluruh transaksi: tidak ada proses Reflow yang tersisa.
	list, err := f.s.List(ctx, f.engineer)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range list {
		names = append(names, p.Name+"/"+strings.Join(p.Aliases, ","))
	}
	if !slices.Equal(names, []string{"IQC/Incoming QC", "SMT/"}) {
		t.Errorf("List: %v", names)
	}

	// Nama dan alias boleh bertukar dalam satu perubahan: alias lama dilepas
	// dulu, baru nama tampilan diganti.
	swapped, err := f.s.Update(ctx, f.engineer, iqc.ID, Input{Name: "Incoming QC", Aliases: []string{"IQC"}})
	if err != nil || swapped.Name != "Incoming QC" || !slices.Equal(swapped.Aliases, []string{"IQC"}) {
		t.Errorf("tukar nama dan alias: %+v, %v", swapped, err)
	}
	// Mengganti huruf besar kecil nama sendiri bukan bentrok.
	if lower, err := f.s.Update(ctx, f.engineer, smt.ID, Input{Name: "smt"}); err != nil || lower.Name != "smt" {
		t.Errorf("ganti huruf nama sendiri: %+v, %v", lower, err)
	}
}

// TestCreateConcurrent: pendaftaran paralel dengan nama yang sama menghasilkan
// tepat satu proses. Yang lain mendapat name:taken, bukan error server.
func TestCreateConcurrent(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	const writers = 10
	errs := make([]error, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Go(func() {
			_, errs[i] = f.s.Create(ctx, f.engineer, Input{Name: "AOI"})
		})
	}
	wg.Wait()

	created := 0
	for _, err := range errs {
		if err == nil {
			created++
			continue
		}
		if got := problemCodes(t, err); !slices.Equal(got, []string{"name:taken"}) {
			t.Errorf("penolakan: %v", got)
		}
	}
	list, err := f.s.List(ctx, f.engineer)
	if created != 1 || err != nil || len(list) != 1 {
		t.Errorf("%d berhasil dan %d proses tersimpan (%v), ingin 1 dan 1", created, len(list), err)
	}
}

// TestUpdateConcurrent: perubahan paralel pada proses yang sama berjalan
// bergantian, karena Update mengunci nama tampilan proses dulu. Buktinya:
// setiap entri audit dimulai dari hasil entri sebelumnya, dan alias akhir
// tepat milik satu perubahan, bukan gabungan beberapa perubahan.
func TestUpdateConcurrent(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	p, err := f.s.Create(ctx, f.engineer, Input{Name: "IQC"})
	if err != nil {
		t.Fatal(err)
	}
	const writers = 10
	errs := make([]error, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Go(func() {
			_, errs[i] = f.s.Update(ctx, f.engineer, p.ID, Input{Name: "IQC", Aliases: []string{fmt.Sprintf("Alias %02d", i)}})
		})
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("perubahan %d: %v", i, err)
		}
	}

	entries := auditOf(t, f.pool, p.ID)
	if len(entries) != writers+1 {
		t.Fatalf("%d entri audit, ingin %d", len(entries), writers+1)
	}
	previous := "-"
	for _, r := range entries[1:] {
		var details map[string]string
		if err := json.Unmarshal([]byte(r.Details), &details); err != nil {
			t.Fatal(err)
		}
		before, after, _ := strings.Cut(details["aliases"], " → ")
		if before != previous {
			t.Errorf("entri %d mulai dari %q, padahal entri sebelumnya berakhir di %q", r.Seq, before, previous)
		}
		previous = after
	}
	final, err := f.s.Get(ctx, f.engineer, p.ID)
	if err != nil || strings.Join(final.Aliases, ", ") != previous {
		t.Errorf("alias akhir %v, ingin %q dari entri terakhir (%v)", final.Aliases, previous, err)
	}
}

// TestValidation: aturan isian nama dan alias (ASUMSI E-32).
func TestValidation(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	p, err := f.s.Create(ctx, f.engineer, Input{Name: "IQC"})
	if err != nil {
		t.Fatal(err)
	}
	tooMany := make([]string, maxAliases+1)
	for i := range tooMany {
		tooMany[i] = "Alias " + string(rune('A'+i))
	}

	tests := []struct {
		name string
		err  error
		want []string
	}{
		{"nama kosong", second(f.s.Create(ctx, f.engineer, Input{Name: "  "})), []string{"name:required"}},
		{"nama terlalu panjang", second(f.s.Create(ctx, f.engineer, Input{Name: strings.Repeat("a", 201)})), []string{"name:too_long"}},
		{"nama dua baris", second(f.s.Create(ctx, f.engineer, Input{Name: "IQC\nSMT"})), []string{"name:invalid_format"}},
		{"alias kosong", second(f.s.Create(ctx, f.engineer, Input{Name: "SMT", Aliases: []string{" "}})), []string{"aliases:required"}},
		{"alias terlalu panjang", second(f.s.Create(ctx, f.engineer, Input{Name: "SMT", Aliases: []string{strings.Repeat("a", 201)}})), []string{"aliases:too_long"}},
		{"alias terlalu banyak", second(f.s.Create(ctx, f.engineer, Input{Name: "SMT", Aliases: tooMany})), []string{"aliases:too_long"}},
		{"nama dan alias salah", second(f.s.Update(ctx, f.engineer, p.ID, Input{Name: "", Aliases: []string{"a\tb"}})), []string{"name:required", "aliases:invalid_format"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := problemCodes(t, tt.err); !slices.Equal(got, tt.want) {
				t.Errorf("masalah %v, ingin %v", got, tt.want)
			}
		})
	}

	for _, id := range []string{"bukan-uuid", "01a180aa-a98b-7bf2-8e0d-b97a25ecc489"} {
		if _, err := f.s.Get(ctx, f.engineer, id); !errors.Is(err, problem.ErrNotFound) {
			t.Errorf("Get %s: %v, ingin problem.ErrNotFound", id, err)
		}
		if _, err := f.s.Update(ctx, f.engineer, id, Input{Name: "X"}); !errors.Is(err, problem.ErrNotFound) {
			t.Errorf("Update %s: %v, ingin problem.ErrNotFound", id, err)
		}
	}
}

// TestPermissions: master proses diubah pemegang E-6, yaitu Pengelola
// template milik PPAP Engineer (D-64). Akun admin dan penyusun biasa tidak
// bisa mengubahnya, tapi semua akun bisa melihat (ASUMSI E-24).
func TestPermissions(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	p, err := f.s.Create(ctx, f.engineer, Input{Name: "IQC"})
	if err != nil {
		t.Fatal(err)
	}
	admin := newActor(t, f.pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	author := newActor(t, f.pool, "penyusun.uji", access.AccountWork, access.RoleAuthor)

	for _, actor := range []access.Actor{admin, author} {
		if _, err := f.s.Create(ctx, actor, Input{Name: "SMT"}); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("Create oleh %s: %v, ingin ErrForbidden", actor.Login, err)
		}
		if _, err := f.s.Update(ctx, actor, p.ID, Input{Name: "SMT"}); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("Update oleh %s: %v, ingin ErrForbidden", actor.Login, err)
		}
		if list, err := f.s.List(ctx, actor); err != nil || len(list) != 1 {
			t.Errorf("List oleh %s: %+v, %v", actor.Login, list, err)
		}
		if got, err := f.s.Get(ctx, actor, p.ID); err != nil || got.Name != "IQC" {
			t.Errorf("Get oleh %s: %+v, %v", actor.Login, got, err)
		}
	}
	// Tanpa hak, penolakan datang sebelum isian diperiksa.
	if _, err := f.s.Create(ctx, author, Input{}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Create kosong oleh penyusun: %v, ingin ErrForbidden", err)
	}
}

// TestAudit: proses baru dan setiap perubahan tercatat beserta nilai lama dan
// barunya. Simpan tanpa perubahan tidak dicatat.
func TestAudit(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	p, err := f.s.Create(ctx, f.engineer, Input{Name: "Parts Receiving", Aliases: []string{"Part Receiving"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Update(ctx, f.engineer, p.ID, Input{Name: "Receiving", Aliases: []string{"Parts Receiving", "Part Receiving"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Update(ctx, f.engineer, p.ID, Input{Name: "Receiving", Aliases: []string{"Part Receiving", "Parts Receiving"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Update(ctx, f.engineer, p.ID, Input{Name: "Receiving"}); err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, r := range auditOf(t, f.pool, p.ID) {
		got = append(got, r.Action+" "+r.Details)
		if r.TargetType == nil || *r.TargetType != audit.TargetProcess {
			t.Errorf("jenis sasaran %v, ingin %s", r.TargetType, audit.TargetProcess)
		}
	}
	want := []string{
		`process.created {"aliases":"Part Receiving","name":"Parts Receiving"}`,
		`process.updated {"aliases":"Part Receiving → Part Receiving, Parts Receiving","name":"Parts Receiving → Receiving"}`,
		`process.updated {"aliases":"Part Receiving, Parts Receiving → -"}`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("entri audit:\n%s\ningin:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
