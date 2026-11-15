import type { MessageKey } from './i18n/index.ts';

// Menu atas berkelompok (D-127, bagian dari UX-05). Isinya hanya halaman yang
// sudah dibangun; menu fitur yang belum ada (misalnya Yokoten) tidak dipasang.
//
// Menyembunyikan menu bukan pengaman: server tetap memeriksa setiap operasi
// (M1.2). Menu hanya tidak menawarkan halaman yang pasti ditolak.

export type MenuItem = {
	href: string;
	label: MessageKey;
};

export type MenuGroup = {
	id: 'home' | 'documents' | 'templates' | 'master' | 'admin';
	label: MessageKey;
	items: MenuItem[];
};

// Satu entri menu sebelum disaring menurut hak. Tanpa permission berarti
// boleh dilihat semua akun yang login (hak dasar Lihat).
type Entry = MenuItem & { permission?: string };

const groups: { id: MenuGroup['id']; label: MessageKey; entries: Entry[] }[] = [
	{ id: 'home', label: 'navHome', entries: [{ href: '/', label: 'navHome' }] },
	{
		id: 'documents',
		label: 'navGroupDocuments',
		entries: [
			// Dokumen draft (M4.1): butuh L-4, jadi viewer dan admin tidak melihatnya.
			{ href: '/documents', label: 'navDocuments', permission: 'L-4' },
			{ href: '/changes', label: 'navChanges', permission: 'L-4' },
			{ href: '/yokoten', label: 'navYokoten', permission: 'L-4' },
			// Dasbor ukuran (M7.7): semua akun (L-2).
			{ href: '/dashboard', label: 'navDashboard' },
			{ href: '/numbers', label: 'navNumbers' },
			// Impor nomor lama hanya untuk document control (E-8, M2.5).
			{ href: '/numbers/import', label: 'numberImportLink', permission: 'E-8' }
		]
	},
	{
		id: 'templates',
		label: 'navGroupTemplates',
		entries: [
			{ href: '/format-templates', label: 'navFormatTemplates' },
			{ href: '/process-templates', label: 'navProcessTemplates' },
			// Form set (langkah D, D-133): semua akun melihat; admin mengubah, penyetuju form menerbitkan.
			{ href: '/form-sets', label: 'navFormSets' },
			{ href: '/processes', label: 'navProcesses' },
			{ href: '/referenced-documents', label: 'navReferencedDocuments' },
			{ href: '/print-layouts', label: 'navPrintLayouts' }
		]
	},
	{
		id: 'master',
		label: 'navGroupMaster',
		entries: [
			{ href: '/customers', label: 'navCustomers' },
			{ href: '/models', label: 'navModels' },
			{ href: '/document-types', label: 'navDocumentTypes' },
			{ href: '/change-types', label: 'navChangeTypes' },
			{ href: '/positions', label: 'navPositions' },
			{ href: '/approval-routes', label: 'navApprovalRoutes' }
		]
	},
	{
		id: 'admin',
		label: 'navGroupAdmin',
		entries: [
			{ href: '/admin/accounts', label: 'navAccounts', permission: 'A-1' },
			// Isi peran diatur admin (M8.1, D-66).
			{ href: '/admin/roles', label: 'navRoles', permission: 'A-1' },
			// Audit log hanya lewat akun admin (A-4, D-75).
			{ href: '/admin/audit', label: 'navAuditLog', permission: 'A-4' },
			// Pengaturan sistem untuk pemegang A-3 (UC-07).
			{ href: '/admin/settings', label: 'navSettings', permission: 'A-3' }
		]
	}
];

// buildMenu menyusun menu untuk akun yang login. has adalah pemeriksa hak,
// biasanya can() dari session.svelte.ts. Kelompok yang isinya habis tersaring
// tidak ikut ditampilkan.
export function buildMenu(has: (permission: string) => boolean): MenuGroup[] {
	return groups
		.map((group) => ({
			id: group.id,
			label: group.label,
			items: group.entries
				.filter((entry) => !entry.permission || has(entry.permission))
				.map(({ href, label }) => ({ href, label }))
		}))
		.filter((group) => group.items.length > 0);
}

// activeGroup mengembalikan id kelompok yang memuat halaman di pathname, atau
// null bila halaman itu tidak ada di menu. Alamat yang paling panjang dan cocok
// yang menang, supaya /numbers/import tidak tertukar dengan halaman lain.
export function activeGroup(menu: MenuGroup[], pathname: string): MenuGroup['id'] | null {
	let best: { id: MenuGroup['id']; length: number } | null = null;
	for (const group of menu) {
		for (const item of group.items) {
			const matches =
				item.href === '/'
					? pathname === '/'
					: pathname === item.href || pathname.startsWith(item.href + '/');
			if (matches && (!best || item.href.length > best.length)) {
				best = { id: group.id, length: item.href.length };
			}
		}
	}
	return best?.id ?? null;
}
