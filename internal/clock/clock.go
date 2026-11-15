// Package clock memberi zona waktu pabrik untuk tanggal dan periode laporan:
// jatuh tempo review berkala (D-196) dan minggu laporan kinerja (D-197).
package clock

import "time"

// Zone adalah zona waktu pabrik. ASUMSI E-131: WIB (UTC+7), tanpa pengaturan.
// WIB tidak punya waktu musim panas, jadi zona tetap cukup dan tidak butuh
// basis data zona waktu di Windows.
var Zone = time.FixedZone("WIB", 7*60*60)

// Date memberi tanggal t di zona pabrik, pukul 00.00.
func Date(t time.Time) time.Time {
	t = t.In(Zone)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, Zone)
}

// Days memberi selisih hari kalender dari a ke b (b - a).
func Days(a, b time.Time) int {
	a, b = Date(a), Date(b)
	// Dibulatkan, karena zona tetap selalu 24 jam per hari.
	return int(b.Sub(a).Hours() / 24)
}
