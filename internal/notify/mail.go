package notify

// Pengirim email (D-194). Tiga pilihan, diatur IT lewat pengaturan:
//
//   - graph: Microsoft Graph API (POST /users/{pengirim}/sendMail) dengan
//     app registration di Microsoft Entra ID dan izin aplikasi Mail.Send.
//     Ini jalur yang disarankan Microsoft untuk aplikasi yang mengirim email.
//   - smtp: SMTP dengan STARTTLS, misalnya smtp.office365.com port 587 dengan
//     akun pengirim, atau relay Exchange internal perusahaan.
//   - log: tidak mengirim; isi email ditulis ke log server. Untuk demo.
//
// Belum diuji dengan tenant Microsoft perusahaan (ASUMSI E-126): test memakai
// server tiruan.

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Mail adalah satu email yang dikirim.
type Mail struct {
	To      string
	Subject string
	Body    string // teks biasa
}

// Mailer mengirim satu email.
type Mailer interface {
	Send(ctx context.Context, m Mail) error
}

// Config adalah pengaturan pengirim, dibaca dari .env (lihat .env.example).
type Config struct {
	Provider string // graph, smtp, log, atau kosong (email tidak dikirim)
	From     string // alamat pengirim, misalnya revioten@perusahaan.co.id

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPStartTLS bool

	GraphTenantID     string
	GraphClientID     string
	GraphClientSecret string
}

// Describe memberi keterangan pengaturan tanpa rahasia, untuk layar admin.
func (c Config) Describe() (provider, from, server string) {
	switch c.Provider {
	case "smtp":
		return "smtp", c.From, net.JoinHostPort(c.SMTPHost, strconv.Itoa(c.SMTPPort))
	case "graph":
		return "graph", c.From, "graph.microsoft.com (tenant " + c.GraphTenantID + ")"
	case "log":
		return "log", c.From, "server log"
	}
	return "", c.From, ""
}

// NewMailer membuat pengirim dari pengaturan. Hasil nil tanpa error berarti
// email tidak dikirim (Provider kosong).
func NewMailer(c Config, log *slog.Logger) (Mailer, error) {
	if log == nil {
		log = slog.Default()
	}
	need := func(pairs ...string) error {
		var missing []string
		for i := 0; i < len(pairs); i += 2 {
			if strings.TrimSpace(pairs[i+1]) == "" {
				missing = append(missing, pairs[i])
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("pengaturan email belum lengkap: %s", strings.Join(missing, ", "))
		}
		return nil
	}
	switch c.Provider {
	case "":
		return nil, nil
	case "log":
		return logMailer{log: log}, nil
	case "smtp":
		if err := need("MAIL_FROM", c.From, "SMTP_HOST", c.SMTPHost); err != nil {
			return nil, err
		}
		return &SMTPMailer{Config: c}, nil
	case "graph":
		if err := need("MAIL_FROM", c.From, "GRAPH_TENANT_ID", c.GraphTenantID, "GRAPH_CLIENT_ID", c.GraphClientID, "GRAPH_CLIENT_SECRET", c.GraphClientSecret); err != nil {
			return nil, err
		}
		return &GraphMailer{Config: c, Client: &http.Client{Timeout: 30 * time.Second}}, nil
	}
	return nil, fmt.Errorf("MAIL_PROVIDER %q tidak dikenal: pakai graph, smtp, atau log", c.Provider)
}

// logMailer menulis email ke log server, tanpa mengirim.
type logMailer struct{ log *slog.Logger }

func (l logMailer) Send(_ context.Context, m Mail) error {
	l.log.Info("email (mode log, tidak dikirim)", "to", m.To, "subject", m.Subject, "body", m.Body)
	return nil
}

// message menyusun email MIME teks biasa UTF-8. Subjek dikodekan supaya
// huruf di luar ASCII tetap terbaca.
func mimeMessage(from, to, subject, body string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	for len(encoded) > 76 {
		b.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	b.WriteString(encoded + "\r\n")
	return b.Bytes()
}

// validAddress menolak alamat yang bisa menyisipkan header lain.
func validAddress(a string) bool {
	return a != "" && !strings.ContainsAny(a, "\r\n<>,;") && strings.Count(a, "@") == 1
}

// SMTPMailer mengirim lewat SMTP. Dengan STARTTLS koneksi dienkripsi sebelum
// login, dan login tanpa enkripsi ditolak.
type SMTPMailer struct {
	Config Config
	// TLS dipakai test untuk mempercayai sertifikat server tiruan.
	TLS *tls.Config
}

func (s *SMTPMailer) Send(ctx context.Context, m Mail) error {
	if !validAddress(m.To) || !validAddress(s.Config.From) {
		return errors.New("alamat email tidak sah")
	}
	addr := net.JoinHostPort(s.Config.SMTPHost, strconv.Itoa(s.Config.SMTPPort))
	var d net.Dialer
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("menghubungi server SMTP: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	c, err := smtp.NewClient(conn, s.Config.SMTPHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("membuka sesi SMTP: %w", err)
	}
	defer c.Close()
	if s.Config.SMTPStartTLS {
		cfg := s.TLS
		if cfg == nil {
			cfg = &tls.Config{ServerName: s.Config.SMTPHost, MinVersion: tls.VersionTLS12}
		}
		if err := c.StartTLS(cfg); err != nil {
			return fmt.Errorf("STARTTLS: %w", err)
		}
	}
	if s.Config.SMTPUsername != "" {
		if ok, _ := c.Extension("AUTH"); !ok {
			return errors.New("server SMTP tidak menawarkan login")
		}
		if err := c.Auth(&loginAuth{user: s.Config.SMTPUsername, pass: s.Config.SMTPPassword, host: s.Config.SMTPHost}); err != nil {
			// Pesan server bisa memuat nama akun, tetapi tidak pernah password.
			return fmt.Errorf("login SMTP ditolak: %w", err)
		}
	}
	if err := c.Mail(s.Config.From); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := c.Rcpt(m.To); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(mimeMessage(s.Config.From, m.To, m.Subject, m.Body)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mengakhiri DATA: %w", err)
	}
	return c.Quit()
}

// loginAuth memilih PLAIN bila server menawarkannya, dan LOGIN bila tidak.
// Microsoft 365 menawarkan LOGIN, bukan PLAIN. Keduanya hanya dipakai di
// koneksi terenkripsi atau localhost (aturan net/smtp).
type loginAuth struct {
	user, pass, host string
	mech             string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && !isLocal(server.Name) {
		return "", nil, errors.New("login SMTP tanpa enkripsi ditolak: nyalakan STARTTLS")
	}
	if slices.Contains(server.Auth, "PLAIN") {
		a.mech = "PLAIN"
		return "PLAIN", []byte("\x00" + a.user + "\x00" + a.pass), nil
	}
	if slices.Contains(server.Auth, "LOGIN") {
		a.mech = "LOGIN"
		return "LOGIN", nil, nil
	}
	return "", nil, errors.New("server SMTP tidak menawarkan PLAIN atau LOGIN")
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more || a.mech != "LOGIN" {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(string(fromServer))) {
	case "username:":
		return []byte(a.user), nil
	case "password:":
		return []byte(a.pass), nil
	}
	return nil, fmt.Errorf("tantangan LOGIN tidak dikenal")
}

func isLocal(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// GraphMailer mengirim lewat Microsoft Graph dengan alur client credentials:
// minta token akses ke Microsoft Entra ID, lalu panggil sendMail atas nama
// kotak surat pengirim. Token akses disimpan di memori sampai hampir habis.
type GraphMailer struct {
	Config Config
	Client *http.Client
	// LoginURL dan GraphURL bisa diganti test; kosong berarti alamat Microsoft.
	LoginURL string
	GraphURL string

	mu      sync.Mutex
	token   string
	expires time.Time
}

func (g *GraphMailer) accessToken(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.token != "" && time.Now().Before(g.expires) {
		return g.token, nil
	}
	base := g.LoginURL
	if base == "" {
		base = "https://login.microsoftonline.com"
	}
	form := url.Values{
		"client_id":     {g.Config.GraphClientID},
		"client_secret": {g.Config.GraphClientSecret},
		"scope":         {"https://graph.microsoft.com/.default"},
		"grant_type":    {"client_credentials"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/"+url.PathEscape(g.Config.GraphTenantID)+"/oauth2/v2.0/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := g.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("meminta token Graph: %w", err)
	}
	defer resp.Body.Close()
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil || resp.StatusCode != http.StatusOK || body.AccessToken == "" {
		// Hanya kode error yang dicatat; isi jawaban bisa memuat data tenant.
		return "", fmt.Errorf("token Graph ditolak: status %d %s", resp.StatusCode, body.Error)
	}
	g.token = body.AccessToken
	g.expires = time.Now().Add(time.Duration(max(body.ExpiresIn-120, 60)) * time.Second)
	return g.token, nil
}

func (g *GraphMailer) Send(ctx context.Context, m Mail) error {
	if !validAddress(m.To) || !validAddress(g.Config.From) {
		return errors.New("alamat email tidak sah")
	}
	token, err := g.accessToken(ctx)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"message": map[string]any{
			"subject":      m.Subject,
			"body":         map[string]string{"contentType": "Text", "content": m.Body},
			"toRecipients": []map[string]any{{"emailAddress": map[string]string{"address": m.To}}},
		},
		"saveToSentItems": false,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	base := g.GraphURL
	if base == "" {
		base = "https://graph.microsoft.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1.0/users/"+url.PathEscape(g.Config.From)+"/sendMail", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.Client.Do(req)
	if err != nil {
		return fmt.Errorf("mengirim lewat Graph: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		// Token mungkin dicabut: minta yang baru pada percobaan berikutnya.
		g.mu.Lock()
		g.token = ""
		g.mu.Unlock()
	}
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Graph menolak email: status %d", resp.StatusCode)
	}
	return nil
}
