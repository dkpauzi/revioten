package numbering

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/db"
)

// NewSet adalah isian pendaftaran sebagai set (UC-11, R-NUM-05).
type NewSet struct {
	CustomerID      string
	InitialRevision int
	IdempotencyKey  string
}

// RegisterSet mendaftarkan PFD, PFMEA, dan CP satu model dengan nomor berjalan
// yang sama (M2.4, R-NUM-05, E-1). Nomornya adalah nomor bebas berikutnya di
// ketiga jenis: nilai penghitung tertinggi ditambah satu. Penghitung yang
// tertinggal melewati beberapa nomor; nomor itu dicatat dengan status
// Skipped, supaya register tetap tanpa lubang (R-NUM-03) dan nomornya tidak
// pernah diberikan lagi (ASUMSI E-64, usulan M2.4 dan Q-05).
func (s *Service) RegisterSet(ctx context.Context, actor access.Actor, in NewSet) (Registration, error) {
	if err := s.authz.Authorize(ctx, actor, access.RegisterNumbers, access.Resource{CustomerID: in.CustomerID}); err != nil {
		return Registration{}, err
	}
	settings, err := s.q.GetNumberSettings(ctx)
	if err != nil {
		return Registration{}, fmt.Errorf("membaca pengaturan nomor: %w", err)
	}
	list, err := s.checkCustomer(ctx, in.CustomerID)
	if err != nil {
		return Registration{}, err
	}
	list = append(list, checkRevisionAndKey(in.InitialRevision, in.IdempotencyKey, int(settings.RevisionDigits))...)
	if err := problems(list...); err != nil {
		return Registration{}, err
	}
	pattern, err := ParsePattern(settings.Pattern)
	if err != nil {
		return Registration{}, fmt.Errorf("pola nomor di database tidak sah: %w", err)
	}
	types, err := s.setTypes(ctx)
	if err != nil {
		return Registration{}, err
	}
	hash := requestHash(OriginSet, in.CustomerID, strconv.Itoa(in.InitialRevision))

	var batchID string
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		code, err := customerCodeForNumber(ctx, q, in.CustomerID)
		if err != nil {
			return err
		}
		batchID, err = startBatch(ctx, q, actor, OriginSet, in.CustomerID, in.IdempotencyKey, hash)
		if err != nil {
			return err
		}
		// Ketiga penghitung dikunci menurut urutan ID jenis dokumen, urutan
		// yang sama dengan impor (M2.5), supaya tidak terjadi deadlock.
		last := map[string]int{}
		next := FirstSequence
		for _, d := range types {
			v, err := q.LockCounter(ctx, db.LockCounterParams{CustomerCode: code, DocumentTypeID: d.ID, FirstValue: FirstSequence - 1})
			if err != nil {
				return err
			}
			last[d.ID] = int(v)
			next = max(next, int(v)+1)
		}
		for _, d := range types {
			base := newRow{customerID: in.CustomerID, code: code, docType: d, batchID: batchID}
			for v := last[d.ID] + 1; v < next; v++ {
				skipped := base
				skipped.sequence, skipped.status = v, StatusSkipped
				if err := insertNumber(ctx, q, actor, pattern, skipped); err != nil {
					return err
				}
			}
			reserved := base
			reserved.sequence, reserved.status, reserved.initialRevision = next, StatusReserved, in.InitialRevision
			if err := insertNumber(ctx, q, actor, pattern, reserved); err != nil {
				return err
			}
			if err := q.SetCounter(ctx, db.SetCounterParams{CustomerCode: code, DocumentTypeID: d.ID, LastValue: int32(next)}); err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, errReplay) {
		return s.replay(ctx, actor, in.IdempotencyKey, hash)
	}
	if err := registerResult(err); err != nil {
		return Registration{}, err
	}
	numbers, err := s.batchNumbers(ctx, batchID)
	return Registration{Numbers: numbers}, err
}

// setTypes mengembalikan jenis dokumen yang dinomori sistem (PFD, PFMEA, dan
// CP), urut menurut ID untuk urutan kunci.
func (s *Service) setTypes(ctx context.Context) ([]DocumentType, error) {
	rows, err := s.q.ListDocumentTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil jenis dokumen: %w", err)
	}
	var types []DocumentType
	for _, r := range rows {
		if r.Key != nil {
			types = append(types, DocumentType{ID: r.ID, Key: *r.Key, Name: r.Name, Letter: r.Letter, Prefix: r.Prefix})
		}
	}
	slices.SortFunc(types, func(a, b DocumentType) int {
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		}
		return 0
	})
	return types, nil
}
