import { describe, expect, it } from 'vitest';
import { en } from './i18n/en.ts';
import { canStartNewVersion, isDraft, latestVersion, versionOptionText } from './versions.ts';

const published = { id: 'v-1', version: 1, status: 'published' as const, published_at: '2026-11-10T09:30:00Z' };
const draft = { id: 'v-2', version: 2, status: 'draft' as const };

describe('latestVersion', () => {
	it('mengambil versi dengan nomor terbesar', () => {
		expect(latestVersion([published, draft])).toBe(draft);
		expect(latestVersion([draft, published])).toBe(draft);
	});

	it('mengembalikan null bila tidak ada versi', () => {
		expect(latestVersion([])).toBeNull();
	});
});

describe('isDraft', () => {
	it('hanya versi draft yang bisa diubah', () => {
		expect(isDraft(draft)).toBe(true);
		expect(isDraft(published)).toBe(false);
		expect(isDraft(null)).toBe(false);
	});
});

describe('canStartNewVersion', () => {
	it('boleh bila versi terakhir sudah terbit', () => {
		expect(canStartNewVersion([published])).toBe(true);
	});

	it('tidak boleh selama masih ada draft, atau belum ada versi', () => {
		expect(canStartNewVersion([published, draft])).toBe(false);
		expect(canStartNewVersion([draft])).toBe(false);
		expect(canStartNewVersion([])).toBe(false);
	});
});

describe('versionOptionText', () => {
	it('menulis nomor dan status versi', () => {
		expect(versionOptionText(published)).toBe(
			en.versionOption.replace('{version}', '1').replace('{status}', en.versionStatusPublished)
		);
		expect(versionOptionText(draft)).toBe(en.versionOption.replace('{version}', '2').replace('{status}', en.versionStatusDraft));
	});
});
