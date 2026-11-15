package server

// Handler email notifikasi dan alamat email (D-194). Aturannya ada di
// internal/notify.

import (
	"context"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/notify"
)

// NotificationService adalah operasi notifikasi dan alamat email.
type NotificationService interface {
	Overview(ctx context.Context, actor access.Actor) (notify.Overview, error)
	SetEvent(ctx context.Context, actor access.Actor, key string, enabled bool) error
	SendTest(ctx context.Context, actor access.Actor) error
	SetEmail(ctx context.Context, actor access.Actor, accountID, email string) (string, error)
	OwnEmail(ctx context.Context, actor access.Actor) (string, error)
}

// GetOwnEmail menjawab GET /api/v1/account/email.
func (a *apiServer) GetOwnEmail(ctx context.Context, _ GetOwnEmailRequestObject) (GetOwnEmailResponseObject, error) {
	email, err := a.notifications.OwnEmail(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return GetOwnEmail200JSONResponse{Email: email}, nil
}

// SetOwnEmail menjawab PUT /api/v1/account/email.
func (a *apiServer) SetOwnEmail(ctx context.Context, req SetOwnEmailRequestObject) (SetOwnEmailResponseObject, error) {
	actor := actorFrom(ctx)
	email, err := a.notifications.SetEmail(ctx, actor, actor.AccountID, req.Body.Email)
	if err != nil {
		return nil, err
	}
	return SetOwnEmail200JSONResponse{Email: email}, nil
}

// SetAccountEmail menjawab PUT /api/v1/accounts/{accountId}/email.
func (a *apiServer) SetAccountEmail(ctx context.Context, req SetAccountEmailRequestObject) (SetAccountEmailResponseObject, error) {
	email, err := a.notifications.SetEmail(ctx, actorFrom(ctx), req.AccountId, req.Body.Email)
	if err != nil {
		return nil, err
	}
	return SetAccountEmail200JSONResponse{Email: email}, nil
}

// GetNotifications menjawab GET /api/v1/notifications.
func (a *apiServer) GetNotifications(ctx context.Context, _ GetNotificationsRequestObject) (GetNotificationsResponseObject, error) {
	o, err := a.notifications.Overview(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	out := NotificationOverview{Provider: o.Provider, From: o.From, Server: o.Server, Outbox: make([]OutboxEntry, 0, len(o.Outbox))}
	out.Events = make([]struct {
		Enabled bool   `json:"enabled"`
		Key     string `json:"key"`
	}, len(o.Events))
	for i, e := range o.Events {
		out.Events[i].Key, out.Events[i].Enabled = e.Key, e.Enabled
	}
	for _, e := range o.Outbox {
		out.Outbox = append(out.Outbox, OutboxEntry{Id: e.ID, Event: e.Event, Recipient: e.Recipient, Email: optional(e.Email), Subject: e.Subject,
			Status: OutboxEntryStatus(e.Status), Reason: optional(e.Reason), Attempts: e.Attempts, CreatedAt: e.CreatedAt, SentAt: e.SentAt})
	}
	return GetNotifications200JSONResponse(out), nil
}

// SetNotificationEvent menjawab PUT /api/v1/notifications/events/{event}.
func (a *apiServer) SetNotificationEvent(ctx context.Context, req SetNotificationEventRequestObject) (SetNotificationEventResponseObject, error) {
	if err := a.notifications.SetEvent(ctx, actorFrom(ctx), req.Event, req.Body.Enabled); err != nil {
		return nil, err
	}
	return SetNotificationEvent204Response{}, nil
}

// SendTestEmail menjawab POST /api/v1/notifications/test.
func (a *apiServer) SendTestEmail(ctx context.Context, _ SendTestEmailRequestObject) (SendTestEmailResponseObject, error) {
	if err := a.notifications.SendTest(ctx, actorFrom(ctx)); err != nil {
		return nil, err
	}
	return SendTestEmail204Response{}, nil
}
