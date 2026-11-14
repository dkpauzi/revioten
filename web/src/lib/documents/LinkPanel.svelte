<script lang="ts">
	// Penautan baris (M7.4, R-MIG-05): sistem mengusulkan poin template untuk
	// baris yang diketik sendiri lewat kemiripan teks; orang mencentang yang
	// benar lalu mengonfirmasi. Sistem tidak pernah menautkan sendiri.
	import { linkDocumentRows, listLinkSuggestions } from '#lib/api/client.ts';
	import type { Document, LinkSuggestion } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let { doc, onlinked }: { doc: Document; onlinked: (doc: Document) => void } = $props();

	let items = $state<LinkSuggestion[] | null>(null);
	let chosen = $state<string[]>([]);
	let message = $state('');
	let failure = $state('');

	async function find() {
		message = '';
		failure = '';
		const result = await listLinkSuggestions(doc.id);
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		items = result.data.items;
		// Yang sama persis dicentang dulu; yang hanya mirip diputuskan orang.
		chosen = items.filter((s) => s.score === 100).map((s) => s.row_id);
	}

	async function confirmLinks(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		const links = (items ?? []).filter((s) => chosen.includes(s.row_id)).map((s) => ({ row_id: s.row_id, point_id: s.point_id, version_id: s.version_id }));
		const result = await linkDocumentRows(doc.id, links);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		message = t('linkDone', { count: String(links.length) });
		items = null;
		chosen = [];
		onlinked(result.data);
	}
</script>

<section>
	<h2>{t('linkTitle')}</h2>
	<p class="hint">{t('linkHint')}</p>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<p><button type="button" onclick={find}>{t('linkFind')}</button></p>
	{#if items !== null}
		{#if items.length === 0}
			<p>{t('linkNone')}</p>
		{:else}
			<form onsubmit={confirmLinks} novalidate>
				<div class="scroll">
					<table>
						<thead>
							<tr>
								<th></th>
								<th>{t('excelCompareRow')}</th>
								<th>{t('linkRowText')}</th>
								<th>{t('linkPoint')}</th>
								<th>{t('linkScore')}</th>
							</tr>
						</thead>
						<tbody>
							{#each items as s (s.row_id)}
								<tr>
									<td><input type="checkbox" bind:group={chosen} value={s.row_id} aria-label="{t('excelCompareRow')} {s.position}" /></td>
									<td>{s.position}</td>
									<td>{s.row_text}</td>
									<td>{s.point_text}{s.matched !== s.point_text ? ` · ${s.matched}` : ''}</td>
									<td>{s.score}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<p><button type="submit" class="primary" disabled={chosen.length === 0}>{t('linkConfirm')}</button></p>
			</form>
		{/if}
	{/if}
</section>

<style>
	.scroll {
		overflow-x: auto;
	}
</style>
