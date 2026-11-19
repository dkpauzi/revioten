package metrics

// Laporan kinerja PPAP Engineer (D-197, D-201, D-202): per minggu (Jumat
// sampai Kamis, karena laporan mingguan disampaikan setiap Jumat), per bulan,
// dan per tahun. Semua angka dihitung dari kejadian yang tercatat: permintaan
// perubahan, yokoten, audit log, tanda siap cetak, catatan cetak, unggahan
// scan, dan rilis. Tidak ada angka yang diketik (brief bagian 15, butir 5),
// dan tidak ada target (D-201): laporan hanya mengumpulkan lamanya.

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

// Kelompok pemicu di laporan (D-202). Kelompok diambil dari kunci laporan
// sumber perubahan (migrasi 00048), jadi tetap benar walau namanya diganti.
const (
	CategoryECR            = "ecr"
	CategoryCustomerClaim  = "customer_claim"
	CategoryAuditFinding   = "audit_finding"
	CategoryYokoten        = "yokoten"
	CategoryPeriodicReview = "periodic_review"
	CategoryOther          = "other"
)

// Categories adalah urutan kelompok pemicu di layar.
var Categories = []string{CategoryECR, CategoryCustomerClaim, CategoryAuditFinding, CategoryYokoten, CategoryPeriodicReview, CategoryOther}

// Category memberi kelompok laporan satu perubahan, untuk paket lain (dasbor
// admin, D-205).
func Category(reportKey, mechanism string) string { return category(reportKey, mechanism) }

// category memberi kelompok laporan satu perubahan.
func category(reportKey, mechanism string) string {
	if mechanism == "yokoten" {
		return CategoryYokoten
	}
	if reportKey != "" {
		return reportKey
	}
	return CategoryOther
}

// Period adalah satu periode laporan: From termasuk, To tidak termasuk,
// keduanya pukul 00.00 di zona pabrik.
type Period struct {
	Kind string
	From time.Time
	To   time.Time
}

// Week memberi nomor minggu laporan: minggu ISO hari Kamisnya, hari terakhir
// minggu laporan.
func (p Period) Week() int {
	_, w := p.To.AddDate(0, 0, -1).ISOWeek()
	return w
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

// previous memberi periode sebelumnya yang sejenis.
func (p Period) previous() Period {
	q, _ := PeriodOf(p.Kind, p.From.AddDate(0, 0, -1))
	return q
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

// Stages adalah lama setiap tahap revisi, dari diterima sampai terbit.
type Stages struct {
	// AcceptedToRelease: dari perubahan diterima sampai rilis dicatat.
	AcceptedToRelease Duration
	// EditToReady: dari edit pertama draft sampai ditandai siap cetak, atau
	// sampai dicetak untuk tanda tangan bila tidak ditandai (ASUMSI E-130).
	EditToReady Duration
	// WaitingToPrint: dari siap cetak sampai dicetak untuk tanda tangan.
	WaitingToPrint Duration
	// PrintToSigned: dari cetak untuk tanda tangan sampai scan diunggah.
	PrintToSigned Duration
	// SignedToReleased: dari scan diunggah sampai rilis dicatat.
	SignedToReleased Duration
}

// PersonPerformance adalah angka satu orang dalam periode.
type PersonPerformance struct {
	PersonID      string // "" untuk aplikasi lain (API integrasi)
	Name          string
	Submitted     int // permintaan perubahan yang diajukan (diterima PPAP)
	Accepted      int // permintaan orang ini yang diterima verifikator
	Rejected      int
	Open          int // dokumen dari permintaannya yang belum terbit di akhir periode
	Yokotens      int // yokoten yang dibuat
	Releases      int // revisi terbit yang dicetak orang ini untuk tanda tangan
	CustomerClaim int // permintaan berpemicu customer claim
	AuditFinding  int // permintaan berpemicu temuan audit
	Yokoten       int // permintaan hasil yokoten
	Stages        Stages
}

// ReleaseItem adalah satu revisi terbit dalam periode beserta lama setiap tahapnya.
type ReleaseItem struct {
	RevisionID string
	Number     string
	TypeKey    string
	Revision   int
	Category   string
	PrintedBy  string // "" bila tidak ada catatan cetak untuk tanda tangan
	AcceptedAt time.Time
	PrintedAt  *time.Time
	ScanAt     *time.Time
	ReleasedAt time.Time
	// Lama dalam jam; nil bila salah satu ujungnya tidak tercatat.
	AcceptedToReleaseHours *float64
	EditToReadyHours       *float64
	WaitingToPrintHours    *float64
	PrintToSignedHours     *float64
	SignedToReleasedHours  *float64
}

// YokotenItem adalah satu yokoten yang dibuat dalam periode beserta cakupannya.
type YokotenItem struct {
	Ref           string // YKT-0001
	Reason        string
	Started       time.Time
	Targets       int
	Applied       int // diterapkan (apply atau adapt)
	NotApplicable int
	Pending       int
	// DaysToClose: hari dari dibuat sampai keputusan terakhir; nil bila
	// masih ada target yang belum diputuskan.
	DaysToClose *int
}

// TrendPoint adalah satu titik tren: minggu laporan (untuk minggu dan
// bulan) atau bulan (untuk tahun).
type TrendPoint struct {
	Label                   string // W47 atau 2026-03
	From, To                time.Time
	Releases                int
	AcceptedToReleaseMedian float64 // jam; 0 bila tidak ada rilis
	ByCategory              map[string]int
}

// Summary adalah angka utama periode sebelumnya, untuk perbandingan.
type Summary struct {
	Submitted               int
	Releases                int
	AcceptedToReleaseMedian float64
}

// Performance adalah isi laporan kinerja satu periode.
type Performance struct {
	Period    Period
	PersonID  string // saringan satu orang; "" untuk semua
	Submitted int    // permintaan perubahan yang diajukan dalam periode
	Accepted  int    // permintaan yang diterima dalam periode
	Rejected  int
	Waiting   int // permintaan yang diajukan dalam periode dan belum diverifikasi
	// ByTrigger menghitung permintaan yang diajukan dalam periode per pemicu.
	ByTrigger []TriggerCount
	// ByCategory menghitung revisi yang terbit dalam periode per kelompok
	// pemicu: setiap revisi dihitung sekali, di bawah pemicu perubahannya.
	ByCategory     map[string]int
	Yokotens       int // yokoten yang dibuat
	YokotenTargets int // model target yokoten itu
	YokotenChanges int // permintaan perubahan hasil yokoten yang diajukan
	YokotenItems   []YokotenItem
	Releases       int // revisi terbit lewat perubahan
	OpenAtEnd      int // dokumen dari perubahan diterima yang belum terbit di akhir periode
	Stages         Stages
	Previous       Summary
	People         []PersonPerformance // tidak pernah nil
	ReleaseItems   []ReleaseItem       // tidak pernah nil
	Trend          []TrendPoint        // tidak pernah nil
}

// Performance menghitung laporan kinerja satu periode (L-5). customerID dan
// personID kosong berarti semua customer dan semua orang.
func (s *Service) Performance(ctx context.Context, actor access.Actor, kind string, date time.Time, customerID, personID string) (Performance, error) {
	if customerID != "" && !db.IsUUID(customerID) {
		return Performance{}, &problem.ValidationError{Problems: []problem.Problem{{Field: "customer_id", Code: problem.InvalidFormat, Message: "Choose an existing customer."}}}
	}
	if personID != "" && !db.IsUUID(personID) {
		return Performance{}, &problem.ValidationError{Problems: []problem.Problem{{Field: "person_id", Code: problem.InvalidFormat, Message: "Choose a person from the list."}}}
	}
	if err := s.authz.Authorize(ctx, actor, access.ViewPerformance, access.Resource{CustomerID: customerID}); err != nil {
		return Performance{}, err
	}
	period, err := PeriodOf(kind, date)
	if err != nil {
		return Performance{}, err
	}
	f := filter{customer: optional(customerID), person: personID}
	out, err := s.compute(ctx, period, f)
	if err != nil {
		return Performance{}, err
	}
	prev, err := s.compute(ctx, period.previous(), f)
	if err != nil {
		return Performance{}, err
	}
	out.Previous = Summary{Submitted: prev.Submitted, Releases: prev.Releases, AcceptedToReleaseMedian: prev.Stages.AcceptedToRelease.MedianHours}
	if out.Trend, err = s.trend(ctx, period, f); err != nil {
		return Performance{}, err
	}
	out.PersonID = personID
	return out, nil
}

type filter struct {
	customer *string
	person   string
}

// mine melaporkan apakah kejadian milik orang id masuk saringan orang.
func (f filter) mine(id *string) bool {
	if f.person == "" {
		return true
	}
	return id != nil && *id == f.person
}

// compute menghitung semua angka satu periode, tanpa tren dan perbandingan.
func (s *Service) compute(ctx context.Context, period Period, f filter) (Performance, error) {
	out := Performance{Period: period, ByTrigger: []TriggerCount{}, ByCategory: map[string]int{}, YokotenItems: []YokotenItem{},
		People: []PersonPerformance{}, ReleaseItems: []ReleaseItem{}, Trend: []TrendPoint{}}
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

	changes, err := s.q.PerformanceChanges(ctx, db.PerformanceChangesParams{FromAt: period.From, ToAt: period.To, CustomerID: f.customer})
	if err != nil {
		return Performance{}, fmt.Errorf("membaca perubahan: %w", err)
	}
	triggers := map[string]*TriggerCount{}
	for _, c := range changes {
		if !f.mine(c.RequestedByPersonID) {
			continue
		}
		p := person(c.RequestedByPersonID, c.RequestedBy)
		created := c.CreatedAt
		if in(&created) {
			out.Submitted++
			p.Submitted++
			if c.Status == "submitted" {
				out.Waiting++
			}
			switch category(c.ReportKey, c.Mechanism) {
			case CategoryYokoten:
				out.YokotenChanges++
				p.Yokoten++
			case CategoryCustomerClaim:
				p.CustomerClaim++
			case CategoryAuditFinding:
				p.AuditFinding++
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

	yokotens, err := s.q.PerformanceYokotens(ctx, db.PerformanceYokotensParams{FromAt: period.From, ToAt: period.To, CustomerID: f.customer})
	if err != nil {
		return Performance{}, fmt.Errorf("membaca yokoten: %w", err)
	}
	for _, y := range yokotens {
		id := y.CreatedByPersonID
		if !f.mine(&id) {
			continue
		}
		out.Yokotens++
		out.YokotenTargets += int(y.Targets)
		person(&id, y.CreatedBy).Yokotens++
		item := YokotenItem{Reason: y.Reason, Started: y.CreatedAt.UTC(), Targets: int(y.Targets), Applied: int(y.Applied),
			NotApplicable: int(y.NotApplicable), Pending: int(y.Pending)}
		if y.Seq != nil {
			item.Ref = fmt.Sprintf("YKT-%04d", *y.Seq)
		}
		if y.Pending == 0 && y.Targets > 0 {
			days := clock.Days(y.CreatedAt, y.LastDecidedAt)
			item.DaysToClose = &days
		}
		out.YokotenItems = append(out.YokotenItems, item)
	}

	releases, err := s.releases(ctx, period.From, period.To, f)
	if err != nil {
		return Performance{}, err
	}
	var all [5][]float64
	personal := map[*PersonPerformance]*[5][]float64{}
	for _, item := range releases {
		out.Releases++
		out.ByCategory[item.Category]++
		out.ReleaseItems = append(out.ReleaseItems, item.ReleaseItem)
		p := person(item.printerID, item.PrintedBy)
		p.Releases++
		if personal[p] == nil {
			personal[p] = &[5][]float64{}
		}
		for i, h := range item.hours() {
			if h != nil {
				all[i] = append(all[i], *h)
				personal[p][i] = append(personal[p][i], *h)
			}
		}
	}
	out.Stages = stagesOf(all)

	end := period.To
	open, err := s.q.PerformanceOpen(ctx, db.PerformanceOpenParams{At: &end, CustomerID: f.customer})
	if err != nil {
		return Performance{}, fmt.Errorf("membaca pekerjaan terbuka: %w", err)
	}
	for _, o := range open {
		if !f.mine(o.RequestedByPersonID) {
			continue
		}
		out.OpenAtEnd += int(o.Open)
		person(o.RequestedByPersonID, o.RequestedBy).Open += int(o.Open)
	}

	for _, p := range people {
		if v := personal[p]; v != nil {
			p.Stages = stagesOf(*v)
		}
		out.People = append(out.People, *p)
	}
	slices.SortFunc(out.People, func(a, b PersonPerformance) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// release adalah satu revisi terbit beserta pencetaknya.
type release struct {
	ReleaseItem
	printerID *string
}

func (r release) hours() [5]*float64 {
	return [5]*float64{r.AcceptedToReleaseHours, r.EditToReadyHours, r.WaitingToPrintHours, r.PrintToSignedHours, r.SignedToReleasedHours}
}

func stagesOf(v [5][]float64) Stages {
	return Stages{AcceptedToRelease: summarize(v[0]), EditToReady: summarize(v[1]), WaitingToPrint: summarize(v[2]),
		PrintToSigned: summarize(v[3]), SignedToReleased: summarize(v[4])}
}

// releases membaca revisi terbit dalam [from, to) beserta lama setiap tahapnya.
func (s *Service) releases(ctx context.Context, from, to time.Time, f filter) ([]release, error) {
	rows, err := s.q.PerformanceReleases(ctx, db.PerformanceReleasesParams{FromAt: &from, ToAt: &to, CustomerID: f.customer})
	if err != nil {
		return nil, fmt.Errorf("membaca rilis: %w", err)
	}
	out := make([]release, 0, len(rows))
	for _, r := range rows {
		if r.ReleasedAt == nil || !f.mine(r.PrintedByPersonID) {
			continue
		}
		item := ReleaseItem{RevisionID: r.ID, Number: r.Number, Revision: int(r.Revision), AcceptedAt: r.AcceptedAt.UTC(),
			ReleasedAt: r.ReleasedAt.UTC(), PrintedBy: deref(r.PrintedBy), Category: category(r.ReportKey, r.Mechanism)}
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
		ready := r.ReadyAt
		if ready == nil {
			// Tanpa tanda siap cetak, cetakan untuk tanda tangan dianggap
			// saat siap cetak, dan tahap menunggu cetak tidak dihitung.
			ready = r.PrintedAt
		} else {
			item.WaitingToPrintHours = hours(r.ReadyAt, r.PrintedAt)
		}
		item.AcceptedToReleaseHours = hours(&r.AcceptedAt, r.ReleasedAt)
		item.EditToReadyHours = hours(&r.FirstEditAt, ready)
		item.PrintToSignedHours = hours(r.PrintedAt, r.ScanUploadedAt)
		item.SignedToReleasedHours = hours(r.ScanUploadedAt, r.ReleasedAt)
		out = append(out, release{ReleaseItem: item, printerID: r.PrintedByPersonID})
	}
	return out, nil
}

// trend memberi titik tren: 8 minggu laporan sampai minggu ini, minggu
// laporan yang hari Kamisnya jatuh di bulan itu, atau 12 bulan tahun itu.
func (s *Service) trend(ctx context.Context, period Period, f filter) ([]TrendPoint, error) {
	var points []TrendPoint
	switch period.Kind {
	case PeriodWeek:
		for i := 7; i >= 0; i-- {
			p, _ := PeriodOf(PeriodWeek, period.From.AddDate(0, 0, -7*i))
			points = append(points, TrendPoint{Label: fmt.Sprintf("W%d", p.Week()), From: p.From, To: p.To})
		}
	case PeriodMonth:
		// Minggu laporan masuk bulan hari Kamisnya.
		w, _ := PeriodOf(PeriodWeek, period.From)
		for ; w.From.Before(period.To); w, _ = PeriodOf(PeriodWeek, w.To) {
			thursday := w.To.AddDate(0, 0, -1)
			if !thursday.Before(period.From) && thursday.Before(period.To) {
				points = append(points, TrendPoint{Label: fmt.Sprintf("W%d", w.Week()), From: w.From, To: w.To})
			}
		}
	case PeriodYear:
		for m := period.From; m.Before(period.To); m = m.AddDate(0, 1, 0) {
			points = append(points, TrendPoint{Label: m.Format("2006-01"), From: m, To: m.AddDate(0, 1, 0)})
		}
	}
	if len(points) == 0 {
		return []TrendPoint{}, nil
	}
	rels, err := s.releases(ctx, points[0].From, points[len(points)-1].To, f)
	if err != nil {
		return nil, err
	}
	for i := range points {
		p := &points[i]
		p.ByCategory = map[string]int{}
		var times []float64
		for _, r := range rels {
			if r.ReleasedAt.Before(p.From) || !r.ReleasedAt.Before(p.To) {
				continue
			}
			p.Releases++
			p.ByCategory[r.Category]++
			if r.AcceptedToReleaseHours != nil {
				times = append(times, *r.AcceptedToReleaseHours)
			}
		}
		p.AcceptedToReleaseMedian = summarize(times).MedianHours
	}
	return points, nil
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

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
