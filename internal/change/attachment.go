package change

// Lampiran perubahan (M5.2, R-CHG-01, R-SEC-04). Berkas pendukung, misalnya
// laporan 8D, foto NG, atau surat customer, disimpan di folder privat dengan
// nama SHA-256 isinya (internal/storage). Unduhan selalu lewat layanan ini,
// yang memeriksa hak akses. Pemindaian antivirus menyusul di Tahap 8.

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/storage"
)

// MaxAttachmentBytes adalah ukuran lampiran paling besar: 20 MB (ASUMSI E-99).
const MaxAttachmentBytes = 20 << 20

const maxFileName = 200

// allowedTypes adalah jenis berkas yang boleh dilampirkan, menurut akhiran,
// beserta jenis yang dipakai saat diunduh (ASUMSI E-99).
var allowedTypes = map[string]string{
	".pdf":  "application/pdf",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xls":  "application/vnd.ms-excel",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".doc":  "application/msword",
	".msg":  "application/vnd.ms-outlook",
	".txt":  "text/plain",
	".csv":  "text/csv",
}

// Attachment adalah satu lampiran perubahan.
type Attachment struct {
	ID          string
	FileName    string
	ContentType string
	Size        int64
	SHA256      string
	UploadedBy  string // nama orang pengunggah
	UploadedAt  time.Time
}

// WithStorage memasang penyimpanan lampiran. Tanpa penyimpanan, unggahan ditolak.
func (s *RequestService) WithStorage(store *storage.Store) *RequestService {
	s.store = store
	return s
}

// AddAttachment menyimpan satu berkas dan menautkannya ke perubahan (E-3).
func (s *RequestService) AddAttachment(ctx context.Context, actor access.Actor, id, fileName string, r io.Reader) (Change, error) {
	if s.store == nil {
		return Change{}, errors.New("penyimpanan lampiran belum dipasang")
	}
	name := cleanFileName(fileName)
	contentType, ok := allowedTypes[strings.ToLower(filepath.Ext(name))]
	if name == "" || !ok {
		return Change{}, problems(invalid("file", problem.NotAllowed, "Attach a PDF, picture, Excel, Word, Outlook, text, or CSV file."))
	}
	c, err := s.load(ctx, id)
	if err != nil {
		return Change{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.ProposeChanges, access.Resource{CustomerID: c.CustomerID}); err != nil {
		return Change{}, err
	}
	// Berkas disimpan dulu, di luar transaksi. Bila pencatatan gagal, berkas
	// tanpa catatan tidak berbahaya: namanya hash, dan tidak bisa diunduh.
	sum, size, err := s.store.Put(r, MaxAttachmentBytes)
	switch {
	case errors.Is(err, storage.ErrEmpty):
		return Change{}, problems(invalid("file", problem.Required, "The file is empty."))
	case errors.Is(err, storage.ErrTooLarge):
		return Change{}, problems(invalid("file", problem.TooLong, "A file can be at most 20 MB."))
	case err != nil:
		return Change{}, err
	}
	return s.edit(ctx, actor, id, func(q *db.Queries, c Change) (string, map[string]string, error) {
		_, err := q.AddChangeAttachment(ctx, db.AddChangeAttachmentParams{
			ChangeID: c.ID, FileName: name, ContentType: contentType, SizeBytes: size, Sha256: sum, UploadedByPersonID: actor.PersonID,
		})
		return audit.ActionChangeAttachmentAdded, map[string]string{"file": name, "sha256": sum}, err
	})
}

// RemoveAttachment melepas satu lampiran dari perubahan yang masih diajukan
// (E-3). Catatannya tetap ada; berkasnya tidak bisa diunduh lagi.
func (s *RequestService) RemoveAttachment(ctx context.Context, actor access.Actor, id, attachmentID string) (Change, error) {
	return s.edit(ctx, actor, id, func(q *db.Queries, c Change) (string, map[string]string, error) {
		name := ""
		for _, a := range c.Attachments {
			if a.ID == attachmentID {
				name = a.FileName
			}
		}
		if name == "" {
			return "", nil, problem.ErrNotFound
		}
		if _, err := q.RemoveChangeAttachment(ctx, db.RemoveChangeAttachmentParams{ID: attachmentID, ChangeID: c.ID}); err != nil {
			return "", nil, err
		}
		return audit.ActionChangeAttachmentRemoved, map[string]string{"file": name}, nil
	})
}

// OpenAttachment membuka berkas satu lampiran untuk diunduh (L-4).
// Pemanggil menutup berkasnya.
func (s *RequestService) OpenAttachment(ctx context.Context, actor access.Actor, id, attachmentID string) (Attachment, *os.File, error) {
	c, err := s.Get(ctx, actor, id)
	if err != nil {
		return Attachment{}, nil, err
	}
	for _, a := range c.Attachments {
		if a.ID != attachmentID {
			continue
		}
		if s.store == nil {
			return Attachment{}, nil, errors.New("penyimpanan lampiran belum dipasang")
		}
		f, err := s.store.Open(a.SHA256)
		if err != nil {
			return Attachment{}, nil, err
		}
		return a, f, nil
	}
	return Attachment{}, nil, problem.ErrNotFound
}

// cleanFileName membuang folder dari nama berkas (misalnya ../), karakter
// kendali, dan memotongnya sampai 200 karakter.
func cleanFileName(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	name = filepath.Base("/" + name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "/" || name == "." {
		return ""
	}
	for utf8.RuneCountInString(name) > maxFileName {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return name
}

// attachments membaca lampiran aktif satu perubahan.
func attachments(ctx context.Context, q *db.Queries, id string) ([]Attachment, error) {
	rows, err := q.ListChangeAttachments(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Attachment, 0, len(rows))
	for _, r := range rows {
		out = append(out, Attachment{
			ID: r.ID, FileName: r.FileName, ContentType: r.ContentType, Size: r.SizeBytes, SHA256: r.Sha256,
			UploadedBy: r.UploadedBy, UploadedAt: r.UploadedAt.UTC(),
		})
	}
	return out, nil
}
