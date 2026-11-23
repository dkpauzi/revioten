package change

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/settings"
)

// D-223 (E-145 opsi a): ganti logo seperti revisi dokumen. Revisi terbit
// membekukan logo saat rilis; admin menjawab apakah dokumen direvisi; bila ya,
// PPAP Engineer mengajukan satu perubahan "Logo change" per customer.

func pngLogo(t *testing.T, w int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, w, 4))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func snapshotLogo(t *testing.T, e doctest.Env, documentID string, revision int) string {
	t.Helper()
	var text string
	if err := e.Pool.QueryRow(t.Context(), "SELECT snapshot FROM document_revisions WHERE document_id = $1 AND revision = $2", documentID, revision).Scan(&text); err != nil {
		t.Fatal(err)
	}
	snap, err := document.ParseSnapshot(text)
	if err != nil {
		t.Fatal(err)
	}
	return snap.LogoSHA256
}

func TestLogoChangeAsRevision(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	s := NewRequestService(e.Pool, e.Checker)
	logos := settings.NewService(e.Pool, e.Checker)
	m := readMasters(t, e)
	k125 := e.NewSet(t, "k125", e.CustomerAB, e.ModelK125)
	e.NewSet(t, "b1", e.CustomerCD, e.ModelB1)

	// CP K-125 terbit di revisi 0 dengan logo bawaan.
	recorder := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	doctest.AttachScan(t, e.Pool, k125["cp"].ID)
	if _, err := e.Docs.Release(ctx, recorder, k125["cp"].ID); err != nil {
		t.Fatal(err)
	}
	if got := snapshotLogo(t, e, k125["cp"].ID, 0); got != settings.DefaultLogoSHA {
		t.Fatalf("logo di snapshot revisi 0: %q, ingin logo bawaan", got)
	}

	// Tanpa logo baru, tidak ada yang menunggu.
	if p, err := s.PendingLogoChange(ctx, e.Author, e.CustomerAB); err != nil || p != nil {
		t.Fatalf("sebelum ganti logo: %+v, %v", p, err)
	}

	// Admin mengganti logo dan menjawab "ya, revisi dokumen".
	newLogo := pngLogo(t, 30)
	c, err := logos.SetLogo(ctx, e.Admin, newLogo, true)
	if err != nil {
		t.Fatal(err)
	}
	// Revisi 0 tetap memakai logo bawaan; logo bawaan dan logo baru keduanya bisa dibaca lagi.
	if got := snapshotLogo(t, e, k125["cp"].ID, 0); got != settings.DefaultLogoSHA {
		t.Errorf("revisi 0 berubah logo: %q", got)
	}
	if old, _ := logos.LogoBySHA(ctx, settings.DefaultLogoSHA); !bytes.Equal(old, settings.DefaultLogo) {
		t.Error("logo bawaan tidak bisa dibaca lagi")
	}
	if cur, _ := logos.LogoBySHA(ctx, c.LogoSHA256); !bytes.Equal(cur, newLogo) {
		t.Error("logo baru tidak tersimpan di riwayat logo")
	}

	pending, err := s.PendingLogoChange(ctx, e.Author, e.CustomerAB)
	if err != nil || pending == nil || pending.LogoSHA256 != c.LogoSHA256 {
		t.Fatalf("perubahan logo yang menunggu: %+v, %v", pending, err)
	}

	in := LogoChange{CustomerID: e.CustomerAB, LogoChangeID: pending.ID, ChangeTypeID: m.content, Reason: "Change company logo"}
	if _, err := s.SubmitLogoChange(ctx, e.Viewer, in); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("viewer: %v", err)
	}
	ch, err := s.SubmitLogoChange(ctx, e.Author, in)
	if err != nil {
		t.Fatal(err)
	}
	if ch.Mechanism != MechanismLogo || ch.Reason != "Change company logo" || len(ch.Documents) != 3 {
		t.Fatalf("perubahan logo: %+v", ch)
	}
	// Sekali per customer per penggantian logo.
	if _, err := s.SubmitLogoChange(ctx, e.Author, in); !slices.Equal(problemCodes(t, err), []string{"logo_change_id:not_allowed"}) {
		t.Errorf("perubahan logo kedua: %v", err)
	}
	if p, _ := s.PendingLogoChange(ctx, e.Author, e.CustomerAB); p != nil {
		t.Errorf("customer AB masih menunggu: %+v", p)
	}
	if p, _ := s.PendingLogoChange(ctx, e.Author, e.CustomerCD); p == nil {
		t.Error("customer CD belum mengajukan, tetapi tidak menunggu")
	}

	// Jawaban "tidak": penggantian berikutnya tidak menunggu perubahan.
	if _, err := logos.SetLogo(ctx, e.Admin, pngLogo(t, 40), false); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.PendingLogoChange(ctx, e.Author, e.CustomerCD); p != nil {
		t.Errorf("sesudah jawaban tidak: %+v", p)
	}
}
