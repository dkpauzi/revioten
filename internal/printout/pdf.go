package printout

// PDF lewat browser headless (M6.3, hasil uji S5, ASUMSI E-108): HTML ditulis
// ke folder sementara, lalu Chromium atau Edge yang sudah terpasang mencetaknya
// dengan --print-to-pdf. Tidak ada mesin PDF tambahan yang diunduh. Bila tidak
// ada browser, PDF tidak tersedia dan Excel tetap bisa dipakai (opsi D).

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// ErrNoBrowser berarti tidak ada browser untuk membuat PDF.
var ErrNoBrowser = errors.New("tidak ada Chromium, Chrome, atau Edge untuk membuat PDF")

// FindBrowser mencari browser: PDF_BROWSER bila diisi, lalu lokasi biasa.
func FindBrowser(configured string) (string, error) {
	if configured != "" {
		if _, err := os.Stat(configured); err != nil {
			return "", fmt.Errorf("PDF_BROWSER %s tidak ada: %w", configured, err)
		}
		return configured, nil
	}
	var candidates []string
	if runtime.GOOS == "windows" {
		for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
			if base == "" {
				continue
			}
			candidates = append(candidates,
				filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
				filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"))
		}
	} else {
		candidates = []string{"/usr/bin/chromium", "/usr/bin/chromium-browser", "/usr/bin/google-chrome", "/opt/pw-browsers/chromium",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "chrome", "msedge"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", ErrNoBrowser
}

// PDF mencetak HTML ke PDF dengan browser di lokasi browser.
func PDF(ctx context.Context, browser string, html []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "revioten-print-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	in := filepath.Join(dir, "page.html")
	out := filepath.Join(dir, "page.pdf")
	if err := os.WriteFile(in, html, 0o600); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, browser,
		"--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check", "--disable-extensions",
		"--no-pdf-header-footer", "--user-data-dir="+filepath.Join(dir, "profile"),
		"--print-to-pdf="+out, "file:///"+filepath.ToSlash(in))
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("browser gagal membuat PDF: %w (%s)", err, truncateOutput(output))
	}
	// Di Windows, proses yang dijalankan bisa keluar lebih dulu sementara
	// proses anaknya masih mencetak. Tunggu sampai berkas ada dan ukurannya
	// tidak berubah lagi.
	last := int64(-1)
	for {
		if st, err := os.Stat(out); err == nil && st.Size() > 0 {
			if st.Size() == last {
				break
			}
			last = st.Size()
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("browser tidak menulis PDF dalam 60 detik: %w", ctx.Err())
		case <-time.After(300 * time.Millisecond):
		}
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("membaca PDF: %w", err)
	}
	return data, nil
}

func truncateOutput(b []byte) string {
	if len(b) > 300 {
		b = b[:300]
	}
	return string(b)
}
