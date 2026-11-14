import { describe, expect, it } from 'vitest';
import type { AuditEntry } from './api/gen/index.ts';
import { actionLabel, actorLabel, detailsText, targetLabel, verificationText } from './audit.ts';
import { en } from './i18n/en.ts';

// entry membuat entri contoh; field yang diberikan menimpa nilai bawaan.
function entry(fields: Partial<AuditEntry> = {}): AuditEntry {
	return {
		seq: 1,
		occurred_at: '2026-10-28T01:02:03Z',
		actor: { type: 'human', login: 'admin.contoh', person_name: 'Admin Contoh' },
		action: 'account.created',
		details: {},
		prev_hash: '0'.repeat(64),
		hash: 'a'.repeat(64),
		...fields
	};
}

describe('teks audit log', () => {
	it('memberi nama peristiwa, atau kodenya bila belum dikenal', () => {
		expect(actionLabel('account.created')).toBe(en.auditActionAccountCreated);
		expect(actionLabel('customer.created')).toBe(en.auditActionCustomerCreated);
		expect(actionLabel('portal.published')).toBe('portal.published');
		expect(actionLabel('customer.code_changed')).toBe(en.auditActionCustomerCodeChanged);
		expect(actionLabel('part_number.updated')).toBe(en.auditActionPartNumberUpdated);
		expect(actionLabel('process.created')).toBe(en.auditActionProcessCreated);
		expect(actionLabel('process.updated')).toBe(en.auditActionProcessUpdated);
		expect(actionLabel('number_settings.updated')).toBe(en.auditActionNumberSettingsUpdated);
		expect(actionLabel('document_type.created')).toBe(en.auditActionDocumentTypeCreated);
		expect(actionLabel('special_code.updated')).toBe(en.auditActionSpecialCodeUpdated);
		expect(actionLabel('change_source.created')).toBe(en.auditActionChangeSourceCreated);
		expect(actionLabel('four_m_category.updated')).toBe(en.auditActionFourMCategoryUpdated);
		expect(actionLabel('position.holder_added')).toBe(en.auditActionPositionHolderAdded);
		expect(actionLabel('approval_route.updated')).toBe(en.auditActionApprovalRouteUpdated);
		expect(actionLabel('customer.default_route_changed')).toBe(en.auditActionCustomerRouteChanged);
		expect(actionLabel('system_settings.updated')).toBe(en.auditActionSystemSettingsUpdated);
	});

	// Aktor manusia ditampilkan bersama orang pemegang akunnya (D-74).
	it('menyebut akun dan orangnya, atau jenis aktor lain', () => {
		expect(actorLabel(entry())).toBe(
			en.auditActorHuman.replace('{name}', 'Admin Contoh').replace('{login}', 'admin.contoh')
		);
		expect(actorLabel(entry({ actor: { type: 'system' } }))).toBe(en.auditActorSystem);
		expect(actorLabel(entry({ actor: { type: 'anonymous' } }))).toBe(en.auditActorAnonymous);
	});

	it('memakai nama tampilan sasaran, atau ID-nya', () => {
		expect(targetLabel(entry({ target: { type: 'account', id: 'akun-1', label: 'penyusun.contoh' } }))).toBe(
			'penyusun.contoh'
		);
		expect(targetLabel(entry({ target: { type: 'customer', id: 'customer-1' } }))).toBe('customer-1');
		expect(targetLabel(entry())).toBe('');
	});

	it('menulis rincian sebagai pasangan nama dan nilai, urut menurut kunci', () => {
		expect(detailsText({ login: 'penyusun.contoh', kind: 'work' })).toBe(
			`${en.auditDetailKind}: work; ${en.auditDetailLogin}: penyusun.contoh`
		);
		expect(detailsText({ via: 'create-admin', baru: 'x' })).toBe(
			`baru: x; ${en.auditDetailVia}: create-admin`
		);
		expect(detailsText({})).toBe('');
		expect(detailsText({ project_group: 'GR1 → GR2', model_group: '- → Family' })).toBe(
			`${en.auditDetailModelGroup}: - → Family; ${en.auditDetailProjectGroup}: GR1 → GR2`
		);
		expect(detailsText({ name: 'IQC', aliases: 'Incoming QC' })).toBe(
			`${en.auditDetailAliases}: Incoming QC; ${en.auditDetailName}: IQC`
		);
		expect(detailsText({ factory_code: 'S → T', letter: 'H → P' })).toBe(
			`${en.auditDetailFactoryCode}: S → T; ${en.auditDetailLetter}: H → P`
		);
		expect(detailsText({ person: 'Orang Contoh', default_route: '- → Up to GM' })).toBe(
			`${en.auditDetailDefaultRoute}: - → Up to GM; ${en.auditDetailPerson}: Orang Contoh`
		);
	});

	it('menjelaskan hasil pemeriksaan rantai', () => {
		expect(verificationText({ valid: true, entries: 12, last_seq: 12, last_hash: 'abc' })).toBe(
			en.auditChainValid.replace('{entries}', '12').replace('{hash}', 'abc')
		);
		expect(verificationText({ valid: true, entries: 0 })).toBe(en.auditChainEmpty);
		expect(
			verificationText({ valid: false, entries: 3, broken_seq: 3, reason: 'hash_mismatch' })
		).toBe(en.auditChainBroken.replace('{seq}', '3').replace('{reason}', en.auditReasonHashMismatch));
	});
});
