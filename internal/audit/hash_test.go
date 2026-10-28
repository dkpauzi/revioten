package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

// exampleFields adalah satu entri contoh dengan semua field terisi.
func exampleFields() entryFields {
	return entryFields{
		PrevHash:       genesisHash,
		Seq:            "1",
		OccurredAt:     "2026-10-28T01:02:03.000004Z",
		ActorType:      "human",
		ActorAccountID: "01a180aa-a98b-7bf2-8e0d-b97a25ecc489",
		ActorPersonID:  "01a180aa-a98b-7bf2-8e0d-b97a25ecc48a",
		RequestID:      "UJI1",
		SourceIP:       "127.0.0.1",
		Action:         "customer.created",
		TargetType:     "customer",
		TargetID:       "01a180aa-a98b-7bf2-8e0d-b97a25ecc48b",
		Details:        `{"code":"AB","name":"Example Customer"}`,
	}
}

// TestCanonicalForm menetapkan bentuk baku isi entri yang di-hash. Bentuk ini
// bagian dari spesifikasi (D-33): sistem lain yang ingin memeriksa rantai
// harus menyusun teks yang sama persis. Setiap field ditulis sebagai
// <panjang byte>:<isi>, dengan urutan tetap, diawali hash entri sebelumnya.
func TestCanonicalForm(t *testing.T) {
	want := "64:0000000000000000000000000000000000000000000000000000000000000000," +
		"1:1," +
		"27:2026-10-28T01:02:03.000004Z," +
		"5:human," +
		"36:01a180aa-a98b-7bf2-8e0d-b97a25ecc489," +
		"36:01a180aa-a98b-7bf2-8e0d-b97a25ecc48a," +
		"4:UJI1," +
		"9:127.0.0.1," +
		"16:customer.created," +
		"8:customer," +
		"36:01a180aa-a98b-7bf2-8e0d-b97a25ecc48b," +
		`39:{"code":"AB","name":"Example Customer"},`
	f := exampleFields()
	if got := f.canonical(); got != want {
		t.Errorf("bentuk baku:\n dapat %s\ningin %s", got, want)
	}
	sum := sha256.Sum256([]byte(want))
	if got := f.hash(); got != hex.EncodeToString(sum[:]) {
		t.Errorf("hash = %s, ingin SHA-256 dari bentuk baku", got)
	}
}

// TestCanonicalFormSeparatesFields: memindah satu huruf dari satu field ke
// field sebelahnya mengubah hash. Tanpa panjang di depan setiap field, kedua
// isi di bawah bisa menghasilkan teks gabungan yang sama.
func TestCanonicalFormSeparatesFields(t *testing.T) {
	a, b := exampleFields(), exampleFields()
	a.TargetType, a.TargetID = "customer", "x"
	b.TargetType, b.TargetID = "custome", "rx"
	if a.hash() == b.hash() {
		t.Error("isi field berbeda memberi hash yang sama")
	}
}

// TestTimeText: waktu ditulis dalam UTC dengan tepat enam angka pecahan detik
// (mikrodetik), sama dengan ketelitian timestamptz PostgreSQL. Dengan begitu
// waktu yang dibaca ulang dari database memberi teks yang sama.
func TestTimeText(t *testing.T) {
	jakarta := time.FixedZone("WIB", 7*60*60)
	at := time.Date(2026, 10, 28, 8, 2, 3, 4567, jakarta)
	if got, want := timeText(at), "2026-10-28T01:02:03.000004Z"; got != want {
		t.Errorf("timeText = %s, ingin %s", got, want)
	}
}

// TestVerifyChain memeriksa rantai tanpa database: rantai utuh lolos, dan
// setiap jenis kerusakan dilaporkan dengan nomor entri pertama yang rusak.
func TestVerifyChain(t *testing.T) {
	chain := buildChain(5)

	if v := verifyChain(nil); !v.Valid || v.Entries != 0 {
		t.Errorf("rantai kosong: %+v, ingin utuh dengan 0 entri", v)
	}
	if v := verifyChain(chain); !v.Valid || v.Entries != 5 || v.LastSeq != 5 || v.LastHash != chain[4].Hash {
		t.Errorf("rantai utuh: %+v", v)
	}

	tests := []struct {
		name   string
		damage func([]entryFields) []entryFields
		seq    int64
		reason string
	}{
		{"isi entri diubah", func(c []entryFields) []entryFields {
			c[2].Details = `{"code":"XY"}`
			return c
		}, 3, ReasonHashMismatch},
		{"entri di tengah dihapus", func(c []entryFields) []entryFields {
			return append(c[:2], c[3:]...)
		}, 4, ReasonSequenceGap},
		// Pengubah menghitung ulang hash entri yang diubah, tapi rantai sesudahnya
		// masih menunjuk hash lama.
		{"isi diubah dan hash dihitung ulang", func(c []entryFields) []entryFields {
			c[2].Details = `{"code":"XY"}`
			c[2].Hash = c[2].hash()
			return c
		}, 4, ReasonLinkMismatch},
		{"entri pertama bukan nomor 1", func(c []entryFields) []entryFields {
			return c[1:]
		}, 2, ReasonSequenceGap},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := verifyChain(tt.damage(buildChain(5)))
			if v.Valid || v.BrokenSeq != tt.seq || v.Reason != tt.reason {
				t.Errorf("hasil %+v, ingin rusak di entri %d (%s)", v, tt.seq, tt.reason)
			}
		})
	}
}

// buildChain membuat n entri contoh yang saling terhubung dengan benar.
func buildChain(n int) []entryFields {
	chain := make([]entryFields, 0, n)
	prev := genesisHash
	for i := 1; i <= n; i++ {
		f := exampleFields()
		f.PrevHash = prev
		f.Seq = seqText(int64(i))
		f.Hash = f.hash()
		chain = append(chain, f)
		prev = f.Hash
	}
	return chain
}
