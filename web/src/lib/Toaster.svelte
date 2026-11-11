<script lang="ts">
	// Notifikasi pop-up di pojok kanan atas (masukan pemilik, 11 November 2026).
	//
	// Setiap halaman sudah menulis hasil aksinya sebagai pesan di dalam halaman:
	// role="status" untuk berhasil dan role="alert" untuk gagal. Pesan itu bisa
	// tidak terlihat bila layar sedang digulir ke bawah. Komponen ini mengamati
	// isi halaman dan menyalin setiap pesan baru ke pop-up, jadi halaman tidak
	// perlu diubah satu per satu, dan pesan aslinya tetap di tempatnya.
	//
	// Teks pop-up ditulis lewat atribut data-text dan CSS, bukan sebagai teks
	// di DOM, dan wadahnya aria-hidden. Dengan begitu pembaca layar tidak
	// membacakan pesan yang sama dua kali.
	import { onMount } from 'svelte';
	import { t } from '#lib/i18n/index.ts';
	import { language } from '#lib/i18n/language.svelte.ts';

	type Toast = { id: number; text: string; kind: 'status' | 'alert' };

	// Lama pop-up tampil (milidetik). Pesan gagal lebih lama, dan pesan panjang
	// seperti kata sandi sementara paling lama.
	const duration = { status: 6000, alert: 10000, long: 20000 };

	let toasts = $state<Toast[]>([]);
	let nextId = 1;
	// Teks terakhir setiap elemen pesan, supaya pesan yang sama tidak diulang.
	const seen = new WeakMap<Element, string>();

	function dismiss(id: number) {
		toasts = toasts.filter((toast) => toast.id !== id);
	}

	function show(text: string, kind: Toast['kind'], long: boolean) {
		const id = nextId++;
		// Paling banyak tiga pop-up sekaligus; yang paling lama dibuang.
		toasts = [...toasts, { id, text, kind }].slice(-3);
		setTimeout(() => dismiss(id), long ? duration.long : duration[kind]);
	}

	// Saat bahasa diganti, semua pesan di halaman berubah teksnya sekaligus.
	// Itu bukan pesan baru, jadi selama sesaat perubahan hanya dicatat, tidak
	// dijadikan pop-up.
	let quietUntil = 0;
	let lastLanguage = language.current;
	$effect(() => {
		// Hanya saat bahasa benar-benar berganti, bukan saat halaman baru dibuka.
		if (language.current !== lastLanguage) {
			lastLanguage = language.current;
			quietUntil = Date.now() + 500;
		}
	});

	// scan mencari pesan baru atau pesan yang teksnya berubah.
	function scan(root: Element) {
		for (const el of root.querySelectorAll('[role="status"], [role="alert"]')) {
			const text = (el.textContent ?? '').replace(/\s+/g, ' ').trim();
			if (text === '' || seen.get(el) === text) {
				continue;
			}
			seen.set(el, text);
			if (Date.now() < quietUntil) {
				continue;
			}
			show(text, el.getAttribute('role') === 'alert' ? 'alert' : 'status', el.hasAttribute('data-toast-long'));
		}
	}

	onMount(() => {
		const root = document.querySelector('main');
		if (root === null) {
			return;
		}
		const observer = new MutationObserver(() => scan(root));
		observer.observe(root, { childList: true, subtree: true, characterData: true });
		return () => observer.disconnect();
	});
</script>

<div class="toaster" aria-hidden="true">
	{#each toasts as toast (toast.id)}
		<!-- Klik menutup pop-up lebih cepat; pesan aslinya tetap ada di halaman. -->
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div class="toast" data-kind={toast.kind} data-text={toast.text} title={t('toastDismiss')} onclick={() => dismiss(toast.id)}></div>
	{/each}
</div>

<style>
	.toaster {
		position: fixed;
		top: 76px;
		right: 20px;
		z-index: 50;
		display: flex;
		flex-direction: column;
		gap: 10px;
		width: min(380px, calc(100vw - 32px));
		pointer-events: none;
	}

	.toast {
		pointer-events: auto;
		padding: 12px 16px;
		border: 1px solid var(--border);
		border-left: 4px solid var(--success);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--ink);
		font-size: 14px;
		line-height: 1.45;
		overflow-wrap: anywhere;
		box-shadow: 0 8px 24px rgba(15, 23, 42, 0.16);
		cursor: pointer;
		animation: toast-in 160ms ease-out;
	}

	.toast::after {
		content: attr(data-text);
	}

	.toast[data-kind='alert'] {
		border-left-color: var(--danger);
	}

	@keyframes toast-in {
		from {
			opacity: 0;
			transform: translateY(-6px);
		}
	}

	/* HP: selebar layar, di bawah bilah atas. */
	@media (max-width: 639px) {
		.toaster {
			right: 16px;
			left: 16px;
			width: auto;
		}
	}
</style>
