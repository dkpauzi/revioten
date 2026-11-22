// Kamera tur (rencana T, docs/rencana/import-tur-dan-logo.md): sebelum setiap
// klik, isian, centang, atau pilihan, layar digulir sampai elemen itu di
// tengah, lalu elemen diberi sorotan sebentar. Dengan begitu penonton tur
// tahu bagian mana yang sedang dikerjakan.
//
// Caranya: method Locator Playwright (click, fill, dan seterusnya) dibungkus
// sekali untuk seluruh tur, jadi langkah-langkah tur tidak perlu diubah. Bila
// elemen belum siap disorot (misalnya masih dimuat), sorotan dilewati dan
// aksinya tetap berjalan seperti biasa. Aplikasinya sendiri tidak berubah;
// sorotan hanya ditambahkan tur, seperti pita keterangan.
import type { Locator, Page } from '@playwright/test';

// Method yang mendapat sorotan.
const actions = ['click', 'dblclick', 'fill', 'check', 'uncheck', 'selectOption', 'setInputFiles', 'press', 'pressSequentially'] as const;

const overlayId = 'tour-spotlight';

// show menggulir elemen ke tengah layar dan memasang sorotan di sekelilingnya.
// Cincin sorotan mengikuti posisi elemen setiap frame, jadi tetap tepat
// selama gulir halus berjalan.
async function show(locator: Locator, waitMs: number): Promise<void> {
	const page = locator.page();
	try {
		await locator.evaluate(
			(el, id) => {
				el.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'smooth' });
				document.getElementById(id)?.remove();
				const ring = document.createElement('div');
				ring.id = id;
				ring.style.cssText =
					'position:fixed;z-index:2147483646;pointer-events:none;border:3px solid #f59e0b;border-radius:8px;' +
					'box-shadow:0 0 0 4px rgba(245,158,11,.35),0 0 18px rgba(245,158,11,.6)';
				document.body.appendChild(ring);
				const pad = 4;
				const follow = () => {
					if (!ring.isConnected) {
						return;
					}
					const r = el.getBoundingClientRect();
					ring.style.left = `${r.left - pad}px`;
					ring.style.top = `${r.top - pad}px`;
					ring.style.width = `${r.width + pad * 2}px`;
					ring.style.height = `${r.height + pad * 2}px`;
					requestAnimationFrame(follow);
				};
				follow();
			},
			overlayId,
			{ timeout: 3000 }
		);
		await page.waitForTimeout(waitMs);
	} catch {
		// Elemen belum bisa disorot: lewati, aksinya tetap dikerjakan.
	}
}

// hide melepas sorotan sesudah aksi.
async function hide(page: Page): Promise<void> {
	try {
		await page.evaluate((id) => document.getElementById(id)?.remove(), overlayId);
	} catch {
		// Halaman sedang berpindah; sorotan ikut hilang bersama halaman lama.
	}
}

// installSpotlight memasang kamera tur. waitMs adalah lama sorotan sebelum
// aksi; 0 berarti mati (TOUR_SPEED=fast), supaya tur cepat tetap cepat.
export function installSpotlight(page: Page, waitMs: number): void {
	if (waitMs <= 0) {
		return;
	}
	const proto = Object.getPrototypeOf(page.locator('body')) as Record<string, unknown> & { __tourSpotlight?: boolean };
	if (proto.__tourSpotlight) {
		return;
	}
	proto.__tourSpotlight = true;
	for (const name of actions) {
		const original = proto[name] as (...args: unknown[]) => Promise<unknown>;
		proto[name] = async function (this: Locator, ...args: unknown[]) {
			await show(this, waitMs);
			try {
				return await original.apply(this, args);
			} finally {
				await hide(this.page());
			}
		};
	}
}
