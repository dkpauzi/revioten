<script lang="ts">
	// Salin komposisi dari model lain (M4.7, R-CMP-05): dokumen yang masih
	// kosong dimulai dari baris dokumen model serupa. Salinan mandiri; yang
	// disalin baris dan poin yang dikecualikan, untuk seluruh set dokumen.
	import { onMount } from 'svelte';
	import { copyDocumentComposition, listDocuments } from '#lib/api/client.ts';
	import type { Document, DocumentSummary } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		doc,
		oncopied,
		oncancel
	}: {
		doc: Document;
		oncopied: (doc: Document, source: string) => void;
		oncancel: () => void;
	} = $props();

	let sources = $state<DocumentSummary[] | null>(null);
	let sourceId = $state('');
	let problem = $state('');
	let busy = $state(false);

	onMount(async () => {
		const result = await listDocuments(doc.customer_id);
		if (!result.ok) {
			problem = failureText(result.error);
			sources = [];
			return;
		}
		// Sumber: dokumen lain berjenis sama yang sudah punya baris.
		const own = new Set(doc.set.map((m) => m.id));
		sources = result.data.items.filter((d) => !own.has(d.id) && d.document_type.key === doc.document_type.key && d.row_count > 0);
	});

	async function run(event: SubmitEvent) {
		event.preventDefault();
		problem = '';
		const source = sources?.find((d) => d.id === sourceId);
		if (!source) {
			problem = t('copySourceRequired');
			return;
		}
		busy = true;
		const result = await copyDocumentComposition(doc.id, source.id);
		busy = false;
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			problem = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		oncopied(result.data, source.number);
	}
</script>

<h2>{t('copyTitle')}</h2>
<p class="hint">{t('copyHint')}</p>
{#if sources === null}
	<p>{t('formatLoading')}</p>
{:else if sources.length === 0 && !problem}
	<p>{t('copyNone', { type: doc.document_type.name })}</p>
	<p><button type="button" onclick={oncancel}>{t('composeClose')}</button></p>
{:else}
	<form onsubmit={run} novalidate>
		<div class="field">
			<label for="copy-source">{t('copySource')}</label>
			<select id="copy-source" bind:value={sourceId}>
				<option value="">{t('copySourceChoose')}</option>
				{#each sources as s (s.id)}
					<option value={s.id}>{t('copyOption', { number: s.number, models: s.models.join(', '), rows: String(s.row_count) })}</option>
				{/each}
			</select>
		</div>
		{#if problem}<p class="error" role="alert">{problem}</p>{/if}
		<p class="actions">
			<button type="submit" class="primary" disabled={busy}>{t('copyRun')}</button>
			<button type="button" onclick={oncancel}>{t('composeClose')}</button>
		</p>
	</form>
{/if}

<style>
	.actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
</style>