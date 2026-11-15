package printout_test

import (
	"os"
	"testing"

	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/notify"
)

// TestMain menyalakan PostgreSQL untuk test layanan cetak.
func TestMain(m *testing.M) {
	// Pendengar notifikasi email terpasang, supaya setiap peristiwa di test
	// ini juga menguji penyusunan emailnya (D-194).
	audit.SetListener(notify.NewNotifier("http://contoh.test").Strict().Listener())
	os.Exit(dbtest.Main(m))
}
