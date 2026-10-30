package auth

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
)

// TestAuditTrail: setiap operasi akun yang mengubah data menulis satu entri
// audit dengan jenis aktor, akun, dan sasaran yang benar (M1.3, R-SEC-10), dan
// audit log tidak pernah memuat password, password sementara, atau token (E-18).
func TestAuditTrail(t *testing.T) {
	s, _ := newTestService(t)
	ctx := audit.WithRequest(t.Context(), audit.Request{ID: "UJI-AUDIT", IP: "192.0.2.30"})

	admin, err := s.CreateFirstAdmin(ctx, adminName, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	_, adminToken, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	const wrongPassword = "bukan password yang benar"
	if _, _, err := s.Login(ctx, adminLogin, wrongPassword); err == nil {
		t.Fatal("login dengan password salah berhasil")
	}
	// Nama login yang tidak ada tidak mengubah data apa pun, jadi tidak dicatat.
	if _, _, err := s.Login(ctx, "tidak.ada", wrongPassword); err == nil {
		t.Fatal("login dengan nama tidak dikenal berhasil")
	}

	info, temporary, err := s.CreateAccount(ctx, admin, NewAccount{
		PersonName: "Penyusun Uji", Kind: KindWork, Login: "penyusun.uji",
		Roles: []access.RoleGrant{{Role: access.RoleAuthor, Scope: access.ScopeAll}},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = s.SetRoles(ctx, admin, info.ID, []access.RoleGrant{
		{Role: access.RoleAuthor, Scope: access.ScopeAll}, {Role: access.RoleVerifier, Scope: access.ScopeOwnProjectGroup},
	})
	if err != nil {
		t.Fatal(err)
	}
	worker, workerToken, err := s.Login(ctx, "penyusun.uji", temporary)
	if err != nil {
		t.Fatal(err)
	}
	const newPassword = "lampu jalan menyala sore hari"
	if _, err := s.ChangePassword(ctx, worker, wrongPassword, newPassword); err == nil {
		t.Fatal("ganti password dengan password lama yang salah berhasil")
	}
	newToken, err := s.ChangePassword(ctx, worker, temporary, newPassword)
	if err != nil {
		t.Fatal(err)
	}
	resetTemporary, err := s.ResetPassword(ctx, admin, info.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DisableAccount(ctx, admin, info.ID); err != nil {
		t.Fatal(err)
	}
	// Menonaktifkan akun yang sudah nonaktif tidak mengubah apa pun, jadi tidak dicatat.
	if _, err := s.DisableAccount(ctx, admin, info.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EnableAccount(ctx, admin, info.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Logout(ctx, admin, adminToken); err != nil {
		t.Fatal(err)
	}

	rows, err := db.New(s.pool).ListAuditChain(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	type entry struct{ action, actorType, actor, target, details string }
	var got []entry
	for _, r := range rows {
		got = append(got, entry{r.Action, r.ActorType, deref(r.ActorAccountID), deref(r.TargetID), r.Details})
		if deref(r.RequestID) != "UJI-AUDIT" || deref(r.SourceIp) != "192.0.2.30" {
			t.Errorf("%s: request ID %q dan IP %q", r.Action, deref(r.RequestID), deref(r.SourceIp))
		}
	}
	a, w := admin.AccountID, info.ID
	want := []entry{
		{audit.ActionAccountCreated, "system", "", a, `{"kind":"admin","login":"admin.uji","roles":"admin(all)","via":"create-admin"}`},
		{audit.ActionLogin, "human", a, a, `{}`},
		{audit.ActionLoginFailed, "anonymous", "", a, `{"failed_logins":"1"}`},
		{audit.ActionAccountCreated, "human", a, w, `{"kind":"work","login":"penyusun.uji","roles":"author(all)"}`},
		{audit.ActionRolesChanged, "human", a, w, `{"after":"author(all), verifier(own_project_group)","before":"author(all)"}`},
		{audit.ActionLogin, "human", w, w, `{}`},
		{audit.ActionPasswordChangeFailed, "human", w, w, `{"failed_logins":"1"}`},
		{audit.ActionPasswordChanged, "human", w, w, `{}`},
		{audit.ActionPasswordReset, "human", a, w, `{}`},
		{audit.ActionAccountDisabled, "human", a, w, `{}`},
		{audit.ActionAccountEnabled, "human", a, w, `{}`},
		{audit.ActionLogout, "human", a, a, `{}`},
	}
	if !slices.Equal(got, want) {
		t.Errorf("entri audit:\n dapat %v\ningin %v", got, want)
	}

	// Tidak ada rahasia di audit log, juga tidak hash password-nya.
	var all strings.Builder
	for _, r := range rows {
		all.WriteString(r.Details + " " + r.Action + " " + deref(r.TargetID) + "\n")
	}
	for _, secret := range []string{adminPassword, wrongPassword, temporary, newPassword, resetTemporary,
		adminToken, workerToken, newToken, "$argon2id$"} {
		if strings.Contains(all.String(), secret) {
			t.Errorf("audit log memuat rahasia %q", secret)
		}
	}

	v, err := audit.NewService(s.pool, access.NewChecker(s.pool)).Verify(ctx, admin)
	if err != nil || !v.Valid || v.Entries != int64(len(want)) {
		t.Errorf("rantai audit: %+v, %v", v, err)
	}
}

// TestAuditLockAndDisable: kunci sementara dan penonaktifan karena gagal login
// berulang tercatat di rincian entri login_failed. Penonaktifan hanya untuk
// akun kerja; akun admin hanya dikunci (E-22, D-101).
func TestAuditLockAndDisable(t *testing.T) {
	s, _ := newServiceWithAdmin(t)
	ctx := t.Context()
	admin, _, err := s.Login(ctx, adminLogin, adminPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.CreateAccount(ctx, admin, NewAccount{PersonName: "Penyusun Uji", Kind: KindWork, Login: "penyusun.uji"}); err != nil {
		t.Fatal(err)
	}
	for range lockThreshold {
		if _, _, err := s.Login(ctx, adminLogin, "bukan password yang benar"); err == nil {
			t.Fatal("login dengan password salah berhasil")
		}
	}
	rows, err := db.New(s.pool).ListAuditChain(ctx)
	if err != nil {
		t.Fatal(err)
	}
	last := rows[len(rows)-1]
	if last.Action != audit.ActionLoginFailed || !strings.Contains(last.Details, `"failed_logins":"10"`) ||
		!strings.Contains(last.Details, `"locked_until":`) {
		t.Errorf("entri terakhir: %s %s", last.Action, last.Details)
	}

	// Satu percobaan sebelum batas penonaktifan (100 kali gagal), untuk kedua akun.
	if _, err := s.pool.Exec(ctx, "UPDATE accounts SET failed_logins = $1, locked_until = NULL", disableThreshold-1); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ login, want, notWant string }{
		{"penyusun.uji", `"disabled":"true"`, `"locked_until":`},
		{adminLogin, `"locked_until":`, `"disabled"`},
	} {
		if _, _, err := s.Login(ctx, tc.login, "bukan password yang benar"); err == nil {
			t.Fatal("login dengan password salah berhasil")
		}
		rows, err = db.New(s.pool).ListAuditChain(ctx)
		if err != nil {
			t.Fatal(err)
		}
		last = rows[len(rows)-1]
		if !strings.Contains(last.Details, tc.want) || strings.Contains(last.Details, tc.notWant) {
			t.Errorf("%s: rincian kegagalan ke-100 %s, ingin memuat %s", tc.login, last.Details, tc.want)
		}
	}
}

// TestLockAccountAllowsAudit: kunci baris akun di SetRoles tidak boleh
// menghalangi transaksi lain yang menulis entri audit tentang akun itu. Bila
// menghalangi, dua transaksi bisa saling menunggu (deadlock): SetRoles memegang
// baris akun sambil menunggu kunci rantai audit, sedangkan transaksi lain
// memegang kunci rantai sambil menunggu baris akun untuk pemeriksaan foreign key.
func TestLockAccountAllowsAudit(t *testing.T) {
	s, admin := newAdminActor(t)
	ctx := t.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := s.q.WithTx(tx).LockAccount(ctx, admin.AccountID); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- db.WithTx(ctx, s.pool, func(other pgx.Tx) error {
			return audit.Record(ctx, s.q.WithTx(other), admin, audit.Event{
				Action: audit.ActionLogout, TargetType: audit.TargetAccount, TargetID: admin.AccountID,
			})
		})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("menulis entri audit: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("entri audit terhalang kunci baris akun")
	}
}
