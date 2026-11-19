package server

// Handler tanda tangan digital (fase 3; D-198, D-203, D-204). Aturannya ada
// di internal/signature.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/signature"
)

// SignatureService adalah operasi tanda tangan digital.
type SignatureService interface {
	SetMode(ctx context.Context, actor access.Actor, customerID, mode string) error
	Plan(ctx context.Context, actor access.Actor, documentID string) (signature.Plan, error)
	Send(ctx context.Context, actor access.Actor, documentID string, in signature.SendInput) (signature.Request, error)
	Get(ctx context.Context, actor access.Actor, requestID string) (signature.Request, error)
	Sign(ctx context.Context, actor access.Actor, requestID, password string) (signature.Request, error)
	Reject(ctx context.Context, actor access.Actor, requestID, reason, password string) (signature.Request, error)
	Withdraw(ctx context.Context, actor access.Actor, requestID string) (signature.Request, error)
	ForDocument(ctx context.Context, actor access.Actor, documentID string) ([]signature.Request, error)
	List(ctx context.Context, actor access.Actor) (signature.Overview, error)
}

func toAPISignatureRequest(r signature.Request) SignatureRequest {
	out := SignatureRequest{
		Id: r.ID, DocumentId: r.DocumentID, Number: r.Number, DocumentType: r.DocumentType, CustomerId: r.CustomerID,
		CustomerCode: r.CustomerCode, Revision: r.Revision, RouteName: r.RouteName, ContentSha256: r.ContentSHA256,
		Note: optional(r.Note), Status: SignatureRequestStatus(r.Status), SentBy: r.SentBy, SentById: r.SentByID, SentAt: r.SentAt,
		ClosedAt: r.ClosedAt, ClosedBy: optional(r.ClosedBy), RejectReason: optional(r.RejectReason),
		NextSlot: r.NextSlot, Changes: r.Changes, Slots: make([]SignatureSlot, 0, len(r.Slots)),
	}
	if d := releaseDate(r.ReleaseDate); d != nil {
		out.ReleaseDate = *d
	}
	for _, s := range r.Slots {
		out.Slots = append(out.Slots, SignatureSlot{
			Order: s.Order, Stage: SignatureSlotStage(s.Stage), PrintTitle: s.PrintTitle, Position: s.Position,
			PersonId: s.PersonID, PersonName: s.PersonName, SignedAt: s.SignedAt,
		})
	}
	return out
}

func toAPISignatureRequests(list []signature.Request) []SignatureRequest {
	out := make([]SignatureRequest, 0, len(list))
	for _, r := range list {
		out = append(out, toAPISignatureRequest(r))
	}
	return out
}

// SetCustomerSignatureMode menjawab PUT /api/v1/customers/{customerId}/signature-mode.
func (a *apiServer) SetCustomerSignatureMode(ctx context.Context, req SetCustomerSignatureModeRequestObject) (SetCustomerSignatureModeResponseObject, error) {
	actor := actorFrom(ctx)
	if err := a.signatures.SetMode(ctx, actor, req.CustomerId, string(req.Body.Mode)); err != nil {
		return nil, err
	}
	list, err := a.customers.List(ctx, actor)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		if c.ID == req.CustomerId {
			return SetCustomerSignatureMode200JSONResponse(toAPICustomer(c)), nil
		}
	}
	return nil, problem.ErrNotFound
}

// GetSignaturePlan menjawab GET /api/v1/documents/{documentId}/signature-plan.
func (a *apiServer) GetSignaturePlan(ctx context.Context, req GetSignaturePlanRequestObject) (GetSignaturePlanResponseObject, error) {
	p, err := a.signatures.Plan(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	out := SignaturePlan{DocumentId: p.DocumentID, Number: p.Number, Revision: p.Revision, Mode: SignaturePlanMode(p.Mode), Slots: make([]SignaturePlanSlot, 0, len(p.Slots))}
	if p.RouteID != "" {
		out.Route = &ApprovalRouteRef{Id: p.RouteID, Name: p.RouteName}
	}
	for _, s := range p.Slots {
		slot := SignaturePlanSlot{Order: s.Order, Stage: SignaturePlanSlotStage(s.Stage), PrintTitle: s.PrintTitle, Candidates: make([]SignatureCandidate, 0, len(s.Candidates))}
		for _, c := range s.Candidates {
			slot.Candidates = append(slot.Candidates, SignatureCandidate{PersonId: c.PersonID, Name: c.Name, PositionKey: c.PositionKey, Position: c.Position})
		}
		out.Slots = append(out.Slots, slot)
	}
	return GetSignaturePlan200JSONResponse(out), nil
}

// ListDocumentSignatureRequests menjawab GET /api/v1/documents/{documentId}/signature-requests.
func (a *apiServer) ListDocumentSignatureRequests(ctx context.Context, req ListDocumentSignatureRequestsRequestObject) (ListDocumentSignatureRequestsResponseObject, error) {
	list, err := a.signatures.ForDocument(ctx, actorFrom(ctx), req.DocumentId)
	if err != nil {
		return nil, err
	}
	return ListDocumentSignatureRequests200JSONResponse(SignatureRequestList{Items: toAPISignatureRequests(list)}), nil
}

// SendForSignature menjawab POST /api/v1/documents/{documentId}/signature-requests.
func (a *apiServer) SendForSignature(ctx context.Context, req SendForSignatureRequestObject) (SendForSignatureResponseObject, error) {
	in := signature.SendInput{Signers: map[int]string{}, Note: value(req.Body.Note), Password: req.Body.Password}
	for _, s := range req.Body.Signers {
		in.Signers[s.Slot] = s.PersonId
	}
	r, err := a.signatures.Send(ctx, actorFrom(ctx), req.DocumentId, in)
	if err != nil {
		return nil, err
	}
	return SendForSignature201JSONResponse(toAPISignatureRequest(r)), nil
}

// ListSignatureRequests menjawab GET /api/v1/signature-requests.
func (a *apiServer) ListSignatureRequests(ctx context.Context, _ ListSignatureRequestsRequestObject) (ListSignatureRequestsResponseObject, error) {
	o, err := a.signatures.List(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return ListSignatureRequests200JSONResponse(SignatureOverview{
		Waiting: toAPISignatureRequests(o.Waiting), Sent: toAPISignatureRequests(o.Sent), ToRelease: toAPISignatureRequests(o.ToRelease),
	}), nil
}

// GetSignatureRequest menjawab GET /api/v1/signature-requests/{requestId}.
func (a *apiServer) GetSignatureRequest(ctx context.Context, req GetSignatureRequestRequestObject) (GetSignatureRequestResponseObject, error) {
	r, err := a.signatures.Get(ctx, actorFrom(ctx), req.RequestId)
	if err != nil {
		return nil, err
	}
	return GetSignatureRequest200JSONResponse(toAPISignatureRequest(r)), nil
}

// SignSignatureRequest menjawab POST /api/v1/signature-requests/{requestId}/sign.
func (a *apiServer) SignSignatureRequest(ctx context.Context, req SignSignatureRequestRequestObject) (SignSignatureRequestResponseObject, error) {
	r, err := a.signatures.Sign(ctx, actorFrom(ctx), req.RequestId, req.Body.Password)
	if err != nil {
		return nil, err
	}
	return SignSignatureRequest200JSONResponse(toAPISignatureRequest(r)), nil
}

// RejectSignatureRequest menjawab POST /api/v1/signature-requests/{requestId}/reject.
func (a *apiServer) RejectSignatureRequest(ctx context.Context, req RejectSignatureRequestRequestObject) (RejectSignatureRequestResponseObject, error) {
	r, err := a.signatures.Reject(ctx, actorFrom(ctx), req.RequestId, req.Body.Reason, req.Body.Password)
	if err != nil {
		return nil, err
	}
	return RejectSignatureRequest200JSONResponse(toAPISignatureRequest(r)), nil
}

// WithdrawSignatureRequest menjawab POST /api/v1/signature-requests/{requestId}/withdraw.
func (a *apiServer) WithdrawSignatureRequest(ctx context.Context, req WithdrawSignatureRequestRequestObject) (WithdrawSignatureRequestResponseObject, error) {
	r, err := a.signatures.Withdraw(ctx, actorFrom(ctx), req.RequestId)
	if err != nil {
		return nil, err
	}
	return WithdrawSignatureRequest200JSONResponse(toAPISignatureRequest(r)), nil
}
