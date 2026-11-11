// copyText menyalin teks ke clipboard dan melaporkan berhasil atau tidak.
// Browser hanya mengizinkan clipboard di halaman aman (https atau localhost),
// dan bisa menolak, misalnya bila jendelanya sedang tidak aktif. Karena itu
// pemanggil selalu menyediakan jalan lain: teksnya tetap tampil di layar.
export async function copyText(text: string): Promise<boolean> {
	try {
		if (!navigator.clipboard) {
			return false;
		}
		await navigator.clipboard.writeText(text);
		return true;
	} catch {
		return false;
	}
}
