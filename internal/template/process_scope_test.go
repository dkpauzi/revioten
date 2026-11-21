package template

import (
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// Test M3.3: cakupan poin dan prioritas (D-129, R-TPL-03, D-39). Satu poin
// punya cakupan umum, customer, group model, atau model, dan boleh punya
// varian untuk cakupan yang lebih sempit. Untuk satu model, varian yang
// paling spesifik menang: model, lalu group model, lalu customer, lalu umum.

// scopeOf membuat cakupan untuk test fungsi murni resolve.
func scopeOf(level, target string) Scope {
	return Scope{Level: level, TargetID: target}
}

// texts meringkas hasil resolve menjadi "sub-proses/teks" supaya mudah dibandingkan.
func texts(list []ResolvedSubprocess) []string {
	var out []string
	for _, s := range list {
		for _, p := range s.Points {
			out = append(out, s.Name+"/"+p.Text)
		}
	}
	return out
}

// TestResolvePriority menguji aturan prioritas tanpa database. Contoh
// pemilik: proses laser marking, poin laser type umum, tetapi laser type
// sebenarnya berbeda per model.
func TestResolvePriority(t *testing.T) {
	pfd := DocumentTypeRef{ID: "pfd", Key: "pfd", Name: "PFD"}
	laserType := Point{ID: "laser-type", DocumentType: pfd, Text: "Laser type sesuai spesifikasi", Position: 1,
		Scope: scopeOf(ScopeGeneral, ""),
		Variants: []Variant{
			{ID: "v-cust", Scope: scopeOf(ScopeCustomer, "cust-a"), Text: "Laser type customer A"},
			{ID: "v-group", Scope: scopeOf(ScopeModelGroup, "group-1"), Text: "Laser type group 1"},
			{ID: "v-model", Scope: scopeOf(ScopeModel, "model-x"), Text: "Laser type model X"},
			{ID: "v-hidden", Scope: scopeOf(ScopeModel, "model-y"), Text: "Laser type model Y (tersembunyi)", Hidden: true},
		}}
	content := Point{ID: "content", DocumentType: pfd, Text: "Isi marking", Position: 2, Scope: scopeOf(ScopeGeneral, ""), Variants: []Variant{}}
	onlyB := Point{ID: "only-b", DocumentType: pfd, Text: "Cek khusus customer B", Position: 3, Scope: scopeOf(ScopeCustomer, "cust-b"), Variants: []Variant{}}
	hiddenPoint := Point{ID: "old", DocumentType: pfd, Text: "Poin lama", Position: 4, Hidden: true, Scope: scopeOf(ScopeGeneral, ""), Variants: []Variant{}}
	v := ProcessVersion{Subprocesses: []Subprocess{
		{ID: "s1", Name: "Laser", Position: 1, Points: []Point{laserType, content, onlyB, hiddenPoint}},
		{ID: "s2", Name: "Lama", Position: 2, Hidden: true, Points: []Point{content}},
	}}

	cases := []struct {
		name  string
		model modelScope
		want  []string
	}{
		{"varian model menang", modelScope{ID: "model-x", CustomerID: "cust-a", GroupID: "group-1"},
			[]string{"Laser/Laser type model X", "Laser/Isi marking"}},
		{"tanpa varian model, varian group menang", modelScope{ID: "model-z", CustomerID: "cust-a", GroupID: "group-1"},
			[]string{"Laser/Laser type group 1", "Laser/Isi marking"}},
		// Usulan milestone M3.3: model tanpa group jatuh ke tingkat customer.
		{"model tanpa group jatuh ke customer", modelScope{ID: "model-z", CustomerID: "cust-a"},
			[]string{"Laser/Laser type customer A", "Laser/Isi marking"}},
		// Usulan milestone M3.3: group tanpa varian juga jatuh ke tingkat customer.
		{"group tanpa varian jatuh ke customer", modelScope{ID: "model-z", CustomerID: "cust-a", GroupID: "group-2"},
			[]string{"Laser/Laser type customer A", "Laser/Isi marking"}},
		// Varian yang disembunyikan dianggap tidak ada (asumsi rencana M3.3).
		{"varian tersembunyi diabaikan", modelScope{ID: "model-y", CustomerID: "cust-c"},
			[]string{"Laser/Laser type sesuai spesifikasi", "Laser/Isi marking"}},
		// Poin yang cakupannya customer B hanya muncul untuk model customer B.
		{"poin khusus customer hanya untuk customer itu", modelScope{ID: "model-b", CustomerID: "cust-b"},
			[]string{"Laser/Laser type sesuai spesifikasi", "Laser/Isi marking", "Laser/Cek khusus customer B"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := texts(resolve(v, c.model)); !slices.Equal(got, c.want) {
				t.Errorf("hasil %q, ingin %q", got, c.want)
			}
		})
	}

	// Poin hasil resolve menyimpan ID poin asal, supaya baris dokumen nanti
	// tetap tertaut ke poin template yang sama (R-CMP-04).
	got := resolve(v, modelScope{ID: "model-x", CustomerID: "cust-a", GroupID: "group-1"})
	first := got[0].Points[0]
	if first.PointID != "v-model" || first.OriginID != "laser-type" || first.Scope.Level != ScopeModel {
		t.Errorf("poin pertama %+v, ingin varian model dari laser-type", first)
	}
}

// scopeFixture menambah customer, group model, dan model ke processFixture.
type scopeFixture struct {
	processFixture
	custA, custB   string // id customer
	groupA, groupB string // group model milik customer A
	modelX         string // model customer A di group A
	modelLone      string // model customer A tanpa group
	modelB         string // model customer B
}

func newScopeFixture(t *testing.T) scopeFixture {
	t.Helper()
	f := scopeFixture{processFixture: newProcessFixture(t)}
	ctx := t.Context()
	pg, err := f.q.CreateProjectGroup(ctx, db.CreateProjectGroupParams{Code: "GR1", Name: "Project Group 1"})
	if err != nil {
		t.Fatal(err)
	}
	customer := func(code string) string {
		c, err := f.q.CreateCustomer(ctx, db.CreateCustomerParams{Code: code, Name: "Customer " + code, ProjectGroupID: &pg.ID})
		if err != nil {
			t.Fatal(err)
		}
		return c.ID
	}
	group := func(customerID, name string) string {
		g, err := f.q.CreateModelGroup(ctx, db.CreateModelGroupParams{CustomerID: customerID, Name: name})
		if err != nil {
			t.Fatal(err)
		}
		return g.ID
	}
	model := func(customerID string, groupID *string, name string) string {
		id, err := f.q.CreateModel(ctx, db.CreateModelParams{CustomerID: customerID, ModelGroupID: groupID, Name: name})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	f.custA, f.custB = customer("AB"), customer("CD")
	f.groupA, f.groupB = group(f.custA, "Group Laser"), group(f.custA, "Group Lain")
	f.modelX = model(f.custA, &f.groupA, "K-125")
	f.modelLone = model(f.custA, nil, "K-200")
	f.modelB = model(f.custB, nil, "B-1")
	return f
}

// laserTemplate membuat proses Laser Marking dengan satu poin PFD umum "Laser type".
func (f scopeFixture) laserTemplate(t *testing.T) (ProcessVersion, Point) {
	t.Helper()
	v := f.newTemplate(t, "Laser Marking", "Laser")
	v = f.addPoint(t, v, "Laser", "cp", "Laser type sesuai spesifikasi")
	return v, subprocessNamed(t, v, "Laser").Points[0]
}

// TestAddVariant: Pengelola template menambah varian; cakupan, teks, dan
// kepemilikan customer diperiksa, dan satu cakupan hanya punya satu varian.
func TestAddVariant(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()
	v, laser := f.laserTemplate(t)
	if laser.Scope.Level != ScopeGeneral || len(laser.Variants) != 0 {
		t.Fatalf("poin baru: cakupan %+v, varian %d; ingin umum tanpa varian", laser.Scope, len(laser.Variants))
	}

	v, err := f.s.AddVariant(ctx, f.maintainer, v.ID, laser.ID, NewVariant{
		Scope: ScopeInput{Level: ScopeModel, TargetID: f.modelX}, Text: "  Laser type: fiber 20 W  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := subprocessNamed(t, v, "Laser").Points[0]
	if len(got.Variants) != 1 {
		t.Fatalf("varian %d, ingin 1", len(got.Variants))
	}
	variant := got.Variants[0]
	if variant.Text != "Laser type: fiber 20 W" || variant.Scope.Level != ScopeModel ||
		variant.Scope.TargetID != f.modelX || variant.Scope.Label != "K-125" {
		t.Errorf("varian %+v", variant)
	}
	// Varian tidak menambah poin di daftar sub-proses: posisinya mengikuti poin asal.
	if n := len(subprocessNamed(t, v, "Laser").Points); n != 1 {
		t.Errorf("poin di sub-proses %d, ingin 1", n)
	}

	for _, c := range []struct {
		name string
		in   NewVariant
		want []string
	}{
		{"cakupan umum bukan varian", NewVariant{Scope: ScopeInput{Level: ScopeGeneral}, Text: "X"}, []string{"scope_level:not_allowed"}},
		{"tingkat tidak dikenal", NewVariant{Scope: ScopeInput{Level: "plant", TargetID: f.custA}, Text: "X"}, []string{"scope_level:invalid_format"}},
		{"target kosong", NewVariant{Scope: ScopeInput{Level: ScopeCustomer}, Text: "X"}, []string{"scope_target_id:required"}},
		{"target tidak ada", NewVariant{Scope: ScopeInput{Level: ScopeModel, TargetID: "00000000-0000-7000-8000-000000000000"}, Text: "X"}, []string{"scope_target_id:invalid_format"}},
		{"teks kosong", NewVariant{Scope: ScopeInput{Level: ScopeCustomer, TargetID: f.custA}, Text: " "}, []string{"text:required"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := f.s.AddVariant(ctx, f.maintainer, v.ID, laser.ID, c.in)
			if got := problemCodes(t, err); !slices.Equal(got, c.want) {
				t.Errorf("masalah %q, ingin %q", got, c.want)
			}
		})
	}

	// Satu cakupan hanya punya satu varian dari poin yang sama.
	_, err = f.s.AddVariant(ctx, f.maintainer, v.ID, laser.ID, NewVariant{Scope: ScopeInput{Level: ScopeModel, TargetID: f.modelX}, Text: "Lagi"})
	if got := problemCodes(t, err); !slices.Equal(got, []string{"scope_target_id:taken"}) {
		t.Errorf("varian ganda: %q", got)
	}

	// Varian dari varian tidak ada: ID varian bukan poin di versi ini.
	if _, err := f.s.AddVariant(ctx, f.maintainer, v.ID, variant.ID, NewVariant{
		Scope: ScopeInput{Level: ScopeModel, TargetID: f.modelLone}, Text: "X",
	}); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("varian dari varian: %v", err)
	}

	// Hanya Pengelola template (E-6, D-64).
	for _, actor := range []access.Actor{f.admin, f.author} {
		if _, err := f.s.AddVariant(ctx, actor, v.ID, laser.ID, NewVariant{
			Scope: ScopeInput{Level: ScopeCustomer, TargetID: f.custA}, Text: "X",
		}); !errors.Is(err, access.ErrForbidden) {
			t.Errorf("%s: %v, ingin ErrForbidden", actor.Login, err)
		}
	}
}

// TestVariantWithinOrigin: varian harus lebih sempit dari poin asalnya dan
// berada di dalamnya. Poin khusus customer A tidak bisa diberi varian untuk
// model customer B, dan poin khusus model tidak punya varian.
func TestVariantWithinOrigin(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()
	v := f.newTemplate(t, "Laser Marking", "Laser")
	sub := subprocessNamed(t, v, "Laser")
	add := func(scope ScopeInput, text string) Point {
		t.Helper()
		var err error
		v, err = f.s.AddPoint(ctx, f.maintainer, v.ID, NewPoint{SubprocessID: sub.ID, DocumentTypeID: f.types["cp"], Text: text, Scope: scope})
		if err != nil {
			t.Fatal(err)
		}
		points := subprocessNamed(t, v, "Laser").Points
		return points[len(points)-1]
	}
	forA := add(ScopeInput{Level: ScopeCustomer, TargetID: f.custA}, "Cek khusus customer A")
	if forA.Scope.Level != ScopeCustomer || forA.Scope.Label != "AB" {
		t.Fatalf("poin customer A: %+v", forA.Scope)
	}
	forGroup := add(ScopeInput{Level: ScopeModelGroup, TargetID: f.groupA}, "Cek group laser")
	forModel := add(ScopeInput{Level: ScopeModel, TargetID: f.modelX}, "Cek model K-125")

	for _, c := range []struct {
		name   string
		origin Point
		scope  ScopeInput
		want   []string
	}{
		{"model customer lain", forA, ScopeInput{Level: ScopeModel, TargetID: f.modelB}, []string{"scope_target_id:not_allowed"}},
		{"customer lain", forA, ScopeInput{Level: ScopeCustomer, TargetID: f.custB}, []string{"scope_level:not_allowed"}},
		{"group lain", forGroup, ScopeInput{Level: ScopeModel, TargetID: f.modelLone}, []string{"scope_target_id:not_allowed"}},
		{"tidak lebih sempit", forGroup, ScopeInput{Level: ScopeCustomer, TargetID: f.custA}, []string{"scope_level:not_allowed"}},
		{"poin khusus model", forModel, ScopeInput{Level: ScopeModel, TargetID: f.modelLone}, []string{"scope_level:not_allowed"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := f.s.AddVariant(ctx, f.maintainer, v.ID, c.origin.ID, NewVariant{Scope: c.scope, Text: "X"})
			if got := problemCodes(t, err); !slices.Equal(got, c.want) {
				t.Errorf("masalah %q, ingin %q", got, c.want)
			}
		})
	}

	// Yang sah: varian group A dari poin customer A, dan model K-125 dari poin group A.
	if _, err := f.s.AddVariant(ctx, f.maintainer, v.ID, forA.ID, NewVariant{Scope: ScopeInput{Level: ScopeModelGroup, TargetID: f.groupA}, Text: "Group laser"}); err != nil {
		t.Errorf("varian group dari poin customer: %v", err)
	}
	if _, err := f.s.AddVariant(ctx, f.maintainer, v.ID, forGroup.ID, NewVariant{Scope: ScopeInput{Level: ScopeModel, TargetID: f.modelX}, Text: "Model K-125"}); err != nil {
		t.Errorf("varian model dari poin group: %v", err)
	}
}

// TestUpdateVariant: teks varian bisa diganti dan varian bisa disembunyikan,
// tetapi cakupannya tetap.
func TestUpdateVariant(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()
	v, laser := f.laserTemplate(t)
	v, err := f.s.AddVariant(ctx, f.maintainer, v.ID, laser.ID, NewVariant{Scope: ScopeInput{Level: ScopeCustomer, TargetID: f.custA}, Text: "Laser A"})
	if err != nil {
		t.Fatal(err)
	}
	variantID := subprocessNamed(t, v, "Laser").Points[0].Variants[0].ID
	hidden := true
	v, err = f.s.UpdateVariant(ctx, f.maintainer, v.ID, variantID, VariantUpdate{Text: "Laser A, CO2", Hidden: &hidden})
	if err != nil {
		t.Fatal(err)
	}
	got := subprocessNamed(t, v, "Laser").Points[0].Variants[0]
	if got.Text != "Laser A, CO2" || !got.Hidden || got.Scope.TargetID != f.custA {
		t.Errorf("varian sesudah diubah: %+v", got)
	}
	if _, err := f.s.UpdateVariant(ctx, f.maintainer, v.ID, variantID, VariantUpdate{Text: ""}); !slices.Equal(problemCodes(t, err), []string{"text:required"}) {
		t.Errorf("teks kosong: %v", err)
	}
	if _, err := f.s.UpdateVariant(ctx, f.maintainer, v.ID, laser.ID, VariantUpdate{Text: "X"}); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("poin asal bukan varian: %v", err)
	}
	if _, err := f.s.UpdateVariant(ctx, f.author, v.ID, variantID, VariantUpdate{Text: "X"}); !errors.Is(err, access.ErrForbidden) {
		t.Errorf("Penyusun: %v", err)
	}
}

// TestResolvePoints menjalankan contoh laser marking lewat database: model
// K-125 memakai varian modelnya, model tanpa varian memakai poin umum.
func TestResolvePoints(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()
	v, laser := f.laserTemplate(t)
	v = f.addPoint(t, v, "Laser", "cp", "Isi dan lokasi marking sesuai drawing")
	v, err := f.s.AddVariant(ctx, f.maintainer, v.ID, laser.ID, NewVariant{Scope: ScopeInput{Level: ScopeModel, TargetID: f.modelX}, Text: "Laser type: fiber 20 W"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := f.s.ResolvePoints(ctx, f.author, v.ID, f.modelX)
	if err != nil {
		t.Fatal(err)
	}
	if got.Model.Name != "K-125" || got.Model.CustomerCode != "AB" || got.Model.ModelGroupName != "Group Laser" {
		t.Errorf("model %+v", got.Model)
	}
	if want := []string{"Laser/Laser type: fiber 20 W", "Laser/Isi dan lokasi marking sesuai drawing"}; !slices.Equal(texts(got.Subprocesses), want) {
		t.Errorf("K-125: %q, ingin %q", texts(got.Subprocesses), want)
	}
	got, err = f.s.ResolvePoints(ctx, f.author, v.ID, f.modelB)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Laser/Laser type sesuai spesifikasi", "Laser/Isi dan lokasi marking sesuai drawing"}; !slices.Equal(texts(got.Subprocesses), want) {
		t.Errorf("B-1: %q, ingin %q", texts(got.Subprocesses), want)
	}

	if _, err := f.s.ResolvePoints(ctx, f.author, v.ID, "bukan-id"); !slices.Equal(problemCodes(t, err), []string{"model_id:invalid_format"}) {
		t.Errorf("model tidak ada: %v", err)
	}
	if _, err := f.s.ResolvePoints(ctx, f.author, "bukan-id", f.modelX); !errors.Is(err, problem.ErrNotFound) {
		t.Errorf("versi tidak ada: %v", err)
	}
}

// TestPointScopeGuards: database sendiri menolak cakupan yang tidak lengkap
// dan varian ganda, juga bila aplikasi dilewati.
func TestPointScopeGuards(t *testing.T) {
	f := newScopeFixture(t)
	ctx := t.Context()
	_, laser := f.laserTemplate(t)
	var subprocessID, typeID string
	if err := f.s.pool.QueryRow(ctx, "SELECT subprocess_id, document_type_id FROM template_points WHERE id = $1", laser.ID).Scan(&subprocessID, &typeID); err != nil {
		t.Fatal(err)
	}
	insert := func(scope string, customerID, modelID, variantOf *string) error {
		return f.exec(t, `INSERT INTO template_points (subprocess_id, document_type_id, scope, customer_id, model_id, variant_of)
			VALUES ($1, $2, $3, $4, $5, $6)`, subprocessID, typeID, scope, customerID, modelID, variantOf)
	}
	code := func(err error) string {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return pgErr.Code
		}
		return ""
	}
	if err := insert(ScopeCustomer, nil, nil, nil); code(err) != "23514" {
		t.Errorf("customer tanpa id: %v, ingin 23514", err)
	}
	if err := insert(ScopeGeneral, &f.custA, nil, nil); code(err) != "23514" {
		t.Errorf("umum dengan customer: %v, ingin 23514", err)
	}
	if err := insert(ScopeGeneral, nil, nil, &laser.ID); code(err) != "23514" {
		t.Errorf("varian umum: %v, ingin 23514", err)
	}
	if err := insert(ScopeModel, nil, &f.modelX, &laser.ID); err != nil {
		t.Fatalf("varian model: %v", err)
	}
	if err := insert(ScopeModel, nil, &f.modelX, &laser.ID); code(err) != "23505" {
		t.Errorf("varian ganda: %v, ingin 23505", err)
	}
}
