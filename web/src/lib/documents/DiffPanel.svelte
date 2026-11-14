<script lang="ts">
	// Perbandingan dua revisi (M5.6, R-REV-03): per baris dan per sel. Sisi
	// "ke" boleh draft yang sedang dikerjakan. Saringan proses memberi riwayat
	// satu proses di dokumen ini.
	import { untrack } from 'svelte';
	import { getDocumentDiff } from '#lib/api/client.ts';
	import type { Document, DocumentDiff, DocumentRevision } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let { doc, revisions }: { doc: Document; revisions: DocumentRevision[] } = $props();

	let released = $derived(revisions.filter((r) => r.status === 'released'));
	// Proses yang ada di baris dokumen sekarang, untuk saringan.
	let processes = $derived(
		[...new Map(doc.rows.filter((r) => r.process_id).map((r) => [r.process_id as string, r.process_name ?? ''])).entries()].sort((a, b) =>
			a[1].localeCompare(b[1])
		)
	);
	let from = $state('');
	// Bawaannya draft yang terbuka; dokumen tanpa draft dibandingkan dengan revisi terbit terakhir.
	let to = $state(untrack(() => (doc.status === 'draft' ? 'draft' : String(revisions.find((r) => r.status === 'released')?.revision ?? ''))));
	let processId = $state('');
	let diff = $state<DocumentDiff | null>(null);
	let failure = $state('');

	async function compare(event: SubmitEvent) {
		event.preventDefault();
		failure = '';
		diff = null;
		if (from === '') {
			failure = t('diffChooseFrom');
			return;
		}
		const result = await getDocumentDiff(doc.id, Number(from), to === 'draft' ? undefined : Number(to), processId || undefined);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		diff = result.data;
	}

	// value menulis isi sel untuk dibaca: teks apa adanya, selain itu JSON.
	function value(v: unknown): string {
		if (v === undefined || v === null) return '';
		return typeof v === 'string' ? v : JSON.stringify(v);
	}
</script>

{#if released.length > 0}
	<form class="row" onsubmit={compare} novalidate>
		<div class="field">
			<label for="diff-from">{t('diffFrom')}</label>
			<select id="diff-from" bind:value={from}>
				<option value="">{t('changeChoose')}</option>
				{#each released as r (r.id)}<option value={String(r.revision)}>{r.revision}</option>{/each}
			</select>
		</div>
		<div class="field">
			<label for="diff-to">{t('diffTo')}</label>
			<select id="diff-to" bind:value={to}>
				{#if doc.status === 'draft'}<option value="draft">{t('diffDraft', { revision: String(doc.revision) })}</option>{/if}
				{#each released as r (r.id)}<option value={String(r.revision)}>{r.revision}</option>{/each}
			</select>
		</div>
		<div class="field">
			<label for="diff-process">{t('diffProcess')}</label>
			<select id="diff-process" bind:value={processId}>
				<option value="">{t('diffAllProcesses')}</option>
				{#each processes as [id, name] (id)}<option value={id}>{name}</option>{/each}
			</select>
		</div>
		<button type="submit">{t('diffCompare')}</button>
	</form>
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	{#if diff}
		{#if diff.header.length === 0 && diff.rows.length === 0}
			<p role="status">{t('diffNone')}</p>
		{:else}
			<table class="diff">
				<thead>
					<tr>
						<th>{t('diffChange')}</th>
						<th>{t('diffRow')}</th>
						<th>{t('diffField')}</th>
						<th>{t('diffOld')}</th>
						<th>{t('diffNew')}</th>
					</tr>
				</thead>
				<tbody>
					{#each diff.header as c (c.field_id)}
						<tr data-diff="changed">
							<td>{t('diffHeader')}</td>
							<td></td>
							<td>{c.label}</td>
							<td class="old">{value(c.old)}</td>
							<td class="new">{value(c.new)}</td>
						</tr>
					{/each}
					{#each diff.rows as r (r.row_id)}
						{@const where = `${r.operation_number ?? ''} ${r.process_name ?? ''}`.trim() || `#${r.new_position ?? r.old_position}`}
						{#if r.cells.length === 0}
							<tr data-diff={r.status}>
								<td>{t(`diff_${r.status}` as 'diff_added')}</td>
								<td>{where}</td>
								<td colspan="3">{r.status === 'moved' ? t('diffMovedFrom', { from: String(r.old_position), to: String(r.new_position) }) : ''}</td>
							</tr>
						{:else}
							{#each r.cells as c (c.field_id)}
								<tr data-diff={r.status}>
									<td>{t(`diff_${r.status}` as 'diff_added')}</td>
									<td>{where}</td>
									<td>{c.label}</td>
									<td class="old">{value(c.old)}</td>
									<td class="new">{value(c.new)}</td>
								</tr>
							{/each}
						{/if}
					{/each}
				</tbody>
			</table>
		{/if}
	{/if}
{/if}

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	.row > button {
		margin-top: var(--label-space);
	}

	.old {
		color: var(--danger);
		text-decoration: line-through;
		white-space: pre-wrap;
	}

	.new {
		color: var(--success);
		white-space: pre-wrap;
	}
</style>
