package server

import (
	"context"

	"github.com/dkpauzi/revioten/internal/audit"
)

// defaultAuditPage adalah jumlah entri per halaman bila limit tidak diberikan.
const defaultAuditPage = 50

// ListAuditLog menjawab GET /api/v1/audit-log.
func (a *apiServer) ListAuditLog(ctx context.Context, req ListAuditLogRequestObject) (ListAuditLogResponseObject, error) {
	limit := defaultAuditPage
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	entries, err := a.audit.List(ctx, actorFrom(ctx), value(req.Params.Before), limit)
	if err != nil {
		return nil, err
	}
	list := AuditEntryList{Items: make([]AuditEntry, 0, len(entries))}
	for _, e := range entries {
		list.Items = append(list.Items, toAPIAuditEntry(e))
	}
	// Halaman penuh berarti mungkin masih ada entri yang lebih lama.
	if n := len(entries); n > 0 && n == limit && entries[n-1].Seq > 1 {
		next := entries[n-1].Seq
		list.NextBefore = &next
	}
	return ListAuditLog200JSONResponse(list), nil
}

// VerifyAuditLog menjawab GET /api/v1/audit-log/verification.
func (a *apiServer) VerifyAuditLog(ctx context.Context, _ VerifyAuditLogRequestObject) (VerifyAuditLogResponseObject, error) {
	v, err := a.audit.Verify(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	result := AuditVerification{Valid: v.Valid, Entries: v.Entries}
	if v.LastSeq > 0 {
		result.LastSeq, result.LastHash = &v.LastSeq, &v.LastHash
	}
	if !v.Valid {
		reason := AuditVerificationReason(v.Reason)
		result.BrokenSeq, result.Reason = &v.BrokenSeq, &reason
	}
	return VerifyAuditLog200JSONResponse(result), nil
}

// toAPIAuditEntry mengubah satu entri dari paket audit menjadi AuditEntry kontrak API.
func toAPIAuditEntry(e audit.Entry) AuditEntry {
	entry := AuditEntry{
		Seq: e.Seq, OccurredAt: e.OccurredAt, Action: e.Action, Details: e.Details,
		PrevHash: e.PrevHash, Hash: e.Hash,
		Actor: AuditActor{
			Type: AuditActorType(e.ActorType), AccountId: optional(e.ActorAccountID), Login: optional(e.ActorLogin),
			PersonId: optional(e.ActorPersonID), PersonName: optional(e.ActorPersonName),
		},
		RequestId: optional(e.RequestID), SourceIp: optional(e.SourceIP),
	}
	if e.TargetType != "" {
		entry.Target = &AuditTarget{Type: e.TargetType, Id: e.TargetID, Label: optional(e.TargetLabel)}
	}
	return entry
}

// optional mengubah teks kosong menjadi field yang tidak dikirim.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
