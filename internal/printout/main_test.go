package printout_test

import (
	"os"
	"testing"

	"github.com/dkpauzi/revioten/internal/db/dbtest"
)

// TestMain menyalakan PostgreSQL untuk test layanan cetak.
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}
