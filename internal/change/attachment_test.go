package change

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/storage"
)

// M5.2 · Lampiran (R-CHG-01, R-SEC-04): berkas pendukung disimpan di folder
// privat dengan SHA-256 dan tertaut ke perubahan.

func TestAttachments(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := NewRequestService(e.Pool, e.Checker).WithStorage(store)
	m := readMasters(t, e)
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	c, err := s.Create(ctx, e.Author, NewChange{CustomerID: e.CustomerAB, TriggerID: m.trigger, ChangeTypeID: m.content, Reason: "Claim", DocumentIDs: []string{set["cp"].ID}})
	if err != nil {
		t.Fatal(err)
	}

	content := "%PDF-1.4 8D report"
	if _, err := s.AddAttachment(ctx, e.Viewer, c.ID, "8D.pdf", strings.NewReader(content)); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer mengunggah: %v", err)
	}
	c, err = s.AddAttachment(ctx, e.Author, c.ID, "../8D report.PDF", strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Attachments) != 1 {
		t.Fatalf("lampiran: %+v", c.Attachments)
	}
	a := c.Attachments[0]
	// Nama dibersihkan dari folder; jenis dari akhiran; hash 64 huruf.
	if a.FileName != "8D report.PDF" || a.ContentType != "application/pdf" || a.Size != int64(len(content)) || len(a.SHA256) != 64 || a.UploadedBy != "Orang penyusun.uji" {
		t.Errorf("lampiran: %+v", a)
	}
	got, f, err := s.OpenAttachment(ctx, e.Author, c.ID, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(f)
	f.Close()
	if string(body) != content || got.SHA256 != a.SHA256 {
		t.Errorf("unduhan: %q", body)
	}
	if _, _, err := s.OpenAttachment(ctx, e.Viewer, c.ID, a.ID); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer mengunduh: %v", err)
	}

	// Jenis berkas yang tidak diizinkan, kosong, atau terlalu besar ditolak.
	for name, x := range map[string]struct {
		file, content, want string
	}{
		"exe":    {"setup.exe", "MZ", "file:not_allowed"},
		"kosong": {"empty.pdf", "", "file:required"},
		"besar":  {"big.pdf", strings.Repeat("x", MaxAttachmentBytes+1), "file:too_long"},
	} {
		_, err := s.AddAttachment(ctx, e.Author, c.ID, x.file, strings.NewReader(x.content))
		if got := problemCodes(t, err); !slices.Contains(got, x.want) {
			t.Errorf("%s: %v", name, got)
		}
	}

	// Lampiran dilepas: tidak tampil lagi, tetapi catatannya tetap di audit log.
	if c, err = s.RemoveAttachment(ctx, e.Author, c.ID, a.ID); err != nil || len(c.Attachments) != 0 {
		t.Errorf("lepas lampiran: %+v, %v", c.Attachments, err)
	}
	if _, _, err := s.OpenAttachment(ctx, e.Author, c.ID, a.ID); err == nil {
		t.Error("lampiran yang dilepas masih bisa diunduh")
	}
	var actions []string
	rows, err := e.Pool.Query(ctx, "SELECT action FROM audit_log WHERE action LIKE 'change.attachment_%' ORDER BY seq")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var x string
		_ = rows.Scan(&x)
		actions = append(actions, x)
	}
	rows.Close()
	if !slices.Equal(actions, []string{"change.attachment_added", "change.attachment_removed"}) {
		t.Errorf("audit: %v", actions)
	}
}
