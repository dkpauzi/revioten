package document

// Menu migrasi (D-215, D-216).
//
// Satu set dokumen lama (PFD, PFMEA, CP) masuk sekaligus dari nomor aslinya.
// Nomor yang belum ada di register didaftarkan saat itu juga, berstatus
// Released, dalam satu batch, jadi ketiganya satu set dokumen (E-140). Aturan
// migrasi tidak berubah: dokumen masuk di revisinya yang sekarang, nomor
// tidak pernah diubah, dan terbit sesudah orang lain mencocokkannya dengan
// hardcopy (M7.1, M7.5).

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/problem"
)

// LegacySetDocument adalah satu baris isian set: satu dokumen lama.
type LegacySetDocument struct {
	TypeKey       string // pfd, pfmea, atau cp
	Number        string // nomor asli tanpa revisi; boleh kosong bila OneSet
	Revision      int    // revisi saat ini di hardcopy, misalnya 2 untuk -02
	EffectiveDate string // YYYY-MM-DD
	PreparedBy    string // nama penyusun di hardcopy
	ApprovedBy    string // nama approver di hardcopy
}

// LegacySet adalah isian "masukkan satu set" di menu migrasi.
type LegacySet struct {
	CustomerID string
	// OneSet true (bawaan): nomor yang kosong mengikuti nomor berjalan nomor
	// yang diisi (D-216).
	OneSet        bool
	Documents     []LegacySetDocument // satu baris per dokumen yang dibuat
	ModelIDs      []string
	PartNumberIDs []string
}

// Keadaan dokumen lama di papan menu migrasi.
const (
	LegacyEmpty     = "empty"     // belum ada baris
	LegacyToVerify  = "to_verify" // sudah berisi, menunggu dicocokkan dengan hardcopy
	LegacyCompleted = "released"  // Released (migrated)
)

// LegacySummary adalah satu baris papan menu migrasi.
type LegacySummary struct {
	ID              string
	Number          string
	DocumentTypeKey string
	DocumentType    string
	Revision        int
	Rows            int
	EnteredBy       string
	EnteredAt       time.Time
	VerifiedBy      string // "" sebelum diverifikasi
	VerifiedAt      *time.Time
	State           string
}

// MigrationOverview adalah isi menu migrasi satu customer: prasyarat dan
// papan status (D-215).
type MigrationOverview struct {
	MigrationMode    bool
	FormSetPublished bool // form set customer (atau bawaan) punya versi terbit
	Models           int  // model customer yang belum EOL
	Processes        int  // proses di master
	Documents        []LegacySummary
}

// MigrationOverview memberi prasyarat dan papan status menu migrasi (E-10).
func (s *Service) MigrationOverview(ctx context.Context, actor access.Actor, customerID string) (MigrationOverview, error) {
	if !db.IsUUID(customerID) {
		return MigrationOverview{}, problems(invalid("customer_id", problem.InvalidFormat, "Choose an existing customer."))
	}
	if err := s.authz.Authorize(ctx, actor, access.MigrateDocuments, access.Resource{CustomerID: customerID}); err != nil {
		return MigrationOverview{}, err
	}
	customer, err := s.q.GetCustomerForDocument(ctx, customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return MigrationOverview{}, problems(invalid("customer_id", problem.InvalidFormat, "Choose an existing customer."))
	}
	if err != nil {
		return MigrationOverview{}, fmt.Errorf("membaca customer: %w", err)
	}
	ready, err := s.q.MigrationReadiness(ctx, customerID)
	if err != nil {
		return MigrationOverview{}, fmt.Errorf("membaca prasyarat migrasi: %w", err)
	}
	_, setProblem := formSetOf(ctx, s.q, customer.DefaultFormSetID)
	out := MigrationOverview{
		MigrationMode: ready.MigrationMode, FormSetPublished: setProblem == nil,
		Models: int(ready.Models), Processes: int(ready.Processes), Documents: []LegacySummary{},
	}
	rows, err := s.q.ListLegacyDocuments(ctx, customerID)
	if err != nil {
		return MigrationOverview{}, fmt.Errorf("membaca dokumen lama: %w", err)
	}
	for _, r := range rows {
		l := LegacySummary{
			ID: r.ID, Number: r.Number, DocumentTypeKey: deref(r.DocumentTypeKey), DocumentType: r.DocumentTypeName,
			Revision: int(r.Revision), Rows: int(r.Rows), EnteredBy: r.EnteredBy, EnteredAt: r.EnteredAt,
			VerifiedBy: deref(r.VerifiedBy), VerifiedAt: r.VerifiedAt, State: LegacyToVerify,
		}
		switch {
		case r.VerifiedAt != nil:
			l.State = LegacyCompleted
		case r.Rows == 0:
			l.State = LegacyEmpty
		}
		out.Documents = append(out.Documents, l)
	}
	return out, nil
}

// legacyTypeKeys adalah jenis dokumen yang bisa dimigrasi, urut cetak.
var legacyTypeKeys = []string{numbering.KeyPFD, numbering.KeyPFMEA, numbering.KeyCP}

// MigrateSet memasukkan satu set dokumen lama sekaligus (E-10, D-215,
// D-216). Semua atau tidak sama sekali: bila satu dokumen ditolak, nomor yang
// baru didaftarkan ikut batal. Hasilnya urut PFD, PFMEA, CP.
func (s *Service) MigrateSet(ctx context.Context, actor access.Actor, in LegacySet) ([]Document, error) {
	if !db.IsUUID(in.CustomerID) {
		return nil, problems(invalid("customer_id", problem.InvalidFormat, "Choose an existing customer."))
	}
	if err := s.authz.Authorize(ctx, actor, access.MigrateDocuments, access.Resource{CustomerID: in.CustomerID}); err != nil {
		return nil, err
	}
	docs, err := checkLegacySet(in)
	if err != nil {
		return nil, err
	}

	var ids []string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := requireMigrationMode(ctx, q); err != nil {
			return err
		}
		texts := map[string]string{}
		for _, d := range docs {
			texts[d.TypeKey] = d.Number
		}
		numbers, err := s.numbers.RegisterLegacySet(ctx, q, actor, numbering.LegacySetInput{
			CustomerID: in.CustomerID, OneSet: in.OneSet, Numbers: texts, Field: "documents",
		})
		if err != nil {
			return err
		}
		if err := checkSameSet(docs, numbers); err != nil {
			return err
		}
		for _, d := range docs {
			n := numbers[d.TypeKey]
			id, err := s.migrateOne(ctx, q, actor, LegacyDocument{
				NewDocument:   NewDocument{NumberID: n.ID, ModelIDs: in.ModelIDs, PartNumberIDs: in.PartNumberIDs},
				Revision:      d.Revision,
				EffectiveDate: d.EffectiveDate,
				PreparedBy:    d.PreparedBy,
				ApprovedBy:    d.ApprovedBy,
			})
			if err != nil {
				return prefixFields(err, d.TypeKey, n.Number)
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, result(err, "memasukkan set dokumen lama")
	}
	out := make([]Document, 0, len(ids))
	for _, id := range ids {
		d, err := getDocument(ctx, s.q, id)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// checkLegacySet memeriksa baris isian: setiap jenis paling banyak sekali,
// lalu mengurutkannya PFD, PFMEA, CP.
func checkLegacySet(in LegacySet) ([]LegacySetDocument, error) {
	if len(in.Documents) == 0 {
		return nil, problems(invalid("documents", problem.Required, "Add at least one document of the set."))
	}
	seen := map[string]bool{}
	for _, d := range in.Documents {
		field := "documents." + d.TypeKey + ".document_type_key"
		if !slices.Contains(legacyTypeKeys, d.TypeKey) || seen[d.TypeKey] {
			return nil, problems(invalid(field, problem.InvalidFormat, "Each document of the set is a PFD, a PFMEA, or a CP, at most once."))
		}
		seen[d.TypeKey] = true
	}
	out := slices.Clone(in.Documents)
	slices.SortFunc(out, func(a, b LegacySetDocument) int {
		return slices.Index(legacyTypeKeys, a.TypeKey) - slices.Index(legacyTypeKeys, b.TypeKey)
	})
	return out, nil
}

// checkSameSet memastikan semua nomor isian satu batch: nomor baru selalu
// satu batch, tetapi nomor yang sudah terdaftar membawa batch-nya sendiri.
// Set dikenali dari batch (ListDocumentSet), jadi campuran ditolak
// (ASUMSI E-142).
func checkSameSet(docs []LegacySetDocument, numbers map[string]numbering.LegacyNumber) error {
	first := numbers[docs[0].TypeKey]
	for _, d := range docs[1:] {
		n := numbers[d.TypeKey]
		if n.BatchID != first.BatchID {
			// Pesan menunjuk nomor yang sudah terdaftar di set lain.
			field, other := d.TypeKey, n
			if n.New {
				field, other = docs[0].TypeKey, first
			}
			return problems(invalid("documents."+field+".number", problem.NotAllowed,
				fmt.Sprintf("%s is already registered in another set. Enter it together with the numbers of its own set.", other.Number)))
		}
	}
	return nil
}

// prefixFields menambah awalan documents.<jenis>. pada penolakan satu
// dokumen. Model dan part number milik seluruh set, jadi tetap.
func prefixFields(err error, typeKey, number string) error {
	var v *problem.ValidationError
	if !errors.As(err, &v) {
		return err
	}
	out := &problem.ValidationError{}
	for _, p := range v.Problems {
		switch {
		case strings.HasPrefix(p.Field, "model_ids"), strings.HasPrefix(p.Field, "part_number_ids"), p.Field == "form_set", p.Field == "route_id":
		case p.Field == "document_number_id":
			p.Field = "documents." + typeKey + ".number"
			if p.Code == problem.NotAllowed {
				p.Message = fmt.Sprintf("%s already has a document, or is Void, Obsolete, or Skipped.", number)
			}
		default:
			p.Field = "documents." + typeKey + "." + p.Field
		}
		out.Problems = append(out.Problems, p)
	}
	return out
}
