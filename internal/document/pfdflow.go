package document

// Langkah PFD (C4, D-211, D-212). PFD diisi per dokumen tanpa template, dan
// langkahnya menjadi dasar proses di PFMEA dan CP: nomor langkah, proses,
// simbol alur, dan arah NG. CP membawa langkah PFD satu set-nya sebagai
// bagian isinya (Document.Flow), jadi langkah itu ikut tertulis di snapshot
// saat CP ditandatangani dan dirilis (D-212). Cetakan revisi lama tetap
// menggambar alur yang berlaku saat revisi itu terbit.

import (
	"context"
	"fmt"
	"strings"

	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/template"
)

// FlowStep adalah satu langkah PFD. Satu nomor langkah boleh punya beberapa
// proses (di PFD Tokairika langkah 2 berisi IQC untuk beberapa material);
// setiap pasangan nomor dan proses adalah satu langkah di sini.
type FlowStep struct {
	SectionKey  string `json:"section_key,omitempty"` // bagian per PCB (D-155); "" tanpa bagian
	Operation   string `json:"operation"`
	ProcessID   string `json:"process_id,omitempty"`
	ProcessName string `json:"process_name,omitempty"`
	Symbol      string `json:"symbol,omitempty"` // isi kolom Flow Symbol PFD
	NG          string `json:"ng,omitempty"`     // isi kolom NG Disposition PFD
	// Lompatan (D-227): nomor langkah tujuan dan labelnya (ok, ng, yes, no).
	// omitempty: snapshot lama tanpa lompatan tetap sama isinya.
	Jump      string `json:"jump,omitempty"`
	JumpLabel string `json:"jump_label,omitempty"`
}

// key menandai satu langkah di dalam bagiannya.
func (s FlowStep) key() string { return s.SectionKey + "\x00" + s.Operation + "\x00" + s.ProcessID }

// carriesFlow melaporkan apakah dokumen jenis ini membawa langkah PFD di
// isinya. Hanya CP yang menggambar Process Flow dari PFD (D-122, D-212).
func carriesFlow(typeKey string) bool { return typeKey == "cp" }

// pfdOf memberi ID dokumen PFD satu set; "" bila set tanpa PFD.
func pfdOf(d Document) string {
	for _, m := range d.Set {
		if m.DocumentType.Key == "pfd" && m.ID != d.ID {
			return m.ID
		}
	}
	return ""
}

// loadFlow membaca langkah PFD satu set, menurut urutan baris PFD. Kosong
// (tidak nil) bila set tanpa PFD atau PFD-nya belum berisi.
func loadFlow(ctx context.Context, q *db.Queries, d Document) ([]FlowStep, error) {
	id := pfdOf(d)
	if id == "" {
		return []FlowStep{}, nil
	}
	pfd, err := getDocument(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("membaca PFD satu set: %w", err)
	}
	format, err := template.LoadFormatVersion(ctx, q, pfd.FormatVersionID)
	if err != nil {
		return nil, fmt.Errorf("membaca format PFD: %w", err)
	}
	return StepsOfPFD(pfd, format), nil
}

// StepsOfPFD menyusun langkah dari baris PFD: baris berurutan dengan nomor
// langkah dan proses yang sama adalah satu langkah. Baris tanpa nomor
// langkah dilewati, karena tidak bisa dicocokkan dengan PFMEA dan CP
// (ASUMSI E-138).
func StepsOfPFD(pfd Document, format template.FormatVersion) []FlowStep {
	symbol, ng, jump, label := "", "", "", ""
	for _, f := range format.Fields {
		switch f.Key {
		case "flow_symbol":
			symbol = f.ID
		case "ng_disposition":
			ng = f.ID
		case "jump_to":
			jump = f.ID
		case "jump_label":
			label = f.ID
		}
	}
	keys := map[string]string{}
	for _, s := range pfd.Sections {
		keys[s.ID] = s.Key
	}
	out := []FlowStep{}
	seen := map[string]bool{}
	for _, r := range pfd.Rows {
		op := strings.TrimSpace(r.OperationNumber)
		if op == "" {
			continue
		}
		step := FlowStep{SectionKey: keys[r.SectionID], Operation: op, ProcessID: r.ProcessID, ProcessName: r.ProcessName,
			Symbol: cellText(r.Content[symbol]), NG: cellText(r.Content[ng]),
			Jump: cellText(r.Content[jump]), JumpLabel: cellText(r.Content[label])}
		if seen[step.key()] {
			continue
		}
		seen[step.key()] = true
		out = append(out, step)
	}
	return out
}

// cellText memberi isi sel sebagai teks; kosong untuk sel tanpa isi.
func cellText(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
