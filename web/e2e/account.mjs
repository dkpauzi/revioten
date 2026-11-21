// Akun admin untuk test E2E. Dibuat ulang di database revioten_e2e setiap kali
// test berjalan (lihat server.mjs), jadi bukan akun sungguhan. Nama fiktif
// (CLAUDE.md bagian 7).
export const admin = {
	name: 'Admin E2E',
	login: 'admin.e2e',
	password: 'kuda terbang di atas sawah'
};

// Akun PPAP Engineer untuk master data (D-209): customer, model, jenis dokumen,
// jabatan, rute, format template, dan form set. Dibuat admin saat pertama kali
// dipakai (lihat logInAsMaster di helpers.ts).
export const master = {
	name: 'PPAP Engineer E2E',
	login: 'ppap.e2e',
	password: 'perahu layar di danau biru'
};
