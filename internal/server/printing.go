package server

// Handler cetak, scan, layout, dan verifikasi (Tahap 6). Aturannya ada di
// internal/printout.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/dkpauzi/revioten/internal/printout"
	"github.com/dkpauzi/revioten/internal/problem"
)

// PrintDocument menjawab POST /api/v1/documents/{documentId}/print.
func (a *apiServer) PrintDocument(ctx context.Context, req PrintDocumentRequestObject) (PrintDocumentResponseObject, error) {
	b := req.Body
	f, err := a.printing.Print(ctx, actorFrom(ctx), req.DocumentId, printout.Request{Revision: b.Revision, Purpose: b.Purpose, Format: b.Format})
	if err != nil {
		return nil, err
	}
	disposition := "attachment; filename*=UTF-8''" + url.PathEscape(f.Name)
	return PrintDocument200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(f.Body), ContentLength: int64(len(f.Body)),
		Headers: PrintDocument200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

// ListDocumentPrints menjawab GET /api/v1/documents/{documentId}/prints.
func (a *apiServer) ListDocumentPrints(ctx context.Context, req ListDocumentPrintsRequestObject) (ListDocumentPrintsResponseObject, error) {
	logs, err := a.printing.Logs(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	out := PrintLogList{Items: make([]PrintLog, 0, len(logs))}
	for _, l := range logs {
		out.Items = append(out.Items, PrintLog{Id: l.ID, Revision: l.Revision, Purpose: PrintLogPurpose(l.Purpose), Format: PrintLogFormat(l.Format),
			CopyNumber: l.CopyNumber, Sha256: l.SHA256, PrintedBy: l.PrintedBy, PrintedAt: l.PrintedAt})
	}
	return ListDocumentPrints200JSONResponse(out), nil
}

// UploadSignedScan menjawab POST /api/v1/documents/{documentId}/scan.
func (a *apiServer) UploadSignedScan(ctx context.Context, req UploadSignedScanRequestObject) (UploadSignedScanResponseObject, error) {
	for {
		part, err := req.Body.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: problem.InvalidFormat, Message: "Send the file as multipart form data."}}}
		}
		if part.FormName() != "file" {
			continue
		}
		sc, err := a.printing.UploadScan(ctx, actorFrom(ctx), req.DocumentId, part.FileName(), part)
		if err != nil {
			return nil, err
		}
		return UploadSignedScan201JSONResponse(toAPIScan(sc)), nil
	}
	return nil, &problem.ValidationError{Problems: []problem.Problem{{Field: "file", Code: problem.Required, Message: "Choose the signed scan."}}}
}

// GetSignedScan menjawab GET /api/v1/documents/{documentId}/revisions/{revision}/scan.
func (a *apiServer) GetSignedScan(ctx context.Context, req GetSignedScanRequestObject) (GetSignedScanResponseObject, error) {
	sc, f, err := a.printing.OpenScan(ctx, actorFrom(ctx), req.DocumentId, req.Revision)
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, err
	}
	disposition := "attachment; filename*=UTF-8''" + url.PathEscape(sc.FileName)
	return GetSignedScan200ApplicationoctetStreamResponse{
		Body: bytes.NewReader(content), ContentLength: int64(len(content)),
		Headers: GetSignedScan200ResponseHeaders{ContentDisposition: &disposition},
	}, nil
}

func toAPIScan(sc printout.Scan) SignedScan {
	return SignedScan{Revision: sc.Revision, FileName: sc.FileName, ContentType: sc.ContentType, Size: sc.Size, Sha256: sc.SHA256, UploadedBy: sc.UploadedBy, UploadedAt: sc.UploadedAt}
}

// VerifyRevision menjawab GET /api/v1/verify/{revisionId}.
func (a *apiServer) VerifyRevision(ctx context.Context, req VerifyRevisionRequestObject) (VerifyRevisionResponseObject, error) {
	v, err := a.printing.Verify(ctx, actorFrom(ctx), req.RevisionId)
	if err != nil {
		return nil, err
	}
	return VerifyRevision200JSONResponse(RevisionVerification{
		RevisionId: v.RevisionID, DocumentId: v.DocumentID, Number: v.Number, DocumentType: v.DocumentType, CustomerCode: v.CustomerCode,
		Revision: v.Revision, Status: RevisionVerificationStatus(v.Status), ReleasedAt: v.ReleasedAt, Sha256: optional(v.SHA256),
	}), nil
}

// ListPrintLayouts menjawab GET /api/v1/print-layouts.
func (a *apiServer) ListPrintLayouts(ctx context.Context, _ ListPrintLayoutsRequestObject) (ListPrintLayoutsResponseObject, error) {
	list, err := a.printing.Layouts(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := PrintLayoutList{Items: make([]PrintLayout, 0, len(list))}
	for _, l := range list {
		out.Items = append(out.Items, toAPILayout(l))
	}
	return ListPrintLayouts200JSONResponse(out), nil
}

// CreatePrintLayout menjawab POST /api/v1/print-layouts.
func (a *apiServer) CreatePrintLayout(ctx context.Context, req CreatePrintLayoutRequestObject) (CreatePrintLayoutResponseObject, error) {
	in, err := toLayoutInput(*req.Body)
	if err != nil {
		return nil, err
	}
	l, err := a.printing.CreateLayout(ctx, actorFrom(ctx), in)
	if err != nil {
		return nil, err
	}
	return CreatePrintLayout201JSONResponse(toAPILayout(l)), nil
}

// UpdatePrintLayout menjawab PUT /api/v1/print-layouts/{layoutId}.
func (a *apiServer) UpdatePrintLayout(ctx context.Context, req UpdatePrintLayoutRequestObject) (UpdatePrintLayoutResponseObject, error) {
	in, err := toLayoutInput(*req.Body)
	if err != nil {
		return nil, err
	}
	l, err := a.printing.UpdateLayout(ctx, actorFrom(ctx), req.LayoutId, in)
	if err != nil {
		return nil, err
	}
	return UpdatePrintLayout200JSONResponse(toAPILayout(l)), nil
}

// ListCustomerPrintLayouts menjawab GET /api/v1/customers/{customerId}/print-layouts.
func (a *apiServer) ListCustomerPrintLayouts(ctx context.Context, req ListCustomerPrintLayoutsRequestObject) (ListCustomerPrintLayoutsResponseObject, error) {
	m, err := a.printing.CustomerLayouts(ctx, actorFrom(ctx), req.CustomerId)
	if err != nil {
		return nil, err
	}
	return ListCustomerPrintLayouts200JSONResponse(toCustomerLayouts(m)), nil
}

// SetCustomerPrintLayout menjawab PUT /api/v1/customers/{customerId}/print-layouts/{documentTypeId}.
func (a *apiServer) SetCustomerPrintLayout(ctx context.Context, req SetCustomerPrintLayoutRequestObject) (SetCustomerPrintLayoutResponseObject, error) {
	if err := a.printing.SetCustomerLayout(ctx, actorFrom(ctx), req.CustomerId, req.DocumentTypeId, deref(req.Body.LayoutId)); err != nil {
		return nil, err
	}
	m, err := a.printing.CustomerLayouts(ctx, actorFrom(ctx), req.CustomerId)
	if err != nil {
		return nil, err
	}
	return SetCustomerPrintLayout200JSONResponse(toCustomerLayouts(m)), nil
}

func toCustomerLayouts(m map[string]string) CustomerPrintLayoutList {
	out := CustomerPrintLayoutList{Items: make([]CustomerPrintLayout, 0, len(m))}
	for t, l := range m {
		out.Items = append(out.Items, CustomerPrintLayout{DocumentTypeId: t, LayoutId: l})
	}
	return out
}

func toLayoutInput(b PrintLayoutInput) (printout.LayoutInput, error) {
	raw, err := json.Marshal(b.Layout)
	if err != nil {
		return printout.LayoutInput{}, err
	}
	return printout.LayoutInput{Name: b.Name, DocumentTypeID: deref(b.DocumentTypeId), Layout: raw}, nil
}

func toAPILayout(l printout.PrintLayout) PrintLayout {
	lay := l.Layout
	widths := map[string]float32{}
	for k, v := range lay.Widths {
		widths[k] = float32(v)
	}
	m := lay.MarginsMM
	return PrintLayout{
		Id: l.ID, Name: l.Name, DocumentTypeId: l.DocumentTypeID, DocumentTypeKey: l.DocumentTypeKey, IsDefault: l.IsDefault, UpdatedAt: l.UpdatedAt.Truncate(time.Microsecond),
		Layout: PrintLayoutContent{
			Paper: PrintLayoutContentPaper(lay.Paper), Orientation: PrintLayoutContentOrientation(lay.Orientation), FontSize: float32(lay.FontSize),
			Cover: lay.Cover, RevisionMarks: lay.RevisionMarks, Widths: widths,
			MarginsMm: struct {
				Bottom float32 `json:"bottom"`
				Left   float32 `json:"left"`
				Right  float32 `json:"right"`
				Top    float32 `json:"top"`
			}{Bottom: float32(m.Bottom), Left: float32(m.Left), Right: float32(m.Right), Top: float32(m.Top)},
		},
	}
}

// GetPagePreview menjawab GET /api/v1/documents/{documentId}/page-preview (C3).
func (a *apiServer) GetPagePreview(ctx context.Context, req GetPagePreviewRequestObject) (GetPagePreviewResponseObject, error) {
	v, err := a.printing.Preview(ctx, actorFrom(ctx), req.DocumentId, req.Params.Revision)
	if err != nil {
		return nil, err
	}
	out := PagePreview{Title: v.Title, Number: v.Number, Revision: v.Revision, PaperWMm: float32(v.Plan.PaperW), PaperHMm: float32(v.Plan.PaperH),
		FontSize: float32(v.FontSize), HeadMm: float32(v.Plan.HeadMM), GroupRowMm: float32(v.Plan.GroupRowMM), LabelRowMm: float32(v.Plan.LabelRowMM),
		MarkMm: float32(v.Plan.MarkMM), FlowColumn: v.FlowColumn, FlowLeftMm: float32(v.FlowLeftMM),
		ColumnsMm: make([]float32, 0, len(v.Plan.ColumnsMM)), Header: []struct {
			Label string `json:"label"`
			Value string `json:"value"`
		}{}, Columns: []struct {
			Group string `json:"group"`
			Label string `json:"label"`
		}{}, Pages: make([]PreviewPage, 0, len(v.Pages))}
	if v.HeadHTML != "" {
		head := v.HeadHTML
		out.HeadHtml = &head
	}
	if v.Legend != "" {
		legend, h := v.Legend, float32(v.Plan.LegendMM)
		out.LegendSvg, out.LegendMm = &legend, &h
	}
	out.MarginsMm.Top, out.MarginsMm.Right, out.MarginsMm.Bottom, out.MarginsMm.Left = float32(v.MarginsMM.Top), float32(v.MarginsMM.Right), float32(v.MarginsMM.Bottom), float32(v.MarginsMM.Left)
	for _, w := range v.Plan.ColumnsMM {
		out.ColumnsMm = append(out.ColumnsMm, float32(w))
	}
	for _, p := range v.Header {
		out.Header = append(out.Header, struct {
			Label string `json:"label"`
			Value string `json:"value"`
		}{p.Label, p.Value})
	}
	for _, c := range v.Columns {
		out.Columns = append(out.Columns, struct {
			Group string `json:"group"`
			Label string `json:"label"`
		}{c.Group, c.Label})
	}
	for _, p := range v.Pages {
		page := PreviewPage{FlowSvg: string(p.Flow)}
		if p.Diagram != "" {
			svg := string(p.Diagram)
			page.DiagramSvg = &svg
		}
		for _, r := range p.Rows {
			row := struct {
				Cells []struct {
					Flow     bool    `json:"flow"`
					HeightMm float32 `json:"height_mm"`
					Merged   bool    `json:"merged"`
					Rowspan  int     `json:"rowspan"`
					Skip     bool    `json:"skip"`
					Text     string  `json:"text"`
				} `json:"cells"`
				HeightMm float32 `json:"height_mm"`
				Mark     string  `json:"mark"`
				Section  string  `json:"section"`
			}{HeightMm: float32(r.HeightMM), Mark: r.Mark, Section: r.Section}
			for _, c := range r.Cells {
				row.Cells = append(row.Cells, struct {
					Flow     bool    `json:"flow"`
					HeightMm float32 `json:"height_mm"`
					Merged   bool    `json:"merged"`
					Rowspan  int     `json:"rowspan"`
					Skip     bool    `json:"skip"`
					Text     string  `json:"text"`
				}{c.Flow, float32(c.HeightMM), c.Merged, c.Rowspan, c.Skip, c.Text})
			}
			page.Rows = append(page.Rows, row)
		}
		out.Pages = append(out.Pages, page)
	}
	return GetPagePreview200JSONResponse(out), nil
}
