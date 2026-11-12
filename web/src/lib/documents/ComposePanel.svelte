<script lang="ts">
	// Perakitan dua panel (M4.5, R-CMP-01 sampai R-CMP-03). Kiri: pohon process
	// template dengan kotak centang sub-proses dan poin. Kanan: baris yang akan
	// ada di setiap dokumen set ini. Poin tercentang menjadi baris yang tertaut
	// ke template; poin tanpa centang dicatat sebagai tidak dipakai. Belum ada
	// yang disimpan sebelum tombol Apply ditekan.
	import { onMount } from 'svelte';
	import { composeDocument, getDocumentComposition, listProcessTemplates } from '#lib/api/client.ts';
	import type { Composition, Document } from '#lib/api/gen/index.ts';
	import { composeInput, initialChecks, publishedChoices, resultRows, type Checks, type PublishedChoice } from '#lib/compose.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		doc,
		oncomposed,
		oncancel
	}: {
		doc: Document;
		oncomposed: (doc: Document, process: string) => void;
		oncancel: () => void;
	} = $props();

	let choices = $state<PublishedChoice[] | null>(null);
	let versionId = $state('');
	let view = $state<Composition | null>(null);
	let checks = $state<Checks>({});
	let reasons = $state<Record<string, string>>({});
	let problem = $state('');
	let busy = $state(false);

	onMount(async () => {
		const result = await listProcessTemplates();
		if (!result.ok) {
			problem = failureText(result.error);
			choices = [];
			return;
		}
		choices = publishedChoices(result.data.items);
	});

	async function choose() {
		problem = '';
		view = null;
		if (versionId === '') {
			return;
		}
		const result = await getDocumentComposition(doc.id, versionId);
		if (!result.ok) {
			problem = failureText(result.error);
			return;
		}
		view = result.data;
		checks = initialChecks(result.data);
		reasons = Object.fromEntries(result.data.subprocesses.flatMap((s) => s.points.map((p) => [p.point_id, p.reason ?? ''])));
	}

	// Kotak centang sub-proses mencentang atau mengosongkan semua poinnya.
	function allChecked(points: Composition['subprocesses'][number]['points']): boolean {
		return points.every((p) => checks[p.point_id]);
	}

	function toggleAll(points: Composition['subprocesses'][number]['points'], on: boolean) {
		for (const p of points) {
			checks[p.point_id] = on;
		}
	}

	async function apply() {
		if (!view) {
			return;
		}
		problem = '';
		busy = true;
		const result = await composeDocument(doc.id, composeInput(view, checks, reasons));
		busy = false;
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			problem = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		oncomposed(result.data, view.process.name);
	}
</script>

<h2>{t('composeTitle')}</h2>
<p class="hint">{t('composeHint')}</p>
{#if choices === null}
	<p>{t('formatLoading')}</p>
{:else if choices.length === 0 && !problem}
	<p>{t('composeNoTemplates')}</p>
{:else}
	<div class="field">
		<label for="compose-process">{t('composeProcess')}</label>
		<select id="compose-process" bind:value={versionId} onchange={choose}>
			<option value="">{t('composeProcessChoose')}</option>
			{#each choices as c (c.versionId)}
				<option value={c.versionId}>{c.process} · {t('composeVersion', { version: String(c.version) })}</option>
			{/each}
		</select>
	</div>
{/if}
{#if problem}<p class="error" role="alert">{problem}</p>{/if}

{#if view}
	{#if view.subprocesses.length === 0}
		<p>{t('composeEmpty')}</p>
	{:else}
		<div class="panels">
			<div class="panel">
				<h3>{t('composeLeft')}</h3>
				{#each view.subprocesses as s (s.id)}
					<fieldset>
						<legend>
							<label class="check">
								<input
									type="checkbox"
									aria-label={t('composeAll', { name: s.name })}
									checked={allChecked(s.points)}
									onchange={(event) => toggleAll(s.points, event.currentTarget.checked)}
								/>
								<strong>{s.name}</strong>
							</label>
						</legend>
						<ul>
							{#each s.points as p (p.point_id)}
								<li>
									<label class="check">
										<input type="checkbox" bind:checked={checks[p.point_id]} />
										<span class="type">{p.document_type.name}</span>
										<span>{p.text}</span>
									</label>
									{#if p.state === 'undecided'}<span class="badge">{t('composeUndecided')}</span>{/if}
									{#if !checks[p.point_id]}
										<input
											class="reason"
											aria-label={t('composeReason', { text: p.text })}
											placeholder={t('composeReasonPlaceholder')}
											maxlength="500"
											bind:value={reasons[p.point_id]}
										/>
									{/if}
								</li>
							{/each}
						</ul>
					</fieldset>
				{/each}
			</div>
			<div class="panel">
				<h3>{t('composeRight')}</h3>
				{#each doc.set as member (member.id)}
					{@const rows = resultRows(view, checks, member.id)}
					<h4>{member.document_type.name} · <span class="mono">{member.number}</span></h4>
					{#if rows.length === 0}
						<p class="hint">{t('composeNoRows')}</p>
					{:else}
						<ol data-testid="compose-result-{member.document_type.key}">
							{#each rows as r (r.pointId)}
								<li class:remove={r.change === 'remove'}>
									<span class="sub">{r.subprocess}</span>
									<span class="text">{r.text}</span>
									{#if r.change === 'add'}<em>{t('composeWillAdd')}</em>{/if}
									{#if r.change === 'remove'}<em>{t('composeWillRemove')}</em>{/if}
								</li>
							{/each}
						</ol>
					{/if}
				{/each}
			</div>
		</div>
	{/if}
{/if}

<p class="actions">
	{#if view && view.subprocesses.length > 0}
		<button type="button" class="primary" onclick={apply} disabled={busy}>{t('composeApply')}</button>
	{/if}
	<button type="button" onclick={oncancel}>{t('composeClose')}</button>
</p>

<style>
	.panels {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 24px;
		align-items: start;
	}

	/* Di HP dan tablet kecil kedua panel bersusun (D-128). */
	@media (max-width: 899px) {
		.panels {
			grid-template-columns: 1fr;
		}
	}

	.panel {
		min-width: 0;
	}

	fieldset {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		margin: 0 0 12px;
		padding: 8px 12px;
	}

	ul {
		list-style: none;
		margin: 0;
		padding: 0 0 0 24px;
	}

	li {
		padding: 4px 0;
	}

	.check {
		display: inline-flex;
		gap: 8px;
		align-items: baseline;
	}

	.type {
		color: var(--text-3);
		font-size: 12px;
		min-width: 3.5rem;
	}

	.reason {
		display: block;
		margin: 4px 0 0 24px;
		width: min(100%, 24rem);
	}

	ol {
		margin: 0 0 12px;
		padding-left: 1.5rem;
	}

	.sub {
		color: var(--text-3);
		margin-right: 6px;
	}

	li.remove .text {
		text-decoration: line-through;
		color: var(--text-3);
	}

	em {
		color: var(--warning);
		font-size: 13px;
		margin-left: 6px;
	}

	.actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
