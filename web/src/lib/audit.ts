// Teks layar untuk audit log (M1.3). Kode peristiwa dan kunci rincian datang
// dari server; yang belum punya teks layar ditampilkan apa adanya.
import type { AuditEntry, AuditVerification } from './api/gen/index.ts';
import { t, type MessageKey } from './i18n/index.ts';

const actionKeys: Record<string, MessageKey> = {
	'account.login': 'auditActionLogin',
	'account.login_failed': 'auditActionLoginFailed',
	'account.logout': 'auditActionLogout',
	'account.password_changed': 'auditActionPasswordChanged',
	'account.password_change_failed': 'auditActionPasswordChangeFailed',
	'account.created': 'auditActionAccountCreated',
	'account.roles_changed': 'auditActionRolesChanged',
	'account.password_reset': 'auditActionPasswordReset',
	'account.disabled': 'auditActionAccountDisabled',
	'account.enabled': 'auditActionAccountEnabled',
	'customer.created': 'auditActionCustomerCreated',
	'customer.updated': 'auditActionCustomerUpdated',
	'customer.code_changed': 'auditActionCustomerCodeChanged',
	'project_group.created': 'auditActionProjectGroupCreated',
	'project_group.updated': 'auditActionProjectGroupUpdated',
	'model_group.created': 'auditActionModelGroupCreated',
	'model_group.updated': 'auditActionModelGroupUpdated',
	'model.created': 'auditActionModelCreated',
	'model.updated': 'auditActionModelUpdated',
	'part_number.created': 'auditActionPartNumberCreated',
	'part_number.updated': 'auditActionPartNumberUpdated'
};

const detailKeys: Record<string, MessageKey> = {
	failed_logins: 'auditDetailFailedLogins',
	locked_until: 'auditDetailLockedUntil',
	disabled: 'auditDetailDisabled',
	before: 'auditDetailBefore',
	after: 'auditDetailAfter',
	login: 'auditDetailLogin',
	kind: 'auditDetailKind',
	roles: 'auditDetailRoles',
	via: 'auditDetailVia',
	code: 'auditDetailCode',
	name: 'auditDetailName',
	active: 'auditDetailActive',
	customer: 'auditDetailCustomer',
	project_group: 'auditDetailProjectGroup',
	model_group: 'auditDetailModelGroup',
	model: 'auditDetailModel',
	number: 'auditDetailNumber'
};

const reasonKeys: Record<string, MessageKey> = {
	sequence_gap: 'auditReasonSequenceGap',
	link_mismatch: 'auditReasonLinkMismatch',
	hash_mismatch: 'auditReasonHashMismatch'
};

// actionLabel memberi nama tampilan sebuah peristiwa.
export function actionLabel(action: string): string {
	const key = actionKeys[action];
	return key ? t(key) : action;
}

// actorLabel menyebut siapa pelakunya: akun beserta orang pemegangnya (D-74),
// perintah di komputer server, atau orang yang belum login.
export function actorLabel(entry: AuditEntry): string {
	switch (entry.actor.type) {
		case 'human':
			return t('auditActorHuman', {
				name: entry.actor.person_name ?? '',
				login: entry.actor.login ?? ''
			});
		case 'system':
			return t('auditActorSystem');
		default:
			return t('auditActorAnonymous');
	}
}

// targetLabel memberi nama tampilan sasaran entri, atau ID-nya bila tidak ada.
export function targetLabel(entry: AuditEntry): string {
	return entry.target?.label ?? entry.target?.id ?? '';
}

// detailsText menulis rincian sebagai "Nama: nilai; ...", urut menurut kunci.
export function detailsText(details: Record<string, string>): string {
	return Object.keys(details)
		.sort()
		.map((key) => `${detailKeys[key] ? t(detailKeys[key]) : key}: ${details[key]}`)
		.join('; ');
}

// verificationText menjelaskan hasil pemeriksaan rantai hash.
export function verificationText(v: AuditVerification): string {
	if (!v.valid) {
		const key = v.reason ? reasonKeys[v.reason] : undefined;
		return t('auditChainBroken', {
			seq: String(v.broken_seq ?? ''),
			reason: key ? t(key) : (v.reason ?? '')
		});
	}
	if (v.entries === 0) {
		return t('auditChainEmpty');
	}
	return t('auditChainValid', { entries: String(v.entries), hash: v.last_hash ?? '' });
}
