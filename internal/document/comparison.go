package document

// PFMEA dan CP berdampingan (M4.9, R-DOC-06, D-65).
//
// Konsistensi PFMEA dan CP diperiksa orang, bukan mesin aturan. Sistem
// menampilkan baris kedua dokumen per proses berdampingan, dan pemegang V-2
// (untuk saat ini PPAP Engineer) memberi tanda "dicek oleh, tanggal" per
// proses. Tanda berlaku untuk revisi kedua dokumen saat itu. Bila baris
// proses itu ditambah, diubah, atau dihapus sesudah dicek, tandanya
// ditampilkan perlu dicek ulang. Untuk itu tanda menyimpan sidik jari baris
// proses itu saat dicek.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Comparison adalah PFMEA dan CP satu set, dikelompokkan per proses.
type Comparison struct {
	PFMEA     Document
	CP        Document
	Processes []ProcessGroup // menurut kemunculan pertama; baris tanpa proses di akhir
}

// ProcessGroup adalah baris PFMEA dan CP satu proses di satu bagian beserta
// tanda dicek-nya.
type ProcessGroup struct {
	SectionKey  string // bagian (D-155); "" untuk dokumen tanpa bagian
	SectionName string
	ProcessID   string // "" untuk baris tanpa proses
	ProcessName string
	PFMEARows   []Row // tidak pernah nil
	CPRows      []Row // tidak pernah nil
	Check       *CheckMark
}

// CheckMark adalah tanda "dicek oleh, tanggal" satu proses.
type CheckMark struct {
	By            string // nama orang pemberi tanda
	At            time.Time
	PFMEARevision int
	CPRevision    int
	Stale         bool // baris proses ini berubah sesudah dicek
	// fingerprint adalah sidik jari baris proses saat dicek.
	fingerprint string
}

// sectionKeyOf memberi kunci bagian satu baris ("" untuk dokumen tanpa bagian).
func sectionKeyOf(d Document, r Row) string {
	for _, s := range d.Sections {
		if s.ID == r.SectionID {
			return s.Key
		}
	}
	return ""
}

// fingerprint memberi sidik jari baris satu proses di satu bagian PFMEA dan
// CP: SHA-256 dari ID dan versi setiap baris. Versi naik setiap baris
// disimpan (M4.3), jadi baris yang ditambah, diubah, atau dihapus mengubah
// sidik jarinya.
func fingerprint(pfmea, cp Document, sectionKey, processID string) string {
	h := sha256.New()
	for _, d := range []Document{pfmea, cp} {
		for _, r := range d.Rows {
			if r.ProcessID == processID && sectionKeyOf(d, r) == sectionKey {
				fmt.Fprintf(h, "%s:%s:%d\n", d.ID, r.ID, r.Version)
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// pair mencari dokumen PFMEA dan CP di set dokumen id.
func pair(ctx context.Context, q *db.Queries, d Document) (Document, Document, error) {
	ids := map[string]string{d.DocumentType.Key: d.ID}
	for _, m := range d.Set {
		ids[m.DocumentType.Key] = m.ID
	}
	if ids["pfmea"] == "" || ids["cp"] == "" {
		return Document{}, Document{}, problems(invalid("set", problem.NotAllowed,
			"The side-by-side check needs a PFMEA and a CP in the same document set."))
	}
	pfmea, err := getDocument(ctx, q, ids["pfmea"])
	if err != nil {
		return Document{}, Document{}, err
	}
	cp, err := getDocument(ctx, q, ids["cp"])
	return pfmea, cp, err
}

// Comparison menampilkan PFMEA dan CP set dokumen id per bagian dan proses (L-4).
func (s *Service) Comparison(ctx context.Context, actor access.Actor, id string) (Comparison, error) {
	d, err := s.Get(ctx, actor, id)
	if err != nil {
		return Comparison{}, err
	}
	c, err := comparison(ctx, s.q, d)
	if err != nil {
		return Comparison{}, result(err, "membaca perbandingan PFMEA dan CP")
	}
	return c, nil
}

func comparison(ctx context.Context, q *db.Queries, d Document) (Comparison, error) {
	pfmea, cp, err := pair(ctx, q, d)
	if err != nil {
		return Comparison{}, err
	}
	out, err := groupComparison(ctx, q, pfmea, cp)
	if err != nil {
		return Comparison{}, err
	}
	for i := range out.Processes {
		if g := &out.Processes[i]; g.Check != nil {
			g.Check.Stale = g.Check.fingerprint != fingerprint(pfmea, cp, g.SectionKey, g.ProcessID)
		}
	}
	return out, nil
}

// groupComparison mengelompokkan baris PFMEA dan CP per bagian dan proses,
// lalu memasang tanda dicek untuk revisi kedua dokumen itu. Stale dihitung
// pemanggil, karena sumber barisnya bisa dokumen atau snapshot (D-195).
func groupComparison(ctx context.Context, q *db.Queries, pfmea, cp Document) (Comparison, error) {
	out := Comparison{PFMEA: pfmea, CP: cp, Processes: []ProcessGroup{}}
	// Urutan dan nama bagian dari CP; PFMEA punya bagian yang sama (ASUMSI E-97).
	sectionPos, sectionName := map[string]int{}, map[string]string{}
	for _, doc := range []Document{pfmea, cp} {
		for _, s := range doc.Sections {
			sectionPos[s.Key], sectionName[s.Key] = s.Position, s.Name
		}
	}
	index := map[string]int{}
	group := func(doc Document, r Row) *ProcessGroup {
		key := sectionKeyOf(doc, r)
		k := key + "/" + r.ProcessID
		i, ok := index[k]
		if !ok {
			i = len(out.Processes)
			index[k] = i
			out.Processes = append(out.Processes, ProcessGroup{
				SectionKey: key, SectionName: sectionName[key], ProcessID: r.ProcessID, ProcessName: r.ProcessName,
				PFMEARows: []Row{}, CPRows: []Row{},
			})
		}
		return &out.Processes[i]
	}
	for _, r := range pfmea.Rows {
		g := group(pfmea, r)
		g.PFMEARows = append(g.PFMEARows, r)
	}
	for _, r := range cp.Rows {
		g := group(cp, r)
		g.CPRows = append(g.CPRows, r)
	}
	// Menurut urutan bagian; di setiap bagian, baris tanpa proses di akhir
	// karena tidak bisa dicek per proses.
	slices.SortStableFunc(out.Processes, func(a, b ProcessGroup) int {
		if pa, pb := sectionPos[a.SectionKey], sectionPos[b.SectionKey]; pa != pb {
			return pa - pb
		}
		switch {
		case a.ProcessID == "" && b.ProcessID != "":
			return 1
		case a.ProcessID != "" && b.ProcessID == "":
			return -1
		}
		return 0
	})

	checks, err := q.ListPfmeaCpChecks(ctx, db.ListPfmeaCpChecksParams{
		PfmeaDocumentID: pfmea.ID, CpDocumentID: cp.ID, PfmeaRevision: int16(pfmea.Revision), CpRevision: int16(cp.Revision),
	})
	if err != nil {
		return Comparison{}, err
	}
	for _, c := range checks {
		key := deref(c.SectionKey)
		i := slices.IndexFunc(out.Processes, func(g ProcessGroup) bool { return g.SectionKey == key && g.ProcessID == c.ProcessID })
		if i < 0 {
			continue // proses itu sudah tidak punya baris di bagian ini
		}
		out.Processes[i].Check = &CheckMark{
			By: c.CheckedBy, At: c.CheckedAt.UTC(), PFMEARevision: int(c.PfmeaRevision), CPRevision: int(c.CpRevision),
			fingerprint: c.RowsFingerprint,
		}
	}
	return out, nil
}

// MarkChecked memberi atau mencabut tanda dicek satu proses di satu bagian
// (V-2, D-65). sectionKey wajib bila set punya bagian (D-155). Memberi tanda
// pada proses yang perlu dicek ulang memperbarui tandanya.
func (s *Service) MarkChecked(ctx context.Context, actor access.Actor, id, sectionKey, processID string, checked bool) (Comparison, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return Comparison{}, err
	}
	if err := s.authz.Authorize(ctx, actor, access.CheckPFMEACP, access.Resource{CustomerID: d.CustomerID}); err != nil {
		return Comparison{}, err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		pfmea, cp, err := pair(ctx, q, d)
		if err != nil {
			return err
		}
		if _, err := sectionByKey(cp, sectionKey); err != nil {
			return err
		}
		inUse := func(doc Document) bool {
			return slices.ContainsFunc(doc.Rows, func(r Row) bool { return r.ProcessID == processID && sectionKeyOf(doc, r) == sectionKey })
		}
		if processID == "" || !db.IsUUID(processID) || (!inUse(pfmea) && !inUse(cp)) {
			return problems(invalid("process_id", problem.InvalidFormat, "Choose a process that has rows in the PFMEA or the CP."))
		}
		pfmeaRev, cpRev := int16(pfmea.Revision), int16(cp.Revision)
		var changed int64
		action := audit.ActionPFMEACPChecked
		if checked {
			changed, err = q.AddPfmeaCpCheck(ctx, db.AddPfmeaCpCheckParams{
				PfmeaDocumentID: pfmea.ID, CpDocumentID: cp.ID, ProcessID: processID,
				PfmeaRevision: pfmeaRev, CpRevision: cpRev, CheckedByPersonID: actor.PersonID,
				RowsFingerprint: fingerprint(pfmea, cp, sectionKey, processID), SectionKey: optional(sectionKey),
			})
		} else {
			action = audit.ActionPFMEACPUnchecked
			changed, err = q.RemovePfmeaCpCheck(ctx, db.RemovePfmeaCpCheckParams{
				PfmeaDocumentID: pfmea.ID, CpDocumentID: cp.ID, ProcessID: processID, PfmeaRevision: pfmeaRev, CpRevision: cpRev,
				SectionKey: optional(sectionKey),
			})
		}
		if err != nil || changed == 0 {
			return err // tidak ada yang berubah, jadi tidak ada entri audit
		}
		name, err := q.GetProcessDisplayName(ctx, processID)
		if err != nil {
			return err
		}
		details := map[string]string{"process": name, "pfmea": pfmea.Number, "cp": cp.Number}
		for _, sec := range cp.Sections {
			if sec.Key == sectionKey {
				details["section"] = sec.Name
			}
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: action, TargetType: audit.TargetDocument, TargetID: cp.ID, Details: details})
	})
	if err != nil {
		return Comparison{}, result(err, "memberi tanda dicek")
	}
	d, err = getDocument(ctx, s.q, id)
	if err != nil {
		return Comparison{}, err
	}
	return comparison(ctx, s.q, d)
}
