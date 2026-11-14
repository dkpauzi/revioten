package backup

// Arsip folder lampiran (D-163, M5.2). Isi lampiran tidak ada di dump
// database, jadi backup juga membuat arsip tar terenkripsi dari folder
// ATTACHMENT_DIR dengan passphrase yang sama. Manifest mencatat nama arsip,
// SHA-256, ukuran, dan jumlah berkasnya. Restore mengekstraknya ke folder
// baru yang masih kosong; folder yang sedang dipakai tidak pernah ditimpa.

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const filesSuffix = ".files.tar.age"

// archiveFiles menulis semua berkas biasa di srcDir sebagai tar ke w.
// Nama di dalam arsip memakai garis miring, relatif terhadap srcDir.
func archiveFiles(w io.Writer, srcDir string) (int, error) {
	tw := tar.NewWriter(w)
	count := 0
	err := filepath.WalkDir(srcDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil // folder dan tautan tidak ikut
		}
		rel, err := filepath.Rel(srcDir, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{Name: filepath.ToSlash(rel), Mode: 0o600, Size: info.Size(), ModTime: info.ModTime(), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("mengarsipkan folder %s: %w", srcDir, err)
	}
	return count, tw.Close()
}

// RestoreFiles mengekstrak arsip lampiran milik backup backupPath ke dstDir (D-163).
// dstDir harus belum ada atau masih kosong. Hasilnya jumlah berkas.
func RestoreFiles(backupPath, passphrase, dstDir string) (int, error) {
	m, err := ReadManifest(backupPath)
	if err != nil {
		return 0, err
	}
	if m.Files == "" {
		return 0, errors.New("backup ini tidak punya arsip lampiran")
	}
	archive := filepath.Join(filepath.Dir(backupPath), m.Files)
	if err := verifySum(archive, m.FilesSHA256, m.FilesSize); err != nil {
		return 0, err
	}
	if entries, err := os.ReadDir(dstDir); err == nil && len(entries) > 0 {
		return 0, fmt.Errorf("folder %s tidak kosong; pilih folder baru", dstDir)
	}
	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return 0, err
	}
	f, err := os.Open(archive)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	plain, err := Decrypt(f, passphrase)
	if err != nil {
		return 0, err
	}
	tr := tar.NewReader(plain)
	count := 0
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return count, fmt.Errorf("membaca arsip lampiran: %w", err)
		}
		// Nama yang keluar dari folder tujuan (../) ditolak.
		name := path.Clean(h.Name)
		if h.Typeflag != tar.TypeReg || strings.HasPrefix(name, "../") || name == ".." || path.IsAbs(name) {
			return count, fmt.Errorf("arsip lampiran berisi nama yang tidak aman: %q", h.Name)
		}
		target := filepath.Join(dstDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return count, err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return count, err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return count, err
		}
		if err := out.Close(); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// verifySum memeriksa ukuran dan SHA-256 satu berkas terhadap catatan manifest.
func verifySum(file, sum string, size int64) error {
	got, n, err := fileSum(file)
	if err != nil {
		return err
	}
	if n != size || got != sum {
		return fmt.Errorf("berkas %s tidak sama dengan manifest (rusak atau diganti)", file)
	}
	return nil
}
