package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/dkpauzi/revioten/internal/db"
)

// config berisi pengaturan program. Semua nilainya memuat password, jadi tidak
// pernah ditulis ke log atau pesan error (E-18).
type config struct {
	// OwnerURL adalah alamat database sebagai pemiliknya. Hanya dipakai saat
	// program mulai, untuk migrasi dan memasang password role aplikasi (E-06).
	OwnerURL string
	// AppURL adalah alamat database sebagai role aplikasi (db.AppRole), untuk
	// semua operasi lain. Role ini tidak bisa mengubah atau menghapus audit log.
	AppURL string
	// AppPassword adalah password role aplikasi dari APP_DB_PASSWORD.
	AppPassword string
}

// loadConfig membaca pengaturan dari environment dan dari file .env.
// Nilai di environment menang atas nilai di file, sama seperti Docker Compose.
// File .env boleh tidak ada.
//
// Alamat pemilik database disusun dari POSTGRES_USER, POSTGRES_PASSWORD,
// POSTGRES_DB, dan POSTGRES_PORT, yaitu isian yang sama dengan yang dipakai
// compose.yaml, atau diambil apa adanya dari DATABASE_URL bila diisi. Alamat
// role aplikasi memakai server dan database yang sama, dengan user
// revioten_app dan password APP_DB_PASSWORD.
func loadConfig(getenv func(string) string, envFile string) (config, error) {
	fileValues := map[string]string{}
	data, err := os.ReadFile(envFile)
	switch {
	case err == nil:
		fileValues, err = parseEnvFile(string(data))
		if err != nil {
			return config{}, fmt.Errorf("membaca %s: %w", envFile, err)
		}
	case !errors.Is(err, os.ErrNotExist):
		return config{}, fmt.Errorf("membaca %s: %w", envFile, err)
	}

	// get mengambil nilai dari environment, lalu dari file .env.
	get := func(key string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return fileValues[key]
	}

	dbURL := get("DATABASE_URL")
	required := []string{"APP_DB_PASSWORD"}
	if dbURL == "" {
		required = []string{"POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB", "APP_DB_PASSWORD"}
	}
	var missing []string
	for _, key := range required {
		if get(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return config{}, fmt.Errorf("%s belum diisi di file .env: lihat contoh dan penjelasannya di .env.example",
			strings.Join(missing, ", "))
	}

	var owner *url.URL
	if dbURL != "" {
		owner, err = url.Parse(dbURL)
		if err != nil || owner.Scheme == "" {
			// Isi DATABASE_URL tidak disebut, karena memuat password (E-18).
			return config{}, errors.New("DATABASE_URL harus berbentuk postgres://user:password@host:port/database")
		}
	} else {
		port := get("POSTGRES_PORT")
		if port == "" {
			port = "5432"
		}
		// url.URL mengodekan karakter khusus di user dan password, misalnya @ atau /.
		owner = &url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(get("POSTGRES_USER"), get("POSTGRES_PASSWORD")),
			Host:     net.JoinHostPort("127.0.0.1", port),
			Path:     "/" + get("POSTGRES_DB"),
			RawQuery: "sslmode=disable", // database lokal di Docker, tanpa TLS (D-36)
		}
	}
	app := *owner
	app.User = url.UserPassword(db.AppRole, get("APP_DB_PASSWORD"))
	ownerURL := dbURL
	if ownerURL == "" {
		ownerURL = owner.String()
	}
	return config{OwnerURL: ownerURL, AppURL: app.String(), AppPassword: get("APP_DB_PASSWORD")}, nil
}

// parseEnvFile membaca isi file .env dengan aturan yang sama dengan Docker
// Compose untuk kasus yang dipakai di sini:
//   - baris kosong dan baris yang diawali # dilewati;
//   - setiap baris lain berbentuk NAMA=nilai, spasi di tepi dibuang;
//   - nilai dalam tanda kutip "..." atau '...' diambil isinya saja;
//   - pada nilai tanpa kutip, " #" dan sesudahnya adalah komentar.
//
// Variabel di dalam nilai, seperti ${NAMA}, tidak diganti.
func parseEnvFile(content string) (map[string]string, error) {
	values := map[string]string{}
	for i, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			// Isi baris tidak disebut, karena bisa berupa password (E-18).
			return nil, fmt.Errorf("baris %d tidak berbentuk NAMA=nilai", i+1)
		}
		values[key] = envValue(strings.TrimSpace(value))
	}
	return values, nil
}

// envValue membuang tanda kutip atau komentar dari satu nilai di .env.
func envValue(value string) string {
	for _, quote := range []string{`"`, `'`} {
		if strings.HasPrefix(value, quote) {
			if end := strings.Index(value[1:], quote); end >= 0 {
				return value[1 : end+1]
			}
		}
	}
	if before, _, found := strings.Cut(value, " #"); found {
		return strings.TrimSpace(before)
	}
	return value
}
