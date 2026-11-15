<script lang="ts">
	// Isi revisi yang berlaku satu dokumen (D-195), dibaca dari snapshot rilis.
	// Hanya dibaca: tidak ada tombol edit, dan draft revisi berikutnya tidak
	// pernah tampil. Tab Check membuka PFMEA dan CP terbit berdampingan.
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { getFormatVersion, getReleasedDocument } from '#lib/api/client.ts';
	import type { Document, FormatVersion } from '#lib/api/gen/index.ts';
	import Workbook from '#lib/documents/Workbook.svelte';
	import ReviewBanner from '#lib/ReviewBanner.svelte';
	import { rowsOfSection } from '#lib/documents.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let doc = $state<Document | null>(null);
	let format = $state<FormatVersion | null>(null);
	let loadError = $state('');
	let sectionKey = $state('');
	let section = $derived(doc?.sections.find((s) => s.key === sectionKey) ?? doc?.sections[0] ?? null);
	let shown = $derived(doc ? { ...doc, rows: rowsOfSection(doc.rows, section?.id ?? '') } : null);
	let hasPair = $derived(doc !== null && doc.set.some((m) => m.document_type.key === 'pfmea') && doc.set.some((m) => m.document_type.key === 'cp'));

	let currentId = $derived(page.url.searchParams.get('id') ?? '');
	$effect(() => {
		const id = currentId;
		untrack(() => load(id));
	});

	async function load(id: string) {
		doc = null;
		format = null;
		loadError = '';
		sectionKey = '';
		const result = await getReleasedDocument(id);
		if (!result.ok) {
			loadError = result.error.status === 404 ? t('releasedNotFound') : failureText(result.error);
			return;
		}
		const f = await getFormatVersion(result.data.format_version_id);
		if (!f.ok) {
			loadError = failureText(f.error);
			return;
		}
		format = f.data;
		doc = result.data;
	}

	function headerText(value: unknown): string {
		return value === undefined || value === null ? '' : String(value);
	}
</script>

<p><a href="/documents/released">{t('releasedBack')}</a></p>
{#if loadError}
	<h1>{t('releasedTitle')}</h1>
	<p class="error" role="alert">{loadError}</p>
{:else if !doc || !format || !shown}
	<p>{t('formatLoading')}</p>
{:else}
	<h1 class="mono">{doc.number}</h1>
	<ReviewBanner documentId={doc.id} />
	<p class="hint">{t('releasedViewHint')}</p>
	<dl class="facts">
		<dt>{t('documentCustomer')}</dt>
		<dd>{doc.customer_code}</dd>
		<dt>{t('numberColumnType')}</dt>
		<dd>{doc.document_type.name}</dd>
		<dt>{t('releasedRevision')}</dt>
		<dd>{doc.revision}{doc.released_migrated ? ` · ${t('documentStatusReleasedMigrated')}` : ''}</dd>
		<dt>{t('documentModels')}</dt>
		<dd>{doc.models.map((m) => m.name).join(', ')}</dd>
	</dl>

	{@const headerFields = format.fields.filter((f) => f.area === 'header' && !f.hidden && headerText(doc?.header[f.id]) !== '')}
	{#if headerFields.length > 0}
		<section>
			<h2>{t('formatHeaderFields')}</h2>
			<dl class="facts">
				{#each headerFields as f (f.id)}
					<dt>{f.label}</dt>
					<dd>{headerText(doc.header[f.id])}</dd>
				{/each}
			</dl>
		</section>
	{/if}

	<section>
		<nav class="sheets" aria-label={t('documentSheets')}>
			{#each doc.set as m (m.id)}
				<a href="/documents/released/view?id={m.id}" data-sveltekit-noscroll aria-current={m.id === doc.id ? 'page' : undefined}>{m.document_type.name}</a>
			{/each}
			{#if hasPair}
				<a href="/documents/check?id={doc.id}&released=1">{t('checkTab')}</a>
			{/if}
		</nav>
		<h2>{t('documentRowsTitle')}</h2>
		{#if doc.sections.length > 1}
			<div class="field section-choice">
				<label for="released-section">{t('releasedSection')}</label>
				<select id="released-section" bind:value={sectionKey}>
					{#each doc.sections as s (s.key)}<option value={s.key}>{s.name}</option>{/each}
				</select>
			</div>
		{/if}
		{#key doc.id + (section?.id ?? '')}
			<Workbook doc={shown} {format} processes={[]} canEdit={false} onsave={async () => null} ondelete={() => {}} />
		{/key}
	</section>
{/if}

<style>
	.sheets {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin-bottom: 1rem;
		border-bottom: 1px solid var(--border);
	}

	.sheets a {
		padding: 8px 16px;
		border: 1px solid transparent;
		border-bottom: none;
		border-radius: var(--radius) var(--radius) 0 0;
		color: var(--text-2);
	}

	.sheets a[aria-current='page'] {
		background: var(--surface);
		border-color: var(--border);
		color: var(--accent-ink);
		font-weight: 600;
	}

	.section-choice {
		max-width: 20rem;
	}
</style>
