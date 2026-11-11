// Bantuan tabel Action Priority (form set langkah A, D-134). Aturannya sama
// dengan internal/template/ap.go; server tetap sumber kebenaran, dan layar
// memakai fungsi ini hanya untuk menampilkan tabel dan kalkulator coba.
import type { ApRule } from './api/gen/index.ts';

// actionPriority mencari AP untuk satu kombinasi S, O, D, atau null bila
// tidak ada aturan yang berlaku.
export function actionPriority(rules: ApRule[], s: number, o: number, d: number): ApRule['priority'] | null {
	const rule = rules.find(
		(r) => s >= r.s_min && s <= r.s_max && o >= r.o_min && o <= r.o_max && d >= r.d_min && d <= r.d_max
	);
	return rule?.priority ?? null;
}

// ApMatrix adalah tabel AP untuk layar: satu baris per pasangan rentang S dan
// O, satu kolom per rentang D.
export type ApMatrix = {
	detection: string[];
	rows: { severity: string; occurrence: string; cells: string[] }[];
};

// bands mengumpulkan batas rentang satu skala lalu memotong 1 sampai 10
// menjadi rentang yang tidak terbelah oleh aturan mana pun, dari yang tertinggi.
function bands(starts: number[]): [number, number][] {
	const cuts = [...new Set([1, ...starts])].sort((a, b) => a - b);
	return cuts.map((lo, i) => [lo, (cuts[i + 1] ?? 11) - 1] as [number, number]).reverse();
}

const label = ([lo, hi]: [number, number]) => (lo === hi ? String(lo) : `${lo}–${hi}`);

// apMatrix menyusun aturan menjadi tabel seperti di handbook.
export function apMatrix(rules: ApRule[]): ApMatrix {
	const sBands = bands(rules.map((r) => r.s_min));
	const oBands = bands(rules.map((r) => r.o_min));
	const dBands = bands(rules.map((r) => r.d_min));
	const rows = sBands.flatMap((s) =>
		oBands.map((o) => ({
			severity: label(s),
			occurrence: label(o),
			// Setiap rentang tidak terbelah, jadi batas bawahnya mewakili seluruh rentang.
			cells: dBands.map((d) => actionPriority(rules, s[0], o[0], d[0]) ?? '')
		}))
	);
	return { detection: dBands.map(label), rows };
}
