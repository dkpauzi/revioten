package inbox

import (
	"os"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/change"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/db/dbtest"
	"github.com/dkpauzi/revioten/internal/document/doctest"
	"github.com/dkpauzi/revioten/internal/notify"
)

func TestMain(m *testing.M) {
	audit.SetListener(notify.NewNotifier("http://contoh.test").Strict().Listener())
	os.Exit(dbtest.Main(m))
}

func kinds(tasks []Task) map[string]int {
	out := map[string]int{}
	for _, t := range tasks {
		out[t.Kind]++
	}
	return out
}

// D-202 · "Waiting for you" di Home hanya memuat pekerjaan yang boleh
// dikerjakan orang yang login.
func TestTasks(t *testing.T) {
	e := doctest.Setup(t)
	ctx := t.Context()
	set := e.NewSet(t, "ab", e.CustomerAB, e.ModelK125)
	dcc := doctest.NewActor(t, e.Pool, "dcc.uji", "work", doctest.All(access.RoleDocumentControl))
	doctest.AttachScan(t, e.Pool, set["cp"].ID)
	if _, err := e.Docs.Release(ctx, dcc, set["cp"].ID); err != nil {
		t.Fatal(err)
	}
	var claim, content string
	if err := e.Pool.QueryRow(ctx, "SELECT id FROM change_sources WHERE name = 'Customer claim'").Scan(&claim); err != nil {
		t.Fatal(err)
	}
	if err := e.Pool.QueryRow(ctx, "SELECT id FROM change_types WHERE key = 'content'").Scan(&content); err != nil {
		t.Fatal(err)
	}
	if _, err := change.NewRequestService(e.Pool, e.Checker).Create(ctx, e.Author, change.NewChange{
		CustomerID: e.CustomerAB, TriggerID: claim, ChangeTypeID: content, Reason: "Claim 7", DocumentIDs: []string{set["pfmea"].ID}}); err != nil {
		t.Fatal(err)
	}
	verifier := doctest.NewActor(t, e.Pool, "verifier.uji", "work", doctest.All(access.RoleVerifier))

	s := NewService(e.Pool, e.Checker)
	s.now = func() time.Time { return time.Now().AddDate(0, 37, 0) }
	withSubject := func(a access.Actor) []Task {
		t.Helper()
		tasks, _, err := s.Tasks(db.WithSubject(ctx, a.AccountID), a)
		if err != nil {
			t.Fatal(err)
		}
		return tasks
	}

	// Penyusun: review jatuh tempo, bukan verifikasi permintaannya sendiri.
	if k := kinds(withSubject(e.Author)); k[KindReview] != 1 || k[KindVerify] != 0 {
		t.Errorf("penyusun: %v", k)
	}
	// Verifikator: permintaan perubahan yang menunggu.
	if k := kinds(withSubject(verifier)); k[KindVerify] != 1 || k[KindReview] != 0 {
		t.Errorf("verifikator: %v", k)
	}
	// Viewer: tidak ada pekerjaan.
	if tasks := withSubject(e.Viewer); len(tasks) != 0 {
		t.Errorf("viewer: %+v", tasks)
	}
}
