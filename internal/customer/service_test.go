package customer

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/dkpauzi/revioten/internal/db/dbtest"
)

// TestMain menyalakan PostgreSQL untuk test di paket ini (lihat dbtest).
func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func TestListEmpty(t *testing.T) {
	s := NewService(dbtest.NewPool(t))

	list, err := s.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Daftar kosong harus berupa slice kosong, bukan nil, supaya JSON-nya [].
	if list == nil || len(list) != 0 {
		t.Errorf("List = %#v, ingin slice kosong", list)
	}
}

func TestCreateAndList(t *testing.T) {
	s := NewService(dbtest.NewPool(t))
	ctx := t.Context()
	before := time.Now()

	created, err := s.Create(ctx, NewCustomer{Code: " bb ", Name: " Customer B "})
	if err != nil {
		t.Fatalf("Create BB: %v", err)
	}
	if _, err := s.Create(ctx, NewCustomer{Code: "AA", Name: "Customer A"}); err != nil {
		t.Fatalf("Create AA: %v", err)
	}

	// Hasil Create sudah dirapikan dan diberi id serta waktu oleh database.
	if created.ID == "" {
		t.Error("ID kosong")
	}
	if created.Code != "BB" || created.Name != "Customer B" {
		t.Errorf("Create = %+v, ingin kode BB dan nama Customer B", created)
	}
	if created.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt dalam zona %v, ingin UTC", created.CreatedAt.Location())
	}
	if created.CreatedAt.Before(before.Add(-time.Minute)) || created.CreatedAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("CreatedAt = %v, tidak dekat dengan waktu test", created.CreatedAt)
	}

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var codes []string
	for _, c := range list {
		codes = append(codes, c.Code)
	}
	if len(codes) != 2 || codes[0] != "AA" || codes[1] != "BB" {
		t.Errorf("urutan kode = %v, ingin [AA BB]", codes)
	}
	if list[1] != created {
		t.Errorf("baris BB dari List = %+v, ingin sama dengan hasil Create %+v", list[1], created)
	}
}

func TestCreateDuplicateCode(t *testing.T) {
	s := NewService(dbtest.NewPool(t))
	ctx := t.Context()
	if _, err := s.Create(ctx, NewCustomer{Code: "AB", Name: "Customer Pertama"}); err != nil {
		t.Fatalf("Create pertama: %v", err)
	}

	// Kode yang sama setelah dirapikan (" ab " menjadi AB) harus ditolak.
	_, err := s.Create(ctx, NewCustomer{Code: " ab ", Name: "Customer Kedua"})

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("ingin *ConflictError, dapat %v", err)
	}
	if len(conflict.Problems) != 1 || conflict.Problems[0].Field != "code" || conflict.Problems[0].Code != ProblemTaken {
		t.Errorf("masalah = %+v, ingin satu masalah code:taken", conflict.Problems)
	}
}

func TestCreateInvalidIsNotStored(t *testing.T) {
	s := NewService(dbtest.NewPool(t))
	ctx := t.Context()

	_, err := s.Create(ctx, NewCustomer{Code: "ZZ", Name: "General"})
	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("ingin *ValidationError, dapat %v", err)
	}

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("ada %d customer tersimpan, ingin 0", len(list))
	}
}
