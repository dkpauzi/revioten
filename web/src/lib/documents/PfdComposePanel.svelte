<script lang="ts">
	// Rakit dari PFD (C4, D-211). Setiap langkah PFD tampil dengan process
	// template prosesnya dan keadaannya di PFMEA dan CP: sudah ada, baris yang
	// akan ditambah, atau satu baris "--" bila tanpa template (E-136). Penyusun
	// memilih langkah, dan baris dibuat dengan nomor langkah dan proses dari
	// PFD. Di bawahnya pemeriksaan: langkah PFD yang belum ada, dan nomor
	// langkah PFMEA atau CP yang tidak ada di PFD.
	import { onMount } from 'svelte';
	import { composeFromPfd, getPfdComposition } from '#lib/api/client.ts';
	import type { Document, PfdComposition, PfdStepTarget } from '#lib/api/gen/index.ts';
	import { chosenSteps, initialChoice, open, stepKey, warnings } from '#lib/documents/pfdcompose.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		doc,
		oncomposed,
		oncancel
	}: {
		doc: Document;
		oncomposed: (doc: Document, steps: number) => void;
		oncancel: () => void;
	} = $props();

	let view = $state<PfdComposition | null>(null);
	let chosen = $state<Record<string, boolean>>({});
	let problem = $state('');
	let busy = $state(false);

	let sections = $derived(Object.fromEntries(doc.sections.map((s) => [s.key, s.name])));
	let hasSections = $derived(view?.steps.some((s) => s.section_key !== '') ?? false);
	let openSteps = $derived(view?.steps.filter(open).length ?? 0);
	let checks = $derived(view ? warnings(view) : []);
	let typeName = $derived(Object.fromEntries((view?.documents ?? []).map((d) => [d.document_type_key, d.document_type_name])));

	onMount(load);

	async function load() {
		const result = await getPfdComposition(doc.id);
		if (!result.ok) {
			problem = failureText(result.error);
			return;
		}
		view = result.data;
		chosen = initialChoice(result.data);
	}

	function stateText(target: PfdStepTarget): string {
		switch (target.state) {
			case 'present':
				return t('pfdStatePresent');
			case 'to_add':
				return t('pfdStateToAdd', { points: String(target.points) });
			case 'no_template':
				return t('pfdStateNoTemplate');
			default:
				return t('pfdStateNotDraft');
		}
	}

	async function apply() {
		if (!view) {
			return;
		}
		const steps = chosenSteps(view, chosen);
		problem = '';
		busy = true;
		const result = await composeFromPfd(doc.id, steps);
		busy = false;
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			problem = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		oncomposed(result.data, steps.length);
	}
</script>

<p class="hint">{t('pfdComposeHint')}</p>
{#if problem}<p class="error" role="alert">{problem}</p>{/if}

{#if view === null}
	{#if !problem}<p>{t('formatLoading')}</p>{/if}
{:else if view.pfd_id === ''}
	<p>{t('pfdNoPfd')}</p>
{:else if view.steps.length === 0}
	<p>{t('pfdEmpty')}</p>
	<p><a class="button" href="/documents/view?id={view.pfd_id}">{t('nextPfdOpen')}</a></p>
{:else}
	<div class="table-wrap">
		<table data-testid="pfd-steps">
			<thead>
				<tr>
					<th scope="col"><span class="visually-hidden">{t('pfdApply')}</span></th>
					{#if hasSections}<th scope="col">{t('pfdSection')}</th>{/if}
					<th scope="col">{t('pfdStepNo')}</th>
					<th scope="col">{t('pfdStepProcess')}</th>
					<th scope="col">{t('pfdStepTemplate')}</th>
					{#each view.documents as d (d.id)}<th scope="col">{d.document_type_name}</th>{/each}
				</tr>
			</thead>
			<tbody>
				{#each view.steps as s (stepKey(s))}
					{@const key = stepKey(s)}
					<tr>
						<td>
							{#if open(s)}
								<input type="checkbox" aria-label={t('pfdChoose', { operation: s.operation, process: s.process_name })} bind:checked={chosen[key]} />
							{/if}
						</td>
						{#if hasSections}<td>{sections[s.section_key] ?? s.section_key}</td>{/if}
						<td class="mono">{s.operation}</td>
						<td>{s.process_name}</td>
						<td>
							{#if s.template_version_id}{t('pfdTemplateVersion', { version: String(s.template_version) })}{:else}<span class="muted">{t('pfdNoTemplate')}</span>{/if}
						</td>
						{#each s.targets as target (target.document_id)}
							<td class="state-{target.state}">{stateText(target)}</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	{#if openSteps === 0}<p>{t('pfdNothing')}</p>{/if}

	<section class="check" aria-labelledby="pfd-check-h">
		<h3 id="pfd-check-h">{t('pfdCheckTitle')}</h3>
		{#if checks.length === 0}
			<p>{t('pfdCheckOk')}</p>
		{:else}
			<ul data-testid="pfd-warnings">
				{#each checks as w, i (i)}
					<li>
						{#if hasSections && w.sectionKey}<strong>{sections[w.sectionKey] ?? w.sectionKey}</strong> · {/if}
						{t(w.kind === 'missing' ? 'pfdMissing' : 'pfdExtra', { operation: w.operation, process: w.process, document: typeName[w.documentType] ?? w.documentType })}
					</li>
				{/each}
			</ul>
		{/if}
	</section>
{/if}

<p class="actions">
	{#if view && openSteps > 0}
		<button type="button" class="primary" onclick={apply} disabled={busy}>{t('pfdApply')}</button>
	{/if}
	<button type="button" onclick={oncancel}>{t('composeClose')}</button>
</p>

<style>
	.table-wrap {
		overflow-x: auto;
	}

	td.state-present,
	td.state-not_draft,
	.muted {
		color: var(--text-3);
	}

	td.state-no_template {
		color: var(--warning);
	}

	.check ul {
		margin: 0;
		padding-left: 1.25rem;
	}

	.check li {
		color: var(--warning);
	}

	.actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
</style>
