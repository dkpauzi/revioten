import { describe, expect, it } from 'vitest';
import { activeGroup, buildMenu } from './menu.ts';

// has membuat pemeriksa hak dari daftar kode hak, seperti can() di session.svelte.ts.
function has(...permissions: string[]): (permission: string) => boolean {
	return (permission) => permissions.includes(permission);
}

// hrefs meringkas menu menjadi id kelompok beserta alamat isinya, supaya mudah dibandingkan.
function hrefs(permissions: string[]): Record<string, string[]> {
	return Object.fromEntries(
		buildMenu(has(...permissions)).map((group) => [group.id, group.items.map((item) => item.href)])
	);
}

describe('buildMenu', () => {
	it('menyusun kelompok menu D-127 untuk akun kerja tanpa hak admin', () => {
		expect(hrefs(['L-1', 'E-1'])).toEqual({
			home: ['/'],
			documents: ['/dashboard', '/documents/released', '/periodic-reviews', '/numbers'],
			templates: ['/format-templates', '/process-templates', '/form-sets', '/processes', '/referenced-documents', '/print-layouts'],
			master: [
				'/customers',
				'/models',
				'/document-types',
				'/change-types',
				'/positions',
				'/approval-routes',
				'/document-settings'
			]
		});
	});

	it('menampilkan Documents hanya untuk akun yang melihat draft (L-4, M4.1)', () => {
		expect(hrefs(['L-1', 'L-4']).documents).toEqual(['/documents', '/changes', '/yokoten', '/signatures', '/dashboard', '/documents/released', '/periodic-reviews', '/numbers']);
	});

	it('menampilkan impor nomor lama hanya untuk document control (E-8)', () => {
		expect(hrefs(['L-1', 'E-8']).documents).toEqual(['/dashboard', '/documents/released', '/periodic-reviews', '/numbers', '/numbers/import']);
	});

	it('menampilkan laporan kinerja hanya untuk pemegang L-5 (D-197)', () => {
		expect(hrefs(['L-1', 'L-5']).documents).toContain('/performance');
		expect(hrefs(['L-1', 'L-4']).documents).not.toContain('/performance');
	});

	it('menampilkan kelompok Admin sesuai hak admin yang dipegang', () => {
		expect(hrefs(['A-1', 'A-3', 'A-4']).admin).toEqual([
			'/admin/health',
			'/admin/accounts',
			'/admin/accounts/import',
			'/admin/roles',
			'/admin/service-accounts',
			'/admin/audit',
			'/admin/settings',
			'/admin/notifications'
		]);
		expect(hrefs(['A-4']).admin).toEqual(['/admin/audit']);
	});

	// Kelompok tanpa isi tidak ditampilkan, supaya layar tidak menawarkan menu kosong.
	it('tidak menampilkan kelompok Admin tanpa hak admin', () => {
		expect(buildMenu(has('L-1')).some((group) => group.id === 'admin')).toBe(false);
	});
});

describe('activeGroup', () => {
	const menu = buildMenu(has('E-8', 'A-1', 'A-3', 'A-4'));

	it('menandai kelompok yang memuat halaman aktif', () => {
		expect(activeGroup(menu, '/')).toBe('home');
		expect(activeGroup(menu, '/customers')).toBe('master');
		expect(activeGroup(menu, '/numbers/import')).toBe('documents');
		expect(activeGroup(menu, '/admin/audit')).toBe('admin');
	});

	// /numbers/import tidak boleh ikut menandai kelompok lain hanya karena diawali /numbers.
	it('memilih alamat menu yang paling cocok', () => {
		expect(activeGroup(menu, '/numbers')).toBe('documents');
		expect(activeGroup(menu, '/processes')).toBe('templates');
		expect(activeGroup(menu, '/process-templates')).toBe('templates');
	});

	it('tidak menandai apa pun untuk halaman di luar menu', () => {
		expect(activeGroup(menu, '/about')).toBeNull();
		expect(activeGroup(menu, '/account/password')).toBeNull();
	});
});
