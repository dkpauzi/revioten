package settings

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"slices"
	"testing"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Bagian C (E-144): nama perusahaan dan logo di cetakan diatur admin
// (permintaan pemilik 22 November 2026: "buat logo bisa di atur oleh admin").
// Tanpa logo unggahan, cetakan memakai logo bawaan dari Excel referensi.

// pngOf membuat gambar PNG polos selebar w dan setinggi h.
func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.Black)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func problemCodes(err error) []string {
	var v *problem.ValidationError
	if !errors.As(err, &v) {
		return nil
	}
	var out []string
	for _, p := range v.Problems {
		out = append(out, p.Field+":"+p.Code)
	}
	return out
}

func TestCompanyAndLogo(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := t.Context()
	s := NewService(pool, access.NewChecker(pool))
	admin := newActor(t, pool, "admin.uji", access.AccountAdmin, access.RoleAdmin)
	engineer := newActor(t, pool, "ppap.uji", access.AccountWork, access.RolePPAPEngineer)

	c, err := s.Company(ctx, engineer)
	if err != nil || c.Name != "PT. SIIX EMS INDONESIA" || c.CustomLogo {
		t.Fatalf("isi awal: %+v, %v", c, err)
	}
	logo, err := s.Logo(ctx)
	if err != nil || !bytes.Equal(logo, DefaultLogo) || !bytes.HasPrefix(logo, []byte("\x89PNG")) {
		t.Fatalf("logo bawaan: %d byte, %v", len(logo), err)
	}

	// Hanya admin.
	if _, err := s.SetCompanyName(ctx, engineer, "PT. Lain"); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("PPAP Engineer mengganti nama: %v", err)
	}
	if _, err := s.SetLogo(ctx, engineer, pngOf(t, 10, 10), false); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("PPAP Engineer mengganti logo: %v", err)
	}
	if c, err := s.SetCompanyName(ctx, admin, "  PT. Contoh  "); err != nil || c.Name != "PT. Contoh" {
		t.Errorf("ganti nama: %+v, %v", c, err)
	}
	if _, err := s.SetCompanyName(ctx, admin, ""); !slices.Equal(problemCodes(err), []string{"name:required"}) {
		t.Errorf("nama kosong: %v", err)
	}

	// Bukan PNG, terlalu besar, atau terlalu lebar ditolak.
	for name, body := range map[string][]byte{
		"bukan PNG":     []byte("GIF89a bukan png"),
		"terlalu lebar": pngOf(t, 2100, 10),
	} {
		if _, err := s.SetLogo(ctx, admin, body, false); !slices.Equal(problemCodes(err), []string{"logo:invalid_format"}) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := s.SetLogo(ctx, admin, append([]byte("\x89PNG"), make([]byte, MaxLogoBytes)...), false); !slices.Equal(problemCodes(err), []string{"logo:too_long"}) {
		t.Errorf("terlalu besar: %v", err)
	}

	custom := pngOf(t, 40, 20)
	c, err = s.SetLogo(ctx, admin, custom, false)
	if err != nil || !c.CustomLogo || c.LogoSHA256 == "" || c.LogoUpdatedAt == nil {
		t.Fatalf("unggah logo: %+v, %v", c, err)
	}
	if got, _ := s.Logo(ctx); !bytes.Equal(got, custom) {
		t.Error("logo unggahan tidak dipakai")
	}
	var action, details string
	if err := pool.QueryRow(ctx, "SELECT action, details FROM audit_log ORDER BY seq DESC LIMIT 1").Scan(&action, &details); err != nil || action != "system_settings.logo_changed" {
		t.Errorf("audit: %s %s %v", action, details, err)
	}
	// Kembali ke logo bawaan.
	if c, err := s.ResetLogo(ctx, admin, false); err != nil || c.CustomLogo {
		t.Errorf("logo bawaan lagi: %+v, %v", c, err)
	}
	if got, _ := s.Logo(ctx); !bytes.Equal(got, DefaultLogo) {
		t.Error("logo bawaan tidak kembali")
	}
}
