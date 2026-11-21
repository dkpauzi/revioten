// Package change berisi master untuk perubahan dokumen (M1.6, UC-05): jenis
// perubahan (R-TPL-10), sumber perubahan (R-CHG-06), dan kategori 4M
// (R-CHG-01). Semuanya diubah PPAP Engineer (hak E-11, D-209) dan tidak pernah dihapus. Entitas
// perubahan sendiri menyusul di Tahap 5. Kebijakan approval per jenis
// perubahan (rute yang dipakai) diatur lewat paket approval (M1.8, E-35, D-112).
package change

import "time"

// Key jenis perubahan (R-TPL-10). Namanya hanya tampilan.
const (
	KeyContent        = "content"
	KeyLayout         = "layout"
	KeyAdministrative = "administrative"
)

// Mekanisme sistem yang memakai sumber perubahan (R-CHG-01). Sumber dengan
// mekanisme selalu aktif (E-36, D-113).
const (
	MechanismYokoten  = "yokoten"
	MechanismTemplate = "template"
	// MechanismIntegration: usulan dari aplikasi lain lewat API integrasi (M9.4).
	MechanismIntegration = "integration"
	MechanismSQAM        = "sqam"
)

// ChangeType adalah satu jenis perubahan: isi, tata letak saja, atau administratif.
type ChangeType struct {
	ID   string
	Key  string
	Name string
	// Rute approval untuk jenis ini (R-TPL-10, M1.8). Kosong berarti rute
	// default customer (E-35, D-112). Diatur lewat paket approval.
	ApprovalRouteID   string
	ApprovalRouteName string
}

// ChangeSource adalah satu sumber perubahan, misalnya customer claim.
type ChangeSource struct {
	ID        string
	Name      string
	Mechanism string // kosong untuk pemicu biasa
	Active    bool
	CreatedAt time.Time // selalu UTC
}

// ChangeSourceUpdate adalah isian sumber perubahan yang bisa diubah. Active
// nil berarti status tetap seperti sebelumnya.
type ChangeSourceUpdate struct {
	Name   string
	Active *bool
}

// FourMCategory adalah satu kategori 4M: Man, Machine, Material, atau Method.
type FourMCategory struct {
	ID   string
	Key  string
	Name string
}

// maxNameLength adalah panjang nama paling banyak, dalam karakter.
// ASUMSI E-30: sama dengan master lain.
const maxNameLength = 200
