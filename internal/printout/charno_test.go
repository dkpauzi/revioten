package printout

import (
	"testing"

	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/template"
)

// Nomor karakteristik CP (D-222, seperti kolom F dan I SC-TR-020-04): No di
// depan PRODUCT CHARACTERISTICS menomori baris yang Product Characteristics-nya
// diisi, dan No di depan PROCESS CHARACTERISTICS menomori baris yang Process
// Characteristics-nya diisi. Nomor berurutan dan mulai lagi dari 1 di setiap
// proses (nomor langkah dan nama proses yang sama, berurutan). Karakteristik
// "--" mendapat "--"; yang kosong dibiarkan kosong.
func TestCharacteristicNumbers(t *testing.T) {
	format := template.FormatVersion{Fields: []template.Field{
		{ID: "pn", Area: "row", Kind: template.KindProductCharNo, Label: "No"},
		{ID: "pc", Area: "row", Kind: template.KindText, Key: "product_characteristic", Label: "Product Characteristics"},
		{ID: "qn", Area: "row", Kind: template.KindProcessCharNo, Label: "No"},
		{ID: "qc", Area: "row", Kind: template.KindText, Key: "process_characteristic", Label: "Process Characteristics"},
	}}
	row := func(id, op, process, product, proc string) document.SnapshotRow {
		c := map[string]any{}
		if product != "" {
			c["pc"] = product
		}
		if proc != "" {
			c["qc"] = proc
		}
		return document.SnapshotRow{ID: id, OperationNumber: op, ProcessName: process, Content: c}
	}
	rows := []document.SnapshotRow{
		row("a", "1", "Part Receiving", "--", "Verification Packaging"),
		row("b", "1", "Part Receiving", "--", "Verification actual document"),
		row("c", "1", "Part Receiving", "Label", ""),
		row("d", "2", "IQC", "--", "SAP Barcode Label"),
		row("e", "2", "IQC", "PCB Appearance", "--"),
		row("f", "2", "IQC", "Dimension PCB", "--"),
		row("g", "2", "IQC (PCB)", "--", "Packing Condition"),
	}
	got := charNumbers(format, rows)
	want := map[string][2]string{
		"a": {"--", "1"}, "b": {"--", "2"}, "c": {"1", ""},
		"d": {"--", "1"}, "e": {"1", "--"}, "f": {"2", "--"},
		"g": {"--", "1"}, // proses lain di langkah yang sama: mulai lagi dari 1
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("baris %s: %v, ingin %v", id, got[id], w)
		}
	}

	// Cetakan memakai nomor yang sama.
	sh := Build(Input{Format: format, Snapshot: document.Snapshot{DocumentType: "cp", Rows: rows}})
	if c := sh.Rows[4].Cells; c[0] != "1" || c[2] != "--" {
		t.Errorf("sel cetak baris e: %v", c)
	}
}
