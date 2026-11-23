package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/approval"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/customer"
	"github.com/dkpauzi/revioten/internal/document"
	"github.com/dkpauzi/revioten/internal/numbering"
	"github.com/dkpauzi/revioten/internal/process"
	"github.com/dkpauzi/revioten/internal/product"
	"github.com/dkpauzi/revioten/internal/settings"
	"github.com/dkpauzi/revioten/internal/template"
)

// fakePinger meniru database untuk /health. err nil berarti database hidup.
type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

// fakeCustomers meniru customer.Service, jadi test HTTP tidak butuh database.
type fakeCustomers struct {
	list      []customer.Customer
	listErr   error
	created   customer.Customer
	createErr error
	// gotInput dan gotActor adalah input dan aktor terakhir yang diterima.
	gotInput *customer.NewCustomer
	gotActor access.Actor
	// gotRequest adalah asal permintaan yang sampai ke layanan (untuk audit log).
	gotRequest audit.Request
	// Untuk operasi ubah, ganti kode, dan Project Group.
	groups    []customer.ProjectGroup
	group     customer.ProjectGroup
	gotID     string
	gotUpdate *customer.CustomerUpdate
	gotCode   string
	gotGroup  *customer.ProjectGroupInput
	gotRoute  string
}

func (f *fakeCustomers) List(ctx context.Context, actor access.Actor) ([]customer.Customer, error) {
	f.gotActor = actor
	f.gotRequest = audit.RequestFrom(ctx)
	return f.list, f.listErr
}

func (f *fakeCustomers) Create(_ context.Context, actor access.Actor, in customer.NewCustomer) (customer.Customer, error) {
	f.gotActor = actor
	f.gotInput = &in
	return f.created, f.createErr
}

func (f *fakeCustomers) Update(_ context.Context, actor access.Actor, id string, in customer.CustomerUpdate) (customer.Customer, error) {
	f.gotActor, f.gotID, f.gotUpdate = actor, id, &in
	return f.created, f.createErr
}

func (f *fakeCustomers) ChangeCode(_ context.Context, actor access.Actor, id, code string) (customer.Customer, error) {
	f.gotActor, f.gotID, f.gotCode = actor, id, code
	return f.created, f.createErr
}

func (f *fakeCustomers) SetDefaultRoute(_ context.Context, actor access.Actor, id, routeID string) (customer.Customer, error) {
	f.gotActor, f.gotID, f.gotRoute = actor, id, routeID
	return f.created, f.createErr
}

func (f *fakeCustomers) ListProjectGroups(_ context.Context, actor access.Actor) ([]customer.ProjectGroup, error) {
	f.gotActor = actor
	return f.groups, f.listErr
}

func (f *fakeCustomers) CreateProjectGroup(_ context.Context, actor access.Actor, in customer.ProjectGroupInput) (customer.ProjectGroup, error) {
	f.gotActor, f.gotGroup = actor, &in
	return f.group, f.createErr
}

func (f *fakeCustomers) UpdateProjectGroup(_ context.Context, actor access.Actor, id string, in customer.ProjectGroupInput) (customer.ProjectGroup, error) {
	f.gotActor, f.gotID, f.gotGroup = actor, id, &in
	return f.group, f.createErr
}

// fakeProducts meniru product.Service. Setiap method mencatat argumennya di
// got, lalu mengembalikan nilai yang sudah diisi test.
type fakeProducts struct {
	groups []product.ModelGroup
	group  product.ModelGroup
	models []product.Model
	model  product.Model
	parts  []product.PartNumber
	part   product.PartNumber
	err    error
	// got berisi argumen panggilan terakhir: nama method, ID, dan input.
	got struct {
		method string
		id     string
		input  any
	}
}

func (f *fakeProducts) SetEOL(_ context.Context, _ access.Actor, id string, eol bool) (product.Model, error) {
	f.record("SetEOL", id, eol)
	return product.Model{ID: id}, f.err
}

func (f *fakeProducts) record(method, id string, input any) {
	f.got.method, f.got.id, f.got.input = method, id, input
}

func (f *fakeProducts) ListModelGroups(_ context.Context, _ access.Actor, customerID string) ([]product.ModelGroup, error) {
	f.record("ListModelGroups", customerID, nil)
	return f.groups, f.err
}

func (f *fakeProducts) CreateModelGroup(_ context.Context, _ access.Actor, in product.NewModelGroup) (product.ModelGroup, error) {
	f.record("CreateModelGroup", "", in)
	return f.group, f.err
}

func (f *fakeProducts) RenameModelGroup(_ context.Context, _ access.Actor, id, name string) (product.ModelGroup, error) {
	f.record("RenameModelGroup", id, name)
	return f.group, f.err
}

func (f *fakeProducts) ListModels(_ context.Context, _ access.Actor, customerID string) ([]product.Model, error) {
	f.record("ListModels", customerID, nil)
	return f.models, f.err
}

func (f *fakeProducts) CreateModel(_ context.Context, _ access.Actor, in product.NewModel) (product.Model, error) {
	f.record("CreateModel", "", in)
	return f.model, f.err
}

func (f *fakeProducts) UpdateModel(_ context.Context, _ access.Actor, id string, in product.ModelUpdate) (product.Model, error) {
	f.record("UpdateModel", id, in)
	return f.model, f.err
}

func (f *fakeProducts) ListPartNumbers(_ context.Context, _ access.Actor, customerID string) ([]product.PartNumber, error) {
	f.record("ListPartNumbers", customerID, nil)
	return f.parts, f.err
}

func (f *fakeProducts) CreatePartNumber(_ context.Context, _ access.Actor, in product.NewPartNumber) (product.PartNumber, error) {
	f.record("CreatePartNumber", "", in)
	return f.part, f.err
}

func (f *fakeProducts) UpdatePartNumber(_ context.Context, _ access.Actor, id string, in product.PartNumberUpdate) (product.PartNumber, error) {
	f.record("UpdatePartNumber", id, in)
	return f.part, f.err
}

// fakeProcesses meniru process.Service. Setiap method mencatat argumennya di
// got, lalu mengembalikan nilai yang sudah diisi test.
type fakeProcesses struct {
	list []process.Process
	one  process.Process
	err  error
	got  struct {
		method string
		id     string
		input  process.Input
	}
}

func (f *fakeProcesses) List(context.Context, access.Actor) ([]process.Process, error) {
	f.got.method = "List"
	return f.list, f.err
}

func (f *fakeProcesses) Get(_ context.Context, _ access.Actor, id string) (process.Process, error) {
	f.got.method, f.got.id = "Get", id
	return f.one, f.err
}

func (f *fakeProcesses) Create(_ context.Context, _ access.Actor, in process.Input) (process.Process, error) {
	f.got.method, f.got.input = "Create", in
	return f.one, f.err
}

func (f *fakeProcesses) Update(_ context.Context, _ access.Actor, id string, in process.Input) (process.Process, error) {
	f.got.method, f.got.id, f.got.input = "Update", id, in
	return f.one, f.err
}

// called adalah catatan panggilan terakhir ke service palsu: nama method, ID,
// dan input.
type called struct {
	method string
	id     string
	input  any
}

// fakeNumbering meniru numbering.Service.
type fakeNumbering struct {
	registration numbering.Registration
	number       numbering.DocumentNumber
	report       numbering.ImportReport
	next         string
	settings     numbering.Settings
	types        []numbering.DocumentType
	docType      numbering.DocumentType
	codes        []numbering.SpecialCode
	code         numbering.SpecialCode
	err          error
	got          called
}

func (f *fakeNumbering) GetSettings(context.Context, access.Actor) (numbering.Settings, error) {
	f.got = called{method: "GetSettings"}
	return f.settings, f.err
}

func (f *fakeNumbering) UpdateSettings(_ context.Context, _ access.Actor, in numbering.SettingsInput) (numbering.Settings, error) {
	// Pointer tidak bisa dibandingkan isinya dengan !=, jadi isian ditulis sebagai teks.
	text := func(p *string) string {
		if p == nil {
			return "-"
		}
		return *p
	}
	digits := "-"
	if in.RevisionDigits != nil {
		digits = fmt.Sprint(*in.RevisionDigits)
	}
	f.got = called{method: "UpdateSettings", input: in.FactoryCode + "|" + text(in.Pattern) + "|" + digits}
	return f.settings, f.err
}

func (f *fakeNumbering) Register(_ context.Context, _ access.Actor, in numbering.NewNumber) (numbering.Registration, error) {
	f.got = called{method: "Register", input: in}
	return f.registration, f.err
}

func (f *fakeNumbering) RegisterSet(_ context.Context, _ access.Actor, in numbering.NewSet) (numbering.Registration, error) {
	f.got = called{method: "RegisterSet", input: in}
	return f.registration, f.err
}

func (f *fakeNumbering) Import(_ context.Context, _ access.Actor, in numbering.ImportInput) (numbering.ImportReport, error) {
	f.got = called{method: "Import", input: in}
	return f.report, f.err
}

func (f *fakeNumbering) NextNumber(_ context.Context, _ access.Actor, customerID, documentTypeID string) (string, error) {
	f.got = called{method: "NextNumber", id: customerID, input: documentTypeID}
	return f.next, f.err
}

func (f *fakeNumbering) ListNumbers(_ context.Context, _ access.Actor, filter numbering.ListFilter) ([]numbering.DocumentNumber, string, error) {
	f.got = called{method: "ListNumbers", input: filter}
	return f.registration.Numbers, f.next, f.err
}

func (f *fakeNumbering) Void(_ context.Context, _ access.Actor, id, reason string) (numbering.DocumentNumber, error) {
	f.got = called{method: "Void", id: id, input: reason}
	return f.number, f.err
}

func (f *fakeNumbering) MakeObsolete(_ context.Context, _ access.Actor, id, reason string) (numbering.DocumentNumber, error) {
	f.got = called{method: "MakeObsolete", id: id, input: reason}
	return f.number, f.err
}

func (f *fakeNumbering) ListDocumentTypes(context.Context, access.Actor) ([]numbering.DocumentType, error) {
	f.got = called{method: "ListDocumentTypes"}
	return f.types, f.err
}

func (f *fakeNumbering) CreateDocumentType(_ context.Context, _ access.Actor, in numbering.DocumentTypeInput) (numbering.DocumentType, error) {
	f.got = called{method: "CreateDocumentType", input: in}
	return f.docType, f.err
}

func (f *fakeNumbering) UpdateDocumentType(_ context.Context, _ access.Actor, id string, in numbering.DocumentTypeInput) (numbering.DocumentType, error) {
	f.got = called{method: "UpdateDocumentType", id: id, input: in}
	return f.docType, f.err
}

func (f *fakeNumbering) ListSpecialCodes(context.Context, access.Actor) ([]numbering.SpecialCode, error) {
	f.got = called{method: "ListSpecialCodes"}
	return f.codes, f.err
}

func (f *fakeNumbering) CreateSpecialCode(_ context.Context, _ access.Actor, in numbering.NewSpecialCode) (numbering.SpecialCode, error) {
	f.got = called{method: "CreateSpecialCode", input: in}
	return f.code, f.err
}

func (f *fakeNumbering) RenameSpecialCode(_ context.Context, _ access.Actor, id, name string) (numbering.SpecialCode, error) {
	f.got = called{method: "RenameSpecialCode", id: id, input: name}
	return f.code, f.err
}

// fakeChanges meniru change.Service.
type fakeChanges struct {
	types      []change.ChangeType
	changeType change.ChangeType
	sources    []change.ChangeSource
	source     change.ChangeSource
	categories []change.FourMCategory
	category   change.FourMCategory
	err        error
	got        called
}

func (f *fakeChanges) ListChangeTypes(context.Context, access.Actor) ([]change.ChangeType, error) {
	f.got = called{method: "ListChangeTypes"}
	return f.types, f.err
}

func (f *fakeChanges) RenameChangeType(_ context.Context, _ access.Actor, id, name string) (change.ChangeType, error) {
	f.got = called{method: "RenameChangeType", id: id, input: name}
	return f.changeType, f.err
}

func (f *fakeChanges) ListChangeSources(context.Context, access.Actor) ([]change.ChangeSource, error) {
	f.got = called{method: "ListChangeSources"}
	return f.sources, f.err
}

func (f *fakeChanges) CreateChangeSource(_ context.Context, _ access.Actor, name string) (change.ChangeSource, error) {
	f.got = called{method: "CreateChangeSource", input: name}
	return f.source, f.err
}

func (f *fakeChanges) UpdateChangeSource(_ context.Context, _ access.Actor, id string, in change.ChangeSourceUpdate) (change.ChangeSource, error) {
	f.got = called{method: "UpdateChangeSource", id: id, input: in}
	return f.source, f.err
}

func (f *fakeChanges) SetApprovalRoute(_ context.Context, _ access.Actor, id, routeID string) (change.ChangeType, error) {
	f.got = called{method: "SetApprovalRoute", id: id, input: routeID}
	return f.changeType, f.err
}

func (f *fakeChanges) ListFourMCategories(context.Context, access.Actor) ([]change.FourMCategory, error) {
	f.got = called{method: "ListFourMCategories"}
	return f.categories, f.err
}

func (f *fakeChanges) RenameFourMCategory(_ context.Context, _ access.Actor, id, name string) (change.FourMCategory, error) {
	f.got = called{method: "RenameFourMCategory", id: id, input: name}
	return f.category, f.err
}

// fakeTemplates meniru template.Service. got.input menyimpan isian terakhir;
// gotRatings dan gotSymbols menyimpan daftar, karena slice tidak bisa
// dibandingkan dengan ==.
type fakeTemplates struct {
	templates  []template.FormatTemplate
	template   template.FormatTemplate
	version    template.FormatVersion
	err        error
	got        called
	gotField   template.FieldUpdate
	gotNew     template.NewField
	gotRatings []template.Rating
	gotSymbols []template.Symbol
	gotAPRules []template.APRule
	formSets   template.FormSetList
	formSet    template.FormSet
}

func (f *fakeTemplates) ListFormSets(context.Context, access.Actor) (template.FormSetList, error) {
	f.got = called{method: "ListFormSets"}
	return f.formSets, f.err
}

func (f *fakeTemplates) CreateFormSet(_ context.Context, _ access.Actor, in template.NewFormSet) (template.FormSet, error) {
	f.got = called{method: "CreateFormSet", input: in}
	return f.formSet, f.err
}

func (f *fakeTemplates) SetFormSetSheets(_ context.Context, _ access.Actor, versionID string, sheets template.FormSetSheets) (template.FormSet, error) {
	f.got = called{method: "SetFormSetSheets", id: versionID, input: sheets}
	return f.formSet, f.err
}

func (f *fakeTemplates) NewFormSetVersion(_ context.Context, _ access.Actor, formSetID string) (template.FormSet, error) {
	f.got = called{method: "NewFormSetVersion", id: formSetID}
	return f.formSet, f.err
}

func (f *fakeTemplates) PublishFormSetVersion(_ context.Context, _ access.Actor, versionID string) (template.FormSet, error) {
	f.got = called{method: "PublishFormSetVersion", id: versionID}
	return f.formSet, f.err
}

func (f *fakeTemplates) SetCustomerFormSet(_ context.Context, _ access.Actor, customerID, formSetID string) error {
	f.got = called{method: "SetCustomerFormSet", id: customerID, input: formSetID}
	return f.err
}

func (f *fakeTemplates) SetAPRules(_ context.Context, _ access.Actor, versionID string, rules []template.APRule) (template.FormatVersion, error) {
	f.got, f.gotAPRules = called{method: "SetAPRules", id: versionID}, rules
	return f.version, f.err
}

func (f *fakeTemplates) ActionPriorityOf(_ context.Context, _ access.Actor, versionID string, s, o, d int) (string, error) {
	f.got = called{method: "ActionPriorityOf", id: versionID, input: [3]int{s, o, d}}
	return "H", f.err
}

func (f *fakeTemplates) PublishFormatVersion(_ context.Context, _ access.Actor, versionID string) (template.FormatVersion, error) {
	f.got = called{method: "PublishFormatVersion", id: versionID}
	return f.version, f.err
}

func (f *fakeTemplates) NewFormatVersion(_ context.Context, _ access.Actor, templateID string) (template.FormatVersion, error) {
	f.got = called{method: "NewFormatVersion", id: templateID}
	return f.version, f.err
}

func (f *fakeTemplates) ListFormatTemplates(context.Context, access.Actor) ([]template.FormatTemplate, error) {
	f.got = called{method: "ListFormatTemplates"}
	return f.templates, f.err
}

func (f *fakeTemplates) GetFormatVersion(_ context.Context, _ access.Actor, id string) (template.FormatVersion, error) {
	f.got = called{method: "GetFormatVersion", id: id}
	return f.version, f.err
}

func (f *fakeTemplates) CreateFormatTemplate(_ context.Context, _ access.Actor, in template.NewTemplate) (template.FormatTemplate, error) {
	f.got = called{method: "CreateFormatTemplate", input: in}
	return f.template, f.err
}

func (f *fakeTemplates) RenameFormatTemplate(_ context.Context, _ access.Actor, id, name string) (template.FormatTemplate, error) {
	f.got = called{method: "RenameFormatTemplate", id: id, input: name}
	return f.template, f.err
}

func (f *fakeTemplates) SetFormatTemplateHidden(_ context.Context, _ access.Actor, id string, hidden bool) (template.FormatTemplate, error) {
	f.got = called{method: "SetFormatTemplateHidden", id: id, input: hidden}
	return f.template, f.err
}

func (f *fakeTemplates) SetFormSetHidden(_ context.Context, _ access.Actor, id string, hidden bool) (template.FormSet, error) {
	f.got = called{method: "SetFormSetHidden", id: id, input: hidden}
	return template.FormSet{ID: id, Hidden: hidden, Versions: []template.FormSetVersion{}}, f.err
}

func (f *fakeTemplates) AddField(_ context.Context, _ access.Actor, versionID string, in template.NewField) (template.FormatVersion, error) {
	f.got, f.gotNew = called{method: "AddField", id: versionID}, in
	return f.version, f.err
}

func (f *fakeTemplates) UpdateField(_ context.Context, _ access.Actor, versionID, fieldID string, in template.FieldUpdate) (template.FormatVersion, error) {
	f.got, f.gotField = called{method: "UpdateField", id: versionID, input: fieldID}, in
	return f.version, f.err
}

func (f *fakeTemplates) SetRatingCriteria(_ context.Context, _ access.Actor, versionID string, ratings []template.Rating) (template.FormatVersion, error) {
	f.got, f.gotRatings = called{method: "SetRatingCriteria", id: versionID}, ratings
	return f.version, f.err
}

func (f *fakeTemplates) SaveSymbols(_ context.Context, _ access.Actor, versionID string, symbols []template.Symbol) (template.FormatVersion, error) {
	f.got, f.gotSymbols = called{method: "SaveSymbols", id: versionID}, symbols
	return f.version, f.err
}

func (f *fakeTemplates) SetHighlights(_ context.Context, _ access.Actor, versionID string, in template.HighlightSettings) (template.FormatVersion, error) {
	f.got = called{method: "SetHighlights", id: versionID, input: highlightText(in)}
	return f.version, f.err
}

// highlightText menulis aturan sorot dengan nil sebagai "-", supaya bisa
// dibandingkan di test.
func highlightText(in template.HighlightSettings) string {
	text := func(n *int) string {
		if n == nil {
			return "-"
		}
		return strconv.Itoa(*n)
	}
	return text(in.RPNAbove) + "/" + text(in.SeverityAtLeast)
}

// fakeProcessTemplates meniru template.ProcessService. gotSub dan gotPoint
// menyimpan isian perubahan terakhir, dan gotNew isian poin baru.
type fakeProcessTemplates struct {
	templates []template.ProcessTemplate
	template  template.ProcessTemplate
	version   template.ProcessVersion
	err       error
	got       called
	gotSub    template.SubprocessUpdate
	gotNew    template.NewPoint
	gotPoint  template.PointUpdate
	// Cakupan poin dan varian (M3.3).
	gotVariant       template.NewVariant
	gotVariantUpdate template.VariantUpdate
	resolved         template.Resolved
}

func (f *fakeProcessTemplates) AddVariant(_ context.Context, _ access.Actor, versionID, pointID string, in template.NewVariant) (template.ProcessVersion, error) {
	f.got, f.gotVariant = called{method: "AddVariant", id: versionID, input: pointID}, in
	return f.version, f.err
}

func (f *fakeProcessTemplates) UpdateVariant(_ context.Context, _ access.Actor, versionID, variantID string, in template.VariantUpdate) (template.ProcessVersion, error) {
	f.got, f.gotVariantUpdate = called{method: "UpdateVariant", id: versionID, input: variantID}, in
	return f.version, f.err
}

func (f *fakeProcessTemplates) ResolvePoints(_ context.Context, _ access.Actor, versionID, modelID string) (template.Resolved, error) {
	f.got = called{method: "ResolvePoints", id: versionID, input: modelID}
	return f.resolved, f.err
}

func (f *fakeProcessTemplates) PublishProcessVersion(_ context.Context, _ access.Actor, versionID string) (template.ProcessVersion, error) {
	f.got = called{method: "PublishProcessVersion", id: versionID}
	return f.version, f.err
}

func (f *fakeProcessTemplates) NewProcessVersion(_ context.Context, _ access.Actor, templateID string) (template.ProcessVersion, error) {
	f.got = called{method: "NewProcessVersion", id: templateID}
	return f.version, f.err
}

func (f *fakeProcessTemplates) ListProcessTemplates(context.Context, access.Actor) ([]template.ProcessTemplate, error) {
	f.got = called{method: "ListProcessTemplates"}
	return f.templates, f.err
}

func (f *fakeProcessTemplates) GetProcessVersion(_ context.Context, _ access.Actor, id string) (template.ProcessVersion, error) {
	f.got = called{method: "GetProcessVersion", id: id}
	return f.version, f.err
}

func (f *fakeProcessTemplates) CreateProcessTemplate(_ context.Context, _ access.Actor, processID string) (template.ProcessTemplate, error) {
	f.got = called{method: "CreateProcessTemplate", input: processID}
	return f.template, f.err
}

func (f *fakeProcessTemplates) AddSubprocess(_ context.Context, _ access.Actor, versionID, name string) (template.ProcessVersion, error) {
	f.got = called{method: "AddSubprocess", id: versionID, input: name}
	return f.version, f.err
}

func (f *fakeProcessTemplates) UpdateSubprocess(_ context.Context, _ access.Actor, versionID, subprocessID string, in template.SubprocessUpdate) (template.ProcessVersion, error) {
	f.got, f.gotSub = called{method: "UpdateSubprocess", id: versionID, input: subprocessID}, in
	return f.version, f.err
}

func (f *fakeProcessTemplates) AddPoint(_ context.Context, _ access.Actor, versionID string, in template.NewPoint) (template.ProcessVersion, error) {
	f.got, f.gotNew = called{method: "AddPoint", id: versionID}, in
	return f.version, f.err
}

func (f *fakeProcessTemplates) UpdatePoint(_ context.Context, _ access.Actor, versionID, pointID string, in template.PointUpdate) (template.ProcessVersion, error) {
	f.got, f.gotPoint = called{method: "UpdatePoint", id: versionID, input: pointID}, in
	return f.version, f.err
}

// fakeDocuments meniru document.Service.
type fakeDocuments struct {
	revisions []document.Revision
	diff      document.Diff
	templates []document.TemplateLink
	list      []document.Summary
	doc       document.Document
	err       error
	got       called
	gotNew    document.NewDocument
	gotRow    document.NewRow
	gotUpd    document.RowUpdate
	// gotRows adalah blok baris pada AddRows (M4.4).
	gotRows []document.NewRow
	// composition dan gotCompose: perakitan (M4.5).
	composition document.Composition
	gotSteps    []document.StepChoice
	gotCompose  document.Compose
	gotOrder    document.Order // M4.6
	// differences dan gotReason: perbandingan dan putus tautan (M4.8).
	differences []document.RowDifference
	gotReason   string
	// comparison dan gotChecked: PFMEA dan CP berdampingan (M4.9).
	comparison document.Comparison
	gotChecked bool
	// gotSection dan gotSectionUpd: bagian per PCB (D-155).
	gotSection    document.NewSection
	gotSectionUpd document.SectionUpdate
	header        map[string]any
	// headerVersion adalah versi header yang dikirim klien (M4.3).
	headerVersion int
}

func (f *fakeDocuments) List(_ context.Context, _ access.Actor, customerID string) ([]document.Summary, error) {
	f.got = called{method: "List", id: customerID}
	return f.list, f.err
}

func (f *fakeDocuments) Create(_ context.Context, _ access.Actor, in document.NewDocument) (document.Document, error) {
	f.got, f.gotNew = called{method: "Create"}, in
	return f.doc, f.err
}

func (f *fakeDocuments) MarkReady(_ context.Context, _ access.Actor, id string) ([]document.Revision, error) {
	f.got = called{method: "MarkReady", id: id}
	return f.revisions, f.err
}

func (f *fakeDocuments) ListReleased(_ context.Context, _ access.Actor, customerID string) ([]document.Summary, error) {
	f.got = called{method: "ListReleased", id: customerID}
	return f.list, f.err
}

func (f *fakeDocuments) Released(_ context.Context, _ access.Actor, id string) (document.Document, error) {
	f.got = called{method: "Released", id: id}
	return f.doc, f.err
}

func (f *fakeDocuments) ReleasedComparison(_ context.Context, _ access.Actor, id string) (document.Comparison, error) {
	f.got = called{method: "ReleasedComparison", id: id}
	return document.Comparison{}, f.err
}

func (f *fakeDocuments) Get(_ context.Context, _ access.Actor, id string) (document.Document, error) {
	f.got = called{method: "Get", id: id}
	return f.doc, f.err
}

func (f *fakeDocuments) SetHeader(_ context.Context, _ access.Actor, id string, header map[string]any, version int) (document.Document, error) {
	f.got, f.header, f.headerVersion = called{method: "SetHeader", id: id}, header, version
	return f.doc, f.err
}

func (f *fakeDocuments) AddRow(_ context.Context, _ access.Actor, id string, in document.NewRow) (document.Document, error) {
	f.got, f.gotRow = called{method: "AddRow", id: id}, in
	return f.doc, f.err
}

func (f *fakeDocuments) AddRows(_ context.Context, _ access.Actor, id string, rows []document.NewRow) (document.Document, error) {
	f.got, f.gotRows = called{method: "AddRows", id: id}, rows
	return f.doc, f.err
}

func (f *fakeDocuments) Composition(_ context.Context, _ access.Actor, id, versionID, _ string) (document.Composition, error) {
	f.got = called{method: "Composition", id: id, input: versionID}
	return f.composition, f.err
}

func (f *fakeDocuments) Compose(_ context.Context, _ access.Actor, id string, in document.Compose) (document.Document, error) {
	f.got, f.gotCompose = called{method: "Compose", id: id}, in
	return f.doc, f.err
}

func (f *fakeDocuments) PFDComposition(_ context.Context, _ access.Actor, id string) (document.PFDComposition, error) {
	f.got = called{method: "PFDComposition", id: id}
	return document.PFDComposition{Steps: []document.PFDStepPlan{{FlowStep: document.FlowStep{Operation: "10", ProcessName: "IQC"},
		Targets: []document.StepTarget{{DocumentID: "cp", DocumentType: "cp", Points: 2, State: document.StepToAdd}}}}}, f.err
}

func (f *fakeDocuments) ComposeFromPFD(_ context.Context, _ access.Actor, id string, choices []document.StepChoice) (document.Document, error) {
	f.got, f.gotSteps = called{method: "ComposeFromPFD", id: id}, choices
	return f.doc, f.err
}

func (f *fakeDocuments) Reorder(_ context.Context, _ access.Actor, id string, in document.Order) (document.Document, error) {
	f.got, f.gotOrder = called{method: "Reorder", id: id}, in
	return f.doc, f.err
}

func (f *fakeDocuments) CopyFrom(_ context.Context, _ access.Actor, id, sourceID string) (document.Document, error) {
	f.got = called{method: "CopyFrom", id: id, input: sourceID}
	return f.doc, f.err
}

func (f *fakeDocuments) Differences(_ context.Context, _ access.Actor, id string) ([]document.RowDifference, error) {
	f.got = called{method: "Differences", id: id}
	return f.differences, f.err
}

func (f *fakeDocuments) Detach(_ context.Context, _ access.Actor, id, rowID, reason string) (document.Document, error) {
	f.got, f.gotReason = called{method: "Detach", id: id, input: rowID}, reason
	return f.doc, f.err
}

func (f *fakeDocuments) Release(_ context.Context, _ access.Actor, id string) (document.Document, error) {
	f.got = called{method: "Release", id: id}
	return f.doc, f.err
}

func (f *fakeDocuments) Diff(_ context.Context, _ access.Actor, id string, from int, to *int, processID string) (document.Diff, error) {
	f.got = called{method: "Diff", id: id, input: fmt.Sprint(from, ":", to != nil, ":", processID)}
	return f.diff, f.err
}

func (f *fakeDocuments) Templates(_ context.Context, _ access.Actor, id string) ([]document.TemplateLink, error) {
	f.got = called{method: "Templates", id: id}
	return f.templates, f.err
}

func (f *fakeDocuments) SetTemplateMode(_ context.Context, _ access.Actor, id, templateID, mode, reason string) ([]document.TemplateLink, error) {
	f.got, f.gotReason = called{method: "SetTemplateMode", id: id, input: templateID + ":" + mode}, reason
	return f.templates, f.err
}

func (f *fakeDocuments) Revisions(_ context.Context, _ access.Actor, id string) ([]document.Revision, error) {
	f.got = called{method: "Revisions", id: id}
	return f.revisions, f.err
}

func (f *fakeDocuments) Comparison(_ context.Context, _ access.Actor, id string) (document.Comparison, error) {
	f.got = called{method: "Comparison", id: id}
	return f.comparison, f.err
}

func (f *fakeDocuments) MarkChecked(_ context.Context, _ access.Actor, id, _, processID string, checked bool) (document.Comparison, error) {
	f.got, f.gotChecked = called{method: "MarkChecked", id: id, input: processID}, checked
	return f.comparison, f.err
}

func (f *fakeDocuments) FillReferences(_ context.Context, _ access.Actor, id string) (document.Document, error) {
	f.got = called{method: "FillReferences", id: id}
	return f.doc, f.err
}

func (f *fakeDocuments) AddSection(_ context.Context, _ access.Actor, id string, in document.NewSection) (document.Document, error) {
	f.got, f.gotSection = called{method: "AddSection", id: id}, in
	return f.doc, f.err
}

func (f *fakeDocuments) UpdateSection(_ context.Context, _ access.Actor, id, sectionID string, in document.SectionUpdate) (document.Document, error) {
	f.got, f.gotSectionUpd = called{method: "UpdateSection", id: id, input: sectionID}, in
	return f.doc, f.err
}

func (f *fakeDocuments) RemoveSection(_ context.Context, _ access.Actor, id, sectionID string) (document.Document, error) {
	f.got = called{method: "RemoveSection", id: id, input: sectionID}
	return f.doc, f.err
}

func (f *fakeDocuments) DeleteRow(_ context.Context, _ access.Actor, id, rowID string) (document.Document, error) {
	f.got = called{method: "DeleteRow", id: id, input: rowID}
	return f.doc, f.err
}

func (f *fakeDocuments) UpdateRow(_ context.Context, _ access.Actor, id, rowID string, in document.RowUpdate) (document.Document, error) {
	f.got, f.gotUpd = called{method: "UpdateRow", id: id, input: rowID}, in
	return f.doc, f.err
}

// fakeApproval meniru approval.Service.
type fakeApproval struct {
	positions []approval.Position
	position  approval.Position
	holder    approval.Holder
	routes    []approval.Route
	route     approval.Route
	err       error
	got       called
	// gotHolderID adalah ID pemegang pada RemoveHolder, dan gotRoute isian rute terakhir.
	gotHolderID string
	gotRoute    approval.RouteInput
}

func (f *fakeApproval) ListPositions(context.Context, access.Actor) ([]approval.Position, error) {
	f.got = called{method: "ListPositions"}
	return f.positions, f.err
}

func (f *fakeApproval) ListCandidates(context.Context, access.Actor) ([]approval.Candidate, error) {
	f.got = called{method: "ListCandidates"}
	return []approval.Candidate{{PersonID: "orang-1", Name: "Contoh Orang"}}, f.err
}

func (f *fakeApproval) RenamePosition(_ context.Context, _ access.Actor, id, name string) (approval.Position, error) {
	f.got = called{method: "RenamePosition", id: id, input: name}
	return f.position, f.err
}

func (f *fakeApproval) AddHolder(_ context.Context, _ access.Actor, positionID string, in approval.NewHolder) (approval.Holder, error) {
	f.got = called{method: "AddHolder", id: positionID, input: in}
	return f.holder, f.err
}

func (f *fakeApproval) RemoveHolder(_ context.Context, _ access.Actor, positionID, holderID string) error {
	f.got = called{method: "RemoveHolder", id: positionID}
	f.gotHolderID = holderID
	return f.err
}

func (f *fakeApproval) ListRoutes(context.Context, access.Actor) ([]approval.Route, error) {
	f.got = called{method: "ListRoutes"}
	return f.routes, f.err
}

func (f *fakeApproval) CreateRoute(_ context.Context, _ access.Actor, in approval.RouteInput) (approval.Route, error) {
	f.got = called{method: "CreateRoute"}
	f.gotRoute = in
	return f.route, f.err
}

func (f *fakeApproval) UpdateRoute(_ context.Context, _ access.Actor, id string, in approval.RouteInput) (approval.Route, error) {
	f.got = called{method: "UpdateRoute", id: id}
	f.gotRoute = in
	return f.route, f.err
}

// fakeSettings meniru settings.Service.
type fakeSettings struct {
	current settings.Settings
	err     error
	got     *bool // saklar terakhir yang dikirim ke Apply
	gotMode *string
}

func (f *fakeSettings) Get(context.Context, access.Actor) (settings.Settings, error) {
	return f.current, f.err
}

func (f *fakeSettings) Apply(_ context.Context, _ access.Actor, in settings.Update) (settings.Settings, error) {
	f.got, f.gotMode = in.SignDatesEnabled, in.FormApprovalMode
	if in.FormApprovalMode != nil {
		f.current.FormApprovalMode = *in.FormApprovalMode
	}
	if in.SignDatesEnabled != nil {
		f.current.SignDatesEnabled = *in.SignDatesEnabled
	}
	return f.current, f.err
}

func (f *fakeSettings) Company(context.Context, access.Actor) (settings.Company, error) {
	return settings.Company{Name: "PT. Contoh"}, f.err
}

func (f *fakeSettings) SetCompanyName(_ context.Context, _ access.Actor, name string) (settings.Company, error) {
	return settings.Company{Name: name}, f.err
}

func (f *fakeSettings) Logo(context.Context) ([]byte, error) { return settings.DefaultLogo, f.err }

func (f *fakeSettings) SetLogo(context.Context, access.Actor, []byte, bool) (settings.Company, error) {
	return settings.Company{Name: "PT. Contoh", CustomLogo: true, LogoSHA256: "abc"}, f.err
}

func (f *fakeSettings) ResetLogo(context.Context, access.Actor, bool) (settings.Company, error) {
	return settings.Company{Name: "PT. Contoh"}, f.err
}

// testToken adalah token sesi yang diterima fakeAuth secara bawaan.
const testToken = "token-uji"

// testActor adalah pemilik sesi testToken.
var testActor = access.Actor{
	Type: access.ActorHuman, AccountID: "akun-uji", AccountKind: access.AccountAdmin, Login: "admin.uji",
	PersonID: "orang-uji", PersonName: "Admin Uji",
}

// fakeAuth meniru auth.Service tanpa database. sessions memetakan token ke actor.
type fakeAuth struct {
	sessions map[string]access.Actor
	// loginToken dan loginActor dikembalikan Login bila password-nya "password-benar".
	loginToken string
	loginActor access.Actor
	// changeErr dikembalikan ChangePassword bila diisi.
	changeErr error
	// Hasil dan error untuk operasi akun (A-1).
	accounts  []auth.AccountInfo
	account   auth.AccountInfo
	temporary string
	adminErr  error
	// Catatan panggilan, untuk diperiksa test.
	loggedOut     []string
	changeCurrent string
	changeNew     string
	newAccount    *auth.NewAccount
	targetAccount string
	setGrants     []access.RoleGrant
}

func newFakeAuth() *fakeAuth {
	return &fakeAuth{sessions: map[string]access.Actor{testToken: testActor}}
}

func (f *fakeAuth) Login(_ context.Context, login, password string) (access.Actor, string, error) {
	if login == f.loginActor.Login && password == "password-benar" {
		return f.loginActor, f.loginToken, nil
	}
	return access.Actor{}, "", auth.ErrInvalidCredentials
}

func (f *fakeAuth) Authenticate(_ context.Context, token string) (access.Actor, error) {
	actor, ok := f.sessions[token]
	if !ok {
		return access.Actor{}, auth.ErrUnauthenticated
	}
	return actor, nil
}

func (f *fakeAuth) Logout(_ context.Context, _ access.Actor, token string) error {
	f.loggedOut = append(f.loggedOut, token)
	return nil
}

func (f *fakeAuth) ChangePassword(_ context.Context, _ access.Actor, current, newPassword string) (string, error) {
	f.changeCurrent, f.changeNew = current, newPassword
	if f.changeErr != nil {
		return "", f.changeErr
	}
	return "token-baru", nil
}

func (f *fakeAuth) ListAccounts(context.Context, access.Actor) ([]auth.AccountInfo, error) {
	return f.accounts, f.adminErr
}

func (f *fakeAuth) GetAccount(_ context.Context, _ access.Actor, accountID string) (auth.AccountInfo, error) {
	f.targetAccount = accountID
	return f.account, f.adminErr
}

func (f *fakeAuth) CreateAccount(_ context.Context, _ access.Actor, in auth.NewAccount) (auth.AccountInfo, string, error) {
	f.newAccount = &in
	return f.account, f.temporary, f.adminErr
}

func (f *fakeAuth) ResetPassword(_ context.Context, _ access.Actor, accountID string) (string, error) {
	f.targetAccount = accountID
	return f.temporary, f.adminErr
}

func (f *fakeAuth) DisableAccount(_ context.Context, _ access.Actor, accountID string) (auth.AccountInfo, error) {
	f.targetAccount = accountID
	return f.account, f.adminErr
}

func (f *fakeAuth) EnableAccount(_ context.Context, _ access.Actor, accountID string) (auth.AccountInfo, error) {
	f.targetAccount = accountID
	return f.account, f.adminErr
}

func (f *fakeAuth) SetRoles(_ context.Context, _ access.Actor, accountID string, grants []access.RoleGrant) error {
	f.targetAccount, f.setGrants = accountID, grants
	return f.adminErr
}

// fakeAccess meniru access.Checker untuk hak dan katalog peran.
type fakeAccess struct {
	permissions []access.Permission
	roles       []access.Role
	err         error
}

func (f *fakeAccess) Permissions(context.Context, access.Actor) ([]access.Permission, error) {
	return f.permissions, f.err
}

func (f *fakeAccess) ListRoles(context.Context, access.Actor) ([]access.Role, error) {
	return f.roles, f.err
}

// fakeAudit meniru audit.Service tanpa database.
type fakeAudit struct {
	entries      []audit.Entry
	verification audit.Verification
	err          error
	// Catatan panggilan List.
	gotBefore int64
	gotLimit  int
}

func (f *fakeAudit) List(_ context.Context, _ access.Actor, before int64, limit int) ([]audit.Entry, error) {
	f.gotBefore, f.gotLimit = before, limit
	return f.entries, f.err
}

func (f *fakeAudit) Verify(context.Context, access.Actor) (audit.Verification, error) {
	return f.verification, f.err
}

// testServer adalah handler lengkap dengan dependensi palsu. Log server
// ditampung di log, supaya test bisa memeriksa apa yang dicatat.
type testServer struct {
	handler http.Handler
	log     *bytes.Buffer
}

// newTestServer mengisi dependensi yang kosong dengan versi palsu yang sehat.
func newTestServer(t *testing.T, ui fs.FS, deps Deps) testServer {
	t.Helper()
	if ui == nil {
		ui = fstest.MapFS{}
	}
	if deps.DB == nil {
		deps.DB = fakePinger{}
	}
	if deps.Customers == nil {
		deps.Customers = &fakeCustomers{list: []customer.Customer{}}
	}
	if deps.Auth == nil {
		deps.Auth = newFakeAuth()
	}
	if deps.Access == nil {
		deps.Access = &fakeAccess{permissions: []access.Permission{access.ViewReleased, access.ManageUsers}}
	}
	if deps.Audit == nil {
		deps.Audit = &fakeAudit{}
	}
	if deps.Products == nil {
		deps.Products = &fakeProducts{}
	}
	if deps.Processes == nil {
		deps.Processes = &fakeProcesses{}
	}
	if deps.Numbering == nil {
		deps.Numbering = &fakeNumbering{}
	}
	if deps.Changes == nil {
		deps.Changes = &fakeChanges{}
	}
	if deps.Approval == nil {
		deps.Approval = &fakeApproval{}
	}
	if deps.Settings == nil {
		deps.Settings = &fakeSettings{}
	}
	if deps.Templates == nil {
		deps.Templates = &fakeTemplates{}
	}
	if deps.ProcessTemplates == nil {
		deps.ProcessTemplates = &fakeProcessTemplates{}
	}
	if deps.Documents == nil {
		deps.Documents = &fakeDocuments{}
	}
	var log bytes.Buffer
	deps.Logger = slog.New(slog.NewTextHandler(&log, nil))
	return testServer{handler: NewHandler(ui, deps), log: &log}
}

// do mengirim satu permintaan sebagai pengguna yang sudah login (cookie sesi
// testToken), lalu mengembalikan rekaman jawabannya. header berisi pasangan
// nama dan nilai, misalnya "X-Request-Id", "abc".
func (s testServer) do(method, path, body string, header ...string) *httptest.ResponseRecorder {
	return s.doWithCookie(testToken, method, path, body, header...)
}

// doAnon mengirim permintaan tanpa cookie sesi, seperti pengguna yang belum login.
func (s testServer) doAnon(method, path, body string, header ...string) *httptest.ResponseRecorder {
	return s.doWithCookie("", method, path, body, header...)
}

// doWithCookie mengirim permintaan dengan token sesi tertentu ("" berarti tanpa cookie).
func (s testServer) doWithCookie(token, method, path, body string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	return rec
}

// decodeError membaca jawaban sebagai Error dari kontrak dan memeriksa
// bagian yang wajib ada di setiap error (R-API-13).
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) Error {
	t.Helper()
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type %q, ingin application/json", got)
	}
	var body Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("isi error bukan JSON yang valid: %v\n%s", err, rec.Body.String())
	}
	if body.Message == "" {
		t.Error("message kosong")
	}
	if id := rec.Header().Get("X-Request-Id"); body.RequestId == "" || body.RequestId != id {
		t.Errorf("request_id %q, ingin sama dengan header X-Request-Id %q", body.RequestId, id)
	}
	return body
}

// fakeRefDocs meniru template.RefDocService (D-130).
type fakeRefDocs struct {
	list    []template.RefDocument
	doc     template.RefDocument
	err     error
	got     called
	gotNew  template.NewRefDocument
	gotUpd  template.RefDocumentUpdate
	gotLink template.NewRefLink
}

func (f *fakeRefDocs) List(_ context.Context, _ access.Actor) ([]template.RefDocument, error) {
	f.got = called{method: "List"}
	return f.list, f.err
}

func (f *fakeRefDocs) Create(_ context.Context, _ access.Actor, in template.NewRefDocument) (template.RefDocument, error) {
	f.got, f.gotNew = called{method: "Create"}, in
	return f.doc, f.err
}

func (f *fakeRefDocs) Update(_ context.Context, _ access.Actor, id string, in template.RefDocumentUpdate) (template.RefDocument, error) {
	f.got, f.gotUpd = called{method: "Update", id: id}, in
	return f.doc, f.err
}

func (f *fakeRefDocs) AddLink(_ context.Context, _ access.Actor, id string, in template.NewRefLink) (template.RefDocument, error) {
	f.got, f.gotLink = called{method: "AddLink", id: id}, in
	return f.doc, f.err
}

func (f *fakeRefDocs) RemoveLink(_ context.Context, _ access.Actor, id, linkID string) (template.RefDocument, error) {
	f.got = called{method: "RemoveLink", id: id, input: linkID}
	return f.doc, f.err
}

// fakeChangeRequests meniru change.RequestService (Tahap 5).
type fakeChangeRequests struct {
	list           []change.Change
	c              change.Change
	err            error
	got            called
	gotNew         change.NewChange
	file           string // isi berkas OpenAttachment
	preview        []change.PreviewItem
	gotPropagation change.Propagation
	yokoten        change.Yokoten
	gotYokoten     change.NewYokoten
	gotSQAM        change.SQAMChange
}

func (f *fakeChangeRequests) List(_ context.Context, _ access.Actor, customerID string) ([]change.Change, error) {
	f.got = called{method: "List", id: customerID}
	return f.list, f.err
}

func (f *fakeChangeRequests) ForDocument(_ context.Context, _ access.Actor, documentID string) ([]change.Change, error) {
	f.got = called{method: "ForDocument", id: documentID}
	return f.list, f.err
}

func (f *fakeChangeRequests) Get(_ context.Context, _ access.Actor, id string) (change.Change, error) {
	f.got = called{method: "Get", id: id}
	return f.c, f.err
}

func (f *fakeChangeRequests) Create(_ context.Context, _ access.Actor, in change.NewChange) (change.Change, error) {
	f.got, f.gotNew = called{method: "Create"}, in
	return f.c, f.err
}

func (f *fakeChangeRequests) AddDocument(_ context.Context, _ access.Actor, id, documentID string) (change.Change, error) {
	f.got = called{method: "AddDocument", id: id, input: documentID}
	return f.c, f.err
}

func (f *fakeChangeRequests) RemoveDocument(_ context.Context, _ access.Actor, id, documentID string) (change.Change, error) {
	f.got = called{method: "RemoveDocument", id: id, input: documentID}
	return f.c, f.err
}

func (f *fakeChangeRequests) AddAttachment(_ context.Context, _ access.Actor, id, fileName string, r io.Reader) (change.Change, error) {
	body, _ := io.ReadAll(r)
	f.got = called{method: "AddAttachment", id: id, input: fileName + ":" + string(body)}
	return f.c, f.err
}

func (f *fakeChangeRequests) RemoveAttachment(_ context.Context, _ access.Actor, id, attachmentID string) (change.Change, error) {
	f.got = called{method: "RemoveAttachment", id: id, input: attachmentID}
	return f.c, f.err
}

func (f *fakeChangeRequests) Verify(_ context.Context, _ access.Actor, id string, accept bool, note string) (change.Change, error) {
	f.got = called{method: "Verify", id: id, input: fmt.Sprint(accept, ":", note)}
	return f.c, f.err
}

func (f *fakeChangeRequests) SaveImpacts(_ context.Context, _ access.Actor, id string, items []change.ImpactInput) (change.Change, error) {
	f.got = called{method: "SaveImpacts", id: id, input: fmt.Sprint(items)}
	return f.c, f.err
}

func (f *fakeChangeRequests) Preview(_ context.Context, _ access.Actor, versionID, scope string) ([]change.PreviewItem, error) {
	f.got = called{method: "Preview", id: versionID, input: scope}
	return f.preview, f.err
}

func (f *fakeChangeRequests) Propagate(_ context.Context, _ access.Actor, in change.Propagation) ([]change.Change, error) {
	f.got, f.gotPropagation = called{method: "Propagate", id: in.VersionID, input: in.Scope}, in
	return f.list, f.err
}

func (f *fakeChangeRequests) CreateYokoten(_ context.Context, _ access.Actor, in change.NewYokoten) (change.Yokoten, error) {
	f.got, f.gotYokoten = called{method: "CreateYokoten", id: in.SourceChangeID, input: fmt.Sprint(in.DryRun)}, in
	return f.yokoten, f.err
}

func (f *fakeChangeRequests) GetYokoten(_ context.Context, _ access.Actor, id string) (change.Yokoten, error) {
	f.got = called{method: "GetYokoten", id: id}
	return f.yokoten, f.err
}

func (f *fakeChangeRequests) ListYokotens(_ context.Context, _ access.Actor) ([]change.Yokoten, error) {
	f.got = called{method: "ListYokotens"}
	return []change.Yokoten{f.yokoten}, f.err
}

func (f *fakeChangeRequests) Decide(_ context.Context, _ access.Actor, yokotenID, targetID, decision, reason string) (change.Yokoten, error) {
	f.got = called{method: "Decide", id: yokotenID, input: targetID + ":" + decision + ":" + reason}
	return f.yokoten, f.err
}

func (f *fakeChangeRequests) PendingLogoChange(context.Context, access.Actor, string) (*change.PendingLogo, error) {
	return nil, f.err
}

func (f *fakeChangeRequests) SubmitLogoChange(_ context.Context, _ access.Actor, in change.LogoChange) (change.Change, error) {
	f.got = called{method: "SubmitLogoChange", id: in.CustomerID, input: in.LogoChangeID}
	return change.Change{}, f.err
}

func (f *fakeChangeRequests) SubmitSQAM(_ context.Context, _ access.Actor, in change.SQAMChange) (change.Change, error) {
	f.got, f.gotSQAM = called{method: "SubmitSQAM", id: in.CustomerID, input: in.SQAMReference}, in
	return f.c, f.err
}

// OpenAttachment membuka berkas sementara berisi f.file.
func (f *fakeChangeRequests) OpenAttachment(_ context.Context, _ access.Actor, id, attachmentID string) (change.Attachment, *os.File, error) {
	f.got = called{method: "OpenAttachment", id: id, input: attachmentID}
	if f.err != nil {
		return change.Attachment{}, nil, f.err
	}
	tmp, err := os.CreateTemp("", "fake-attachment-")
	if err != nil {
		return change.Attachment{}, nil, err
	}
	_, _ = tmp.WriteString(f.file)
	_, _ = tmp.Seek(0, io.SeekStart)
	return change.Attachment{ID: attachmentID, FileName: "8D report.pdf"}, tmp, nil
}
