package notify

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"sync"
	"testing"
)

// fakeSMTP adalah server SMTP tiruan tanpa TLS di localhost. Ia menawarkan
// AUTH LOGIN seperti Microsoft 365, dan mencatat perintah serta isi DATA.
func fakeSMTP(t *testing.T) (addr string, got func() (string, string)) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	var mu sync.Mutex
	var commands, data strings.Builder
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		write("220 contoh.test ESMTP")
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			mu.Lock()
			if inData {
				if line == "." {
					inData = false
					mu.Unlock()
					write("250 OK queued")
					continue
				}
				data.WriteString(line + "\n")
				mu.Unlock()
				continue
			}
			commands.WriteString(line + "\n")
			mu.Unlock()
			switch {
			case strings.HasPrefix(line, "EHLO"):
				write("250-contoh.test")
				write("250 AUTH LOGIN")
			case line == "AUTH LOGIN":
				write("334 " + base64.StdEncoding.EncodeToString([]byte("Username:")))
				user, _ := r.ReadString('\n')
				write("334 " + base64.StdEncoding.EncodeToString([]byte("Password:")))
				pass, _ := r.ReadString('\n')
				u, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(user))
				p, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(pass))
				if string(u) == "revioten@contoh.test" && string(p) == "rahasia" {
					write("235 Authentication successful")
				} else {
					write("535 Authentication failed")
				}
			case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"):
				write("250 OK")
			case line == "DATA":
				mu.Lock()
				inData = true
				mu.Unlock()
				write("354 Go ahead")
			case line == "QUIT":
				write("221 Bye")
				return
			default:
				write("250 OK")
			}
		}
	}()
	return ln.Addr().String(), func() (string, string) {
		mu.Lock()
		defer mu.Unlock()
		return commands.String(), data.String()
	}
}

func TestSMTPMailer(t *testing.T) {
	addr, got := fakeSMTP(t)
	host, port, _ := net.SplitHostPort(addr)
	var p int
	for _, c := range port {
		p = p*10 + int(c-'0')
	}
	m := &SMTPMailer{Config: Config{Provider: "smtp", From: "revioten@contoh.test", SMTPHost: host, SMTPPort: p,
		SMTPUsername: "revioten@contoh.test", SMTPPassword: "rahasia"}}
	if err := m.Send(context.Background(), Mail{To: "budi@contoh.test", Subject: "CHG-0001 menunggu verifikasi", Body: "Isi email"}); err != nil {
		t.Fatal(err)
	}
	commands, data := got()
	if !strings.Contains(commands, "AUTH LOGIN") || !strings.Contains(commands, "RCPT TO:<budi@contoh.test>") {
		t.Errorf("perintah: %s", commands)
	}
	if !strings.Contains(data, "Subject: CHG-0001 menunggu verifikasi") || !strings.Contains(data, base64.StdEncoding.EncodeToString([]byte("Isi email"))) {
		t.Errorf("isi: %s", data)
	}
	// Password tidak pernah ditulis polos ke koneksi.
	if strings.Contains(commands+data, "rahasia") {
		t.Error("password terlihat polos")
	}
	// Alamat yang bisa menyisipkan header ditolak sebelum menghubungi server.
	if err := m.Send(context.Background(), Mail{To: "a@contoh.test\r\nBcc: x@y.z", Subject: "x", Body: "x"}); err == nil {
		t.Error("alamat dengan baris baru diterima")
	}
}

// TestSMTPLoginNeedsTLS: login tanpa enkripsi ke server bukan localhost ditolak.
func TestSMTPLoginNeedsTLS(t *testing.T) {
	a := &loginAuth{user: "u", pass: "p", host: "smtp.office365.com"}
	if _, _, err := a.Start(&smtpServerInfo{Name: "smtp.office365.com", TLS: false, Auth: []string{"LOGIN"}}); err == nil {
		t.Error("login tanpa TLS diterima")
	}
	if mech, _, err := a.Start(&smtpServerInfo{Name: "smtp.office365.com", TLS: true, Auth: []string{"LOGIN", "XOAUTH2"}}); err != nil || mech != "LOGIN" {
		t.Errorf("LOGIN dengan TLS: %s, %v", mech, err)
	}
}

func TestGraphMailer(t *testing.T) {
	var tokenCalls, mailCalls int
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tenant-1/oauth2/v2.0/token":
			tokenCalls++
			_ = r.ParseForm()
			if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_secret") != "rahasia" || r.Form.Get("scope") != "https://graph.microsoft.com/.default" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":"invalid_client"}`)
				return
			}
			_, _ = io.WriteString(w, `{"access_token":"token-a","expires_in":3600}`)
		case r.URL.Path == "/v1.0/users/revioten@contoh.test/sendMail":
			mailCalls++
			if r.Header.Get("Authorization") != "Bearer token-a" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewDecoder(r.Body).Decode(&sent)
			w.WriteHeader(http.StatusAccepted)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	g := &GraphMailer{Config: Config{Provider: "graph", From: "revioten@contoh.test", GraphTenantID: "tenant-1", GraphClientID: "app-1", GraphClientSecret: "rahasia"},
		Client: srv.Client(), LoginURL: srv.URL, GraphURL: srv.URL}
	for range 2 {
		if err := g.Send(context.Background(), Mail{To: "budi@contoh.test", Subject: "Uji", Body: "Isi"}); err != nil {
			t.Fatal(err)
		}
	}
	// Token akses dipakai ulang sampai hampir habis.
	if tokenCalls != 1 || mailCalls != 2 {
		t.Errorf("panggilan token %d, email %d", tokenCalls, mailCalls)
	}
	msg := sent["message"].(map[string]any)
	if msg["subject"] != "Uji" || !strings.Contains(mustJSON(msg["toRecipients"]), "budi@contoh.test") {
		t.Errorf("isi Graph: %v", sent)
	}
	bad := &GraphMailer{Config: Config{From: "revioten@contoh.test", GraphTenantID: "tenant-1", GraphClientSecret: "salah"}, Client: srv.Client(), LoginURL: srv.URL, GraphURL: srv.URL}
	err := bad.Send(context.Background(), Mail{To: "budi@contoh.test", Subject: "x", Body: "x"})
	if err == nil || !strings.Contains(err.Error(), "invalid_client") || strings.Contains(err.Error(), "salah") {
		t.Errorf("rahasia salah: %v", err)
	}
}

func TestNewMailerConfig(t *testing.T) {
	if m, err := NewMailer(Config{}, nil); m != nil || err != nil {
		t.Errorf("tanpa pengirim: %v, %v", m, err)
	}
	if _, err := NewMailer(Config{Provider: "graph", From: "a@b.c"}, nil); err == nil || !strings.Contains(err.Error(), "GRAPH_TENANT_ID") {
		t.Errorf("graph belum lengkap: %v", err)
	}
	if _, err := NewMailer(Config{Provider: "outlook"}, nil); err == nil {
		t.Error("penyedia tidak dikenal diterima")
	}
	provider, _, server := Config{Provider: "smtp", SMTPHost: "smtp.office365.com", SMTPPort: 587, SMTPPassword: "rahasia"}.Describe()
	if provider != "smtp" || server != "smtp.office365.com:587" {
		t.Errorf("keterangan: %s %s", provider, server)
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// smtpServerInfo adalah nama pendek untuk smtp.ServerInfo di test ini.
type smtpServerInfo = smtp.ServerInfo
