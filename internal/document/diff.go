package document

// Diff dan riwayat (M5.6, R-REV-03). Dua revisi dibandingkan per baris dan
// per sel. Baris dicocokkan lewat ID barisnya yang tetap (R-DOC-02), jadi
// baris yang dipindah tetap dikenali sebagai baris yang sama. Sisi "ke" boleh
// kosong: artinya isi draft yang sedang dikerjakan.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/template"
)

// Jenis perubahan baris di diff.
const (
	RowAdded   = "added"
	RowRemoved = "removed"
	RowChanged = "changed"
	RowMoved   = "moved" // hanya urutannya yang berubah
)

// Diff adalah perbandingan dua revisi satu dokumen.
type Diff struct {
	From      int
	To        *int // nil berarti draft yang sedang dikerjakan
	ToDraft   bool
	ProcessID string // "" berarti semua proses
	Header    []CellChange
	Rows      []RowChange
}

// CellChange adalah satu sel atau field header yang berubah.
type CellChange struct {
	FieldID string
	Label   string
	Old     any // nil bila sebelumnya kosong
	New     any // nil bila sekarang kosong
}

// RowChange adalah satu baris yang berubah.
type RowChange struct {
	RowID           string
	Status          string
	OldPosition     int // 0 untuk baris baru
	NewPosition     int // 0 untuk baris yang dihapus
	ProcessID       string
	ProcessName     string
	OperationNumber string
	Cells           []CellChange // untuk changed; kosong untuk added, removed, dan moved
}

// Diff membandingkan revisi terbit from dengan revisi terbit to, atau dengan
// draft yang dikerjakan bila to nil (L-4). processID menyaring baris satu
// proses, untuk riwayat per proses.
func (s *Service) Diff(ctx context.Context, actor access.Actor, id string, from int, to *int, processID string) (Diff, error) {
	d, err := s.Get(ctx, actor, id)
	if err != nil {
		return Diff{}, err
	}
	old, err := s.releasedSnapshot(ctx, id, from, "from")
	if err != nil {
		return Diff{}, err
	}
	var cur Snapshot
	if to == nil {
		if d.Status != StatusDraft {
			return Diff{}, problems(invalid("to", problem.Required, "The document has no open draft. Choose a revision to compare with."))
		}
		cur = buildSnapshot(d)
		// Tulis lalu baca ulang, supaya angka di kedua sisi berbentuk sama.
		if cur, err = roundTrip(cur); err != nil {
			return Diff{}, err
		}
	} else if cur, err = s.releasedSnapshot(ctx, id, *to, "to"); err != nil {
		return Diff{}, err
	}
	format, err := template.LoadFormatVersion(ctx, s.q, cur.FormatVersionID)
	if err != nil {
		return Diff{}, err
	}
	labels := map[string]string{}
	for _, f := range format.Fields {
		labels[f.ID] = f.Label
	}
	out := Diff{From: from, To: to, ToDraft: to == nil, ProcessID: processID, Header: cellChanges(old.Header, cur.Header, labels), Rows: []RowChange{}}
	if processID != "" {
		out.Header = []CellChange{}
	}
	out.Rows = rowChanges(old.Rows, cur.Rows, labels, processID)
	return out, nil
}

// releasedSnapshot membaca snapshot satu revisi terbit.
func (s *Service) releasedSnapshot(ctx context.Context, id string, revision int, field string) (Snapshot, error) {
	r, err := s.q.GetDocumentRevision(ctx, db.GetDocumentRevisionParams{DocumentID: id, Revision: int16(revision)})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && r.Snapshot == nil) {
		return Snapshot{}, problems(invalid(field, problem.NotAllowed, fmt.Sprintf("Revision %d is not released.", revision)))
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("membaca revisi: %w", err)
	}
	var snap Snapshot
	if err := json.Unmarshal([]byte(*r.Snapshot), &snap); err != nil {
		return Snapshot{}, fmt.Errorf("membaca snapshot: %w", err)
	}
	return snap, nil
}

func roundTrip(s Snapshot) (Snapshot, error) {
	text, err := json.Marshal(s)
	if err != nil {
		return Snapshot{}, err
	}
	var out Snapshot
	err = json.Unmarshal(text, &out)
	return out, err
}

// cellChanges membandingkan dua isi per ID field, menurut urutan ID.
func cellChanges(old, cur map[string]any, labels map[string]string) []CellChange {
	ids := map[string]bool{}
	for k := range old {
		ids[k] = true
	}
	for k := range cur {
		ids[k] = true
	}
	keys := make([]string, 0, len(ids))
	for k := range ids {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []CellChange{}
	for _, k := range keys {
		a, b := blankToNil(old[k]), blankToNil(cur[k])
		if reflect.DeepEqual(a, b) {
			continue
		}
		label := labels[k]
		if label == "" {
			label = k
		}
		out = append(out, CellChange{FieldID: k, Label: label, Old: a, New: b})
	}
	return out
}

// blankToNil menganggap teks kosong sama dengan sel kosong.
func blankToNil(v any) any {
	if s, ok := v.(string); ok && s == "" {
		return nil
	}
	return v
}

// rowChanges mencocokkan baris lewat ID: baris baru, dihapus, diubah, atau
// hanya dipindah. Hasilnya menurut urutan baru, lalu baris yang dihapus
// menurut urutan lama.
func rowChanges(old, cur []SnapshotRow, labels map[string]string, processID string) []RowChange {
	before := map[string]SnapshotRow{}
	for _, r := range old {
		before[r.ID] = r
	}
	// Urutan relatif baris yang ada di kedua sisi. Baris yang hanya bergeser
	// karena ada baris baru atau dihapus di atasnya tidak dianggap dipindah.
	now := map[string]bool{}
	for _, r := range cur {
		now[r.ID] = true
	}
	var a, b []string
	for _, r := range old {
		if now[r.ID] {
			a = append(a, r.ID)
		}
	}
	for _, r := range cur {
		if _, ok := before[r.ID]; ok {
			b = append(b, r.ID)
		}
	}
	stay := longestCommon(a, b)
	seen := map[string]bool{}
	out := []RowChange{}
	keep := func(r SnapshotRow) bool { return processID == "" || r.ProcessID == processID }
	for _, r := range cur {
		seen[r.ID] = true
		o, existed := before[r.ID]
		if !keep(r) && !(existed && keep(o)) {
			continue
		}
		item := RowChange{RowID: r.ID, NewPosition: r.Position, ProcessID: r.ProcessID, ProcessName: r.ProcessName, OperationNumber: r.OperationNumber, Cells: []CellChange{}}
		if !existed {
			item.Status = RowAdded
			out = append(out, item)
			continue
		}
		item.OldPosition = o.Position
		item.Cells = cellChanges(o.Content, r.Content, labels)
		// Proses dan nomor operasi adalah atribut baris; perubahannya ikut tampil.
		if o.ProcessID != r.ProcessID {
			item.Cells = append(item.Cells, CellChange{FieldID: "process", Label: "Process", Old: nilIfBlank(o.ProcessName), New: nilIfBlank(r.ProcessName)})
		}
		if o.OperationNumber != r.OperationNumber {
			item.Cells = append(item.Cells, CellChange{FieldID: "operation_number", Label: "Operation number", Old: nilIfBlank(o.OperationNumber), New: nilIfBlank(r.OperationNumber)})
		}
		switch {
		case len(item.Cells) > 0:
			item.Status = RowChanged
		case !stay[r.ID]:
			item.Status = RowMoved
		default:
			continue
		}
		out = append(out, item)
	}
	for _, o := range old {
		if seen[o.ID] || !keep(o) {
			continue
		}
		out = append(out, RowChange{RowID: o.ID, Status: RowRemoved, OldPosition: o.Position, ProcessID: o.ProcessID, ProcessName: o.ProcessName, OperationNumber: o.OperationNumber, Cells: []CellChange{}})
	}
	return out
}

func nilIfBlank(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// longestCommon mencari subbarisan sama terpanjang (LCS) dari dua urutan ID.
// Baris di dalamnya tetap di tempatnya; baris di luarnya dipindah. Dengan
// begitu, memindah satu baris hanya menandai baris itu.
func longestCommon(a, b []string) map[string]bool {
	n, m := len(a), len(b)
	// l[i][j] adalah panjang LCS dari a[i:] dan b[j:].
	l := make([][]int, n+1)
	for i := range l {
		l[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				l[i][j] = l[i+1][j+1] + 1
			} else {
				l[i][j] = max(l[i+1][j], l[i][j+1])
			}
		}
	}
	out := map[string]bool{}
	for i, j := 0, 0; i < n && j < m; {
		switch {
		case a[i] == b[j]:
			out[a[i]] = true
			i, j = i+1, j+1
		case l[i+1][j] >= l[i][j+1]:
			i++
		default:
			j++
		}
	}
	return out
}
