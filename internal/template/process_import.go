package template

// Import isi process template dari berkas Excel (I3). PPAP Engineer (E-6,
// D-64) mengunduh template, mengisi satu poin per baris di sheet PFMEA dan
// Control Plan, lalu mengunggahnya. Preview melaporkan setiap baris tanpa
// menyimpan apa pun, termasuk versi draft baru. Import menyimpan poin baru ke
// versi draft process template prosesnya lewat AddSubprocess dan AddPoint,
// jadi aturan, hak, dan audit log sama dengan layar Process templates. Versi
// terbit tidak pernah diubah (M3.4): bila versi terakhir sudah terbit, versi
// berikutnya dibuat dulu sebagai salinannya. Poin yang sudah ada dilewati dan
// tidak diubah, jadi berkas yang sama aman diunggah lagi sesudah reset
// database. PFD tidak punya poin template (D-214), dan varian poin tetap
// ditambah di layar (rencana I3).

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Nama sheet di template import isi.
const (
	SheetContentPFMEA = "PFMEA"
	SheetContentCP    = "Control Plan"
)

// Batas berkas import isi.
const (
	MaxContentImportBytes = 10 << 20 // 10 MB
	MaxContentImportRows  = 5000
)

// Hasil satu baris import isi.
const (
	ContentRowNew      = "new"      // poin baru (disimpan bila Import)
	ContentRowExisting = "existing" // poin yang sama sudah ada, tidak diubah
	ContentRowInvalid  = "invalid"  // ada isian yang salah; lihat Problems
)

// Perubahan versi untuk satu proses.
const (
	VersionNewTemplate = "new_template" // proses belum punya template: template dan versi 1 dibuat
	VersionNewVersion  = "new_version"  // versi terakhir sudah terbit: versi berikutnya dibuat sebagai salinan
	VersionDraft       = "draft"        // poin masuk ke versi draft yang sudah ada
)

// Judul kolom tetap. Kolom isi menyusul, diambil dari format yang berlaku.
const (
	colProcess    = "Process"
	colSubprocess = "Sub-process"
	colText       = "Point text"
	colScope      = "Use point for"
	colCustomer   = "Customer code"
	colGroup      = "Model group"
	colModel      = "Model"
)

var contentFixedHeaders = []string{colProcess, colSubprocess, colText, colScope, colCustomer, colGroup, colModel}

// ContentImportRow adalah satu baris berkas beserta hasilnya.
type ContentImportRow struct {
	Sheet       string // PFMEA atau Control Plan
	Row         int    // nomor baris di Excel, mulai dari 2
	Process     string // nama tampilan proses bila ditemukan, selain itu isi sel
	Subprocess  string
	Text        string
	ScopeLevel  string // general, customer, model_group, atau model; kosong bila salah
	ScopeTarget string // kode customer, nama group model, atau nama model
	Status      string
	Problems    []problem.Problem
}

// ContentImportVersion adalah versi yang menerima poin baru satu proses.
type ContentImportVersion struct {
	Process string
	Version int
	Change  string // new_template, new_version, atau draft
	Points  int    // banyaknya poin baru
}

// ContentImportReport adalah laporan satu Preview atau Import.
type ContentImportReport struct {
	Rows      []ContentImportRow
	Versions  []ContentImportVersion // menurut urutan proses pertama kali muncul
	New       int
	Existing  int
	Invalid   int
	Committed bool
}

// importColumn adalah satu kolom format yang bisa diisi poin (D-146).
type importColumn struct {
	Key   string
	Kind  string // text atau rating
	Label string
}

// contentSheet adalah satu sheet import beserta jenis dokumennya.
type contentSheet struct {
	Name    string
	TypeKey string // pfmea atau cp
	TypeID  string
	Columns []importColumn
}

// rawContentRow adalah isi sel satu baris sebelum diperiksa.
type rawContentRow struct {
	sheet   *contentSheet
	row     int
	fixed   map[string]string // judul kolom tetap ke isi sel
	content map[string]string // key kolom ke isi sel
}

// ContentImportTemplate membuat berkas Excel template (E-6): sheet PFMEA dan
// Control Plan dengan judul kolom, dan sheet Help.
func (s *ProcessService) ContentImportTemplate(ctx context.Context, actor access.Actor) ([]byte, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return nil, err
	}
	sheets, err := s.contentSheets(ctx)
	if err != nil {
		return nil, err
	}
	f := excelize.NewFile()
	defer f.Close()
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	for i, sh := range sheets {
		if i == 0 {
			if err := f.SetSheetName("Sheet1", sh.Name); err != nil {
				return nil, err
			}
		} else if _, err := f.NewSheet(sh.Name); err != nil {
			return nil, err
		}
		headers := []any{}
		for _, h := range contentFixedHeaders {
			headers = append(headers, h)
		}
		for _, c := range sh.Columns {
			headers = append(headers, c.Label)
		}
		_ = f.SetSheetRow(sh.Name, "A1", &headers)
		last, _ := excelize.CoordinatesToCellName(len(headers), 1)
		_ = f.SetCellStyle(sh.Name, "A1", last, bold)
		lastCol, _ := excelize.ColumnNumberToName(len(headers))
		_ = f.SetColWidth(sh.Name, "A", lastCol, 24)
	}

	if _, err := f.NewSheet("Help"); err != nil {
		return nil, err
	}
	help := [][]any{
		{"Column", "What to fill in"},
		{colProcess, "Required. The name or an alias of a registered process. Import the processes first."},
		{colSubprocess, "Required. A sub-process with this name is used, or made when there is none."},
		{colText, fmt.Sprintf("Required, at most %d characters. A point with the same text, sub-process, and use already exists? Then the row is skipped.", maxPointText)},
		{colScope, "General (or empty), Customer, Model group, or Model."},
		{colCustomer, "Required for Customer, Model group, and Model: the customer code."},
		{colGroup, "For Model group: the name of a model group of that customer."},
		{colModel, "For Model: the name of a model of that customer."},
		{"Other columns", fmt.Sprintf("The content of the point per column of the form. Ratings are whole numbers from 1 to %d. Columns that fill themselves (operation number, process, Process Flow, RPN, AP), Relevant Document, and symbol columns are not imported.", maxRating)},
		{"Versions", "Points go to the draft version of the process template. When the last version is published, the next version is made first as its copy. Publish it with Publish version N."},
	}
	for i, line := range help {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		_ = f.SetSheetRow("Help", cell, &line)
	}
	_ = f.SetCellStyle("Help", "A1", "B1", bold)
	_ = f.SetColWidth("Help", "A", "A", 18)
	_ = f.SetColWidth("Help", "B", "B", 110)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("menulis template isi: %w", err)
	}
	return buf.Bytes(), nil
}

// contentSheets menyusun sheet PFMEA dan Control Plan beserta kolom isinya:
// kolom baris bertipe teks atau rating yang punya key, dari versi terakhir
// setiap format template yang tidak disembunyikan (D-146, D-218). Kolom
// dengan key yang sama di dua format hanya muncul sekali.
func (s *ProcessService) contentSheets(ctx context.Context) ([]contentSheet, error) {
	sheets := []contentSheet{{Name: SheetContentPFMEA, TypeKey: "pfmea"}, {Name: SheetContentCP, TypeKey: "cp"}}
	types, err := s.q.ListDocumentTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca jenis dokumen: %w", err)
	}
	formats, err := s.q.ListFormatTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca format template: %w", err)
	}
	versions, err := s.q.ListFormatVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca versi format: %w", err)
	}
	latest := map[string]db.FormatTemplateVersion{}
	for _, v := range versions {
		if old, ok := latest[v.TemplateID]; !ok || v.Version > old.Version {
			latest[v.TemplateID] = v
		}
	}
	for i := range sheets {
		sh := &sheets[i]
		for _, t := range types {
			if derefText(t.Key) == sh.TypeKey {
				sh.TypeID = t.ID
			}
		}
		seen := map[string]bool{}
		for _, ft := range formats {
			v, ok := latest[ft.ID]
			if ft.Hidden || ft.DocumentTypeID != sh.TypeID || !ok {
				continue
			}
			fields, err := s.q.ListVersionFields(ctx, v.ID)
			if err != nil {
				return nil, fmt.Errorf("membaca kolom format: %w", err)
			}
			for _, fl := range fields {
				key := derefText(fl.Key)
				if fl.Area != "row" || fl.Hidden || key == "" || key == KeyRelevantDocument || seen[key] ||
					(fl.Kind != "text" && fl.Kind != "rating") {
					continue
				}
				seen[key] = true
				sh.Columns = append(sh.Columns, importColumn{Key: key, Kind: fl.Kind, Label: fl.Label})
			}
		}
	}
	return sheets, nil
}

// ImportContent membaca berkas (E-6). commit false memberi laporan saja
// (Preview); commit true juga menyimpan poin baru.
func (s *ProcessService) ImportContent(ctx context.Context, actor access.Actor, data []byte, commit bool) (ContentImportReport, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageTemplates, access.Resource{}); err != nil {
		return ContentImportReport{}, err
	}
	sheets, err := s.contentSheets(ctx)
	if err != nil {
		return ContentImportReport{}, err
	}
	raws, err := readContentBook(data, sheets)
	if err != nil {
		return ContentImportReport{}, err
	}
	im, err := s.newContentImporter(ctx, actor, commit)
	if err != nil {
		return ContentImportReport{}, err
	}
	report := ContentImportReport{Rows: make([]ContentImportRow, 0, len(raws)), Committed: commit}
	for _, raw := range raws {
		row, err := im.importRow(ctx, raw)
		if err != nil {
			return ContentImportReport{}, err
		}
		switch row.Status {
		case ContentRowNew:
			report.New++
		case ContentRowExisting:
			report.Existing++
		default:
			report.Invalid++
		}
		report.Rows = append(report.Rows, row)
	}
	report.Versions = []ContentImportVersion{}
	for _, id := range im.order {
		if p := im.procs[id]; p.added > 0 {
			report.Versions = append(report.Versions, ContentImportVersion{Process: p.name, Version: p.targetVersion(), Change: p.change(), Points: p.added})
		}
	}
	return report, nil
}

// contentImporter menyimpan keadaan satu import: proses, customer, dan isi
// versi setiap proses, termasuk yang akan dibuat baris sebelumnya, supaya
// Preview dan Import memberi hasil yang sama.
type contentImporter struct {
	s         *ProcessService
	actor     access.Actor
	commit    bool
	processes map[string]ProcessRef // nama atau alias (huruf kecil) ke proses
	customers map[string]string     // kode customer, juga kode lama (huruf kecil), ke id
	codes     map[string]string     // id customer ke kode sekarang
	groups    map[string]map[string]string
	models    map[string]map[string]string
	procs     map[string]*procState // id proses ke keadaannya
	order     []string              // id proses menurut kemunculan pertama
}

// procState adalah keadaan process template satu proses selama import.
type procState struct {
	name       string
	templateID string // kosong bila belum ada template
	versionID  string // versi terakhir; kosong bila belum ada template
	version    int    // nomor versi terakhir
	draft      bool   // versi terakhir masih draft
	created    string // perubahan versi yang dibuat import ini (atau akan dibuat di Preview)
	subs       []*subState
	added      int
}

// subState adalah satu sub-proses di versi target.
type subState struct {
	id     string // kosong bila baru akan dibuat (Preview)
	name   string
	hidden bool
	count  map[string]int  // id jenis dokumen ke banyaknya poin
	points map[string]bool // kunci poin yang sudah ada
}

// targetVersion adalah nomor versi yang menerima poin. Versi baru (juga di
// Preview) sudah menaikkan version.
func (p *procState) targetVersion() int {
	return p.version
}

func (p *procState) change() string {
	if p.created != "" {
		return p.created
	}
	return VersionDraft
}

func (p *procState) sub(name string) *subState {
	for _, sb := range p.subs {
		if strings.EqualFold(sb.name, name) {
			return sb
		}
	}
	return nil
}

// pointKey adalah identitas poin untuk melewati baris yang sudah ada: jenis
// dokumen, teks (huruf kecil), dan cakupan.
func pointKey(typeID, text, level, targetID string) string {
	return typeID + "|" + strings.ToLower(text) + "|" + level + "|" + targetID
}

func (s *ProcessService) newContentImporter(ctx context.Context, actor access.Actor, commit bool) (*contentImporter, error) {
	im := &contentImporter{s: s, actor: actor, commit: commit, processes: map[string]ProcessRef{},
		customers: map[string]string{}, codes: map[string]string{},
		groups: map[string]map[string]string{}, models: map[string]map[string]string{}, procs: map[string]*procState{}}
	processes, err := s.q.ListProcesses(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca proses: %w", err)
	}
	names := map[string]string{}
	for _, p := range processes {
		names[p.ID] = p.Name
		im.processes[strings.ToLower(p.Name)] = ProcessRef{ID: p.ID, Name: p.Name}
	}
	aliases, err := s.q.ListProcessAliases(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca alias proses: %w", err)
	}
	for _, a := range aliases {
		im.processes[strings.ToLower(a.Name)] = ProcessRef{ID: a.ProcessID, Name: names[a.ProcessID]}
	}
	customers, err := s.q.ListCustomers(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca customer: %w", err)
	}
	for _, c := range customers {
		im.customers[strings.ToLower(c.Code)] = c.ID
		im.codes[c.ID] = c.Code
	}
	// Kode lama juga dikenali (E-11), tetapi kode sekarang menang.
	codes, err := s.q.ListCustomerCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca kode customer: %w", err)
	}
	for _, c := range codes {
		if _, used := im.customers[strings.ToLower(c.Code)]; !used {
			im.customers[strings.ToLower(c.Code)] = c.CustomerID
		}
	}
	templates, err := s.q.ListProcessTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca process template: %w", err)
	}
	versions, err := s.q.ListAllProcessVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca versi process template: %w", err)
	}
	last := map[string]db.ListAllProcessVersionsRow{}
	for _, v := range versions {
		if old, ok := last[v.TemplateID]; !ok || v.Version > old.Version {
			last[v.TemplateID] = v
		}
	}
	for _, t := range templates {
		p := &procState{name: t.ProcessName, templateID: t.ID}
		if v, ok := last[t.ID]; ok {
			p.versionID, p.version, p.draft = v.ID, int(v.Version), v.Status == StatusDraft
			full, err := getProcessVersion(ctx, s.q, v.ID)
			if err != nil {
				return nil, err
			}
			p.load(full)
		}
		im.procs[t.ProcessID] = p
	}
	return im, nil
}

// load mengisi sub-proses dan poin dari satu versi.
func (p *procState) load(v ProcessVersion) {
	p.subs = nil
	for _, sb := range v.Subprocesses {
		st := &subState{id: sb.ID, name: sb.Name, hidden: sb.Hidden, count: map[string]int{}, points: map[string]bool{}}
		for _, pt := range sb.Points {
			st.count[pt.DocumentType.ID]++
			st.points[pointKey(pt.DocumentType.ID, pt.Text, pt.Scope.Level, pt.Scope.TargetID)] = true
		}
		p.subs = append(p.subs, st)
	}
}

// importRow memeriksa, dan bila commit menyimpan, satu baris.
func (im *contentImporter) importRow(ctx context.Context, raw rawContentRow) (ContentImportRow, error) {
	row := ContentImportRow{Sheet: raw.sheet.Name, Row: raw.row, Problems: []problem.Problem{},
		Process: raw.fixed[colProcess], Subprocess: raw.fixed[colSubprocess], Text: raw.fixed[colText]}
	add := func(field, code, message string) {
		row.Problems = append(row.Problems, problem.Problem{Field: field, Code: code, Message: message})
	}

	var proc ProcessRef
	if row.Process == "" {
		add("process", problem.Required, "Write the process.")
	} else if found, ok := im.processes[strings.ToLower(row.Process)]; ok {
		proc, row.Process = found, found.Name
	} else {
		add("process", problem.InvalidFormat, fmt.Sprintf("The process %s is not registered. Import or add it on the Processes page first.", row.Process))
	}
	sub, p := problem.Text("subprocess", row.Subprocess, true, maxSubprocessName)
	row.Subprocess = sub
	if p != nil {
		row.Problems = append(row.Problems, *p)
	}
	text, p := problem.Text("text", row.Text, true, maxPointText)
	row.Text = text
	if p != nil {
		row.Problems = append(row.Problems, *p)
	}
	scope, err := im.scope(ctx, raw, &row, add)
	if err != nil {
		return row, err
	}
	content := map[string]any{}
	for _, c := range raw.sheet.Columns {
		value := strings.TrimSpace(raw.content[c.Key])
		if value == "" {
			continue
		}
		if c.Kind == "rating" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > maxRating {
				add(c.Key, problem.InvalidFormat, fmt.Sprintf("%s is a whole number from 1 to %d.", c.Label, maxRating))
				continue
			}
			content[c.Key] = n
			continue
		}
		if utf8.RuneCountInString(value) > MaxTextLength {
			add(c.Key, problem.TooLong, fmt.Sprintf("%s has at most %d characters.", c.Label, MaxTextLength))
			continue
		}
		content[c.Key] = value
	}
	if len(row.Problems) > 0 {
		row.Status = ContentRowInvalid
		return row, nil
	}

	state := im.procs[proc.ID]
	if state == nil {
		state = &procState{name: proc.Name}
		im.procs[proc.ID] = state
	}
	if !im.seen(proc.ID) {
		im.order = append(im.order, proc.ID)
	}
	st := state.sub(sub)
	key := pointKey(raw.sheet.TypeID, text, scope.Level, scope.TargetID)
	switch {
	case st != nil && st.hidden:
		add("subprocess", problem.NotAllowed, fmt.Sprintf("The sub-process %s is hidden. Show it on the Process templates page first.", st.name))
	case st != nil && st.points[key]:
		row.Status = ContentRowExisting
		return row, nil
	case st == nil && len(state.subs) >= maxSubprocesses:
		add("subprocess", problem.NotAllowed, fmt.Sprintf("A version has at most %d sub-processes.", maxSubprocesses))
	case st != nil && st.count[raw.sheet.TypeID] >= maxPointsPerList:
		add("text", problem.NotAllowed, fmt.Sprintf("A sub-process has at most %d points for one document type.", maxPointsPerList))
	}
	if len(row.Problems) > 0 {
		row.Status = ContentRowInvalid
		return row, nil
	}

	if im.commit {
		problems, err := im.save(ctx, proc, state, sub, NewPoint{
			DocumentTypeID: raw.sheet.TypeID, Text: text, Content: content,
			Scope: ScopeInput{Level: scope.Level, TargetID: scope.TargetID},
		})
		if err != nil {
			return row, err
		}
		if len(problems) > 0 {
			row.Status, row.Problems = ContentRowInvalid, problems
			return row, nil
		}
	} else {
		// Preview: catat apa yang akan dibuat, tanpa menyimpan.
		if state.created == "" && !state.draft {
			if state.templateID == "" {
				state.created = VersionNewTemplate
			} else {
				state.created = VersionNewVersion
			}
			state.version++
		}
		if st == nil {
			st = &subState{name: sub, count: map[string]int{}, points: map[string]bool{}}
			state.subs = append(state.subs, st)
		}
		st.count[raw.sheet.TypeID]++
		st.points[key] = true
	}
	state.added++
	row.Status = ContentRowNew
	return row, nil
}

func (im *contentImporter) seen(processID string) bool {
	for _, id := range im.order {
		if id == processID {
			return true
		}
	}
	return false
}

// save menyimpan satu poin: membuat template atau versi draft bila perlu,
// lalu sub-proses bila belum ada, lalu poinnya. Penolakan dari layanan
// dikembalikan sebagai masalah baris; error lain menghentikan import.
func (im *contentImporter) save(ctx context.Context, proc ProcessRef, state *procState, subName string, in NewPoint) ([]problem.Problem, error) {
	s, actor := im.s, im.actor
	if state.templateID == "" {
		pt, err := s.CreateProcessTemplate(ctx, actor, proc.ID)
		if err != nil {
			return rowProblems(err)
		}
		state.templateID, state.versionID, state.version, state.draft = pt.ID, pt.Versions[0].ID, 1, true
		state.created, state.subs = VersionNewTemplate, nil
	} else if !state.draft {
		v, err := s.NewProcessVersion(ctx, actor, state.templateID)
		if err != nil {
			return rowProblems(err)
		}
		state.versionID, state.version, state.draft, state.created = v.ID, v.Version, true, VersionNewVersion
		state.load(v)
	}
	st := state.sub(subName)
	if st == nil {
		v, err := s.AddSubprocess(ctx, actor, state.versionID, subName)
		if err != nil {
			return rowProblems(err)
		}
		state.load(v)
		st = state.subs[len(state.subs)-1]
	}
	in.SubprocessID = st.id
	v, err := s.AddPoint(ctx, actor, state.versionID, in)
	if err != nil {
		return rowProblems(err)
	}
	state.load(v)
	return nil, nil
}

// rowProblems memisahkan penolakan layanan (masalah baris) dari error lain.
func rowProblems(err error) ([]problem.Problem, error) {
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case errors.As(err, &invalid):
		return invalid.Problems, nil
	case errors.As(err, &conflict):
		return conflict.Problems, nil
	}
	return nil, err
}

// scope membaca kolom Use point for beserta sasarannya.
func (im *contentImporter) scope(ctx context.Context, raw rawContentRow, row *ContentImportRow,
	add func(field, code, message string)) (ScopeInput, error) {
	level := ""
	switch strings.ToLower(strings.Join(strings.Fields(raw.fixed[colScope]), " ")) {
	case "", "general":
		level = ScopeGeneral
	case "customer":
		level = ScopeCustomer
	case "model group", "model_group":
		level = ScopeModelGroup
	case "model":
		level = ScopeModel
	default:
		add("scope", problem.InvalidFormat, "Use point for is General, Customer, Model group, or Model.")
		return ScopeInput{}, nil
	}
	row.ScopeLevel = level
	if level == ScopeGeneral {
		return ScopeInput{Level: level}, nil
	}
	code := strings.TrimSpace(raw.fixed[colCustomer])
	customerID, ok := im.customers[strings.ToLower(code)]
	switch {
	case code == "":
		add("customer_code", problem.Required, "Write the customer code.")
		return ScopeInput{}, nil
	case !ok:
		add("customer_code", problem.InvalidFormat, fmt.Sprintf("There is no customer with the code %s.", code))
		return ScopeInput{}, nil
	}
	row.ScopeTarget = im.codes[customerID]
	if level == ScopeCustomer {
		return ScopeInput{Level: level, TargetID: customerID}, nil
	}
	if err := im.loadProducts(ctx, customerID); err != nil {
		return ScopeInput{}, err
	}
	column, field, list, what := colGroup, "model_group", im.groups[customerID], "model group"
	if level == ScopeModel {
		column, field, list, what = colModel, "model", im.models[customerID], "model"
	}
	name := strings.TrimSpace(raw.fixed[column])
	id, ok := list[strings.ToLower(name)]
	switch {
	case name == "":
		add(field, problem.Required, fmt.Sprintf("Write the %s.", what))
		return ScopeInput{}, nil
	case !ok:
		add(field, problem.InvalidFormat, fmt.Sprintf("The customer %s has no %s named %s.", row.ScopeTarget, what, name))
		return ScopeInput{}, nil
	}
	row.ScopeTarget = name + " (" + row.ScopeTarget + ")"
	return ScopeInput{Level: level, TargetID: id}, nil
}

// loadProducts membaca group model dan model satu customer, sekali saja.
func (im *contentImporter) loadProducts(ctx context.Context, customerID string) error {
	if _, ok := im.groups[customerID]; ok {
		return nil
	}
	groups, err := im.s.q.ListModelGroups(ctx, customerID)
	if err != nil {
		return fmt.Errorf("membaca group model: %w", err)
	}
	models, err := im.s.q.ListModels(ctx, customerID)
	if err != nil {
		return fmt.Errorf("membaca model: %w", err)
	}
	im.groups[customerID], im.models[customerID] = map[string]string{}, map[string]string{}
	for _, g := range groups {
		im.groups[customerID][strings.ToLower(g.Name)] = g.ID
	}
	for _, m := range models {
		im.models[customerID][strings.ToLower(m.Name)] = m.ID
	}
	return nil
}

// readContentBook membaca sheet PFMEA dan Control Plan. Sheet yang tidak ada
// dilewati, tetapi paling tidak satu harus ada. Judul kolom dicocokkan tanpa
// membedakan huruf besar kecil, dengan label kolom atau key tetapnya.
func readContentBook(data []byte, sheets []contentSheet) ([]rawContentRow, error) {
	if len(data) == 0 {
		return nil, contentFileProblem(problem.Required, "Choose the Excel file.")
	}
	if len(data) > MaxContentImportBytes {
		return nil, contentFileProblem(problem.TooLong, "The Excel file can be at most 10 MB.")
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, contentFileProblem(problem.InvalidFormat, "Upload an Excel workbook (.xlsx). Save an old .xls file as .xlsx first.")
	}
	defer book.Close()
	var out []rawContentRow
	found := false
	for i := range sheets {
		sh := &sheets[i]
		if index, err := book.GetSheetIndex(sh.Name); err != nil || index < 0 {
			continue
		}
		found = true
		lines, err := book.GetRows(sh.Name)
		if err != nil {
			return nil, contentFileProblem(problem.InvalidFormat, "The "+sh.Name+" sheet cannot be read.")
		}
		if len(lines) == 0 {
			continue
		}
		fixed := map[int]string{}
		content := map[int]string{}
		for x, h := range lines[0] {
			h = strings.ToLower(strings.TrimSpace(h))
			if h == "" {
				continue
			}
			known := false
			for _, name := range contentFixedHeaders {
				if strings.ToLower(name) == h {
					fixed[x], known = name, true
				}
			}
			for _, c := range sh.Columns {
				if !known && (strings.ToLower(c.Label) == h || c.Key == h) {
					content[x], known = c.Key, true
				}
			}
			if !known {
				return nil, contentFileProblem(problem.InvalidFormat, fmt.Sprintf(
					"The column %s on the %s sheet is not known. Download the template again.", strings.TrimSpace(lines[0][x]), sh.Name))
			}
		}
		for _, need := range []string{colProcess, colSubprocess, colText} {
			has := false
			for _, name := range fixed {
				has = has || name == need
			}
			if !has {
				return nil, contentFileProblem(problem.InvalidFormat, fmt.Sprintf(
					"The %s sheet must have the columns %s. Download the template.", sh.Name, strings.Join(contentFixedHeaders[:3], ", ")))
			}
		}
		for y, line := range lines[1:] {
			r := rawContentRow{sheet: sh, row: y + 2, fixed: map[string]string{}, content: map[string]string{}}
			empty := true
			for x, cell := range line {
				cell = strings.TrimSpace(cell)
				if cell == "" {
					continue
				}
				if name, ok := fixed[x]; ok {
					r.fixed[name], empty = cell, false
				} else if key, ok := content[x]; ok {
					r.content[key], empty = cell, false
				}
			}
			if !empty {
				out = append(out, r)
			}
		}
	}
	switch {
	case !found:
		return nil, contentFileProblem(problem.InvalidFormat, "The Excel file needs a PFMEA or Control Plan sheet. Download the template.")
	case len(out) == 0:
		return nil, contentFileProblem(problem.Required, "The Excel file has no rows to import.")
	case len(out) > MaxContentImportRows:
		return nil, contentFileProblem(problem.TooLong, fmt.Sprintf("Import at most %d rows at a time.", MaxContentImportRows))
	}
	return out, nil
}

func contentFileProblem(code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: code, Message: message}}}
}
