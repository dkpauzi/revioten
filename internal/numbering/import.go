package numbering

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/dkpauzi/revioten/internal/access"
	"github.com/dkpauzi/revioten/internal/audit"
	"github.com/dkpauzi/revioten/internal/db"
	"github.com/dkpauzi/revioten/internal/problem"
)

// MaxImportLines adalah jumlah baris nomor paling banyak dalam satu impor.
// ASUMSI E-65: register lama yang lebih panjang diimpor dalam beberapa kali.
const MaxImportLines = 5000

// maxImportCell adalah panjang kolom pertama paling banyak yang ditulis di
// laporan. Isi yang lebih panjang pasti bukan nomor.
const maxImportCell = 100

// Hasil satu baris impor (M2.5).
const (
	LineImported         = "imported"          // masuk ke register
	LineExisting         = "existing"          // sudah ada di register, tidak diubah
	LineDuplicate        = "duplicate"         // nomor yang sama sudah ada di baris sebelumnya
	LineCustomerMismatch = "customer_mismatch" // kode di nomor bukan kode customer ini (R-POR-01)
	LineInvalid          = "invalid"           // bukan nomor PFD, PFMEA, atau CP menurut pola sekarang
	LineTypeRepeated     = "type_repeated"     // baris yang sama sudah punya nomor jenis ini (D-216)
	// Hanya impor berkas Excel (I4):
	LineUnknownCustomer  = "unknown_customer"  // kode di kolom Customer code tidak dikenal
	LineInactiveCustomer = "inactive_customer" // customer nonaktif (ASUMSI E-67)
	LineInvalidStatus    = "invalid_status"    // kolom Status bukan Released atau Reserved
)

// maxImportColumns adalah jumlah kolom yang dibaca per baris: PFD, PFMEA, dan
// CP satu set (D-216). Kolom sesudahnya, misalnya judul, diabaikan.
const maxImportColumns = 3

// importStatuses adalah status yang boleh dipilih untuk nomor lama. Skipped
// hanya dibuat oleh daftar sebagai set.
var importStatuses = []string{StatusReserved, StatusInUse, StatusReleased, StatusObsolete, StatusVoid}

// ImportInput adalah isian impor nomor lama (UC-14, R-NUM-06).
type ImportInput struct {
	CustomerID string
	// Status untuk semua nomor di impor ini (ASUMSI E-65).
	Status string
	// Text berisi nomor lama, satu set per baris (D-216): sampai tiga kolom
	// nomor PFD, PFMEA, dan CP yang disalin dari register Excel (dipisah
	// tab), dalam urutan apa pun, karena jenisnya dibaca dari awalan nomor.
	// Kolom kosong dilewati, dan kolom sesudah kolom ketiga diabaikan. Setiap baris menjadi satu batch,
	// jadi nomor satu baris adalah satu set dokumen (E-140).
	Text string
	// OneSet true (bawaan di layar): nomor yang tidak ada di baris mengikuti
	// nomor berjalan dan kode customer nomor pertama baris itu.
	OneSet bool
	// DryRun true berarti pratinjau: laporan saja, tidak ada yang disimpan.
	DryRun bool
}

// ImportLine adalah hasil satu baris impor.
type ImportLine struct {
	Line      int    // nomor baris di teks, mulai dari 1; baris kosong tidak dilaporkan
	Text      string // kolom pertama, tanpa spasi di awal dan akhir
	Result    string // LineImported, LineExisting, dan seterusnya
	FirstLine int    // untuk LineDuplicate: baris tempat nomor itu pertama muncul
	Derived   bool   // true bila nomor ini mengikuti nomor lain di barisnya (OneSet)
	// Revision adalah revisi yang ikut ditulis di belakang nomor, misalnya 5
	// untuk SC-KT-001-05 (I4, opsi b); nil bila tanpa revisi. Revisi ini
	// disimpan sebagai revisi awal nomor (D-225) dan mengisi kolom
	// revisi di menu Legacy documents. Text selalu nomor tanpa revisi.
	Revision *int
	// CustomerCode dan Status hanya diisi impor berkas Excel, yang memuat
	// beberapa customer dan status per baris.
	CustomerCode string
	Status       string
}

// SequenceRange adalah rentang nomor berjalan, misalnya 003 sampai 006.
type SequenceRange struct {
	First string
	Last  string
}

// ImportGroup adalah laporan satu kode customer dan satu jenis dokumen.
type ImportGroup struct {
	CustomerCode string
	DocumentType DocumentType
	Count        int             // jumlah nomor di register, semua status
	Highest      string          // nomor berjalan tertinggi; kosong bila Count 0
	Gaps         []SequenceRange // nomor berjalan yang tidak ada, dari 001 sampai Highest
}

// MisalignedRange adalah rentang nomor berjalan yang tidak dimiliki ketiga
// jenis dokumen: set yang tidak sejajar.
type MisalignedRange struct {
	CustomerCode string
	Range        SequenceRange
	Present      []DocumentType // jenis yang punya nomor di rentang ini
}

// ImportReport adalah laporan impor (R-NUM-06).
type ImportReport struct {
	DryRun     bool
	Status     string
	Imported   int // nomor yang masuk, atau yang akan masuk untuk pratinjau
	Lines      []ImportLine
	Groups     []ImportGroup     // per kode customer (kode lama dulu) dan jenis dokumen
	Misaligned []MisalignedRange // per kode customer, urut nomor berjalan
}

// errRollback membatalkan transaksi impor setelah laporannya tersusun: untuk
// pratinjau, dan bila tidak ada nomor yang masuk.
var errRollback = errors.New("impor dibatalkan setelah laporan tersusun")

// Import memasukkan nomor lama satu customer ke register (M2.5, UC-14, E-8).
// Nomor masuk dengan teks aslinya, dan penghitung setiap kode customer dan
// jenis dokumen naik ke nomor tertinggi, supaya nomor baru melanjutkan
// sesudahnya. Lubang, nomor ganda, dan set yang tidak sejajar hanya
// dilaporkan; tidak ada yang diperbaiki atau dinomori ulang, karena nomor
// lama mungkin sudah dipegang customer (R-NUM-06, RK-24). Baris yang bukan
// nomor PFD, PFMEA, atau CP menurut pola sekarang, dan nomor dengan kode
// customer lain, dilaporkan dan tidak diimpor (ASUMSI E-66, R-POR-01).
// Customer nonaktif ditolak seperti pada pendaftaran (ASUMSI E-67).
func (s *Service) Import(ctx context.Context, actor access.Actor, in ImportInput) (ImportReport, error) {
	if err := s.authz.Authorize(ctx, actor, access.ManageNumbers, access.Resource{CustomerID: in.CustomerID}); err != nil {
		return ImportReport{}, err
	}
	list, err := s.checkCustomer(ctx, in.CustomerID)
	if err != nil {
		return ImportReport{}, err
	}
	rows := splitImportLines(in.Text)
	list = append(list, checkImportStatus(in.Status), checkImportCount(len(rows)))
	if err := problems(list...); err != nil {
		return ImportReport{}, err
	}
	for i := range rows {
		rows[i].Status = in.Status
	}
	env, err := s.importEnv(ctx)
	if err != nil {
		return ImportReport{}, err
	}
	report, err := s.importRows(ctx, actor, env, in.CustomerID, rows, in.OneSet, in.DryRun)
	if err != nil {
		return ImportReport{}, err
	}
	// Status impor tempel satu untuk semua baris, jadi hanya ditulis sekali.
	report.Status = in.Status
	for i := range report.Lines {
		report.Lines[i].Status = ""
	}
	return report, nil
}

// importEnv adalah pengaturan nomor yang dipakai satu impor.
type importEnv struct {
	pattern        Pattern
	types          []DocumentType
	revisionDigits int
}

func (s *Service) importEnv(ctx context.Context) (importEnv, error) {
	settings, err := s.q.GetNumberSettings(ctx)
	if err != nil {
		return importEnv{}, fmt.Errorf("membaca pengaturan nomor: %w", err)
	}
	pattern, err := ParsePattern(settings.Pattern)
	if err != nil {
		return importEnv{}, fmt.Errorf("pola nomor di database tidak sah: %w", err)
	}
	types, err := s.setTypes(ctx)
	if err != nil {
		return importEnv{}, err
	}
	return importEnv{pattern: pattern, types: types, revisionDigits: int(settings.RevisionDigits)}, nil
}

// importRows mengimpor baris nomor satu customer dalam satu transaksi. Status
// setiap baris sudah diisi pemanggil, dan hak serta customer sudah diperiksa.
func (s *Service) importRows(ctx context.Context, actor access.Actor, env importEnv, customerID string, rows []importRow, oneSet, dryRun bool) (ImportReport, error) {
	parseImportLines(rows, env.pattern, env.types, env.revisionDigits)
	rows = markRepeatedTypes(rows)
	if oneSet {
		rows = deriveSetNumbers(rows, env.pattern, env.types)
	}

	report := ImportReport{DryRun: dryRun}
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		codes, err := lockCustomerCodes(ctx, q, customerID)
		if err != nil {
			return err
		}
		markMismatchAndDuplicates(rows, codes)
		if err := lockImportCounters(ctx, q, rows); err != nil {
			return err
		}
		if err := markExisting(ctx, q, rows, codes); err != nil {
			return err
		}
		report.Imported, err = insertImported(ctx, q, actor, customerID, rows)
		if err != nil {
			return err
		}
		register, err := registerOfCodes(ctx, q, codes)
		if err != nil {
			return err
		}
		report.Groups, report.Misaligned = buildReport(codes, env.types, register)
		if dryRun || report.Imported == 0 {
			// Laporan sudah tersusun; semua perubahan dibatalkan.
			return errRollback
		}
		return nil
	})
	if err := importResult(err); err != nil {
		return ImportReport{}, err
	}
	for _, r := range rows {
		report.Lines = append(report.Lines, r.ImportLine)
	}
	return report, nil
}

// checkImportStatus memeriksa status pilihan untuk nomor lama.
func checkImportStatus(status string) *problem.Problem {
	switch {
	case status == "":
		return &problem.Problem{Field: "status", Code: problem.Required, Message: "Choose the status of the imported numbers."}
	case !slices.Contains(importStatuses, status):
		return &problem.Problem{Field: "status", Code: problem.InvalidFormat, Message: "Choose Reserved, In use, Released, Obsolete, or Void."}
	}
	return nil
}

// checkImportCount memeriksa jumlah baris yang tidak kosong.
func checkImportCount(n int) *problem.Problem {
	switch {
	case n == 0:
		return &problem.Problem{Field: "text", Code: problem.Required, Message: "Paste at least one number."}
	case n > MaxImportLines:
		return &problem.Problem{Field: "text", Code: problem.TooLong,
			Message: fmt.Sprintf("Paste at most %d numbers at a time.", MaxImportLines)}
	}
	return nil
}

// importRow adalah satu baris impor beserta bagian nomornya. Result kosong
// berarti baris itu masih calon nomor yang diimpor.
type importRow struct {
	ImportLine
	key     numberKey
	docType DocumentType
}

// numberKey mengenali satu nomor: kode customer, jenis dokumen, dan nomor
// berjalan (sama dengan UNIQUE document_numbers_sequence_key).
type numberKey struct {
	code     string
	typeID   string
	sequence int
}

// counterKey mengenali satu baris penghitung: kode customer dan jenis dokumen.
type counterKey struct {
	code   string
	typeID string
}

// splitImportLines memotong teks menjadi baris. Baris kosong dilewati, tapi
// nomor barisnya tetap dihitung, supaya laporan menunjuk baris yang sama
// dengan teks yang ditempel.
func splitImportLines(text string) []importRow {
	var rows []importRow
	for i, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cells := strings.Split(line, "\t")
		for _, cell := range cells[:min(len(cells), maxImportColumns)] {
			cell = strings.TrimSpace(cell)
			if cell == "" {
				continue
			}
			if runes := []rune(cell); len(runes) > maxImportCell {
				cell = string(runes[:maxImportCell])
			}
			rows = append(rows, importRow{ImportLine: ImportLine{Line: i + 1, Text: cell}})
		}
	}
	return rows
}

// parseImportLines membaca setiap baris dengan pola nomor sekarang. Baris
// yang tidak terbaca, atau yang awalannya bukan awalan PFD, PFMEA, atau CP,
// ditandai LineInvalid (ASUMSI E-66). Akhiran revisi, misalnya -05,
// dipisahkan dulu menurut jumlah digit revisi (I4, opsi b).
func parseImportLines(rows []importRow, pattern Pattern, types []DocumentType, revisionDigits int) {
	byPrefix := map[string]DocumentType{}
	for _, d := range types {
		byPrefix[d.Prefix] = d
	}
	for i := range rows {
		if rows[i].Result != "" {
			continue
		}
		if number, revision, ok := pattern.SplitRevision(rows[i].Text, revisionDigits); ok {
			rows[i].Text = number
			rows[i].Revision = &revision
		}
		parts, ok := pattern.Parse(rows[i].Text)
		d, known := byPrefix[parts.Prefix]
		if !ok || !known {
			rows[i].Result = LineInvalid
			continue
		}
		sequence, _ := ParseSequence(parts.Sequence)
		rows[i].key = numberKey{code: parts.Customer, typeID: d.ID, sequence: sequence}
		rows[i].docType = d
	}
}

// markRepeatedTypes menandai nomor kedua dengan jenis yang sama di satu
// baris: satu baris adalah satu set, jadi hanya punya satu nomor per jenis.
func markRepeatedTypes(rows []importRow) []importRow {
	seen := map[int]map[string]bool{}
	for i := range rows {
		r := &rows[i]
		if r.Result != "" {
			continue
		}
		if seen[r.Line] == nil {
			seen[r.Line] = map[string]bool{}
		}
		if seen[r.Line][r.key.typeID] {
			r.Result = LineTypeRepeated
			continue
		}
		seen[r.Line][r.key.typeID] = true
	}
	return rows
}

// deriveSetNumbers menambah nomor yang tidak ada di setiap baris (OneSet):
// awalan jenis itu dengan kode customer dan nomor berjalan nomor sah pertama
// di baris, misalnya SH-TR-020 menjadi juga SF-TR-020 dan SC-TR-020.
func deriveSetNumbers(rows []importRow, pattern Pattern, types []DocumentType) []importRow {
	var out []importRow
	for start := 0; start < len(rows); {
		end := start
		for end < len(rows) && rows[end].Line == rows[start].Line {
			end++
		}
		line := rows[start:end]
		out = append(out, line...)
		var source *importRow
		present := map[string]bool{}
		for i := range line {
			if line[i].Result == "" || line[i].Result == LineTypeRepeated {
				present[line[i].key.typeID] = true
			}
			if line[i].Result == "" && source == nil {
				source = &line[i]
			}
		}
		if source != nil {
			for _, d := range types {
				if present[d.ID] {
					continue
				}
				out = append(out, importRow{
					ImportLine: ImportLine{Line: source.Line, Derived: true, Status: source.Status, CustomerCode: source.CustomerCode,
						Text: pattern.Format(d.Prefix, source.key.code, sequenceText(source.key.sequence))},
					key:     numberKey{code: source.key.code, typeID: d.ID, sequence: source.key.sequence},
					docType: d,
				})
			}
		}
		start = end
	}
	return out
}

// lockCustomerCodes mengunci customer seperti pendaftaran, sebagai langkah
// pertama transaksi (kode tidak bisa diganti selama impor, R-POR-01; urutan
// kunci di customerCodeForNumber), lalu mengembalikan semua kodenya, dari yang
// lama ke yang berlaku (E-11).
func lockCustomerCodes(ctx context.Context, q *db.Queries, customerID string) ([]string, error) {
	if _, err := customerCodeForNumber(ctx, q, customerID); err != nil {
		return nil, err
	}
	history, err := q.ListCodesOfCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("membaca kode customer: %w", err)
	}
	codes := make([]string, 0, len(history))
	for _, h := range history {
		codes = append(codes, h.Code)
	}
	return codes, nil
}

// markMismatchAndDuplicates menandai baris yang kodenya bukan kode customer
// ini (R-POR-01), lalu baris yang nomornya sudah muncul di baris sebelumnya.
// ASUMSI E-66: nomor ganda masuk sekali, dari baris pertamanya; register
// menyimpan setiap nomor hanya sekali.
func markMismatchAndDuplicates(rows []importRow, codes []string) {
	firstLine := map[numberKey]int{}
	for i := range rows {
		r := &rows[i]
		if r.Result != "" {
			continue
		}
		if !slices.Contains(codes, r.key.code) {
			r.Result = LineCustomerMismatch
			continue
		}
		if line, seen := firstLine[r.key]; seen {
			r.Result, r.FirstLine = LineDuplicate, line
			continue
		}
		firstLine[r.key] = r.Line
	}
}

// lockImportCounters mengunci penghitung setiap kode customer dan jenis
// dokumen di impor, menurut urutan kode lalu ID jenis dokumen (urutan yang
// sama dengan daftar sebagai set), supaya dua transaksi tidak saling menunggu
// (deadlock). Selama kunci dipegang, tidak ada transaksi lain yang bisa
// menambah nomor dengan kode dan jenis itu.
func lockImportCounters(ctx context.Context, q *db.Queries, rows []importRow) error {
	var counters []counterKey
	for _, r := range rows {
		c := counterKey{code: r.key.code, typeID: r.key.typeID}
		if r.Result == "" && !slices.Contains(counters, c) {
			counters = append(counters, c)
		}
	}
	slices.SortFunc(counters, func(a, b counterKey) int {
		return cmp.Or(strings.Compare(a.code, b.code), strings.Compare(a.typeID, b.typeID))
	})
	for _, c := range counters {
		_, err := q.LockCounter(ctx, db.LockCounterParams{CustomerCode: c.code, DocumentTypeID: c.typeID, FirstValue: FirstSequence - 1})
		if err != nil {
			return err
		}
	}
	return nil
}

// markExisting menandai baris yang nomornya sudah ada di register: nomor
// berjalan yang sama untuk kode dan jenis itu, atau teks yang sama (misalnya
// nomor yang terdaftar dengan pola lain). Nomor itu tidak diubah.
func markExisting(ctx context.Context, q *db.Queries, rows []importRow, codes []string) error {
	register, err := registerOfCodes(ctx, q, codes)
	if err != nil {
		return err
	}
	keys := map[numberKey]bool{}
	for _, r := range register {
		keys[numberKey{code: r.code, typeID: r.typeID, sequence: r.sequence}] = true
	}
	var texts []string
	for _, r := range rows {
		if r.Result == "" {
			texts = append(texts, r.Text)
		}
	}
	found, err := q.ListNumbersByText(ctx, texts)
	if err != nil {
		return fmt.Errorf("mencari nomor di register: %w", err)
	}
	taken := map[string]bool{}
	for _, text := range found {
		taken[text] = true
	}
	for i := range rows {
		r := &rows[i]
		if r.Result == "" && (keys[r.key] || taken[r.Text]) {
			r.Result = LineExisting
		}
	}
	return nil
}

// insertImported menyimpan baris yang tersisa dengan teks aslinya, mencatat
// entri audit setiap nomor, lalu menaikkan penghitung ke nomor tertinggi.
// Setiap baris yang punya nomor masuk mendapat batch sendiri, jadi nomor satu
// baris adalah satu set (D-216, E-140). Hasilnya jumlah nomor yang masuk.
func insertImported(ctx context.Context, q *db.Queries, actor access.Actor, customerID string, rows []importRow) (int, error) {
	batches := map[int]string{} // nomor baris -> batch
	highest := map[counterKey]int{}
	for i := range rows {
		r := &rows[i]
		if r.Result != "" {
			continue
		}
		batchID := batches[r.Line]
		if batchID == "" {
			// Impor tanpa kunci idempoten: impor yang diulang hanya
			// menghasilkan baris LineExisting.
			id, err := q.InsertNumberBatch(ctx, db.InsertNumberBatchParams{
				Kind: OriginImport, CustomerID: customerID, AccountID: actor.AccountID, PersonID: actor.PersonID,
			})
			if err != nil {
				return 0, err
			}
			batchID = id
			batches[r.Line] = id
		}
		revision := 0
		details := map[string]string{"number": r.Text, "status": r.Status}
		if r.Revision != nil {
			revision = *r.Revision
			details["initial_revision"] = strconv.Itoa(revision)
		}
		id, err := q.InsertDocumentNumber(ctx, db.InsertDocumentNumberParams{
			Number: r.Text, CustomerID: customerID, CustomerCode: r.key.code, DocumentTypeID: r.key.typeID,
			Sequence: int32(r.key.sequence), Status: r.Status, InitialRevision: int16(revision), BatchID: batchID,
		})
		if err != nil {
			return 0, err
		}
		err = audit.Record(ctx, q, actor, audit.Event{
			Action: audit.ActionDocumentNumberImported, TargetType: audit.TargetDocumentNumber, TargetID: id,
			Details: details,
		})
		if err != nil {
			return 0, err
		}
		r.Result = LineImported
		c := counterKey{code: r.key.code, typeID: r.key.typeID}
		highest[c] = max(highest[c], r.key.sequence)
	}
	for c, v := range highest {
		if err := q.SetCounter(ctx, db.SetCounterParams{CustomerCode: c.code, DocumentTypeID: c.typeID, LastValue: int32(v)}); err != nil {
			return 0, err
		}
	}
	imported := 0
	for _, r := range rows {
		if r.Result == LineImported {
			imported++
		}
	}
	return imported, nil
}

// importResult menerjemahkan error transaksi impor. Teks nomor yang
// didaftarkan transaksi lain di antara pemeriksaan dan penyimpanan menjadi
// ConflictError; seluruh impor batal, jadi impor bisa diulang.
func importResult(err error) error {
	var invalid *problem.ValidationError
	var conflict *problem.ConflictError
	switch {
	case err == nil, errors.Is(err, errRollback):
		return nil
	case db.IsUniqueViolation(err, "document_numbers_number_key"), db.IsUniqueViolation(err, "document_numbers_sequence_key"):
		return &problem.ConflictError{Problems: []problem.Problem{{
			Field: "text", Code: problem.Taken,
			Message: "A number in the list was registered while importing. Run the import again.",
		}}}
	case errors.As(err, &invalid), errors.As(err, &conflict), errors.Is(err, problem.ErrNotFound):
		return err
	}
	return fmt.Errorf("mengimpor nomor: %w", err)
}

// registerRow adalah satu nomor di register, untuk laporan impor.
type registerRow struct {
	code     string
	typeID   string
	sequence int
	status   string
}

// registerOfCodes mengambil semua nomor dengan kode-kode customer itu.
func registerOfCodes(ctx context.Context, q *db.Queries, codes []string) ([]registerRow, error) {
	list, err := q.ListRegisterOfCodes(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("membaca register: %w", err)
	}
	rows := make([]registerRow, 0, len(list))
	for _, r := range list {
		rows = append(rows, registerRow{code: r.CustomerCode, typeID: r.DocumentTypeID, sequence: int(r.Sequence), status: r.Status})
	}
	return rows, nil
}

// buildReport menyusun laporan register setelah impor, untuk setiap kode
// customer yang punya nomor (urutan codes) dan setiap jenis dokumen (urutan
// types):
//   - jumlah nomor, nomor tertinggi, dan lubang. Semua status dihitung, jadi
//     nomor Skipped bukan lubang (ASUMSI E-64);
//   - rentang nomor berjalan yang tidak dimiliki ketiga jenis (set tidak
//     sejajar). Di sini Skipped tidak dihitung, karena nomor itu tidak
//     pernah punya dokumen (ASUMSI E-65).
func buildReport(codes []string, types []DocumentType, rows []registerRow) ([]ImportGroup, []MisalignedRange) {
	// sequences[kode][ID jenis] berisi semua nomor berjalan; present[kode][nomor
	// berjalan] berisi ID jenis yang punya nomor itu dengan status selain Skipped.
	sequences := map[string]map[string][]int{}
	present := map[string]map[int][]string{}
	for _, r := range rows {
		if sequences[r.code] == nil {
			sequences[r.code] = map[string][]int{}
			present[r.code] = map[int][]string{}
		}
		sequences[r.code][r.typeID] = append(sequences[r.code][r.typeID], r.sequence)
		if r.status != StatusSkipped {
			present[r.code][r.sequence] = append(present[r.code][r.sequence], r.typeID)
		}
	}

	var groups []ImportGroup
	var misaligned []MisalignedRange
	for _, code := range codes {
		if sequences[code] == nil {
			continue
		}
		for _, d := range types {
			list := sequences[code][d.ID]
			slices.Sort(list)
			g := ImportGroup{CustomerCode: code, DocumentType: d, Count: len(list)}
			if len(list) > 0 {
				g.Highest = sequenceText(list[len(list)-1])
				g.Gaps = gaps(list)
			}
			groups = append(groups, g)
		}
		misaligned = append(misaligned, misalignedRanges(code, types, present[code])...)
	}
	return groups, misaligned
}

// gaps mengembalikan rentang nomor berjalan yang tidak ada, dari nomor
// pertama (001) sampai nomor tertinggi. list sudah urut dan tanpa ganda.
func gaps(list []int) []SequenceRange {
	var out []SequenceRange
	next := FirstSequence
	for _, v := range list {
		if v > next {
			out = append(out, SequenceRange{First: sequenceText(next), Last: sequenceText(v - 1)})
		}
		next = max(next, v+1)
	}
	return out
}

// misalignedRanges mengembalikan rentang nomor berjalan satu kode customer
// yang dimiliki sebagian jenis dokumen saja. Nomor berjalan berurutan dengan
// jenis yang sama digabung menjadi satu rentang.
func misalignedRanges(code string, types []DocumentType, present map[int][]string) []MisalignedRange {
	values := make([]int, 0, len(present))
	for v := range present {
		values = append(values, v)
	}
	slices.Sort(values)

	var out []MisalignedRange
	last := 0 // nomor berjalan terakhir di rentang terakhir out
	for _, v := range values {
		var docs []DocumentType
		for _, d := range types {
			if slices.Contains(present[v], d.ID) {
				docs = append(docs, d)
			}
		}
		if len(docs) == len(types) {
			continue
		}
		n := len(out)
		if n > 0 && last == v-1 && slices.Equal(typeIDs(out[n-1].Present), typeIDs(docs)) {
			out[n-1].Range.Last = sequenceText(v)
		} else {
			out = append(out, MisalignedRange{
				CustomerCode: code, Range: SequenceRange{First: sequenceText(v), Last: sequenceText(v)}, Present: docs,
			})
		}
		last = v
	}
	return out
}

// typeIDs mengembalikan ID setiap jenis dokumen.
func typeIDs(list []DocumentType) []string {
	ids := make([]string, 0, len(list))
	for _, d := range list {
		ids = append(ids, d.ID)
	}
	return ids
}

// sequenceText menulis nomor berjalan untuk laporan, misalnya 7 menjadi 007.
// Nilai di register selalu bisa ditulis; angka biasa hanya pengaman.
func sequenceText(v int) string {
	s, err := FormatSequence(v)
	if err != nil {
		return strconv.Itoa(v)
	}
	return s
}
