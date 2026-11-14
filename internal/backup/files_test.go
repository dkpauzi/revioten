package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateWithFiles: folder lampiran ikut terarsip terenkripsi (D-163), dan
// restore mengekstraknya utuh hanya ke folder kosong.
func TestCreateWithFiles(t *testing.T) {
	dir, files := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(files, "ab"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(files, "ab", "ab12"), []byte("8D report"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(files, "cd34"), []byte("foto NG"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := CreateWithFiles(t.Context(), fakeDumper{content: "isi dump"}, dir, passphraseUji, Info{Database: "revioten"}, waktuUji, files)
	if err != nil {
		t.Fatal(err)
	}
	if m.Files != "revioten-20261030-134500.files.tar.age" || m.FilesCount != 2 || len(m.FilesSHA256) != 64 {
		t.Fatalf("manifest: %+v", m)
	}
	// Arsip terenkripsi: isi lampiran tidak terbaca apa adanya.
	raw, _ := os.ReadFile(filepath.Join(dir, m.Files))
	if strings.Contains(string(raw), "8D report") {
		t.Error("arsip lampiran tidak terenkripsi")
	}

	backup := filepath.Join(dir, m.File)
	out := filepath.Join(t.TempDir(), "pulih")
	n, err := RestoreFiles(backup, passphraseUji, out)
	if err != nil || n != 2 {
		t.Fatalf("restore: %d, %v", n, err)
	}
	if got, _ := os.ReadFile(filepath.Join(out, "ab", "ab12")); string(got) != "8D report" {
		t.Errorf("isi pulih: %q", got)
	}
	// Folder yang tidak kosong tidak pernah ditimpa.
	if _, err := RestoreFiles(backup, passphraseUji, out); err == nil {
		t.Error("restore menimpa folder yang berisi")
	}
	// Arsip yang diganti ketahuan.
	if err := os.WriteFile(filepath.Join(dir, m.Files), []byte("palsu"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreFiles(backup, passphraseUji, t.TempDir()); err == nil {
		t.Error("arsip yang diganti tidak ketahuan")
	}
	// Tanpa folder lampiran: tidak ada arsip.
	m2, err := CreateWithFiles(t.Context(), fakeDumper{content: "x"}, t.TempDir(), passphraseUji, Info{}, waktuUji, filepath.Join(files, "tidak-ada"))
	if err != nil || m2.Files != "" {
		t.Errorf("tanpa folder: %+v, %v", m2, err)
	}
}
