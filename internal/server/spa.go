package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaFallback adalah halaman yang dibuat SvelteKit (adapter-static) untuk
// semua alamat halaman. Setelah halaman ini dimuat, routing dikerjakan di browser.
const spaFallback = "200.html"

// handleSPA melayani tampilan web dari hasil build SvelteKit di ui:
//   - hanya GET dan HEAD; method lain dijawab 405;
//   - file yang ada (misalnya /_app/immutable/....js) dikirim apa adanya;
//   - aset yang tidak ada dijawab 404;
//   - alamat halaman lain dijawab dengan 200.html.
func handleSPA(ui fs.FS) http.Handler {
	files := http.FileServerFS(ui)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		name := strings.TrimPrefix(r.URL.Path, "/")

		if name != "" && isFile(ui, name) {
			files.ServeHTTP(w, r)
			return
		}

		if isAsset(name) {
			http.NotFound(w, r)
			return
		}

		if !isFile(ui, spaFallback) {
			http.Error(w, "Frontend not built yet. Run: npm --prefix web run build", http.StatusServiceUnavailable)
			return
		}

		http.ServeFileFS(w, r, ui, spaFallback)
	})
}

// isFile memeriksa apakah name adalah file (bukan folder) di ui.
func isFile(ui fs.FS, name string) bool {
	info, err := fs.Stat(ui, name)
	return err == nil && !info.IsDir()
}

// isAsset menandai alamat yang tidak boleh diganti halaman fallback: aset
// SvelteKit (_app/) dan nama file yang berekstensi. Alamat API tidak sampai
// ke sini, karena /api/ sudah ditangani routeAPI (server.go).
// Nama kosong (alamat "/") bukan file: path.Base("") menghasilkan ".",
// jadi nama kosong harus diperiksa lebih dulu.
func isAsset(name string) bool {
	if name == "" {
		return false
	}
	return strings.HasPrefix(name, "_app/") || strings.Contains(path.Base(name), ".")
}
