package document_test

import (
	"errors"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
)

// M8.1 · Akses lintas customer (R-SEC-12, D-71, RK-14). Project Leader
// Project Group 1 mencoba menyentuh dokumen customer Project Group 2: semua
// percobaan ditolak fungsi izin, dan database sendiri menyembunyikan barisnya
// (row-level security, migrasi 00042).
func TestCrossCustomerAccess(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()

	// Project Group 2 dengan customer EF dan model E-1.
	var group2, customerEF, modelE1 string
	if err := e.Pool.QueryRow(ctx, `INSERT INTO project_groups (code, name) VALUES ('GR2', 'Project Group 2') RETURNING id`).Scan(&group2); err != nil {
		t.Fatal(err)
	}
	ef, err := customer.NewService(e.Pool, e.Checker).Create(ctx, e.Master, customer.NewCustomer{Code: "EF", Name: "Customer EF", ProjectGroupID: group2})
	if err != nil {
		t.Fatal(err)
	}
	customerEF = ef.ID
	if err := e.Pool.QueryRow(ctx, `INSERT INTO models (customer_id, name) VALUES ($1, 'E-1') RETURNING id`, customerEF).Scan(&modelE1); err != nil {
		t.Fatal(err)
	}
	own := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)["cp"]
	other := e.NewSet(t, "ef", customerEF, modelE1)["cp"]
	d, err := e.Docs.AddRow(ctx, e.Author, other.ID, document.NewRow{Content: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	otherRow := d.Rows[0]

	// Project Leader Project Group 1: Penyusun bercakupan Project Group sendiri.
	leader := doctest.NewActor(t, e.Pool, "pl.gr1", "work", access.RoleGrant{Role: access.RoleAuthor, Scope: access.ScopeOwnProjectGroup})
	if _, err := e.Pool.Exec(ctx, `INSERT INTO position_holders (position_id, per_project_group, person_id, project_group_id)
		SELECT id, per_project_group, $1, $2 FROM positions WHERE key = 'project_leader'`, leader.PersonID, e.GroupID); err != nil {
		t.Fatal(err)
	}

	// Pagar pertama: fungsi izin. Customer sendiri boleh, customer lain ditolak.
	if _, err := e.Docs.Get(ctx, leader, own.ID); err != nil {
		t.Fatalf("dokumen customer sendiri: %v", err)
	}
	attempts := map[string]func() error{
		"daftar dokumen": func() error { _, err := e.Docs.List(ctx, leader, customerEF); return err },
		"baca dokumen":   func() error { _, err := e.Docs.Get(ctx, leader, other.ID); return err },
		"tambah baris": func() error {
			_, err := e.Docs.AddRow(ctx, leader, other.ID, document.NewRow{Content: map[string]any{}})
			return err
		},
		"ubah baris": func() error {
			_, err := e.Docs.UpdateRow(ctx, leader, other.ID, otherRow.ID, document.RowUpdate{Content: map[string]any{}, Version: otherRow.Version})
			return err
		},
		"hapus baris": func() error { _, err := e.Docs.DeleteRow(ctx, leader, other.ID, otherRow.ID); return err },
		"ubah header": func() error {
			_, err := e.Docs.SetHeader(ctx, leader, other.ID, map[string]any{}, other.HeaderVersion)
			return err
		},
		"revisi":                  func() error { _, err := e.Docs.Revisions(ctx, leader, other.ID); return err },
		"perbedaan":               func() error { _, err := e.Docs.Differences(ctx, leader, other.ID); return err },
		"usulan tautan":           func() error { _, err := e.Docs.LinkSuggestions(ctx, leader, other.ID); return err },
		"salin dari dokumen lain": func() error { _, err := e.Docs.CopyFrom(ctx, leader, other.ID, own.ID); return err },
	}
	for name, try := range attempts {
		if err := try(); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s lintas customer: %v", name, err)
		}
	}

	// Pagar kedua: row-level security. Dengan akun itu sebagai subjek, query
	// langsung ke tabel tidak melihat dan tidak bisa mengubah baris customer lain.
	as := db.WithSubject(ctx, leader.AccountID)
	count := func(sql string, args ...any) int {
		t.Helper()
		var n int
		if err := e.Pool.QueryRow(as, sql, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count(`SELECT count(*) FROM documents WHERE customer_id = $1`, customerEF); n != 0 {
		t.Errorf("dokumen customer lain terlihat: %d", n)
	}
	if n := count(`SELECT count(*) FROM documents WHERE customer_id = $1`, e.CustomerAB); n != 3 {
		t.Errorf("dokumen customer sendiri: %d", n)
	}
	if n := count(`SELECT count(*) FROM document_rows WHERE document_id = $1`, other.ID); n != 0 {
		t.Errorf("baris customer lain terlihat: %d", n)
	}
	if n := count(`SELECT count(*) FROM document_revisions WHERE document_id = $1`, other.ID); n != 0 {
		t.Errorf("revisi customer lain terlihat: %d", n)
	}
	tag, err := e.Pool.Exec(as, `UPDATE document_rows SET content = '{}' WHERE document_id = $1`, other.ID)
	if err != nil || tag.RowsAffected() != 0 {
		t.Errorf("ubah baris customer lain: %d baris, %v", tag.RowsAffected(), err)
	}
	// Tanpa subjek (CLI, job) dan sebagai Penyusun semua customer: terlihat.
	var all int
	if err := e.Pool.QueryRow(ctx, `SELECT count(*) FROM documents`).Scan(&all); err != nil || all != 6 {
		t.Errorf("tanpa subjek: %d, %v", all, err)
	}
	if err := e.Pool.QueryRow(db.WithSubject(ctx, e.Author.AccountID), `SELECT count(*) FROM documents`).Scan(&all); err != nil || all != 6 {
		t.Errorf("penyusun semua customer: %d, %v", all, err)
	}
	// Viewer (tanpa L-4) tidak melihat draft, tetapi melihat dokumen yang sudah terbit (R-SEC-12).
	viewer := db.WithSubject(ctx, e.Viewer.AccountID)
	if err := e.Pool.QueryRow(viewer, `SELECT count(*) FROM documents`).Scan(&all); err != nil || all != 0 {
		t.Errorf("viewer melihat draft: %d, %v", all, err)
	}
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	doctest.AttachScan(t, e.Pool, other.ID)
	if _, err := e.Docs.Release(ctx, dcc, other.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.Pool.QueryRow(viewer, `SELECT count(*) FROM documents`).Scan(&all); err != nil || all != 1 {
		t.Errorf("viewer melihat dokumen terbit: %d, %v", all, err)
	}
	if n := count(`SELECT count(*) FROM documents WHERE customer_id = $1`, customerEF); n != 1 {
		t.Errorf("dokumen terbit customer lain: %d", n)
	}
}
