// Teks layar untuk peran dan cakupannya. Sejak D-199 peran punya nama sendiri
// dalam dua bahasa di server, karena admin bisa menambah dan mengganti nama
// peran. Teks di katalog layar hanya cadangan bila katalog peran belum dimuat.
// E-19 (D-98): nama Inggris mengikuti kolom istilah layar di glosarium U1.
import type { Role, RoleGrant } from './api/gen/index.ts';
import { t, type MessageKey } from './i18n/index.ts';
import { language } from './i18n/language.svelte.ts';

const roleKeys: Record<string, MessageKey> = {
	ppap_engineer: 'rolePpapEngineer',
	ppap_engineer_supervisor: 'rolePpapEngineerSupervisor',
	project_leader: 'roleProjectLeader',
	manager: 'roleManager',
	author: 'roleAuthor',
	template_maintainer: 'roleTemplateMaintainer',
	pfmea_cp_checker: 'rolePfmeaCpChecker',
	verifier: 'roleVerifier',
	form_approver: 'roleFormApprover',
	document_control: 'roleDocumentControl',
	migration: 'roleMigration',
	viewer: 'roleViewer',
	admin: 'roleAdmin'
};

// roleLabel memberi nama tampilan sebuah peran dalam bahasa yang aktif. role
// boleh berupa peran dari katalog server (dengan nama), atau hanya ID-nya;
// ID yang tidak dikenal ditampilkan apa adanya.
export function roleLabel(role: Role | string, catalog: Role[] = []): string {
	const found = typeof role === 'string' ? catalog.find((r) => r.id === role) : role;
	if (found) {
		return language.current === 'id' ? found.name_id : found.name_en;
	}
	const id = typeof role === 'string' ? role : '';
	const key = roleKeys[id];
	return key ? t(key) : id;
}

// scopeLabel memberi nama tampilan cakupan peran.
export function scopeLabel(scope: string): string {
	return scope === 'own_project_group' ? t('scopeOwnProjectGroup') : t('scopeAll');
}

// grantLabel memberi nama tampilan satu peran yang dipegang akun. Cakupan hanya
// disebut bila peran dibatasi ke Project Group sendiri; tanpa keterangan
// berarti semua customer.
export function grantLabel(grant: RoleGrant, catalog: Role[] = []): string {
	const name = roleLabel(grant.role, catalog);
	if (grant.scope === 'own_project_group') {
		return t('roleWithScope', { role: name, scope: scopeLabel(grant.scope) });
	}
	return name;
}

// notAssignable adalah peran yang tidak pernah diberikan ke akun kerja: Viewer
// dimiliki semua akun kerja tanpa diberikan (U3 bagian 1), dan Admin hanya
// untuk akun admin (D-72). Server tetap menolak keduanya (R-SEC-11).
const notAssignable = ['viewer', 'admin'];

// assignableRoles menyaring katalog peran menjadi peran yang bisa dipilih untuk
// akun kerja: peran aktif, ditambah peran nonaktif yang masih dipegang akun
// itu (supaya tetap tampil dan bisa dilepas). Urutan katalog dipertahankan.
export function assignableRoles(catalog: Role[], held: RoleGrant[] = []): Role[] {
	return catalog.filter(
		(role) => !notAssignable.includes(role.id) && (role.active || held.some((g) => g.role === role.id))
	);
}
