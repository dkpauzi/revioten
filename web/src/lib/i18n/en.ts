// Teks layar dalam bahasa Inggris (Q-14, keputusan pemilik 28 Oktober 2026).
//
// Setiap teks punya kunci tetap. Kode layar tidak menulis teks langsung, tapi
// memanggil t('kunci'). Untuk bahasa lain cukup dibuat file baru dengan kunci
// yang sama, misalnya id.ts untuk bahasa Indonesia (lihat index.ts).
//
// ASUMSI E-19: istilah Inggris diambil dari brief bagian 3 dan 4 (dan dari form
// serta WI), sampai glosarium U1 punya kolom istilah layar berbahasa Inggris.
export const en = {
	appTitle: 'PFD, PFMEA, and CP System',
	navHome: 'Home',
	navAbout: 'About',
	homeTitle: 'Home',
	healthChecking: 'Checking the server…',
	healthOk: 'Server status: OK',
	healthError: 'Server status: not reachable',
	healthRetry: 'Check again',
	aboutTitle: 'About',
	aboutText:
		'An internal web system to prepare, revise, number, print, and record the release of PFD, PFMEA, and Control Plan documents. Approval stays on paper with wet signatures.'
};

// MessageKey adalah semua kunci teks yang tersedia.
export type MessageKey = keyof typeof en;

// Messages adalah bentuk satu katalog bahasa: setiap kunci wajib punya teks.
export type Messages = Record<MessageKey, string>;
