package notify

// Layanan admin notifikasi dan pekerja kotak keluar (D-194).

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
	"github.com/dkpauzi/revioten/internal/review"
)

// MaxAttempts adalah jumlah percobaan kirim sebelum email ditandai gagal.
// Jedanya 2, 4, 8, lalu 16 menit.
const MaxAttempts = 5

// Service mengurus pengaturan notifikasi, alamat email, dan pengiriman.
type Service struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	authz  access.Authorizer
	mailer Mailer
	config Config
	log    *slog.Logger
}

// NewService membuat Service. mailer nil berarti email tidak dikirim: email
// di kotak keluar ditandai skipped dengan alasan mail_not_configured.
func NewService(pool *pgxpool.Pool, authz access.Authorizer, mailer Mailer, config Config, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{pool: pool, q: db.New(pool), authz: authz, mailer: mailer, config: config, log: log}
}

// Event adalah satu jenis peristiwa beserta saklarnya.
type Event struct {
	Key     string
	Enabled bool
}

// Entry adalah satu email di kotak keluar.
type Entry struct {
	ID        string
	Event     string
	Recipient string
	Email     string
	Subject   string
	Status    string // pending, sending, sent, failed, atau skipped
	Reason    string
	Attempts  int
	CreatedAt time.Time
	SentAt    *time.Time
}

// Overview adalah isi halaman Notifications.
type Overview struct {
	Provider string // graph, smtp, log, atau kosong
	From     string
	Server   string
	Events   []Event
	Outbox   []Entry // 100 terbaru
}

func invalid(field, code, message string) error {
	return &problem.ValidationError{Problems: []problem.Problem{{Field: field, Code: code, Message: message}}}
}

// Overview memberi pengaturan dan kotak keluar (A-3).
func (s *Service) Overview(ctx context.Context, actor access.Actor) (Overview, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageSettings, access.Resource{}); err != nil {
		return Overview{}, err
	}
	out := Overview{Events: []Event{}, Outbox: []Entry{}}
	out.Provider, out.From, out.Server = s.config.Describe()
	if s.mailer == nil {
		out.Provider = ""
	}
	events, err := s.q.ListNotificationEvents(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("membaca jenis notifikasi: %w", err)
	}
	for _, e := range events {
		if e.Key != EventTest {
			out.Events = append(out.Events, Event{Key: e.Key, Enabled: e.Enabled})
		}
	}
	rows, err := s.q.ListOutbox(ctx)
	if err != nil {
		return Overview{}, fmt.Errorf("membaca kotak keluar: %w", err)
	}
	for _, r := range rows {
		out.Outbox = append(out.Outbox, Entry{ID: r.ID, Event: r.Event, Recipient: r.Recipient, Email: text(r.RecipientEmail), Subject: r.Subject,
			Status: r.Status, Reason: text(r.Reason), Attempts: int(r.Attempts), CreatedAt: r.CreatedAt.UTC(), SentAt: r.SentAt})
	}
	return out, nil
}

// SetEvent menyalakan atau mematikan satu jenis notifikasi (A-3).
func (s *Service) SetEvent(ctx context.Context, actor access.Actor, key string, enabled bool) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageSettings, access.Resource{}); err != nil {
		return err
	}
	if key == EventTest {
		return problem.ErrNotFound
	}
	return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		n, err := q.SetNotificationEvent(ctx, db.SetNotificationEventParams{Key: key, Enabled: enabled})
		if err != nil {
			return err
		}
		if n == 0 {
			return problem.ErrNotFound
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionNotificationEventChanged,
			Details: map[string]string{"event": key, "enabled": fmt.Sprint(enabled)}})
	})
}

// SendTest menulis satu email uji untuk orang pemegang akun aktor (A-3).
func (s *Service) SendTest(ctx context.Context, actor access.Actor) error {
	if err := s.authz.Authorize(ctx, actor, access.ManageSettings, access.Resource{}); err != nil {
		return err
	}
	email, err := s.q.GetPersonEmail(ctx, actor.PersonID)
	if err != nil {
		return err
	}
	if email == nil {
		return invalid("email", problem.Required, "Set your own email address first.")
	}
	return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.EnqueueNotification(ctx, db.EnqueueNotificationParams{Event: EventTest, PersonID: actor.PersonID,
			Subject: "Test email from the PFD, PFMEA, and CP system", Body: "This is a test email. If you can read it, the email settings work.\nIni email uji. Bila terbaca, pengaturan email sudah benar."}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionNotificationTestSent})
	})
}

// cleanEmail memeriksa satu alamat email. Kosong berarti dihapus.
func cleanEmail(raw string) (*string, error) {
	email := strings.TrimSpace(raw)
	if email == "" {
		return nil, nil
	}
	at := strings.Index(email, "@")
	if len(email) > 254 || at < 1 || strings.Count(email, "@") != 1 || !strings.Contains(email[at:], ".") ||
		strings.ContainsAny(email, " \t\r\n<>,;") || strings.HasSuffix(email, ".") {
		return nil, invalid("email", problem.InvalidFormat, "Enter an email address such as name@company.co.id.")
	}
	return &email, nil
}

// SetEmail mengganti alamat email orang pemegang satu akun: admin (A-1) untuk
// akun siapa pun, atau orangnya sendiri (D-104). Kosong menghapus alamatnya.
func (s *Service) SetEmail(ctx context.Context, actor access.Actor, accountID, raw string) (string, error) {
	if !db.IsUUID(accountID) {
		return "", problem.ErrNotFound
	}
	if accountID != actor.AccountID {
		if err := s.authz.Authorize(ctx, actor, access.ManageUsers, access.Resource{}); err != nil {
			return "", err
		}
	} else if err := s.authz.Authorize(ctx, actor, access.ManageOwnAccount, access.Resource{AccountID: accountID}); err != nil {
		return "", err
	}
	email, err := cleanEmail(raw)
	if err != nil {
		return "", err
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		a, err := q.GetAccountPerson(ctx, accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return problem.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := q.SetPersonEmail(ctx, db.SetPersonEmailParams{ID: a.PersonID, Email: email}); err != nil {
			return err
		}
		return audit.Record(ctx, q, actor, audit.Event{Action: audit.ActionPersonEmailChanged, TargetType: audit.TargetAccount, TargetID: accountID,
			Details: map[string]string{"login": a.Login, "email": audit.Change(text(a.Email), text(email))}})
	})
	if err != nil {
		return "", err
	}
	return text(email), nil
}

// OwnEmail memberi alamat email orang pemegang akun aktor.
func (s *Service) OwnEmail(ctx context.Context, actor access.Actor) (string, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageOwnAccount, access.Resource{AccountID: actor.AccountID}); err != nil {
		return "", err
	}
	email, err := s.q.GetPersonEmail(ctx, actor.PersonID)
	if err != nil {
		return "", err
	}
	return text(email), nil
}

// SendPending mengirim satu kelompok email dari kotak keluar. Hasilnya
// jumlah email yang diambil.
func (s *Service) SendPending(ctx context.Context, batch int) (int, error) {
	rows, err := s.q.ClaimOutbox(ctx, int32(batch))
	if err != nil {
		return 0, fmt.Errorf("mengambil kotak keluar: %w", err)
	}
	for _, r := range rows {
		if s.mailer == nil {
			if err := s.q.MarkOutboxSkipped(ctx, db.MarkOutboxSkippedParams{ID: r.ID, Reason: ptr("mail_not_configured")}); err != nil {
				return len(rows), err
			}
			continue
		}
		err := s.mailer.Send(ctx, Mail{To: text(r.RecipientEmail), Subject: r.Subject, Body: r.Body})
		if err == nil {
			if err := s.q.MarkOutboxSent(ctx, r.ID); err != nil {
				return len(rows), err
			}
			continue
		}
		s.log.Warn("email gagal dikirim", "id", r.ID, "attempt", r.Attempts+1, "error", err.Error())
		reason := err.Error()
		if len(reason) > 300 {
			reason = reason[:300]
		}
		if err := s.q.MarkOutboxRetry(ctx, db.MarkOutboxRetryParams{ID: r.ID, Reason: &reason, MaxAttempts: MaxAttempts}); err != nil {
			return len(rows), err
		}
	}
	return len(rows), nil
}

func ptr(s string) *string { return &s }

// CheckExpiringTokens menulis email ke admin untuk token integrasi yang
// kedaluwarsa dalam 7 hari, sekali per token.
func (s *Service) CheckExpiringTokens(ctx context.Context) error {
	tokens, err := s.q.ExpiringTokens(ctx)
	if err != nil {
		return err
	}
	for _, t := range tokens {
		err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := s.q.WithTx(tx)
			admins, err := q.AdminPeople(ctx)
			if err != nil {
				return err
			}
			m := message{event: EventTokenExpiring,
				subject: fmt.Sprintf("Integration token %s of %s expires on %s", t.Prefix, t.Name, t.ExpiresAt.UTC().Format(time.DateOnly)),
				body: fmt.Sprintf("The token %s of the service account %s expires on %s. Issue a new token, give it to the owning team, then revoke the old one.\nToken %s milik %s kedaluwarsa pada %s.",
					t.Prefix, t.Name, t.ExpiresAt.UTC().Format(time.DateOnly), t.Prefix, t.Name, t.ExpiresAt.UTC().Format(time.DateOnly))}
			if err := enqueue(ctx, q, access.Actor{}, m, admins); err != nil {
				return err
			}
			return q.MarkTokenExpiryNotified(ctx, t.ID)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// CheckDueReviews menulis email ke penyusun (E-3) customer itu untuk set
// dokumen yang akan atau sudah lewat jatuh tempo review berkala (D-196),
// sekali per set per tahap.
func (s *Service) CheckDueReviews(ctx context.Context, now time.Time) error {
	// Pekerja melihat semua customer; penerimanya dipilih menurut hak.
	ctx = db.AsSystem(ctx)
	o, err := review.Compute(ctx, s.q, review.Filter{}, now)
	if err != nil {
		return err
	}
	for _, set := range o.Sets {
		due := pgtype.Date{Time: set.DueOn, Valid: true}
		sent, err := s.q.ReviewNoticeSent(ctx, db.ReviewNoticeSentParams{CustomerCode: set.CustomerCode, Sequence: int32(set.Sequence), DueOn: due, Stage: set.Status})
		if err != nil {
			return err
		}
		if sent {
			continue
		}
		err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := s.q.WithTx(tx)
			people, err := q.PeopleWithPermission(ctx, db.PeopleWithPermissionParams{Permission: string(access.ProposeChanges), CustomerID: &set.CustomerID})
			if err != nil {
				return err
			}
			numbers := make([]string, 0, len(set.Documents))
			for _, d := range set.Documents {
				numbers = append(numbers, fmt.Sprintf("%s rev %d", d.Number, d.Revision))
			}
			name := fmt.Sprintf("%s · %04d", strings.ToUpper(set.CustomerCode), set.Sequence)
			subject := fmt.Sprintf("[%s] Periodic review of set %s is due on %s", strings.ToUpper(set.CustomerCode), name, set.DueOn.Format(time.DateOnly))
			if set.Status == review.StatusOverdue {
				subject = fmt.Sprintf("[%s] Periodic review of set %s is overdue since %s", strings.ToUpper(set.CustomerCode), name, set.DueOn.Format(time.DateOnly))
			}
			m := message{event: EventReviewDue, subject: subject,
				body: fmt.Sprintf("The documents %s (models %s) had no revision since %s. Review them: with changes (a change request with the trigger Periodic review) or with no change (a review record).\nSet dokumen %s harus direview berkala paling lambat %s.",
					strings.Join(numbers, ", "), strings.Join(set.Models, ", "), set.LastRelease.Format(time.DateOnly), name, set.DueOn.Format(time.DateOnly)) + s.link("/periodic-reviews")}
			if err := enqueue(ctx, q, access.Actor{}, m, people); err != nil {
				return err
			}
			return q.AddReviewNotice(ctx, db.AddReviewNoticeParams{CustomerCode: set.CustomerCode, Sequence: int32(set.Sequence), DueOn: due, Stage: set.Status})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) link(path string) string {
	if s.config.BaseURL == "" {
		return ""
	}
	return "\n\n" + strings.TrimRight(s.config.BaseURL, "/") + path
}

// Run mengirim kotak keluar sampai ctx selesai: setiap every, dan memeriksa
// token yang segera kedaluwarsa sekali setiap jam.
func (s *Service) Run(ctx context.Context, every time.Duration) {
	// Email yang tertahan berstatus sending (server mati) diantrekan lagi.
	if err := s.q.RequeueSendingOutbox(ctx); err != nil {
		s.log.Error("mengantrekan ulang email", "error", err.Error())
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	var lastTokenCheck time.Time
	for {
		if time.Since(lastTokenCheck) >= time.Hour {
			if err := s.CheckExpiringTokens(ctx); err != nil {
				s.log.Error("memeriksa token yang segera kedaluwarsa", "error", err.Error())
			}
			if err := s.CheckDueReviews(ctx, time.Now()); err != nil {
				s.log.Error("memeriksa review berkala yang jatuh tempo", "error", err.Error())
			}
			lastTokenCheck = time.Now()
		}
		for {
			n, err := s.SendPending(ctx, 20)
			if err != nil {
				s.log.Error("mengirim email", "error", err.Error())
				break
			}
			if n == 0 || ctx.Err() != nil {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
