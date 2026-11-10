package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/auth"
	"github.com/dkpauzi/revioten/internal/problem"
)

// createAdminOptions adalah isian perintah create-admin.
type createAdminOptions struct {
	name    string
	login   string
	envFile string
}

// parseCreateAdminFlags membaca flag perintah create-admin. Password sengaja
// tidak bisa diberikan lewat flag, supaya tidak tersimpan di riwayat perintah.
func parseCreateAdminFlags(args []string, out io.Writer) (createAdminOptions, error) {
	var opts createAdminOptions
	fs := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&opts.name, "name", "", "nama orang pemegang akun admin (wajib)")
	fs.StringVar(&opts.login, "login", "", "nama login akun admin (wajib)")
	fs.StringVar(&opts.envFile, "env", ".env", "file berisi pengaturan database")
	if err := fs.Parse(args); err != nil {
		return createAdminOptions{}, errors.New("flag create-admin tidak dikenal; yang tersedia: -name, -login, -env")
	}
	if opts.name == "" || opts.login == "" {
		return createAdminOptions{}, errors.New(`isi -name dan -login, contoh: revioten create-admin -name "Nama Pemegang" -login admin.nama`)
	}
	return opts, nil
}

// runCreateAdmin menjalankan perintah `revioten create-admin` (E-07): membuat
// orang dan akun admin pertama, hanya bila belum ada akun admin sama sekali.
func runCreateAdmin(args []string, in io.Reader, out io.Writer) error {
	opts, err := parseCreateAdminFlags(args, out)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(os.Getenv, opts.envFile)
	if err != nil {
		return err
	}
	password, err := readPassword(in, out)
	if err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := openDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	actor, err := auth.NewService(pool, access.NewChecker(pool)).CreateFirstAdmin(ctx, opts.name, opts.login, password)
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case errors.Is(err, auth.ErrAdminExists):
		return errors.New("akun admin sudah ada: perintah ini hanya untuk akun admin pertama (E-07); akun berikutnya dibuat admin lewat aplikasi")
	case errors.As(err, &invalid):
		printProblems(out, invalid.Problems)
		return errors.New("akun admin tidak dibuat karena input tidak valid")
	case errors.As(err, &conflict):
		printProblems(out, conflict.Problems)
		return errors.New("akun admin tidak dibuat karena bentrok dengan data yang ada")
	case err != nil:
		return err
	}
	fmt.Fprintf(out, "Akun admin %q untuk %s sudah dibuat. Login lewat halaman /login.\n", actor.Login, actor.PersonName)
	return nil
}

// printProblems menulis masalah input per field. Pesannya tidak pernah memuat password.
func printProblems(out io.Writer, problems []problem.Problem) {
	for _, p := range problems {
		fmt.Fprintf(out, "- %s: %s\n", p.Field, p.Message)
	}
}

// readPassword membaca password admin. Bila in adalah terminal, password
// diketik tanpa tampil di layar dan diminta dua kali. Bila tidak (misalnya
// dikirim lewat pipa oleh skrip), satu baris dibaca dari in.
func readPassword(in io.Reader, out io.Writer) (string, error) {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		first, err := promptHidden(f, out, "Password admin (minimal 8 karakter): ")
		if err != nil {
			return "", err
		}
		second, err := promptHidden(f, out, "Ulangi password: ")
		if err != nil {
			return "", err
		}
		if first != second {
			return "", errors.New("kedua password tidak sama")
		}
		return first, nil
	}

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		return "", errors.New("password tidak terbaca dari input: kirim password sebagai satu baris")
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// promptHidden menampilkan pertanyaan lalu membaca jawaban tanpa menampilkannya.
func promptHidden(f *os.File, out io.Writer, prompt string) (string, error) {
	fmt.Fprint(out, prompt)
	value, err := term.ReadPassword(int(f.Fd()))
	fmt.Fprintln(out)
	if err != nil {
		return "", fmt.Errorf("membaca password dari terminal: %w", err)
	}
	return string(value), nil
}
