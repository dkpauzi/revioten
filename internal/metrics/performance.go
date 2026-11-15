package metrics

// Laporan kinerja PPAP Engineer (D-197): per minggu (Jumat sampai Kamis,
// karena laporan mingguan disampaikan setiap Jumat), per bulan, dan per
// tahun. Semua angka dihitung dari kejadian yang tercatat: permintaan
// perubahan, yokoten, catatan cetak, unggahan scan, dan rilis. Tidak ada
// angka yang diketik (brief bagian 15, butir 5).

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/clock"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Jenis periode laporan.
const (
	PeriodWeek  = "week"
	PeriodMonth = "month"
	PeriodYear  = "year"
)

// WeekStart adalah hari pertama minggu laporan: Jumat (pernyataan pemilik,
// 15 November 2026). Minggu berakhir hari Kamis.
const WeekStart = time.Friday

// Period adalah satu periode laporan: From termasuk, To tidak termasuk,
// keduanya pukul 00.00 di zona pabrik.
type Period struct {
	Kind string
	From time.Time
	To   time.Time
}

// PeriodOf memberi periode jenis kind yang memuat tanggal date.
func PeriodOf(kind string, date time.Time) (Period, error) {
	d := clock.Date(date)
	switch kind {
	case PeriodWeek:
		back := (int(d.Weekday()) - int(WeekStart) + 7) % 7
		from := d.AddDate(0, 0, -back)
		return Period{Kind: kind, From: from, To: from.AddDate(0, 0, 7)}, nil
	case PeriodMonth:
		from := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, clock.Zone)
		return Period{Kind: kind, From: from, To: from.AddDate(0, 1, 0)}, nil
	case PeriodYear:
		from := time.Date(d.Year(), 1, 1, 0, 0, 0, 0, clock.Zone)
		return Period{Kind: kind, From: from, To: from.AddDate(1, 0, 0)}, nil
	}
	return Period{}, &problem.ValidationError{Problems: []problem.Problem{{Field: "period", Code: problem.InvalidFormat, Message: "Choose week, month, or year."}}}
}

// Duration meringkas lama satu tahap: berapa kali, rata-rata, dan median.
type Duration struct {
	Count        int
	AverageHours float64
	MedianHours  float64
}

// TriggerCount adalah jumlah permintaan perubahan dengan satu pemicu.
type TriggerCount struct {
	Name      string // nama sumber perubahan (R-CHG-01); "" bila tanpa sumber
	Mechanism string // manual, yokoten, template, sqam, atau integration
	Count     int
}

// PersonPerformance adalah angka satu orang dalam periode.
type PersonPerformance struct {
	PersonID          string // "" untuk aplikasi lain (API integrasi)
	Name              string
	Submitted         int // permintaan perubahan yang diajukan
	Accepted          int // permintaan orang ini yang diterima
	Rejected          int
	Yokotens          int // yokoten yang dibuat
	Releases          int // revisi terbit yang dicetak orang ini untuk tanda tangan
	AcceptedToRelease Duration
	EditToPrint       Duration
	PrintToSigned     Duration
}

// ReleaseItem adalah satu revisi terbit dalam periode beserta lama setiap tahapnya.
type ReleaseItem struct {
	RevisionID string
	Number     string
	TypeKey    string
	Revision   int
	PrintedBy  string // "" bila tidak ada catatan cetak untuk tanda tangan
	AcceptedAt time.Time
	PrintedAt  *time.Time
	ScanAt     *time.Time
	ReleasedAt time.Time
	// Lama dalam jam; nil bila salah satu ujungnya tidak tercatat.
	AcceptedToReleaseHours *float64
	EditToPrintHours       *float64
	PrintToSignedHours     *float64
}

// Performance adalah isi laporan kinerja satu periode.
type Performance struct {
	Period    Period
	Submitted int // permintaan perubahan yang diajukan dalam periode
	Accepted  int // permintaan yang diterima dalam periode
	Rejected  int
	// ByTrigger menghitung permintaan yang diajukan dalam periode per pemicu,
	// termasuk Customer claim dan Audit finding. Tidak pernah nil.
	ByTrigger      []TriggerCount
	Yokotens       int // yokoten yang dibuat
	YokotenTargets int // model target yokoten itu
	YokotenChanges int // permintaan perubahan hasil yokoten yang diajukan
	Releases       int // revisi terbit lewat perubahan
	// AcceptedToRelease: dari perubahan diterima sampai rilis dicatat.
	AcceptedToRelease Duration
	// EditToPrint: dari draft dibuka (mulai diedit) sampai dicetak untuk
	// tanda tangan (siap cetak). Cetak ulang dihitung dari cetakan terakhir
	// sebelum scan diunggah, yaitu cetakan yang ditandatangani (ASUMSI E-130).
	EditToPrint Duration
	// PrintToSigned: dari cetak untuk tanda tangan sampai scan bertanda
	// tangan diunggah.
	PrintToSigned Duration
	People        []PersonPerformance // tidak pernah nil
	ReleaseItems  []ReleaseItem       // tidak pernah nil
}

// Performance menghitung laporan kinerja satu periode (L-5). customerID
// kosong berarti semua customer.
func (s *Service) Performance(ctx context.Context, actor access.Actor, kind string, date time.Time, customerID string) (Performance, error) {
	if customerID != "" && !db.IsUUID(customerID) {
		return Performance{}, &problem.ValidationError{Problems: []problem.Problem{{Field: "customer_id", Code: problem.InvalidFormat, Message: "Choose an existing customer."}}}
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewPerformance, access.Resource{CustomerID: customerID}); err != nil {
		return Performance{}, err
	}
	period, err := PeriodOf(kind, date)
	if err != nil {
		return Performance{}, err
	}
	var filter *string
	if customerID != "" {
		filter = &customerID
	}
	out := Performance{Period: period, ByTrigger: []TriggerCount{}, People: []PersonPerformance{}, ReleaseItems: []ReleaseItem{}}
	people := map[string]*PersonPerformance{}
	person := func(id *string, name string) *PersonPerformance {
		key := ""
		if id != nil {
			key = *id
		}
		if key == "" && name == "" {
			name = "—"
		}
		p, ok := people[key+"/"+name]
		if !ok {
			p = &PersonPerformance{PersonID: key, Name: name}
			people[key+"/"+name] = p
		}
		return p
	}
	in := func(t *time.Time) bool { return t != nil && !t.Before(period.From) && t.Before(period.To) }

	changes, err := s.q.PerformanceChanges(ctx, db.PerformanceChangesParams{FromAt: period.From, ToAt: period.To, CustomerID: filter})
	if err != nil {
		return Performance{}, fmt.Errorf("membaca perubahan: %w", err)
	}
	triggers := map[string]*TriggerCount{}
	for _, c := range changes {
		p := person(c.RequestedByPersonID, c.RequestedBy)
		created := c.CreatedAt
		if in(&created) {
			out.Submitted++
			p.Submitted++
			if c.Mechanism == "yokoten" {
				out.YokotenChanges++
			}
			key := c.TriggerName + "/" + c.Mechanism
			tc, ok := triggers[key]
			if !ok {
				tc = &TriggerCount{Name: c.TriggerName, Mechanism: c.Mechanism}
				triggers[key] = tc
			}
			tc.Count++
		}
		if in(c.VerifiedAt) {
			switch c.Status {
			case "accepted":
				out.Accepted++
				p.Accepted++
			case "rejected":
				out.Rejected++
				p.Rejected++
			}
		}
	}
	for _, tc := range triggers {
		out.ByTrigger = append(out.ByTrigger, *tc)
	}
	slices.SortFunc(out.ByTrigger, func(a, b TriggerCount) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Name+a.Mechanism, b.Name+b.Mechanism)
	})

	yokotens, err := s.q.PerformanceYokotens(ctx, db.PerformanceYokotensParams{FromAt: period.From, ToAt: period.To, CustomerID: filter})
	if err != nil {
		return Performance{}, fmt.Errorf("membaca yokoten: %w", err)
	}
	for _, y := range yokotens {
		out.Yokotens++
		out.YokotenTargets += int(y.Targets)
		id := y.CreatedByPersonID
		person(&id, y.CreatedBy).Yokotens++
	}

	releases, err := s.q.PerformanceReleases(ctx, db.PerformanceReleasesParams{FromAt: &period.From, ToAt: &period.To, CustomerID: filter})
	if err != nil {
		return Performance{}, fmt.Errorf("membaca rilis: %w", err)
	}
	var all, personal [3]map[*PersonPerformance][]float64
	for i := range all {
		all[i], personal[i] = map[*PersonPerformance][]float64{}, map[*PersonPerformance][]float64{}
	}
	var totals [3][]float64
	for _, r := range releases {
		if r.ReleasedAt == nil {
			continue
		}
		item := ReleaseItem{RevisionID: r.ID, Number: r.Number, Revision: int(r.Revision), AcceptedAt: r.AcceptedAt.UTC(), ReleasedAt: r.ReleasedAt.UTC(),
			PrintedBy: deref(r.PrintedBy)}
		if r.TypeKey != nil {
			item.TypeKey = *r.TypeKey
		}
		if r.PrintedAt != nil {
			at := r.PrintedAt.UTC()
			item.PrintedAt = &at
		}
		if r.ScanUploadedAt != nil {
			at := r.ScanUploadedAt.UTC()
			item.ScanAt = &at
		}
		item.AcceptedToReleaseHours = hours(&r.AcceptedAt, r.ReleasedAt)
		item.EditToPrintHours = hours(&r.CreatedAt, r.PrintedAt)
		item.PrintToSignedHours = hours(r.PrintedAt, r.ScanUploadedAt)
		out.Releases++
		out.ReleaseItems = append(out.ReleaseItems, item)

		p := person(r.PrintedByPersonID, deref(r.PrintedBy))
		p.Releases++
		for i, h := range []*float64{item.AcceptedToReleaseHours, item.EditToPrintHours, item.PrintToSignedHours} {
			if h != nil {
				totals[i] = append(totals[i], *h)
				personal[i][p] = append(personal[i][p], *h)
			}
		}
	}
	out.AcceptedToRelease, out.EditToPrint, out.PrintToSigned = summarize(totals[0]), summarize(totals[1]), summarize(totals[2])
	for _, p := range people {
		p.AcceptedToRelease, p.EditToPrint, p.PrintToSigned = summarize(personal[0][p]), summarize(personal[1][p]), summarize(personal[2][p])
		out.People = append(out.People, *p)
	}
	slices.SortFunc(out.People, func(a, b PersonPerformance) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// hours memberi lama dari a ke b dalam jam; nil bila salah satu tidak ada
// atau b sebelum a.
func hours(a, b *time.Time) *float64 {
	if a == nil || b == nil || b.Before(*a) {
		return nil
	}
	h := b.Sub(*a).Hours()
	return &h
}

// summarize memberi jumlah, rata-rata, dan median.
func summarize(values []float64) Duration {
	if len(values) == 0 {
		return Duration{}
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	median := sorted[len(sorted)/2]
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	}
	return Duration{Count: len(sorted), AverageHours: round(sum / float64(len(sorted))), MedianHours: round(median)}
}

func round(v float64) float64 { return math.Round(v*100) / 100 }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
