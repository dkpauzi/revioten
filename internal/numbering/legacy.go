package numbering

// Nomor lama satu set untuk menu migrasi (D-215, D-216).
//
// Nomor PFD, PFMEA, dan CP satu set diisi dalam tiga kolom. Bawaannya "satu
// set": cukup satu kolom diisi, dan dua lainnya mengikuti nomor berjalan dan
// kode customer yang sama (SH-TR-020 → SF-TR-020, SC-TR-020). Bila nomor
// berjalannya tidak sejajar (misalnya SC-TR-005, SF-TR-005, SH-TR-004),
// setiap nomor diketik. Bagaimanapun, nomor di satu isian adalah satu set
// dokumen, karena semuanya masuk ke satu batch (E-140).

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// LegacySetInput adalah nomor lama satu set.
type LegacySetInput struct {
	CustomerID string
	// OneSet true: nomor yang kosong mengikuti nomor berjalan nomor yang diisi.
	OneSet bool
	// Numbers berisi teks nomor per key jenis dokumen (KeyPFD, KeyPFMEA,
	// KeyCP). Hanya jenis yang ada di map ini yang dipakai; teks kosong
	// hanya boleh bila OneSet.
	Numbers map[string]string
	// Field adalah awalan nama field di pesan penolakan, misalnya "documents".
	// Nama field menjadi <Field>.<key>.number.
	Field string
}

// LegacyNumber adalah satu nomor di register untuk set itu.
type LegacyNumber struct {
	ID      string
	Number  string
	Status  string
	BatchID string
	New     bool // true bila baru didaftarkan oleh isian ini
}

// legacyCell adalah satu nomor isian beserta bagiannya.
type legacyCell struct {
	typeKey string
	field   string
	text    string
	key     numberKey
	docType DocumentType
}

// RegisterLegacySet memastikan nomor lama satu set ada di register, di dalam
// transaksi pemanggil (q). Nomor yang belum ada didaftarkan dengan teks
// aslinya, berstatus Released, dalam satu batch baru, dan penghitungnya naik
// supaya nomor baru melanjutkan sesudahnya (R-NUM-06). Nomor yang sudah ada
// dikembalikan apa adanya; pemanggil yang memeriksa apakah nomor itu masih
// bisa dipakai. Harus menjadi langkah penomoran pertama di transaksi itu,
// karena customer dikunci seperti pendaftaran biasa (R-POR-01).
func (s *Service) RegisterLegacySet(ctx context.Context, q *db.Queries, actor access.Actor, in LegacySetInput) (map[string]LegacyNumber, error) {
	settings, err := q.GetNumberSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca pengaturan nomor: %w", err)
	}
	pattern, err := ParsePattern(settings.Pattern)
	if err != nil {
		return nil, fmt.Errorf("pola nomor di database tidak sah: %w", err)
	}
	types, err := s.setTypes(ctx)
	if err != nil {
		return nil, err
	}
	codes, err := lockCustomerCodes(ctx, q, in.CustomerID)
	if err != nil {
		return nil, err
	}
	cells, err := legacyCells(in, pattern, types, codes)
	if err != nil {
		return nil, err
	}

	// Penghitung dikunci menurut urutan kode lalu ID jenis, sama dengan impor
	// dan daftar sebagai set, supaya tidak ada deadlock.
	order := slices.Clone(cells)
	slices.SortFunc(order, func(a, b legacyCell) int {
		return cmp.Or(strings.Compare(a.key.code, b.key.code), strings.Compare(a.key.typeID, b.key.typeID))
	})
	for _, c := range order {
		if _, err := q.LockCounter(ctx, db.LockCounterParams{CustomerCode: c.key.code, DocumentTypeID: c.key.typeID, FirstValue: FirstSequence - 1}); err != nil {
			return nil, err
		}
	}

	out := map[string]LegacyNumber{}
	var batchID string
	for _, c := range cells {
		found, err := q.FindRegisteredNumber(ctx, db.FindRegisteredNumberParams{
			CustomerCode: c.key.code, DocumentTypeID: c.key.typeID, Sequence: int32(c.key.sequence), Number: c.text,
		})
		if err == nil {
			out[c.typeKey] = LegacyNumber{ID: found.ID, Number: found.Number, Status: found.Status, BatchID: found.BatchID}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("mencari nomor di register: %w", err)
		}
		if batchID == "" {
			if batchID, err = q.InsertNumberBatch(ctx, db.InsertNumberBatchParams{
				Kind: OriginImport, CustomerID: in.CustomerID, AccountID: actor.AccountID, PersonID: actor.PersonID,
			}); err != nil {
				return nil, err
			}
		}
		id, err := q.InsertDocumentNumber(ctx, db.InsertDocumentNumberParams{
			Number: c.text, CustomerID: in.CustomerID, CustomerCode: c.key.code, DocumentTypeID: c.key.typeID,
			Sequence: int32(c.key.sequence), Status: StatusReleased, BatchID: batchID,
		})
		if err != nil {
			return nil, err
		}
		if err := q.SetCounter(ctx, db.SetCounterParams{CustomerCode: c.key.code, DocumentTypeID: c.key.typeID, LastValue: int32(c.key.sequence)}); err != nil {
			return nil, err
		}
		if err := audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionDocumentNumberImported, TargetType: audit.TargetDocumentNumber, TargetID: id,
			Details: map[string]string{"number": c.text, "status": StatusReleased},
		}); err != nil {
			return nil, err
		}
		out[c.typeKey] = LegacyNumber{ID: id, Number: c.text, Status: StatusReleased, BatchID: batchID, New: true}
	}
	return out, nil
}

// legacyCells membaca nomor isian dengan pola sekarang dan mengisi nomor yang
// mengikuti (OneSet). Urutan hasilnya PFD, PFMEA, CP.
func legacyCells(in LegacySetInput, pattern Pattern, types []DocumentType, codes []string) ([]legacyCell, error) {
	var list []*problem.Problem
	var cells []legacyCell
	var source *legacyCell
	for _, d := range types {
		text, wanted := in.Numbers[d.Key]
		if !wanted {
			continue
		}
		c := legacyCell{typeKey: d.Key, field: in.Field + "." + d.Key + ".number", text: strings.TrimSpace(text), docType: d}
		if c.text == "" {
			if !in.OneSet {
				list = append(list, &problem.Problem{Field: c.field, Code: problem.Required, Message: "Enter the " + d.Name + " number."})
			}
			cells = append(cells, c)
			continue
		}
		parts, ok := pattern.Parse(c.text)
		if !ok || parts.Prefix != d.Prefix {
			list = append(list, &problem.Problem{Field: c.field, Code: problem.InvalidFormat,
				Message: fmt.Sprintf("Enter a %s number in the current pattern, for example %s, without the revision.",
					d.Name, pattern.Format(d.Prefix, codes[len(codes)-1], "020"))})
			cells = append(cells, c)
			continue
		}
		if !slices.Contains(codes, parts.Customer) {
			list = append(list, &problem.Problem{Field: c.field, Code: problem.NotAllowed,
				Message: fmt.Sprintf("%s is not a code of this customer.", parts.Customer)})
			cells = append(cells, c)
			continue
		}
		sequence, _ := ParseSequence(parts.Sequence)
		c.key = numberKey{code: parts.Customer, typeID: d.ID, sequence: sequence}
		cells = append(cells, c)
		if source == nil {
			source = &cells[len(cells)-1]
		}
	}
	if len(cells) == 0 {
		return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: in.Field, Code: problem.Required, Message: "Enter at least one document number."}}}
	}
	if err := problems(list...); err != nil {
		return nil, err
	}
	if source == nil {
		return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: cells[0].field, Code: problem.Required, Message: "Enter one document number of the set."}}}
	}
	if in.OneSet {
		// Satu set: semua nomor memakai nomor berjalan dan kode yang sama.
		src := *source
		for i := range cells {
			c := &cells[i]
			want := numberKey{code: src.key.code, typeID: c.docType.ID, sequence: src.key.sequence}
			if c.text == "" {
				c.key = want
				c.text = pattern.Format(c.docType.Prefix, src.key.code, sequenceText(src.key.sequence))
				continue
			}
			if c.key != want {
				list = append(list, &problem.Problem{Field: c.field, Code: problem.NotAllowed,
					Message: fmt.Sprintf("In one set every number has the running number %s. Untick One set when the numbers are not aligned.", sequenceText(src.key.sequence))})
			}
		}
	}
	if err := problems(list...); err != nil {
		return nil, err
	}
	return cells, nil
}
