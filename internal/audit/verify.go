package audit

import "github.com/dkpauzi/revioten/internal/db"

// Alasan rantai dinyatakan rusak.
const (
	// ReasonSequenceGap: nomor urut tidak menyambung, berarti ada entri yang hilang.
	ReasonSequenceGap = "sequence_gap"
	// ReasonLinkMismatch: prev_hash tidak sama dengan hash entri sebelumnya.
	ReasonLinkMismatch = "link_mismatch"
	// ReasonHashMismatch: isi entri tidak cocok lagi dengan hash-nya, berarti isinya diubah.
	ReasonHashMismatch = "hash_mismatch"
)

// Verification adalah hasil pemeriksaan rantai.
type Verification struct {
	Valid   bool
	Entries int64 // jumlah entri yang diperiksa
	// LastSeq dan LastHash adalah ujung rantai. Bila dicatat di luar database,
	// penghapusan entri terakhir juga bisa ketahuan (usulan R-SEC-03).
	LastSeq  int64
	LastHash string
	// BrokenSeq dan Reason diisi bila rantai rusak: entri pertama yang tidak cocok.
	BrokenSeq int64
	Reason    string
}

// verifyChain memeriksa entri yang sudah urut menurut seq: nomor urut
// menyambung dari 1, setiap prev_hash sama dengan hash entri sebelumnya, dan
// setiap hash sama dengan hash yang dihitung ulang dari isinya.
func verifyChain(chain []entryFields) Verification {
	v := Verification{Valid: true}
	prevHash := genesisHash
	for i, f := range chain {
		seq := int64(i) + 1
		switch {
		case f.Seq != seqText(seq):
			return broken(v, f, ReasonSequenceGap)
		case f.PrevHash != prevHash:
			return broken(v, f, ReasonLinkMismatch)
		case f.Hash != f.hash():
			return broken(v, f, ReasonHashMismatch)
		}
		v.Entries, v.LastSeq, v.LastHash = seq, seq, f.Hash
		prevHash = f.Hash
	}
	return v
}

// broken menandai rantai rusak di entri f.
func broken(v Verification, f entryFields, reason string) Verification {
	v.Valid = false
	v.Reason = reason
	// Seq entri yang rusak selalu angka yang ditulis Record atau oleh database.
	v.BrokenSeq = parseSeq(f.Seq)
	v.Entries++
	return v
}

// fieldsOf mengubah satu baris tabel audit_log menjadi teks yang di-hash.
func fieldsOf(r db.AuditLog) entryFields {
	return entryFields{
		PrevHash: r.PrevHash, Seq: seqText(r.Seq), OccurredAt: timeText(r.OccurredAt),
		ActorType: r.ActorType, ActorAccountID: text(r.ActorAccountID), ActorPersonID: text(r.ActorPersonID),
		RequestID: text(r.RequestID), SourceIP: text(r.SourceIp),
		Action: r.Action, TargetType: text(r.TargetType), TargetID: text(r.TargetID),
		Details: r.Details, Hash: r.Hash,
	}
}
