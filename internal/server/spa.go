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
//   - file yang ada (misalnya /_app/immutable/....js) dikirim apa adanya;
//   - aset atau alamat API yang tidak ada dijawab 404;
//   - alamat halaman lain dijawab dengan 200.html.
func handleSPA(ui fs.FS) http.Handler {
	files := http.FileServerFS(ui)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")

		if name != "" && isFile(ui, name) {
			files.ServeHTTP(w, r)
			return
		}

		if isAssetOrAPI(name) {
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

// isAssetOrAPI menandai alamat yang tidak boleh diganti halaman fallback:
// aset SvelteKit (_app/), API (api/), dan nama file yang berekstensi.
// Nama kosong (alamat "/") bukan file: path.Base("") menghasilkan ".",
// jadi nama kosong harus diperiksa lebih dulu.
func isAssetOrAPI(name string) bool {
	if name == "" {
		return false
	}
	return strings.HasPrefix(name, "_app/") ||
		name == "api" || strings.HasPrefix(name, "api/") ||
		strings.Contains(path.Base(name), ".")
}
