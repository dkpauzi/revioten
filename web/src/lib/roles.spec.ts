import { afterEach, describe, expect, it } from 'vitest';
import type { Role } from './api/gen/index.ts';
import { en } from './i18n/en.ts';
import { language } from './i18n/language.svelte.ts';
import { assignableRoles, grantLabel, roleLabel, scopeLabel } from './roles.ts';

const role = (id: string, extra: Partial<Role> = {}): Role => ({
	id,
	level: 'edit',
	name_en: id,
	name_id: id,
	active: true,
	builtin: true,
	group_scope: false,
	holders: 0,
	permissions: [],
	...extra
});

afterEach(() => {
	language.current = 'en';
});

describe('label peran', () => {
	// Sejak D-199 nama peran datang dari server dalam dua bahasa.
	it('memakai nama dari katalog server menurut bahasa', () => {
		const auditor = role('qa_auditor', { name_en: 'QA Auditor', name_id: 'Auditor QA', builtin: false });
		expect(roleLabel(auditor)).toBe('QA Auditor');
		language.current = 'id';
		expect(roleLabel('qa_auditor', [auditor])).toBe('Auditor QA');
	});

	it('memakai teks layar bila katalog belum dimuat, dan ID bila tidak dikenal', () => {
		expect(roleLabel('ppap_engineer')).toBe(en.rolePpapEngineer);
		expect(roleLabel('approver')).toBe('approver');
		expect(scopeLabel('own_project_group')).toBe(en.scopeOwnProjectGroup);
	});

	it('menyebut cakupan hanya bila peran dibatasi ke Project Group sendiri', () => {
		const catalog = [role('project_leader', { name_en: 'Project Leader', group_scope: true })];
		expect(grantLabel({ role: 'project_leader', scope: 'all' }, catalog)).toBe('Project Leader');
		expect(grantLabel({ role: 'project_leader', scope: 'own_project_group' }, catalog)).toBe(
			`Project Leader (${en.scopeOwnProjectGroup})`
		);
	});
});

describe('assignableRoles', () => {
	// Viewer dimiliki semua akun kerja tanpa diberikan, dan Admin hanya untuk
	// akun admin (D-72). Peran nonaktif tidak ditawarkan, kecuali masih dipegang.
	it('menyisakan peran aktif yang bisa diberikan ke akun kerja, dengan urutan tetap', () => {
		const catalog = [
			role('ppap_engineer'),
			role('project_leader'),
			role('author', { active: false }),
			role('viewer', { level: 'view' }),
			role('admin', { level: 'admin' })
		];
		expect(assignableRoles(catalog).map((r) => r.id)).toEqual(['ppap_engineer', 'project_leader']);
		expect(assignableRoles(catalog, [{ role: 'author', scope: 'all' }]).map((r) => r.id)).toEqual([
			'ppap_engineer',
			'project_leader',
			'author'
		]);
	});
});
