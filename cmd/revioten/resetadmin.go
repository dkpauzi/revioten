package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/auth"
)

// resetAdminOptions adalah isian perintah reset-admin-password.
type resetAdminOptions struct {
	login   string
	envFile string
}

// parseResetAdminFlags membaca flag perintah reset-admin-password.
func parseResetAdminFlags(args []string, out io.Writer) (resetAdminOptions, error) {
	var opts resetAdminOptions
	fs := flag.NewFlagSet("reset-admin-password", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&opts.login, "login", "", "nama login akun admin (wajib)")
	fs.StringVar(&opts.envFile, "env", ".env", "file berisi pengaturan database")
	if err := fs.Parse(args); err != nil {
		return resetAdminOptions{}, errors.New("flag reset-admin-password tidak dikenal; yang tersedia: -login, -env")
	}
	if opts.login == "" {
		return resetAdminOptions{}, errors.New("isi -login, contoh: revioten reset-admin-password -login admin.nama")
	}
	return opts, nil
}

// runResetAdmin menjalankan perintah `revioten reset-admin-password` (E-22,
// D-101): akun admin yang lupa password, terkunci, atau nonaktif dibuka lagi
// dengan password sementara yang wajib diganti saat login. Perintah ini hanya
// bisa dijalankan di komputer server, karena butuh .env dan akses database.
func runResetAdmin(args []string, out io.Writer) error {
	opts, err := parseResetAdminFlags(args, out)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(os.Getenv, opts.envFile)
	if err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := openDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	temporary, err := auth.NewService(pool, access.NewChecker(pool)).ResetAdminPassword(ctx, opts.login)
	if errors.Is(err, auth.ErrNotAdminAccount) {
		return errors.New("tidak ada akun admin dengan nama login itu; akun kerja direset admin lewat menu Accounts")
	}
	if err != nil {
		return err
	}
	// Password sementara ditampilkan sekali di layar, tidak ditulis ke log (E-18, D-95).
	fmt.Fprintf(out, "Password sementara akun %s: %s\n", opts.login, temporary)
	fmt.Fprintln(out, "Login dengan password ini, lalu ganti password saat diminta. Peristiwanya tercatat di audit log.")
	return nil
}
