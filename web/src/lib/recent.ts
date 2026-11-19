// Daftar "Opened recently" di Home (D-202). Disimpan di browser orang itu
// (localStorage), karena hanya kemudahan pribadi, bukan data yang dibagi.
// Bila penyimpanan ditolak browser (mode privat), daftarnya kosong.

export type RecentDocument = {
	id: string;
	number: string;
	title: string; // model, misalnya "K-125"
	status: 'editing' | 'released';
	revision: number;
	view: 'draft' | 'released'; // halaman yang dibuka
};

const key = 'revioten.recent';
const max = 5;

export function readRecent(): RecentDocument[] {
	try {
		const list = JSON.parse(localStorage.getItem(key) ?? '[]');
		return Array.isArray(list) ? list.slice(0, max) : [];
	} catch {
		return [];
	}
}

// rememberRecent menaruh dokumen di urutan teratas, tanpa duplikat.
export function rememberRecent(doc: RecentDocument): RecentDocument[] {
	const list = [doc, ...readRecent().filter((d) => d.id !== doc.id)].slice(0, max);
	try {
		localStorage.setItem(key, JSON.stringify(list));
	} catch {
		// Penyimpanan ditolak: daftar tetap berjalan tanpa diingat.
	}
	return list;
}
