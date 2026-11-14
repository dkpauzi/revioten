package document

import (
	"testing"
)

// TestRowChangesMoved: memindah satu baris hanya menandai baris itu (LCS).
func TestRowChangesMoved(t *testing.T) {
	row := func(id string, pos int) SnapshotRow {
		return SnapshotRow{ID: id, Position: pos, Content: map[string]any{}}
	}
	old := []SnapshotRow{row("a", 1), row("b", 2), row("c", 3), row("d", 4)}
	cur := []SnapshotRow{row("d", 1), row("a", 2), row("b", 3), row("c", 4)}
	got := rowChanges(old, cur, nil, "")
	if len(got) != 1 || got[0].RowID != "d" || got[0].Status != RowMoved || got[0].OldPosition != 4 || got[0].NewPosition != 1 {
		t.Errorf("dipindah: %+v", got)
	}
	// Baris baru di atas tidak membuat baris lain dianggap dipindah.
	cur = []SnapshotRow{row("x", 1), row("a", 2), row("b", 3), row("c", 4), row("d", 5)}
	got = rowChanges(old, cur, nil, "")
	if len(got) != 1 || got[0].Status != RowAdded {
		t.Errorf("baris baru: %+v", got)
	}
}
