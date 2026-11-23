// Bantuan untuk halaman akun (M1.2): teks status akun dan daftar orang.
import type { AccountInfo, PersonInfo } from './api/gen/index.ts';
import { formatDateTime } from './format.ts';
import { t } from './i18n/index.ts';

// accountStatus memberi teks semua keadaan akun yang berlaku pada waktu now.
// Akun tanpa keadaan khusus disebut aktif.
export function accountStatus(account: AccountInfo, now: Date = new Date()): string[] {
	const status: string[] = [];
	if (account.disabled) {
		status.push(t('accountStatusDisabled'));
	}
	// Kunci yang waktunya sudah lewat tidak berlaku lagi (M1.1).
	if (account.locked_until && new Date(account.locked_until) > now) {
		status.push(t('accountStatusLocked', { time: formatDateTime(account.locked_until) }));
	}
	if (account.must_change_password) {
		status.push(t('accountStatusMustChange'));
	}
	// Password sementara akun hasil import kedaluwarsa 7 hari (D-224).
	if (account.temporary_password_expires_at) {
		const at = account.temporary_password_expires_at;
		status.push(new Date(at) <= now ? t('accountStatusTemporaryExpired') : t('accountStatusTemporaryUntil', { time: formatDateTime(at) }));
	}
	return status.length > 0 ? status : [t('accountStatusActive')];
}

// peopleOf mengambil setiap orang pemegang akun satu kali, diurutkan menurut
// nama. Dipakai untuk memilih orang yang sudah ada, misalnya saat membuat akun
// admin untuk orang yang sudah punya akun kerja (D-72).
export function peopleOf(accounts: AccountInfo[]): PersonInfo[] {
	const byId = new Map<string, PersonInfo>();
	for (const account of accounts) {
		byId.set(account.person.id, account.person);
	}
	return [...byId.values()].sort((a, b) => a.name.localeCompare(b.name));
}
