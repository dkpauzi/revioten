import { describe, expect, it } from 'vitest';
import type { AccountInfo } from './api/gen/index.ts';
import { accountStatus, peopleOf } from './accounts.ts';
import { formatDateTime } from './format.ts';
import { en } from './i18n/en.ts';

// account membuat akun contoh; field yang diberikan menimpa nilai bawaan.
function account(fields: Partial<AccountInfo> = {}): AccountInfo {
	return {
		id: 'akun-1',
		kind: 'work',
		login: 'penyusun.contoh',
		person: { id: 'orang-1', name: 'Penyusun Contoh' },
		roles: [],
		must_change_password: false,
		disabled: false,
		created_at: '2026-10-28T01:00:00Z',
		...fields
	};
}

describe('accountStatus', () => {
	const now = new Date('2026-10-28T08:00:00Z');

	it('akun tanpa catatan apa pun aktif', () => {
		expect(accountStatus(account(), now)).toEqual([en.accountStatusActive]);
	});

	it('menyebut semua keadaan yang berlaku', () => {
		const lockedUntil = '2026-10-28T08:15:00Z';
		expect(
			accountStatus(
				account({ disabled: true, locked_until: lockedUntil, must_change_password: true }),
				now
			)
		).toEqual([
			en.accountStatusDisabled,
			en.accountStatusLocked.replace('{time}', formatDateTime(lockedUntil)),
			en.accountStatusMustChange
		]);
	});

	// Kunci yang waktunya sudah lewat tidak berlaku lagi (M1.1).
	it('kunci yang sudah lewat tidak disebut', () => {
		expect(accountStatus(account({ locked_until: '2026-10-28T07:59:00Z' }), now)).toEqual([
			en.accountStatusActive
		]);
	});
});

describe('peopleOf', () => {
	it('mengambil setiap orang satu kali, diurutkan menurut nama', () => {
		const verifier = { id: 'orang-2', name: 'Verifikator Contoh' };
		const adminHolder = { id: 'orang-1', name: 'Admin Contoh', employee_number: 'K-001' };
		const accounts = [
			account({ id: 'a', person: verifier }),
			account({ id: 'b', person: adminHolder, kind: 'work' }),
			account({ id: 'c', person: adminHolder, kind: 'admin' })
		];
		expect(peopleOf(accounts)).toEqual([adminHolder, verifier]);
	});
});
