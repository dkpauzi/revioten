// Kerangka workbook dokumen (D-145, C1): laci panel dan satu langkah
// berikutnya di kepala halaman. Logikanya di sini, terpisah dari halaman,
// supaya bisa dites dengan Vitest.
import type { MessageKey } from '#lib/i18n/index.ts';

// DrawerId adalah panel yang dibuka dari toolbar di laci samping.
export type DrawerId =
	| 'compose'
	| 'copy'
	| 'compare'
	| 'templates'
	| 'order'
	| 'paste'
	| 'revisions'
	| 'print'
	| 'signature'
	| 'submission'
	| 'migration'
	| 'excel'
	| 'link';

// drawerTitle adalah judul laci, sama dengan judul panelnya, supaya tombol
// toolbar, laci, dan isi panel memakai nama yang sama.
export const drawerTitle: Record<DrawerId, MessageKey> = {
	compose: 'composeOpen',
	copy: 'copyOpen',
	compare: 'compareOpen',
	templates: 'templateLinksTitle',
	order: 'orderOpen',
	paste: 'pasteOpen',
	revisions: 'revisions',
	print: 'printTitle',
	signature: 'sigTitle',
	submission: 'subTitle',
	migration: 'migrationTitle',
	excel: 'excelCompareTitle',
	link: 'linkTitle'
};

// StepInput adalah keadaan dokumen dan hak akun yang menentukan langkah berikutnya.
export type StepInput = {
	status: 'draft' | 'released';
	rows: number;
	hasRoute: boolean;
	signing?: 'waiting' | 'completed';
	// migrated dan verified: dokumen lama (Tahap 7) dan apakah sudah dicocokkan dengan hardcopy.
	migrated?: boolean;
	verified?: boolean;
	canEdit: boolean; // E-2 pada draft yang tidak terkunci
	canPropose: boolean; // E-3
	canRecord: boolean; // E-7
};

// Step adalah satu langkah berikutnya: kalimatnya, dan laci yang dibuka atau alamat tujuannya.
export type Step = { text: MessageKey; open?: DrawerId; href?: string };

// nextStep memilih satu langkah berikutnya ("satu aksi utama", D-145).
// Urutannya mengikuti alur dokumen: isi, cocokkan (dokumen lama), tanda
// tangan, catat rilis. Akun yang tidak bisa melakukan langkah itu tidak
// mendapat langkah.
export function nextStep(d: StepInput): Step | null {
	if (d.status === 'released') {
		return d.canPropose ? { text: 'nextChange', href: '/changes' } : null;
	}
	if (d.signing === 'completed') {
		return d.canRecord ? { text: 'nextRecord', open: 'revisions' } : null;
	}
	if (d.signing === 'waiting') {
		return d.canEdit || d.canRecord ? { text: 'nextWaiting', open: 'signature' } : null;
	}
	if (!d.canEdit) {
		return null;
	}
	if (d.migrated && !d.verified) {
		return { text: 'nextVerify', open: 'migration' };
	}
	if (d.rows === 0) {
		return { text: 'nextCompose', open: 'compose' };
	}
	return d.hasRoute ? { text: 'nextSign', open: 'signature' } : { text: 'nextPrint', open: 'print' };
}
