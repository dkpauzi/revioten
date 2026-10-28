// Teks layar untuk peran dan cakupannya. Peran adalah data di server (D-66),
// jadi ID yang belum punya teks tetap ditampilkan apa adanya.
// ASUMSI E-19: nama Inggris diambil dari brief (author, verifier, document
// control, viewer, admin, migration role) dan dari uraian peran di brief
// bagian 1 untuk Template maintainer dan PFMEA–CP checker.
import type { Role, RoleGrant } from './api/gen/index.ts';
import { t, type MessageKey } from './i18n/index.ts';

const roleKeys: Record<string, MessageKey> = {
	author: 'roleAuthor',
	template_maintainer: 'roleTemplateMaintainer',
	pfmea_cp_checker: 'rolePfmeaCpChecker',
	verifier: 'roleVerifier',
	document_control: 'roleDocumentControl',
	migration: 'roleMigration',
	viewer: 'roleViewer',
	admin: 'roleAdmin'
};

// roleLabel memberi nama tampilan sebuah peran.
export function roleLabel(roleId: string): string {
	const key = roleKeys[roleId];
	return key ? t(key) : roleId;
}

// scopeLabel memberi nama tampilan cakupan peran.
export function scopeLabel(scope: string): string {
	return scope === 'own_project_group' ? t('scopeOwnProjectGroup') : t('scopeAll');
}

// grantLabel memberi nama tampilan satu peran yang dipegang akun. Cakupan hanya
// disebut bila peran dibatasi ke Project Group sendiri; tanpa keterangan
// berarti semua customer.
export function grantLabel(grant: RoleGrant): string {
	if (grant.scope === 'own_project_group') {
		return t('roleWithScope', { role: roleLabel(grant.role), scope: scopeLabel(grant.scope) });
	}
	return roleLabel(grant.role);
}

// groupScopedRoles adalah peran yang boleh dibatasi ke Project Group sendiri
// (U3 bagian 2). Server tetap memeriksa aturan ini.
export const groupScopedRoles = ['author', 'verifier'];

// notAssignable adalah peran yang tidak pernah diberikan ke akun kerja: Viewer
// dimiliki semua akun tanpa diberikan (U3 bagian 1), dan Admin hanya untuk akun
// admin (D-72). Server tetap menolak keduanya (R-SEC-11).
const notAssignable = ['viewer', 'admin'];

// assignableRoles menyaring katalog peran menjadi peran yang bisa dipilih untuk
// akun kerja. Urutan katalog dari server dipertahankan.
export function assignableRoles(catalog: Role[]): Role[] {
	return catalog.filter((role) => !notAssignable.includes(role.id));
}
