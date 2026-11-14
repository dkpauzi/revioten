<script lang="ts">
	// Yokoten (M5.9, R-CHG-04, D-40, D-118). Tiga tampilan di satu halaman:
	// tanpa parameter daftar yokoten; ?change=ID membuat yokoten dari perubahan
	// yang sudah diterima; ?id=ID papan status per model, tempat pemilik target
	// memutuskan apply, adapt, atau not applicable.
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import {
		createYokoten,
		decideYokotenTarget,
		getChange,
		getYokoten,
		listModelGroups,
		listModels,
		listProcesses,
		listYokotens,
		searchYokotenTargets,
		type ApiFailure
	} from '#lib/api/client.ts';
	import type { Change, Model, ModelGroup, Process, Yokoten, YokotenTarget, YokotenTargetFilter } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let changeId = $derived(page.url.searchParams.get('change') ?? '');
	let yokotenId = $derived(page.url.searchParams.get('id') ?? '');

	let list = $state<Yokoten[]>([]);
	let board = $state<Yokoten | null>(null);
	let source = $state<Change | null>(null);
	let groups = $state<ModelGroup[]>([]);
	let models = $state<Model[]>([]);
	let processes = $state<Process[]>([]);
	let targetKind = $state('customer');
	let groupId = $state('');
	let modelIds = $state<string[]>([]);
	let processId = $state('');
	let text = $state('');
	let reason = $state('');
	let found = $state<YokotenTarget[] | null>(null);
	let message = $state('');
	let failure = $state('');
	// Target yang sedang diputuskan tidak berlaku, dan alasannya.
	let declining = $state<string | null>(null);
	let declineReason = $state('');
	let canPropose = $derived(can('E-3'));

	$effect(() => {
		const c = changeId;
		const y = yokotenId;
		untrack(() => load(c, y));
	});

	async function load(c: string, y: string) {
		message = '';
		failure = '';
		if (y) {
			const result = await getYokoten(y);
			board = result.ok ? result.data : null;
			if (!result.ok) failure = failureText(result.error);
			return;
		}
		if (c) {
			const result = await getChange(c);
			if (!result.ok) {
				failure = failureText(result.error);
				return;
			}
			source = result.data;
			reason = result.data.reason;
			const [g, m, p] = await Promise.all([listModelGroups(result.data.customer_id), listModels(result.data.customer_id), listProcesses()]);
			groups = g.ok ? g.data.items : [];
			models = m.ok ? m.data.items : [];
			processes = p.ok ? p.data.items : [];
			return;
		}
		const result = await listYokotens();
		list = result.ok ? result.data.items : [];
	}

	function filter(): YokotenTargetFilter {
		const f: YokotenTargetFilter = { process_id: processId || undefined, text: text.trim() || undefined };
		if (targetKind === 'customer') f.customer_id = source?.customer_id;
		if (targetKind === 'group') f.model_group_id = groupId || undefined;
		if (targetKind === 'models') f.model_ids = modelIds;
		return f;
	}

	function showError(error: ApiFailure) {
		const details = error.body?.details ?? [];
		failure = details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	async function search(event: SubmitEvent) {
		event.preventDefault();
		failure = '';
		if (!source) return;
		const result = await searchYokotenTargets({ source_change_id: source.id, targets: filter() });
		if (!result.ok) {
			showError(result.error);
			return;
		}
		found = result.data.items;
	}

	async function create() {
		failure = '';
		if (!source) return;
		const result = await createYokoten({ source_change_id: source.id, reason: reason.trim(), targets: filter() });
		if (!result.ok) {
			showError(result.error);
			return;
		}
		location.assign(`/yokoten?id=${encodeURIComponent(result.data.id)}`);
	}

	async function decide(target: YokotenTarget, decision: string, why = '') {
		message = '';
		failure = '';
		if (!board || !target.id) return;
		const result = await decideYokotenTarget(board.id, target.id, { decision, reason: why.trim() || undefined });
		if (!result.ok) {
			showError(result.error);
			return;
		}
		board = result.data;
		declining = null;
		declineReason = '';
		message = t('yokotenDecided', { model: target.model_name, decision: t(`yokotenDecision_${decision}` as 'yokotenDecision_apply') });
	}

	function progressText(p: string): string {
		return t(`yokotenProgress_${p}` as 'yokotenProgress_undecided');
	}
</script>

{#if yokotenId}
	<p><a href="/yokoten">{t('yokotenBack')}</a></p>
	{#if board}
		<h1 class="mono">{board.ref}</h1>
		<dl class="facts">
			<dt>{t('yokotenSource')}</dt>
			<dd><a class="mono" href="/changes/view?id={board.source_change.id}">{board.source_change.name}</a></dd>
			<dt>{t('changeReason')}</dt>
			<dd class="reason">{board.reason}</dd>
			{#if board.process_filter}<dt>{t('diffProcess')}</dt><dd>{board.process_filter}</dd>{/if}
			{#if board.text_filter}<dt>{t('yokotenText')}</dt><dd>{board.text_filter}</dd>{/if}
			<dt>{t('changeRequestedBy')}</dt>
			<dd>{board.created_by} · {formatDateTime(board.created_at)}</dd>
		</dl>
		{#if message}<p role="status">{message}</p>{/if}
		{#if failure}<p class="error" role="alert">{failure}</p>{/if}
		<h2>{t('yokotenBoard')}</h2>
		<table class="board">
			<thead>
				<tr>
					<th>{t('documentCustomer')}</th>
					<th>{t('yokotenModel')}</th>
					<th>{t('changeDocuments')}</th>
					<th>{t('yokotenProgress')}</th>
					<th>{t('impactDecision')}</th>
				</tr>
			</thead>
			<tbody>
				{#each board.targets as target (target.id)}
					<tr data-progress={target.progress}>
						<td>{target.customer_code}</td>
						<td>{target.model_name}</td>
						<td class="mono">{target.documents.map((d) => `${d.number} (${d.status === 'released' ? t('documentStatusReleased') : t('versionStatusDraft')} ${d.revision})`).join(', ')}</td>
						<td>
							<span class="badge" data-yokoten={target.progress}>{progressText(target.progress)}</span>
							{#if target.change}<a class="mono" href="/changes/view?id={target.change.id}">{target.change.name}</a>{/if}
						</td>
						<td>
							{#if target.decision !== 'pending'}
								{t(`yokotenDecision_${target.decision}` as 'yokotenDecision_apply')}{target.reason ? ` · ${target.reason}` : ''}
								{#if target.decided_by}<br /><span class="hint">{target.decided_by}</span>{/if}
							{:else if canPropose}
								{#if declining === target.id}
									<div class="row">
										<input aria-label={t('yokotenDeclineReasonFor', { model: target.model_name })} maxlength="500" bind:value={declineReason} />
										<button type="button" onclick={() => decide(target, 'not_applicable', declineReason)}>{t('yokotenDecline')}</button>
										<button type="button" onclick={() => (declining = null)}>{t('masterCancel')}</button>
									</div>
								{:else}
									<div class="row">
										<button type="button" aria-label={t('yokotenApplyFor', { model: target.model_name })} onclick={() => decide(target, 'apply')}>{t('yokotenDecision_apply')}</button>
										<button type="button" aria-label={t('yokotenAdaptFor', { model: target.model_name })} onclick={() => decide(target, 'adapt')}>{t('yokotenDecision_adapt')}</button>
										<button type="button" aria-label={t('yokotenNotApplicableFor', { model: target.model_name })} onclick={() => (declining = target.id ?? null)}>
											{t('yokotenDecision_not_applicable')}
										</button>
									</div>
								{/if}
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{:else if failure}
		<p class="error" role="alert">{failure}</p>
	{/if}
{:else if changeId}
	<p><a href="/changes/view?id={changeId}">{t('changeBack')}</a></p>
	<h1>{t('yokotenNew')}</h1>
	{#if source}
		<p class="hint">{t('yokotenNewHint', { ref: source.ref })}</p>
		<form onsubmit={search} novalidate>
			<div class="row">
				<div class="field">
					<label for="yokoten-kind">{t('yokotenTargets')}</label>
					<select id="yokoten-kind" bind:value={targetKind}>
						<option value="customer">{t('yokotenTargetCustomer', { customer: source.customer_code })}</option>
						<option value="group">{t('yokotenTargetGroup')}</option>
						<option value="models">{t('yokotenTargetModels')}</option>
					</select>
				</div>
				{#if targetKind === 'group'}
					<div class="field">
						<label for="yokoten-group">{t('yokotenTargetGroup')}</label>
						<select id="yokoten-group" bind:value={groupId}>
							<option value="">{t('changeChoose')}</option>
							{#each groups as g (g.id)}<option value={g.id}>{g.name}</option>{/each}
						</select>
					</div>
				{/if}
				<div class="field">
					<label for="yokoten-process">{t('diffProcess')}</label>
					<select id="yokoten-process" bind:value={processId}>
						<option value="">{t('diffAllProcesses')}</option>
						{#each processes as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
					</select>
				</div>
				<div class="field">
					<label for="yokoten-text">{t('yokotenText')}</label>
					<input id="yokoten-text" maxlength="200" bind:value={text} />
				</div>
			</div>
			{#if targetKind === 'models'}
				<fieldset>
					<legend>{t('yokotenTargetModels')}</legend>
					{#each models as m (m.id)}
						<label class="check"><input type="checkbox" value={m.id} bind:group={modelIds} /> {m.name}</label>
					{/each}
				</fieldset>
			{/if}
			<button type="submit">{t('yokotenSearch')}</button>
		</form>
		{#if failure}<p class="error" role="alert">{failure}</p>{/if}
		{#if found}
			{#if found.length === 0}
				<p>{t('yokotenNoTargets')}</p>
			{:else}
				<table>
					<thead>
						<tr>
							<th>{t('documentCustomer')}</th>
							<th>{t('yokotenModel')}</th>
							<th>{t('changeDocuments')}</th>
						</tr>
					</thead>
					<tbody>
						{#each found as target (target.model_id)}
							<tr>
								<td>{target.customer_code}</td>
								<td>{target.model_name}</td>
								<td class="mono">{target.documents.map((d) => d.number).join(', ')}</td>
							</tr>
						{/each}
					</tbody>
				</table>
				<div class="field">
					<label for="yokoten-reason">{t('changeReason')}</label>
					<textarea id="yokoten-reason" rows="2" maxlength="2000" bind:value={reason}></textarea>
				</div>
				<button type="button" class="primary" onclick={create}>{t('yokotenCreate', { count: String(found.length) })}</button>
			{/if}
		{/if}
	{:else if failure}
		<p class="error" role="alert">{failure}</p>
	{/if}
{:else}
	<h1>{t('yokotenTitle')}</h1>
	<p class="hint">{t('yokotenHint')}</p>
	{#if list.length === 0}
		<p>{t('yokotenNone')}</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>{t('yokotenRef')}</th>
					<th>{t('yokotenSource')}</th>
					<th>{t('changeReason')}</th>
					<th>{t('yokotenProgress')}</th>
				</tr>
			</thead>
			<tbody>
				{#each list as y (y.id)}
					<tr>
						<td><a class="mono" href="/yokoten?id={y.id}">{y.ref}</a></td>
						<td class="mono">{y.source_change.name}</td>
						<td>{y.reason}</td>
						<td>{t('yokotenSummary', { done: String(y.targets.filter((x) => x.progress === 'released' || x.progress === 'not_applicable').length), total: String(y.targets.length) })}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
{/if}

<style>
	.facts {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 4px 16px;
		margin: 0 0 1.5rem;
	}

	.facts dt {
		color: var(--text-3);
	}

	.facts dd {
		margin: 0;
	}

	.reason {
		white-space: pre-wrap;
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1rem;
		align-items: flex-start;
	}

	.check {
		display: inline-flex;
		gap: 0.25rem;
		margin-right: 1rem;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
