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
		expect(actionLabel('document.released')).toBe('document.released');
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
