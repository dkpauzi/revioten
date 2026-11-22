package settings

// Nama perusahaan dan logo di cetakan (bagian C, E-144). Admin mengaturnya di
// Admin → Settings, sesuai permintaan pemilik (22 November 2026). Tanpa logo
// unggahan, cetakan memakai logo bawaan: gambar asli dari Excel referensi
// TR-020 (xl/media/image1.png), yang dibundel di binary.

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"image/png"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// DefaultLogo adalah logo bawaan (PNG).
//
//go:embed logo-default.png
var DefaultLogo []byte

// Batas logo unggahan.
const (
	MaxLogoBytes = 1 << 20 // 1 MB
	maxLogoWidth = 2000    // piksel
)

// Company adalah nama perusahaan dan keterangan logo cetakan.
type Company struct {
	Name          string
	CustomLogo    bool       // false berarti logo bawaan
	LogoSHA256    string     // "" untuk logo bawaan
	LogoUpdatedAt *time.Time // waktu logo terakhir diganti; nil bila belum pernah
}

// Company mengembalikan nama perusahaan dan keterangan logo (semua akun).
func (s *Service) Company(ctx context.Context, actor access.Actor) (Company, error) {
	if err := s.authz.Authorize(ctx, actor, access.ViewRegister, access.Resource{}); err != nil {
		return Company{}, err
	}
	return s.company(ctx, s.q)
}

func (s *Service) company(ctx context.Context, q *db.Queries) (Company, error) {
	row, err := q.GetCompany(ctx)
	if err != nil {
		return Company{}, fmt.Errorf("membaca nama perusahaan: %w", err)
	}
	c := Company{Name: row.CompanyName, CustomLogo: row.CompanyLogoSha256 != nil}
	if row.CompanyLogoSha256 != nil {
		c.LogoSHA256 = *row.CompanyLogoSha256
	}
	if row.CompanyLogoUpdatedAt != nil {
		at := row.CompanyLogoUpdatedAt.UTC()
		c.LogoUpdatedAt = &at
	}
	return c, nil
}

// Logo mengembalikan logo yang dipakai cetakan: unggahan admin, atau logo
// bawaan. Tanpa pemeriksaan izin, karena dipanggil saat mencetak dan saat
// menampilkan gambar yang sama ke akun yang sudah login.
func (s *Service) Logo(ctx context.Context) ([]byte, error) {
	logo, err := s.q.GetCompanyLogo(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca logo: %w", err)
	}
	if len(logo) == 0 {
		return DefaultLogo, nil
	}
	return logo, nil
}

// SetCompanyName mengganti nama perusahaan di cetakan (admin).
func (s *Service) SetCompanyName(ctx context.Context, actor access.Actor, name string) (Company, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageSettings, access.Resource{}); err != nil {
		return Company{}, err
	}
	name, p := problem.Text("name", name, true, 200)
	if p != nil {
		return Company{}, &problem.ValidationError{Problems: []problem.Problem{*p}}
	}
	return s.change(ctx, actor, func(q *db.Queries, before Company) (map[string]string, error) {
		if before.Name == name {
			return nil, nil
		}
		if err := q.SetCompanyName(ctx, name); err != nil {
			return nil, err
		}
		return map[string]string{"company_name": audit.Change(before.Name, name)}, nil
	}, audit.ActionSystemSettingsUpdated)
}

// SetLogo mengganti logo cetakan dengan PNG unggahan (admin).
func (s *Service) SetLogo(ctx context.Context, actor access.Actor, data []byte) (Company, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageSettings, access.Resource{}); err != nil {
		return Company{}, err
	}
	if p := checkLogo(data); p != nil {
		return Company{}, &problem.ValidationError{Problems: []problem.Problem{*p}}
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	return s.change(ctx, actor, func(q *db.Queries, before Company) (map[string]string, error) {
		if err := q.SetCompanyLogo(ctx, db.SetCompanyLogoParams{CompanyLogo: data, CompanyLogoSha256: &hash, CompanyLogoUpdatedByPersonID: optionalPerson(actor)}); err != nil {
			return nil, err
		}
		// Isi gambar tidak masuk audit log; cukup hash-nya.
		return map[string]string{"logo_sha256": audit.Change(orDefault(before.LogoSHA256), hash)}, nil
	}, audit.ActionCompanyLogoChanged)
}

// ResetLogo kembali ke logo bawaan (admin).
func (s *Service) ResetLogo(ctx context.Context, actor access.Actor) (Company, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageSettings, access.Resource{}); err != nil {
		return Company{}, err
	}
	return s.change(ctx, actor, func(q *db.Queries, before Company) (map[string]string, error) {
		if !before.CustomLogo {
			return nil, nil
		}
		if err := q.SetCompanyLogo(ctx, db.SetCompanyLogoParams{CompanyLogoUpdatedByPersonID: optionalPerson(actor)}); err != nil {
			return nil, err
		}
		return map[string]string{"logo_sha256": audit.Change(before.LogoSHA256, "default")}, nil
	}, audit.ActionCompanyLogoChanged)
}

// change menjalankan satu perubahan dalam transaksi dengan baris pengaturan
// terkunci, lalu mencatat audit bila ada yang berubah.
func (s *Service) change(ctx context.Context, actor access.Actor, apply func(q *db.Queries, before Company) (map[string]string, error), action string) (Company, error) {
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		row, err := q.LockSystemSettings(ctx)
		if err != nil {
			return err
		}
		before, err := s.company(ctx, q)
		if err != nil {
			return err
		}
		details, err := apply(q, before)
		if err != nil || details == nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: action, TargetType: audit.TargetSystemSettings, TargetID: row.ID, Details: details})
	})
	if err != nil {
		return Company{}, fmt.Errorf("menyimpan pengaturan perusahaan: %w", err)
	}
	return s.company(ctx, s.q)
}

// checkLogo memeriksa unggahan: PNG sungguhan (dari isi, bukan nama berkas),
// paling besar 1 MB, dan paling lebar 2000 piksel.
func checkLogo(data []byte) *problem.Problem {
	if len(data) > MaxLogoBytes {
		return &problem.Problem{Field: "logo", Code: problem.TooLong, Message: "The logo can be at most 1 MB."}
	}
	invalid := &problem.Problem{Field: "logo", Code: problem.InvalidFormat, Message: "Choose a PNG image at most 2000 pixels wide."}
	if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return invalid
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Width > maxLogoWidth || cfg.Height < 1 || cfg.Height > maxLogoWidth {
		return invalid
	}
	return nil
}

func optionalPerson(actor access.Actor) *string {
	if actor.PersonID == "" {
		return nil
	}
	id := actor.PersonID
	return &id
}

func orDefault(hash string) string {
	if hash == "" {
		return "default"
	}
	return hash
}
