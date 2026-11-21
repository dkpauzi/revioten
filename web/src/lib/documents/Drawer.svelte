<script lang="ts">
	// Laci samping workbook (D-145, C1): satu panel dibuka dari toolbar, di
	// kanan layar, tanpa menutup lembar dokumen. Escape atau tombol Close
	// menutupnya. Fokus pindah ke laci saat dibuka, supaya pengguna keyboard
	// langsung berada di panelnya.
	import type { Snippet } from 'svelte';
	import { t } from '#lib/i18n/index.ts';

	let { title, onclose, children }: { title: string; onclose: () => void; children: Snippet } = $props();

	let element: HTMLElement;

	$effect(() => {
		element.focus();
	});

	function keydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			onclose();
		}
	}
</script>

<svelte:window onkeydown={keydown} />

<aside class="drawer" tabindex="-1" bind:this={element}>
	<div class="drawer-head">
		<p class="drawer-title">{title}</p>
		<button type="button" class="drawer-close" onclick={onclose}>{t('drawerClose')}</button>
	</div>
	<div class="drawer-body">
		{@render children()}
	</div>
</aside>

<style>
	/* Laci di kanan lembar, ikut tata letak halaman (tidak menutupi toolbar
	   atau bilah atas), dan menempel saat halaman digulir. Di layar di bawah
	   1024px laci tampil di atas lembar, selebar halaman. */
	.drawer {
		position: sticky;
		top: 16px;
		flex: 0 0 min(480px, 40%);
		max-height: calc(100vh - 96px);
		display: flex;
		flex-direction: column;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		box-shadow: 0 8px 24px rgba(16, 24, 40, 0.08);
		outline: none;
	}

	.drawer-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 12px 18px;
		border-bottom: 1px solid var(--border);
	}

	.drawer-title {
		margin: 0;
		font-weight: 600;
		color: var(--text-3);
	}

	.drawer-body {
		flex: 1;
		overflow-y: auto;
		padding: 4px 18px 20px;
	}

	/* Panel di dalam laci tidak tampil sebagai kartu kedua. */
	.drawer-body :global(section) {
		border: none;
		box-shadow: none;
		padding: 0;
		margin: 0;
		background: none;
	}

	@media (max-width: 1023px) {
		.drawer {
			position: static;
			order: -1;
			flex: 1 1 100%;
			max-height: none;
		}
	}
</style>
