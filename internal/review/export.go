package review

// Unduh daftar review berkala sebagai Excel (D-202, tombol "Download list").

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/clock"
)

// statusText memberi status satu set dalam bahasa Inggris untuk berkas Excel.
func statusText(s Set) string {
	switch s.Status {
	case StatusOverdue:
		return fmt.Sprintf("%d days overdue", -s.DaysLeft)
	case StatusDueSoon, StatusDueYear:
		return fmt.Sprintf("in %d days", s.DaysLeft)
	case StatusEOL:
		return "EOL"
	}
	return "not due"
}

// Export membuat berkas Excel daftar review berkala dengan saringan f (L-2).
// Kolomnya mengikuti layar: model, nomor, setiap dokumen dengan revisi dan
// tanggal rilisnya, jatuh tempo, status, dan engineer.
func (s *Service) Export(ctx context.Context, actor access.Actor, f Filter) ([]byte, string, error) {
	o, err := s.Sets(ctx, actor, f)
	if err != nil {
		return nil, "", err
	}
	book := excelize.NewFile()
	defer book.Close()
	const sheet = "Review"
	book.SetSheetName("Sheet1", sheet)
	header := []any{"Set", "Customer", "Models", "Documents", "PFD", "PFMEA", "Control Plan", "Review due", "Status", "Engineer", "Last review"}
	if err := book.SetSheetRow(sheet, "A1", &header); err != nil {
		return nil, "", err
	}
	for i, set := range o.Sets {
		cells := map[string]string{}
		numbers := make([]string, 0, len(set.Documents))
		for _, d := range set.Documents {
			numbers = append(numbers, d.Number)
			text := fmt.Sprintf("rev %02d · %s", d.Revision, d.LastRelease.Format(time.DateOnly))
			if d.Overdue {
				text += " · overdue"
			}
			cells[d.TypeKey] = text
		}
		last := ""
		if set.LastReview != nil {
			last = set.LastReview.Format(time.DateOnly)
		}
		row := []any{
			fmt.Sprintf("%s · %04d", strings.ToUpper(set.CustomerCode), set.Sequence), strings.ToUpper(set.CustomerCode),
			strings.Join(set.Models, ", "), strings.Join(numbers, ", "), cells["pfd"], cells["pfmea"], cells["cp"],
			set.DueOn.Format(time.DateOnly), statusText(set), set.Engineer, last,
		}
		if err := book.SetSheetRow(sheet, fmt.Sprintf("A%d", i+2), &row); err != nil {
			return nil, "", err
		}
	}
	_ = book.SetColWidth(sheet, "A", "K", 18)
	var buf bytes.Buffer
	if err := book.Write(&buf); err != nil {
		return nil, "", fmt.Errorf("menulis Excel review: %w", err)
	}
	name := fmt.Sprintf("periodic-review-%s.xlsx", clock.Date(s.now()).Format("20060102"))
	return buf.Bytes(), name, nil
}
