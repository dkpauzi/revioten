package metrics

// Unduh laporan kinerja sebagai Excel (D-202, "Download weekly report").

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/clock"
)

// CategoryNames adalah nama kelompok pemicu di berkas Excel.
var CategoryNames = map[string]string{
	CategoryECR: "ECN / ECR", CategoryCustomerClaim: "Customer claim", CategoryAuditFinding: "Internal audit finding",
	CategoryYokoten: "Yokoten", CategoryPeriodicReview: "3-year review", CategoryOther: "Other",
}

// days menulis jam sebagai hari dengan satu angka di belakang koma.
func days(h float64) float64 { return float64(int(h/24*10+0.5)) / 10 }

func hoursCell(h *float64) any {
	if h == nil {
		return ""
	}
	return float64(int(*h*10+0.5)) / 10
}

// Export membuat berkas Excel laporan kinerja (L-5): ringkasan, per orang,
// pemicu, revisi terbit, dan yokoten.
func (s *Service) Export(ctx context.Context, actor access.Actor, kind string, date time.Time, customerID, personID string) ([]byte, string, error) {
	p, err := s.Performance(ctx, actor, kind, date, customerID, personID)
	if err != nil {
		return nil, "", err
	}
	book := excelize.NewFile()
	defer book.Close()
	write := func(sheet string, rows [][]any) error {
		for i, row := range rows {
			if err := book.SetSheetRow(sheet, fmt.Sprintf("A%d", i+1), &row); err != nil {
				return err
			}
		}
		return book.SetColWidth(sheet, "A", "L", 20)
	}
	last := p.Period.To.AddDate(0, 0, -1)
	title := fmt.Sprintf("%s to %s", p.Period.From.Format(time.DateOnly), last.Format(time.DateOnly))
	if p.Period.Kind == PeriodWeek {
		title = fmt.Sprintf("Report week %d · %s", p.Period.Week(), title)
	}
	st := p.Stages
	summary := [][]any{
		{"PPAP Engineer performance", title},
		{"Requests received", p.Submitted},
		{"Accepted", p.Accepted},
		{"Rejected", p.Rejected},
		{"Waiting for verification", p.Waiting},
		{"Revisions released", p.Releases},
		{"Still open at end of period", p.OpenAtEnd},
		{"Yokoten made", p.Yokotens},
		{"Yokoten target models", p.YokotenTargets},
		{},
		{"Stage (median, hours)", "Median hours", "Median days", "Count"},
		{"Approved to released", st.AcceptedToRelease.MedianHours, days(st.AcceptedToRelease.MedianHours), st.AcceptedToRelease.Count},
		{"Edit to ready to print", st.EditToReady.MedianHours, days(st.EditToReady.MedianHours), st.EditToReady.Count},
		{"Waiting to print", st.WaitingToPrint.MedianHours, days(st.WaitingToPrint.MedianHours), st.WaitingToPrint.Count},
		{"Print to signed", st.PrintToSigned.MedianHours, days(st.PrintToSigned.MedianHours), st.PrintToSigned.Count},
		{"Signed to released", st.SignedToReleased.MedianHours, days(st.SignedToReleased.MedianHours), st.SignedToReleased.Count},
		{},
		{"Why documents changed (revisions released)", "Count"},
	}
	for _, c := range Categories {
		summary = append(summary, []any{CategoryNames[c], p.ByCategory[c]})
	}
	book.SetSheetName("Sheet1", "Summary")
	if err := write("Summary", summary); err != nil {
		return nil, "", err
	}

	people := [][]any{{"Engineer", "Requests received", "Released", "Open", "Approved to released (h)", "Edit to ready (h)", "Print to signed (h)", "Customer claim", "Internal audit", "Yokoten"}}
	for _, x := range p.People {
		people = append(people, []any{x.Name, x.Submitted, x.Releases, x.Open, x.Stages.AcceptedToRelease.MedianHours,
			x.Stages.EditToReady.MedianHours, x.Stages.PrintToSigned.MedianHours, x.CustomerClaim, x.AuditFinding, x.Yokoten})
	}
	if _, err := book.NewSheet("People"); err != nil {
		return nil, "", err
	}
	if err := write("People", people); err != nil {
		return nil, "", err
	}

	rels := [][]any{{"Number", "Revision", "Trigger", "Printed by", "Released at", "Approved to released (h)", "Edit to ready (h)", "Waiting to print (h)", "Print to signed (h)", "Signed to released (h)"}}
	for _, r := range p.ReleaseItems {
		rels = append(rels, []any{r.Number, r.Revision, CategoryNames[r.Category], r.PrintedBy, r.ReleasedAt.In(clock.Zone).Format("2006-01-02 15:04"),
			hoursCell(r.AcceptedToReleaseHours), hoursCell(r.EditToReadyHours), hoursCell(r.WaitingToPrintHours), hoursCell(r.PrintToSignedHours), hoursCell(r.SignedToReleasedHours)})
	}
	if _, err := book.NewSheet("Revisions"); err != nil {
		return nil, "", err
	}
	if err := write("Revisions", rels); err != nil {
		return nil, "", err
	}

	yok := [][]any{{"Yokoten", "Item", "Started", "Target models", "Revised", "Not applicable", "Open", "Days to close"}}
	for _, y := range p.YokotenItems {
		close := any("open")
		if y.DaysToClose != nil {
			close = *y.DaysToClose
		}
		yok = append(yok, []any{y.Ref, y.Reason, y.Started.Format(time.DateOnly), y.Targets, y.Applied, y.NotApplicable, y.Pending, close})
	}
	if _, err := book.NewSheet("Yokoten"); err != nil {
		return nil, "", err
	}
	if err := write("Yokoten", yok); err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	if err := book.Write(&buf); err != nil {
		return nil, "", fmt.Errorf("menulis Excel laporan: %w", err)
	}
	name := fmt.Sprintf("ppap-performance-%s-%s.xlsx", p.Period.Kind, p.Period.From.Format("20060102"))
	return buf.Bytes(), name, nil
}
