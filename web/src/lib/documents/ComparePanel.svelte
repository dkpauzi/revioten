<script lang="ts">
	// Tampilan banding (M4.8, R-CMP-04): setiap baris yang dirakit dari
	// process template dibandingkan dengan poin asalnya. Baris yang diubah
	// ditandai berbeda, dengan kolom yang berbeda. Penyusun yang ingin baris
	// berdiri sendiri memutus tautannya, dengan alasan.
	import { onMount } from 'svelte';
	import { detachDocumentRow, getDocumentDifferences } from '#lib/api/client.ts';
	import type { Document, RowDifference } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		doc,
		candetach,
		ondetached,
		oncancel
	}: {
		doc: Document;
		candetach: boolean;
		ondetached: (doc: Document, position: number) => void;
		oncancel: () => void;
	} = $props();

	let items = $state<RowDifference[] | null>(null);
	let problem = $state('');
	// Baris yang sedang diputus tautannya, dan alasannya.
	let detaching = $state<string | null>(null);
	let reason = $state('');
	let reasonError = $state('');

	let different = $derived(items?.filter((d) => d.status === 'different').length ?? 0);

	onMount(load);

	async function load() {
		const result = await getDocumentDifferences(doc.id);
		if (!result.ok) {
			problem = failureText(result.error);
			items = [];
			return;
		}
		items = result.data.items;
	}

	function shown(value: unknown): string {
		return value === '' || value === null || value === undefined ? t('compareEmpty') : String(value);
	}

	async function detach(event: SubmitEvent, d: RowDifference) {
		event.preventDefault();
		reasonError = '';
		const result = await detachDocumentRow(doc.id, d.row_id, reason);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			reasonError = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		detaching = null;
		reason = '';
		ondetached(result.data, d.position);
	}
</script>

<h2>{t('compareTitle')}</h2>
<p class="hint">{t('compareHint')}</p>
{#if problem}<p class="error" role="alert">{problem}</p>{/if}
{#if items === null}
	<p>{t('formatLoading')}</p>
{:else if items.length === 0}
	<p>{t('compareNone')}</p>
{:else}
	<p data-testid="compare-summary">{t('compareSummary', { rows: String(items.length), different: String(different) })}</p>
	<table class="compare">
		<thead>
			<tr>
				<th>#</th>
				<th>{t('compareColumnPoint')}</th>
				<th>{t('compareColumnStatus')}</th>
				<th>{t('compareColumnChanges')}</th>
				{#if candetach}<th>{t('masterActions')}</th>{/if}
			</tr>
		</thead>
		<tbody>
			{#each items as d (d.row_id)}
				<tr data-status={d.status}>
					<td>{d.position}</td>
					<td>{d.point_text}</td>
					<td>
						<span class="badge" data-link={d.status}>
							{d.status === 'same' ? t('compareSame') : d.status === 'different' ? t('compareDifferent') : t('compareDetached')}
						</span>
					</td>
					<td>
						{#if d.status === 'detached'}
							{t('compareDetachReason', { reason: d.reason ?? '' })}
						{:else if d.fields.length > 0}
							<dl class="fields">
								{#each d.fields as f (f.field_id ?? f.label)}
									<dt>{f.label}</dt>
									<dd>
										<span class="template">{t('compareTemplate', { value: shown(f.template) })}</span>
										<span class="row">{t('compareRow', { value: shown(f.row) })}</span>
									</dd>
								{/each}
							</dl>
						{/if}
					</td>
					{#if candetach}
						<td>
							{#if d.status !== 'detached' && detaching !== d.row_id}
								<button
									type="button"
									aria-label={t('compareDetachFor', { position: String(d.position) })}
									onclick={() => {
										detaching = d.row_id;
										reason = '';
										reasonError = '';
									}}>{t('compareDetach')}</button
								>
							{/if}
						</td>
					{/if}
				</tr>
				{#if detaching === d.row_id}
					<tr class="detach">
						<td></td>
						<td colspan={candetach ? 4 : 3}>
							<form onsubmit={(event) => detach(event, d)} novalidate>
								<p class="hint">{t('compareDetachHint')}</p>
								<div class="field">
									<label for="detach-reason-{d.row_id}">{t('compareDetachReasonLabel')}</label>
									<input id="detach-reason-{d.row_id}" bind:value={reason} maxlength="500" size="50" autocomplete="off" />
									{#if reasonError}<p class="error">{reasonError}</p>{/if}
								</div>
								<p class="actions">
									<button type="submit">{t('compareDetach')}</button>
									<button type="button" onclick={() => (detaching = null)}>{t('masterCancel')}</button>
								</p>
							</form>
						</td>
					</tr>
				{/if}
			{/each}
		</tbody>
	</table>
{/if}
<p class="actions"><button type="button" onclick={oncancel}>{t('composeClose')}</button></p>

<style>
	.compare td {
		vertical-align: top;
	}

	.badge[data-link='same'] {
		background: var(--surface-2);
		color: var(--text-2);
	}

	.badge[data-link='different'] {
		background: var(--warning-weak);
		color: var(--warning);
	}

	.badge[data-link='detached'] {
		background: var(--surface-2);
		color: var(--text-3);
	}

	.fields {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 2px 12px;
		margin: 0;
	}

	.fields dt {
		color: var(--text-3);
	}

	.fields dd {
		margin: 0;
	}

	.template {
		display: block;
		color: var(--text-3);
		text-decoration: line-through;
	}

	.row {
		display: block;
	}

	.detach td {
		background: var(--surface-2);
	}

	.actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
</style>