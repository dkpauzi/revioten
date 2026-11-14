package document

// Bagian per PCB (D-155 pola 2, referensi D-159). Satu model bisa terdiri
// dari beberapa PCB, masing-masing dengan alur proses sendiri, tetapi dengan
// satu nomor dokumen, satu revisi, dan satu persetujuan. Contoh SC-KT-040-00:
// PCB A, Heatsink Assy R/L, dan Base Assy R/L, dicetak berurutan dalam satu
// CP dengan judul per bagian.
//
// Bagian berlaku untuk seluruh set (PFD, PFMEA, CP): bagian yang sama di
// ketiga dokumen punya section_key yang sama, supaya perakitan dan cek
// PFMEA-CP berjalan per bagian (ASUMSI E-97). Setiap bagian punya nama dan
// part number pilihan dari part number dokumen; satu bagian boleh punya dua
// part number (R dan L).

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

const maxSectionName = 200

// Section adalah satu bagian dokumen.
type Section struct {
	ID          string // milik satu dokumen
	Key         string // sama untuk bagian yang sama di semua dokumen satu set
	Position    int    // urutan, mulai 1
	Name        string
	PartNumbers []PartRef // menurut nomor; tidak pernah nil
}

// NewSection adalah isian bagian baru.
type NewSection struct {
	Name          string
	PartNumberIDs []string
}

// SectionUpdate adalah isian perubahan bagian. Nama wajib; isian nil tetap (D-111).
type SectionUpdate struct {
	Name          string
	Position      *int
	PartNumberIDs *[]string
}

// rowSection memeriksa bagian baris baru: wajib bila dokumen punya bagian,
// dan harus bagian dokumen itu. Hasilnya nil untuk dokumen tanpa bagian.
func rowSection(d Document, sectionID string) (*string, *problem.Problem) {
	if len(d.Sections) == 0 {
		if sectionID != "" {
			return nil, invalid("section_id", problem.NotAllowed, "This document has no sections.")
		}
		return nil, nil
	}
	if sectionID == "" {
		return nil, invalid("section_id", problem.Required, "Choose the section of the row.")
	}
	if !slices.ContainsFunc(d.Sections, func(s Section) bool { return s.ID == sectionID }) {
		return nil, invalid("section_id", problem.InvalidFormat, "Choose a section of this document.")
	}
	return &sectionID, nil
}

// sectionByKey mencari bagian satu dokumen menurut kuncinya. Untuk dokumen
// tanpa bagian kunci harus kosong, dan hasilnya "" (semua baris).
func sectionByKey(d Document, key string) (string, error) {
	if len(d.Sections) == 0 {
		if key != "" {
			return "", problems(invalid("section_key", problem.NotAllowed, "This document set has no sections."))
		}
		return "", nil
	}
	if key == "" {
		return "", problems(invalid("section_key", problem.Required, "Choose the section."))
	}
	for _, s := range d.Sections {
		if s.Key == key {
			return s.ID, nil
		}
	}
	return "", problems(invalid("section_key", problem.InvalidFormat, "Choose a section of this document set."))
}

// rowsIn memberi baris satu bagian menurut urutan; sectionID "" berarti
// semua baris (dokumen tanpa bagian).
func rowsIn(d Document, sectionID string) []Row {
	if sectionID == "" {
		return d.Rows
	}
	var out []Row
	for _, r := range d.Rows {
		if r.SectionID == sectionID {
			out = append(out, r)
		}
	}
	return out
}

// AddSection menambah satu bagian di akhir, di semua dokumen set (E-2).
// Bila ini bagian pertama, baris yang sudah ada masuk ke bagian ini.
func (s *Service) AddSection(ctx context.Context, actor access.Actor, id string, in NewSection) (Document, error) {
	return s.editSet(ctx, actor, id, func(q *db.Queries, d Document, set []Document) (string, map[string]string, error) {
		name, p := problem.Text("name", in.Name, true, maxSectionName)
		parts, partProblem := sectionParts(d, in.PartNumberIDs)
		if err := problems(p, partProblem); err != nil {
			return "", nil, err
		}
		if err := nameFree(d, name, ""); err != nil {
			return "", nil, err
		}
		if err := partsFree(d, parts, ""); err != nil {
			return "", nil, err
		}
		key, err := q.NewSectionKey(ctx)
		if err != nil {
			return "", nil, err
		}
		for _, doc := range set {
			sectionID, err := q.AddDocumentSection(ctx, db.AddDocumentSectionParams{DocumentID: doc.ID, SectionKey: key, Name: name})
			if err != nil {
				return "", nil, err
			}
			if len(doc.Sections) == 0 {
				if err := q.AssignRowsToSection(ctx, db.AssignRowsToSectionParams{DocumentID: doc.ID, SectionID: &sectionID}); err != nil {
					return "", nil, err
				}
				if err := q.AssignExclusionsToSection(ctx, db.AssignExclusionsToSectionParams{DocumentID: doc.ID, SectionID: &sectionID}); err != nil {
					return "", nil, err
				}
			}
		}
		for _, pid := range parts {
			if err := q.AddSectionPartNumber(ctx, db.AddSectionPartNumberParams{SectionKey: key, PartNumberID: pid}); err != nil {
				return "", nil, err
			}
		}
		return audit.ActionDocumentSectionAdded, map[string]string{"section": name, "part_numbers": partNumbers(d, parts)}, nil
	})
}

// UpdateSection mengganti nama, urutan, atau part number satu bagian, di
// semua dokumen set (E-2).
func (s *Service) UpdateSection(ctx context.Context, actor access.Actor, id, sectionID string, in SectionUpdate) (Document, error) {
	return s.editSet(ctx, actor, id, func(q *db.Queries, d Document, set []Document) (string, map[string]string, error) {
		i := slices.IndexFunc(d.Sections, func(x Section) bool { return x.ID == sectionID })
		if i < 0 {
			return "", nil, problem.ErrNotFound
		}
		old := d.Sections[i]
		name, p := problem.Text("name", in.Name, true, maxSectionName)
		var position *problem.Problem
		if in.Position != nil && (*in.Position < 1 || *in.Position > len(d.Sections)) {
			position = invalid("position", problem.InvalidFormat, "Choose a place between 1 and the number of sections.")
		}
		var parts []string
		var partProblem *problem.Problem
		if in.PartNumberIDs != nil {
			parts, partProblem = sectionParts(d, *in.PartNumberIDs)
		}
		if err := problems(p, position, partProblem); err != nil {
			return "", nil, err
		}
		details := map[string]string{"section": old.Name}
		if name != old.Name {
			if err := nameFree(d, name, old.Key); err != nil {
				return "", nil, err
			}
			if err := q.RenameDocumentSections(ctx, db.RenameDocumentSectionsParams{SectionKey: old.Key, Name: name}); err != nil {
				return "", nil, err
			}
			details["name"] = audit.Change(old.Name, name)
		}
		if in.Position != nil && *in.Position != old.Position {
			for _, doc := range set {
				if err := moveSection(ctx, q, doc, old.Key, *in.Position); err != nil {
					return "", nil, err
				}
			}
			details["position"] = audit.Change(strconv.Itoa(old.Position), strconv.Itoa(*in.Position))
		}
		if in.PartNumberIDs != nil {
			if err := partsFree(d, parts, old.Key); err != nil {
				return "", nil, err
			}
			if err := q.ClearSectionPartNumbers(ctx, old.Key); err != nil {
				return "", nil, err
			}
			for _, pid := range parts {
				if err := q.AddSectionPartNumber(ctx, db.AddSectionPartNumberParams{SectionKey: old.Key, PartNumberID: pid}); err != nil {
					return "", nil, err
				}
			}
			details["part_numbers"] = partNumbers(d, parts)
		}
		if len(details) == 1 {
			return "", nil, nil // tidak ada yang berubah
		}
		return audit.ActionDocumentSectionChanged, details, nil
	})
}

// RemoveSection menghapus satu bagian kosong dari semua dokumen set (E-2).
// Bagian yang masih punya baris tidak bisa dihapus.
func (s *Service) RemoveSection(ctx context.Context, actor access.Actor, id, sectionID string) (Document, error) {
	return s.editSet(ctx, actor, id, func(q *db.Queries, d Document, set []Document) (string, map[string]string, error) {
		i := slices.IndexFunc(d.Sections, func(x Section) bool { return x.ID == sectionID })
		if i < 0 {
			return "", nil, problem.ErrNotFound
		}
		section := d.Sections[i]
		n, err := q.CountSectionRows(ctx, section.Key)
		if err != nil {
			return "", nil, err
		}
		if n > 0 {
			return "", nil, problems(invalid("section", problem.NotAllowed, "Move or remove the rows of this section first."))
		}
		if err := q.DeleteSectionExclusions(ctx, section.Key); err != nil {
			return "", nil, err
		}
		if err := q.ClearSectionPartNumbers(ctx, section.Key); err != nil {
			return "", nil, err
		}
		if err := q.DeleteDocumentSections(ctx, section.Key); err != nil {
			return "", nil, err
		}
		// Urutan bagian lain dirapatkan lagi menjadi 1, 2, 3.
		for _, doc := range set {
			position := 0
			for _, x := range doc.Sections {
				if x.Key == section.Key {
					continue
				}
				position++
				if x.Position != position {
					if err := q.SetDocumentSectionPosition(ctx, db.SetDocumentSectionPositionParams{DocumentID: doc.ID, ID: x.ID, Position: int32(position)}); err != nil {
						return "", nil, err
					}
				}
			}
		}
		return audit.ActionDocumentSectionRemoved, map[string]string{"section": section.Name}, nil
	})
}

// moveSection memindahkan bagian berkunci key ke urutan to di satu dokumen.
// Batasan unik urutan ditunda sampai akhir transaksi.
func moveSection(ctx context.Context, q *db.Queries, doc Document, key string, to int) error {
	list := slices.Clone(doc.Sections)
	from := slices.IndexFunc(list, func(x Section) bool { return x.Key == key })
	if from < 0 {
		return nil
	}
	moving := list[from]
	list = slices.Delete(list, from, from+1)
	list = slices.Insert(list, to-1, moving)
	for i, x := range list {
		if x.Position != i+1 {
			if err := q.SetDocumentSectionPosition(ctx, db.SetDocumentSectionPositionParams{DocumentID: doc.ID, ID: x.ID, Position: int32(i + 1)}); err != nil {
				return err
			}
		}
	}
	return nil
}

// sectionParts memeriksa part number bagian: harus part number dokumen ini.
// Yang ganda dibuang.
func sectionParts(d Document, ids []string) ([]string, *problem.Problem) {
	var out []string
	for _, pid := range ids {
		if slices.Contains(out, pid) {
			continue
		}
		if !slices.ContainsFunc(d.PartNumbers, func(p PartRef) bool { return p.ID == pid }) {
			return nil, invalid("part_number_ids", problem.InvalidFormat, "Choose part numbers of this document.")
		}
		out = append(out, pid)
	}
	return out, nil
}

// partsFree memastikan setiap part number hanya milik satu bagian.
func partsFree(d Document, parts []string, except string) error {
	for _, x := range d.Sections {
		if x.Key == except {
			continue
		}
		for _, p := range x.PartNumbers {
			if slices.Contains(parts, p.ID) {
				return &problem.ConflictError{Problems: []problem.Problem{{Field: "part_number_ids", Code: problem.Taken,
					Message: "Part number " + p.Number + " already belongs to section " + x.Name + "."}}}
			}
		}
	}
	return nil
}

// nameFree memastikan nama bagian unik di dokumen ini.
func nameFree(d Document, name, except string) error {
	for _, x := range d.Sections {
		if x.Key != except && strings.EqualFold(x.Name, name) {
			return &problem.ConflictError{Problems: []problem.Problem{{Field: "name", Code: problem.Taken, Message: "Another section already has this name."}}}
		}
	}
	return nil
}

// partNumbers menulis nomor part untuk rincian audit, dipisah koma.
func partNumbers(d Document, ids []string) string {
	var out []string
	for _, p := range d.PartNumbers {
		if slices.Contains(ids, p.ID) {
			out = append(out, p.Number)
		}
	}
	return strings.Join(out, ", ")
}

// editSet menjalankan satu perubahan yang menyentuh semua dokumen set di
// dalam transaksi: hak E-2, semua dokumen set dikunci menurut urutan ID,
// lalu entri audit bila ada yang berubah.
func (s *Service) editSet(ctx context.Context, actor access.Actor, id string,
	change func(q *db.Queries, d Document, set []Document) (string, map[string]string, error)) (Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.EditDrafts, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Document{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		ids := setIDs(d)
		for _, docID := range ids {
			if err := q.LockDocument(ctx, docID); err != nil {
				return err
			}
		}
		set := make([]Document, 0, len(ids))
		var current Document
		for _, docID := range ids {
			doc, err := getDocument(ctx, q, docID)
			if err != nil {
				return err
			}
			if err := requireDraft(doc); err != nil {
				return err
			}
			if docID == id {
				current = doc
			}
			set = append(set, doc)
		}
		action, details, err := change(q, current, set)
		if err != nil || action == "" {
			return err
		}
		for _, doc := range set {
			if err := q.TouchDocument(ctx, doc.ID); err != nil {
				return err
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: action, TargetType: audit.TargetDocument, TargetID: id, Details: details})
	})
	if err != nil {
		return Document{}, result(err, "mengubah bagian dokumen")
	}
	return getDocument(ctx, s.q, id)
}

// setIDs memberi ID semua dokumen set, termasuk dokumen ini, menurut urutan
// ID supaya penguncian tidak saling menunggu.
func setIDs(d Document) []string {
	ids := []string{d.ID}
	for _, m := range d.Set {
		if m.ID != d.ID {
			ids = append(ids, m.ID)
		}
	}
	slices.Sort(ids)
	return ids
}
