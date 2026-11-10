// Bantuan versi template (M3.4, R-TPL-05), dipakai layar format template dan
// process template. Versi berjalan dari draft ke terbit; versi terbit tidak
// bisa diubah, dan versi baru hanya dibuat setelah versi terakhir terbit.
import { t } from './i18n/index.ts';

// VersionInfo adalah bagian versi yang dibutuhkan di sini. Ringkasan versi
// dan versi lengkap dari kedua jenis template sama-sama memenuhinya.
export type VersionInfo = {
	id: string;
	version: number;
	status: 'draft' | 'published';
	published_at?: string;
};

// latestVersion mengambil versi dengan nomor terbesar, atau null bila kosong.
export function latestVersion<V extends VersionInfo>(versions: V[]): V | null {
	return versions.reduce<V | null>((latest, v) => (latest === null || v.version > latest.version ? v : latest), null);
}

// isDraft melaporkan apakah versi itu masih bisa diubah.
export function isDraft(version: Pick<VersionInfo, 'status'> | null): boolean {
	return version?.status === 'draft';
}

// canStartNewVersion: versi baru hanya dibuat bila versi terakhir sudah
// terbit, jadi satu template paling banyak punya satu draft. Server
// memeriksa hal yang sama.
export function canStartNewVersion(versions: VersionInfo[]): boolean {
	return latestVersion(versions)?.status === 'published';
}

// versionOptionText menulis satu versi untuk daftar pilihan, misalnya
// "Version 2 · Draft".
export function versionOptionText(v: Pick<VersionInfo, 'version' | 'status'>): string {
	return t('versionOption', {
		version: String(v.version),
		status: v.status === 'draft' ? t('versionStatusDraft') : t('versionStatusPublished')
	});
}
