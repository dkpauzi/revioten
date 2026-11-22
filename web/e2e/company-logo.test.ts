// Test E2E bagian C (E-144): admin mengganti nama perusahaan dan logo cetakan
// di Admin → Settings, lalu kembali ke logo bawaan.
import { expect, test } from '@playwright/test';
import { deflateSync } from 'node:zlib';
import { en } from '../src/lib/i18n/en.ts';
import { logIn, openFromMenu } from './helpers.ts';

// tinyPNG membuat PNG 2 x 2 piksel abu-abu tanpa pustaka gambar.
function tinyPNG(): Buffer {
	const crcTable = Array.from({ length: 256 }, (_, n) => {
		let c = n;
		for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
		return c >>> 0;
	});
	const crc = (buf: Buffer) => {
		let c = 0xffffffff;
		for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8);
		return (c ^ 0xffffffff) >>> 0;
	};
	const chunk = (type: string, data: Buffer) => {
		const len = Buffer.alloc(4);
		len.writeUInt32BE(data.length);
		const body = Buffer.concat([Buffer.from(type), data]);
		const sum = Buffer.alloc(4);
		sum.writeUInt32BE(crc(body));
		return Buffer.concat([len, body, sum]);
	};
	const ihdr = Buffer.alloc(13);
	ihdr.writeUInt32BE(2, 0);
	ihdr.writeUInt32BE(2, 4);
	ihdr[8] = 8; // kedalaman bit
	ihdr[9] = 0; // abu-abu
	const raw = Buffer.from([0, 128, 128, 0, 128, 128]); // dua baris, filter 0
	return Buffer.concat([
		Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
		chunk('IHDR', ihdr),
		chunk('IDAT', deflateSync(raw)),
		chunk('IEND', Buffer.alloc(0))
	]);
}

test('admin mengganti nama perusahaan dan logo cetakan, lalu kembali ke logo bawaan', async ({ page }) => {
	await logIn(page);
	await openFromMenu(page, en.navGroupAdmin, en.navSettings);
	const company = page.locator('section', { has: page.getByRole('heading', { name: en.companyTitle }) });
	await expect(company.getByText(en.companyLogoDefault)).toBeVisible();
	await expect(company.getByRole('img', { name: en.companyLogoAlt })).toBeVisible();

	await company.getByLabel(en.companyName, { exact: true }).fill('PT. SIIX EMS INDONESIA');
	await company.getByRole('button', { name: en.companyNameSave }).click();
	await expect(company.getByRole('status')).toHaveText(en.companyNameSaved);

	// Bukan PNG ditolak server.
	await company.getByLabel(en.companyLogoFile).setInputFiles({ name: 'logo.png', mimeType: 'image/png', buffer: Buffer.from('bukan png') });
	await company.getByRole('button', { name: en.companyLogoUpload }).click();
	await expect(company.getByRole('alert')).toBeVisible();

	await company.getByLabel(en.companyLogoFile).setInputFiles({ name: 'logo.png', mimeType: 'image/png', buffer: tinyPNG() });
	await company.getByRole('button', { name: en.companyLogoUpload }).click();
	await expect(company.getByRole('status')).toHaveText(en.companyLogoUploaded);
	await expect(company.getByText(/^Uploaded logo, SHA-256 /)).toBeVisible();

	await company.getByRole('button', { name: en.companyLogoReset }).click();
	await expect(company.getByRole('status')).toHaveText(en.companyLogoResetDone);
	await expect(company.getByText(en.companyLogoDefault)).toBeVisible();
});
