package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// genesisHash adalah prev_hash entri pertama rantai.
var genesisHash = strings.Repeat("0", 64)

// entryFields adalah isi satu entri sebagai teks, dalam urutan yang di-hash.
// Nilai NULL di database ditulis sebagai teks kosong. Hash adalah hash yang
// tersimpan, dan tidak ikut di-hash.
type entryFields struct {
	PrevHash       string
	Seq            string
	OccurredAt     string
	ActorType      string
	ActorAccountID string
	ActorPersonID  string
	RequestID      string
	SourceIP       string
	Action         string
	TargetType     string
	TargetID       string
	Details        string
	Hash           string
}

// canonical menyusun bentuk baku entri: setiap field ditulis sebagai
// <panjang byte>:<isi>, (bentuk netstring), dengan urutan tetap dan diawali
// hash entri sebelumnya. Panjang di depan setiap field membuat batas antarfield
// selalu jelas, jadi dua isi yang berbeda tidak pernah memberi teks yang sama.
func (f entryFields) canonical() string {
	var b strings.Builder
	for _, v := range []string{
		f.PrevHash, f.Seq, f.OccurredAt, f.ActorType, f.ActorAccountID, f.ActorPersonID,
		f.RequestID, f.SourceIP, f.Action, f.TargetType, f.TargetID, f.Details,
	} {
		b.WriteString(strconv.Itoa(len(v)))
		b.WriteByte(':')
		b.WriteString(v)
		b.WriteByte(',')
	}
	return b.String()
}

// hash adalah SHA-256 dari bentuk baku, ditulis sebagai 64 huruf heksadesimal kecil.
func (f entryFields) hash() string {
	sum := sha256.Sum256([]byte(f.canonical()))
	return hex.EncodeToString(sum[:])
}

// timeText menulis waktu dalam UTC dengan tepat enam angka pecahan detik.
func timeText(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000Z")
}

// seqText menulis nomor urut sebagai angka desimal.
func seqText(seq int64) string {
	return strconv.FormatInt(seq, 10)
}

// parseSeq membaca kembali nomor urut dari teks. Teks yang bukan angka memberi 0.
func parseSeq(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
