// Package access adalah titik pemeriksaan izin (M1.2, R-SEC-02, D-35): satu
// fungsi Authorize yang dipanggil setiap operasi domain dengan aktor, aksi, dan
// resource (R-API-03). Peran adalah kumpulan hak yang tersimpan sebagai data
// (D-66, R-SEC-10), dengan isi awal dari matriks U3-peta-pengguna (E-01).
package access

import (
	"context"
	"errors"
)

// ActorType adalah jenis aktor (R-API-03). Saat ini hanya manusia yang bisa
// menjalankan operasi; service account dan customer user menyusul di M8.1 dan
// Tahap 10.
type ActorType string

const (
	// ActorHuman adalah orang yang login dengan akunnya.
	ActorHuman ActorType = "human"
	// ActorSystem dan ActorAnonymous hanya dipakai untuk mencatat pelaku di
	// audit log (M1.3), dan selalu ditolak Authorize karena tidak punya akun.
	// E-27 (D-106): tambahan pada jenis aktor R-API-03.
	//   - ActorSystem: perintah di komputer server, misalnya create-admin.
	//   - ActorAnonymous: belum login, misalnya percobaan login yang gagal.
	// ActorService adalah aplikasi lain yang memanggil API integrasi dengan
	// token service account (Tahap 9). Izin aktor ini diperiksa paket
	// integration (scope dan customer token), bukan lewat peran.
	ActorService   ActorType = "service"
	ActorSystem    ActorType = "system"
	ActorAnonymous ActorType = "anonymous"
)

// Jenis akun (E-02, D-72). Akun admin hanya memegang peran Admin, dan akun
// kerja tidak pernah memegang peran Admin (R-SEC-13).
const (
	AccountWork  = "work"
	AccountAdmin = "admin"
)

// Actor adalah siapa yang memanggil sebuah operasi.
type Actor struct {
	Type        ActorType
	AccountID   string
	AccountKind string // AccountWork atau AccountAdmin
	Login       string
	// PersonID dipakai aturan pemisahan tugas, yang membandingkan orang,
	// bukan akun (E-02).
	PersonID   string
	PersonName string
	// MustChangePassword true bila password direset admin (E-08).
	MustChangePassword bool
}

// Permission adalah kode hak di katalog U3. Kode tetap; nama tampilannya ada
// di katalog teks layar.
type Permission string

const (
	ViewReleased      Permission = "L-1"  // lihat dokumen rilis dan halaman verifikasi
	ViewRegister      Permission = "L-2"  // lihat register nomor, riwayat, diff, diagram, dasbor
	ExportReleased    Permission = "L-3"  // ekspor Excel dan PDF dokumen rilis
	ViewDrafts        Permission = "L-4"  // lihat draft dan permintaan perubahan yang berjalan
	RegisterNumbers   Permission = "E-1"  // daftarkan nomor dokumen
	EditDrafts        Permission = "E-2"  // buat, rakit, dan edit dokumen draft
	ProposeChanges    Permission = "E-3"  // ajukan perubahan
	WorkRevisions     Permission = "E-4"  // draft revisi, yokoten, perubahan surut SQAM
	PrintForSignature Permission = "E-5"  // cetak untuk tanda tangan
	ManageTemplates   Permission = "E-6"  // master proses dan process template
	RecordRelease     Permission = "E-7"  // catat rilis
	ManageNumbers     Permission = "E-8"  // ubah status nomor, impor nomor lama
	ControlledCopies  Permission = "E-9"  // salinan terkendali
	MigrateDocuments  Permission = "E-10" // migrasikan dokumen lama
	VerifyChanges     Permission = "V-1"  // verifikasi permintaan perubahan
	CheckPFMEACP      Permission = "V-2"  // tanda dicek PFMEA dan CP
	ApproveForms      Permission = "V-3"  // setujui (terbitkan) form set dan tabel AP (D-139)
	ManageUsers       Permission = "A-1"  // kelola pengguna dan hak akses
	ManageMasterData  Permission = "A-2"  // kelola master data
	ManageSettings    Permission = "A-3"  // kelola pengaturan sistem
	ViewAuditLog      Permission = "A-4"  // lihat audit log

	// ManageOwnAccount bukan hak di katalog: setiap aktor boleh mengurus
	// akunnya sendiri (lihat sesi, logout, ganti password). Tetap lewat
	// Authorize, supaya semua operasi melewati satu titik pemeriksaan.
	// E-25 (D-104): aturan tetap yang tidak bisa dicabut admin, bukan hak di U3.
	ManageOwnAccount Permission = "own-account"
)

// ID peran (tetap). Isinya ada di tabel role_permissions.
const (
	RoleAuthor             = "author"              // Penyusun
	RoleTemplateMaintainer = "template_maintainer" // Pengelola template
	RolePFMEACPChecker     = "pfmea_cp_checker"    // Pemeriksa PFMEA–CP
	RoleVerifier           = "verifier"            // Verifikator
	RoleFormApprover       = "form_approver"       // Penyetuju form (D-139)
	RoleDocumentControl    = "document_control"    // Dokumen kontrol
	RoleMigration          = "migration"           // Petugas migrasi
	RoleViewer             = "viewer"              // Viewer, dimiliki semua akun
	RoleAdmin              = "admin"               // Admin, hanya akun admin
)

// Cakupan sebuah peran (D-71, E-05).
const (
	ScopeAll             = "all"
	ScopeOwnProjectGroup = "own_project_group"
)

// Resource adalah apa yang disentuh operasi.
type Resource struct {
	// CustomerID diisi bila operasi menyangkut satu customer, supaya cakupan
	// Project Group bisa diperiksa (D-71, E-04).
	CustomerID string
	// AccountID diisi untuk ManageOwnAccount.
	AccountID string
}

// ErrForbidden berarti aktor tidak punya hak untuk operasi itu. API menjawabnya 403.
var ErrForbidden = errors.New("tidak punya hak untuk operasi ini")

// Authorizer adalah fungsi pemeriksaan izin. *Checker memenuhinya; paket lain
// bergantung pada interface ini supaya test bisa memakai versi palsu.
type Authorizer interface {
	Authorize(ctx context.Context, actor Actor, perm Permission, res Resource) error
}
