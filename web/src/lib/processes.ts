// Alias proses ditulis di kotak teks, satu alias per baris, karena nama proses
// bisa memuat koma (M1.5). Server tetap merapikan dan memeriksa setiap alias.

// aliasLines memecah isi kotak teks menjadi daftar alias. Spasi di tepi
// dibuang dan baris kosong dilewati.
export function aliasLines(text: string): string[] {
	return text
		.split('\n')
		.map((line) => line.trim())
		.filter((line) => line !== '');
}

// aliasText menulis daftar alias untuk kotak teks, satu alias per baris.
export function aliasText(aliases: string[]): string {
	return aliases.join('\n');
}
