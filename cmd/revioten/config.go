package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// config berisi pengaturan program.
type config struct {
	// DatabaseURL memuat password, jadi nilainya tidak pernah ditulis ke log
	// atau pesan error (E-18).
	DatabaseURL string
}

// loadConfig membaca pengaturan dari environment dan dari file .env.
// Nilai di environment menang atas nilai di file, sama seperti Docker Compose.
// File .env boleh tidak ada.
//
// Bila DATABASE_URL diisi, nilainya dipakai apa adanya. Bila tidak, alamat
// database disusun dari POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB, dan
// POSTGRES_PORT, yaitu isian yang sama dengan yang dipakai compose.yaml.
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

	if dbURL := get("DATABASE_URL"); dbURL != "" {
		return config{DatabaseURL: dbURL}, nil
	}

	user, password, dbName := get("POSTGRES_USER"), get("POSTGRES_PASSWORD"), get("POSTGRES_DB")
	var missing []string
	for _, item := range []struct{ key, value string }{
		{"POSTGRES_USER", user}, {"POSTGRES_PASSWORD", password}, {"POSTGRES_DB", dbName},
	} {
		if item.value == "" {
			missing = append(missing, item.key)
		}
	}
	if len(missing) > 0 {
		return config{}, fmt.Errorf("%s belum diisi: salin .env.example menjadi .env lalu isi nilainya, atau isi DATABASE_URL",
			strings.Join(missing, ", "))
	}

	port := get("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	// url.URL mengodekan karakter khusus di user dan password, misalnya @ atau /.
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort("127.0.0.1", port),
		Path:     "/" + dbName,
		RawQuery: "sslmode=disable", // database lokal di Docker, tanpa TLS (D-36)
	}
	return config{DatabaseURL: u.String()}, nil
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
