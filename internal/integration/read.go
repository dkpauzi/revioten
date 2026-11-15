package integration

// Bacaan data rilis (M9.3, R-API-12, R-API-13). Hanya dokumen yang punya
// revisi terbit, dan hanya customer yang diizinkan untuk token. Draft tidak
// pernah keluar lewat API ini.

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// Batas halaman (R-API-13: paginasi cursor dengan ukuran halaman terbesar).
const (
	DefaultPageSize = 50
	MaxPageSize     = 200
)

// ReleasedDocument adalah satu dokumen terbit di daftar.
type ReleasedDocument struct {
	ID           string
	Number       string
	CustomerID   string
	CustomerCode string
	DocumentType string
	Revision     int
	ReleasedAt   time.Time
	SHA256       string
	Migrated     bool
}

// Page adalah satu halaman daftar. NextCursor kosong berarti halaman terakhir.
type Page struct {
	Items      []ReleasedDocument
	NextCursor string
}

// ListFilter menyaring daftar dokumen terbit.
type ListFilter struct {
	CustomerID   string
	UpdatedSince *time.Time // hanya yang rilisnya pada atau sesudah waktu ini (sinkronisasi bertahap)
	Cursor       string     // dari NextCursor halaman sebelumnya
	Limit        int        // 0 berarti DefaultPageSize
}

// encodeCursor menyusun cursor dari baris terakhir halaman. Isinya buram
// bagi pemanggil: waktu rilis dan ID, dikodekan base64.
func encodeCursor(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(cursor string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", err
	}
	at, id, ok := strings.Cut(string(raw), "|")
	if !ok || !db.IsUUID(id) {
		return time.Time{}, "", errors.New("cursor rusak")
	}
	t, err := time.Parse(time.RFC3339Nano, at)
	return t, id, err
}

// ListDocuments memberi dokumen terbit yang boleh dibaca token ini (scope
// documents:read), menurut waktu rilis. Untuk sinkronisasi bertahap: simpan
// waktu rilis terakhir yang diterima, lalu kirim sebagai updated_since.
func (s *Service) ListDocuments(ctx context.Context, c Client, f ListFilter) (Page, error) {
	if err := c.require(ScopeDocumentsRead); err != nil {
		return Page{}, err
	}
	limit := f.Limit
	if limit == 0 {
		limit = DefaultPageSize
	}
	if limit < 1 || limit > MaxPageSize {
		return Page{}, invalid("limit", problem.InvalidFormat, fmt.Sprintf("Choose a page size from 1 to %d.", MaxPageSize))
	}
	params := db.ListReleasedDocumentsParams{CustomerIds: c.CustomerIDs, UpdatedSince: f.UpdatedSince, PageSize: int32(limit + 1)}
	if f.CustomerID != "" {
		// Customer di luar daftar token dijawab seperti tidak punya dokumen.
		if !db.IsUUID(f.CustomerID) {
			return Page{}, invalid("customer_id", problem.InvalidFormat, "Choose an existing customer.")
		}
		params.CustomerID = &f.CustomerID
	}
	if f.Cursor != "" {
		at, id, err := decodeCursor(f.Cursor)
		if err != nil {
			return Page{}, invalid("cursor", problem.InvalidFormat, "Use the next_cursor of the previous page.")
		}
		params.AfterTime, params.AfterID = &at, &id
	}
	rows, err := s.q.ListReleasedDocuments(ctx, params)
	if err != nil {
		return Page{}, fmt.Errorf("membaca dokumen terbit: %w", err)
	}
	page := Page{Items: make([]ReleasedDocument, 0, len(rows))}
	for i, r := range rows {
		if i == limit {
			// Baris ke limit+1 hanya penanda bahwa masih ada halaman berikutnya.
			last := page.Items[limit-1]
			page.NextCursor = encodeCursor(last.ReleasedAt, last.ID)
			break
		}
		item := ReleasedDocument{ID: r.ID, Number: r.Number, CustomerID: r.CustomerID, CustomerCode: r.CustomerCode, Migrated: r.Migrated}
		if r.DocumentTypeKey != nil {
			item.DocumentType = *r.DocumentTypeKey
		}
		if r.ReleasedRevision != nil {
			item.Revision = int(*r.ReleasedRevision)
		}
		if r.ReleasedAt != nil {
			item.ReleasedAt = r.ReleasedAt.UTC()
		}
		if r.SnapshotSha256 != nil {
			item.SHA256 = *r.SnapshotSha256
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

// Field adalah satu kolom format: ID-nya menjadi kunci isi baris.
type Field struct {
	ID    string
	Key   string
	Label string
	Area  string // header atau row
	// Automatic: dihitung atau diisi sistem, jadi tidak boleh dikirim dalam usulan.
	Automatic bool
}

// Revision adalah satu revisi terbit lengkap dengan isinya.
type Revision struct {
	DocumentID  string
	Number      string
	CustomerID  string
	Revision    int
	Current     bool // revisi yang berlaku sekarang
	ReleasedAt  time.Time
	ReleaseDate string
	SHA256      string
	Migrated    bool
	Snapshot    document.Snapshot
	Fields      []Field
}

// releasedRevision membaca satu revisi terbit dan memeriksa customer token.
func (s *Service) releasedRevision(ctx context.Context, q *db.Queries, c Client, documentID string, revision *int) (db.GetReleasedRevisionRow, document.Snapshot, error) {
	if !db.IsUUID(documentID) {
		return db.GetReleasedRevisionRow{}, document.Snapshot{}, problem.ErrNotFound
	}
	params := db.GetReleasedRevisionParams{ID: documentID}
	if revision != nil {
		if *revision < 0 || *revision > 32767 {
			return db.GetReleasedRevisionRow{}, document.Snapshot{}, problem.ErrNotFound
		}
		v := int16(*revision)
		params.Revision = &v
	}
	r, err := q.GetReleasedRevision(ctx, params)
	// Dokumen customer lain dijawab sama dengan dokumen yang tidak ada.
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !c.allows(r.CustomerID)) {
		return db.GetReleasedRevisionRow{}, document.Snapshot{}, problem.ErrNotFound
	}
	if err != nil {
		return db.GetReleasedRevisionRow{}, document.Snapshot{}, fmt.Errorf("membaca revisi: %w", err)
	}
	text := ""
	if r.Snapshot != nil {
		text = *r.Snapshot
	}
	snap, err := document.ParseSnapshot(text)
	if err != nil {
		return db.GetReleasedRevisionRow{}, document.Snapshot{}, fmt.Errorf("membaca snapshot: %w", err)
	}
	return r, snap, nil
}

// GetRevision memberi satu revisi terbit beserta barisnya (scope
// revisions:read). revision nil berarti revisi yang berlaku.
func (s *Service) GetRevision(ctx context.Context, c Client, documentID string, revision *int) (Revision, error) {
	if err := c.require(ScopeRevisionsRead); err != nil {
		return Revision{}, err
	}
	r, snap, err := s.releasedRevision(ctx, s.q, c, documentID, revision)
	if err != nil {
		return Revision{}, err
	}
	out := Revision{DocumentID: r.ID, Number: r.Number, CustomerID: r.CustomerID, Revision: int(r.Revision), Migrated: r.Migrated, Snapshot: snap,
		Current: r.ReleasedRevision != nil && *r.ReleasedRevision == r.Revision}
	if r.ReleasedAt != nil {
		out.ReleasedAt = r.ReleasedAt.UTC()
	}
	if r.ReleaseDate.Valid {
		out.ReleaseDate = r.ReleaseDate.Time.Format(time.DateOnly)
	}
	if r.SnapshotSha256 != nil {
		out.SHA256 = *r.SnapshotSha256
	}
	format, err := template.LoadFormatVersion(ctx, s.q, snap.FormatVersionID)
	if err != nil {
		return Revision{}, fmt.Errorf("membaca format: %w", err)
	}
	for _, f := range format.Fields {
		if !f.Hidden {
			out.Fields = append(out.Fields, Field{ID: f.ID, Key: f.Key, Label: f.Label, Area: f.Area, Automatic: !template.Typed(f.Kind)})
		}
	}
	return out, nil
}
