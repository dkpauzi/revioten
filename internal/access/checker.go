package access

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dkpauzi/revioten/internal/db"
)

// Checker memeriksa izin terhadap peran yang tersimpan di database.
type Checker struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewChecker membuat Checker yang memakai pool database.
func NewChecker(pool *pgxpool.Pool) *Checker {
	return &Checker{pool: pool, q: db.New(pool)}
}

// Authorize melaporkan nil bila actor boleh menjalankan perm pada res, atau
// ErrForbidden bila tidak. Hak dibaca dari database setiap kali, jadi
// perubahan peran langsung berlaku.
func (c *Checker) Authorize(ctx context.Context, actor Actor, perm Permission, res Resource) error {
	if actor.AccountID == "" {
		// Pengaman: operasi tanpa aktor selalu ditolak.
		return ErrForbidden
	}
	if perm == ManageOwnAccount {
		if res.AccountID == actor.AccountID {
			return nil
		}
		return ErrForbidden
	}

	grants, err := c.q.ListGrants(ctx, actor.AccountID)
	if err != nil {
		return fmt.Errorf("membaca hak akun: %w", err)
	}
	for _, g := range grants {
		if g.PermissionCode != string(perm) {
			continue
		}
		if g.Scope == ScopeAll {
			return nil
		}
		// Hak bercakupan Project Group hanya berlaku untuk operasi pada satu
		// customer. Operasi tanpa customer ditolak, bukan dianggap boleh
		// (ASUMSI E-41: tolak bila ragu).
		if res.CustomerID == "" {
			continue
		}
		inGroup, err := c.inOwnProjectGroup(ctx, actor.PersonID, res.CustomerID)
		if err != nil {
			return err
		}
		if inGroup {
			return nil
		}
	}
	return ErrForbidden
}

// inOwnProjectGroup melaporkan apakah customer termasuk Project Group tempat
// orang itu menjadi Project Leader atau Project Engineer. Sumbernya master
// Project Group dan pemegang jabatan (E-05, M1.8), bukan isian di peran, jadi
// hak langsung ikut berubah saat pemegang jabatan diganti.
func (c *Checker) inOwnProjectGroup(ctx context.Context, personID, customerID string) (bool, error) {
	if personID == "" || !db.IsUUID(customerID) {
		return false, nil
	}
	in, err := c.q.InOwnProjectGroup(ctx, db.InOwnProjectGroupParams{PersonID: personID, CustomerID: customerID})
	if err != nil {
		return false, fmt.Errorf("memeriksa Project Group: %w", err)
	}
	return in, nil
}

// Permissions mengembalikan semua hak actor, urut seperti katalog U3. Dipakai
// layar untuk menampilkan atau menyembunyikan menu; server tetap memeriksa
// setiap operasi dengan Authorize.
func (c *Checker) Permissions(ctx context.Context, actor Actor) ([]Permission, error) {
	if err := c.Authorize(ctx, actor, ManageOwnAccount, Resource{AccountID: actor.AccountID}); err != nil {
		return nil, err
	}
	grants, err := c.q.ListGrants(ctx, actor.AccountID)
	if err != nil {
		return nil, fmt.Errorf("membaca hak akun: %w", err)
	}
	catalog, err := c.permissionOrder(ctx)
	if err != nil {
		return nil, err
	}
	var perms []Permission
	for _, code := range catalog {
		has := slices.ContainsFunc(grants, func(g db.ListGrantsRow) bool { return g.PermissionCode == string(code) })
		if has {
			perms = append(perms, code)
		}
	}
	return perms, nil
}

// permissionOrder mengembalikan semua kode hak menurut urutan katalog.
func (c *Checker) permissionOrder(ctx context.Context) ([]Permission, error) {
	rows, err := c.q.ListRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca katalog hak: %w", err)
	}
	var order []Permission
	for _, r := range rows {
		if !slices.Contains(order, Permission(r.PermissionCode)) {
			order = append(order, Permission(r.PermissionCode))
		}
	}
	return order, nil
}
