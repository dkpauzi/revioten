// Package web memuat hasil build SvelteKit (SPA statis) ke dalam binary Go (D-01).
//
// npm run build menulis hasilnya ke web/build/app. Folder web/build sendiri
// di-commit bersama file .gitkeep, sehingga go build dan go test tetap jalan
// walaupun frontend belum di-build. Dalam keadaan itu Files mengembalikan
// folder kosong, dan server menjawab dengan petunjuk build.
package web

import (
	"embed"
	"io/fs"
)

// Awalan all: wajib. Tanpa awalan itu, file dan folder yang namanya diawali
// _ atau . tidak ikut di-embed, padahal SvelteKit menaruh JavaScript di _app/.
//
//go:embed all:build
var build embed.FS

// Files mengembalikan isi folder build/app sebagai sistem file.
func Files() fs.FS {
	files, err := fs.Sub(build, "build/app")
	if err != nil {
		// fs.Sub hanya gagal bila nama foldernya tidak valid, dan "build/app" selalu valid.
		panic(err)
	}
	return files
}
