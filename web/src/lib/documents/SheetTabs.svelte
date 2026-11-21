<script lang="ts">
	// Tab sheet di bawah layar (D-145, C1), seperti tab sheet Excel: Cover,
	// dokumen satu set (PFD, PFMEA, CP), lalu Check bila set punya PFMEA dan CP.
	// Tab adalah tautan biasa; halaman dokumen memuat dokumennya dari ?id=.
	import type { DocumentSetMember } from '#lib/api/gen/index.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		set,
		documentId,
		current,
		viewPath = '/documents/view',
		released = false
	}: {
		set: DocumentSetMember[];
		// documentId: dokumen yang sedang dibuka, untuk tab Cover dan Check.
		documentId: string;
		// current: 'cover', 'check', atau ID dokumen yang sheet-nya sedang tampil.
		current: string;
		viewPath?: string;
		released?: boolean;
	} = $props();

	let hasCheck = $derived(set.some((m) => m.document_type.key === 'pfmea') && set.some((m) => m.document_type.key === 'cp'));
</script>

<nav class="sheet-tabs" aria-label={t('documentSheets')}>
	{#if !released}
		<a href="{viewPath}?id={documentId}&sheet=cover" data-sveltekit-noscroll aria-current={current === 'cover' ? 'page' : undefined}>{t('sheetCover')}</a>
	{/if}
	{#each set as m (m.id)}
		<a href="{viewPath}?id={m.id}" data-sveltekit-noscroll aria-current={current === m.id ? 'page' : undefined}>{m.document_type.name}</a>
	{/each}
	{#if hasCheck}
		<a href="/documents/check?id={documentId}{released ? '&released=1' : ''}" aria-current={current === 'check' ? 'page' : undefined}>{t('checkTab')}</a>
	{/if}
</nav>

<style>
	/* Menempel di bawah layar seperti tab sheet Excel. Halaman memberi ruang di
	   bawah (lihat :global(main:has(.sheet-tabs))) supaya isi terakhir tidak tertutup. */
	.sheet-tabs {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 30;
		display: flex;
		gap: 4px;
		padding: 0 16px;
		background: var(--surface-2, #eef0f3);
		border-top: 1px solid var(--border);
		overflow-x: auto;
	}

	.sheet-tabs a {
		padding: 10px 20px;
		border: 1px solid transparent;
		border-top: none;
		border-radius: 0 0 var(--radius) var(--radius);
		color: var(--text-2);
		white-space: nowrap;
	}

	.sheet-tabs a[aria-current='page'] {
		background: var(--surface);
		border-color: var(--border);
		box-shadow: inset 0 3px 0 var(--accent);
		color: var(--ink);
		font-weight: 600;
		margin-top: -1px;
	}

	:global(main:has(.sheet-tabs)) {
		padding-bottom: 96px;
	}
</style>
