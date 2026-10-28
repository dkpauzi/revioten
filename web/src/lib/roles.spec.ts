import { describe, expect, it } from 'vitest';
import type { Role } from './api/gen/index.ts';
import { en } from './i18n/en.ts';
import { assignableRoles, grantLabel, roleLabel, scopeLabel } from './roles.ts';

describe('label peran', () => {
	it('memakai teks layar untuk peran yang dikenal', () => {
		expect(roleLabel('author')).toBe(en.roleAuthor);
		expect(roleLabel('pfmea_cp_checker')).toBe(en.rolePfmeaCpChecker);
		expect(scopeLabel('own_project_group')).toBe(en.scopeOwnProjectGroup);
	});

	// Peran adalah data (D-66): peran baru dari server tetap tampil, walau
	// belum punya teks layar.
	it('memakai ID untuk peran yang belum dikenal', () => {
		expect(roleLabel('approver')).toBe('approver');
	});

	it('menyebut cakupan hanya bila peran dibatasi ke Project Group sendiri', () => {
		expect(grantLabel({ role: 'verifier', scope: 'all' })).toBe(en.roleVerifier);
		expect(grantLabel({ role: 'author', scope: 'own_project_group' })).toBe(
			`${en.roleAuthor} (${en.scopeOwnProjectGroup})`
		);
	});
});

describe('assignableRoles', () => {
	const role = (id: string, level: Role['level']): Role => ({ id, level, permissions: [] });

	// Viewer dimiliki semua akun tanpa diberikan, dan Admin hanya untuk akun
	// admin (D-72). Keduanya tidak ditawarkan di pilihan peran akun kerja.
	it('menyisakan peran yang bisa diberikan ke akun kerja, dengan urutan tetap', () => {
		const catalog = [
			role('author', 'edit'),
			role('verifier', 'verify'),
			role('viewer', 'view'),
			role('admin', 'admin')
		];
		expect(assignableRoles(catalog).map((r) => r.id)).toEqual(['author', 'verifier']);
	});
});
