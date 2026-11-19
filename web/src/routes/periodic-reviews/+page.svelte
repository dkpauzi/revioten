<script lang="ts">
	// Review berkala 3 tahun (D-138, D-196), tampilan mengikuti desain D-202.
	// Jam dihitung per dokumen dari rilis terakhirnya atau review terakhir set
	// itu. Set yang jatuh tempo bisa direview dengan dua hasil:
	//   - Changes needed: membuka permintaan perubahan dengan pemicu Periodic
	//     review; jam mulai lagi saat revisi baru terbit;
	//   - Reviewed, no change: catatan review dengan daftar periksa, yang
	//     dikonfirmasi orang kedua (V-1, ASUMSI E-86); jam mulai lagi.
	// Model EOL (D-200) tidak diperingatkan.
	import { onMount } from 'svelte';
	import { confirmPeriodicReview, listCustomers, listPeriodicReviews, listReviewSets, recordPeriodicReview, reviewExportUrl } from '#lib/api/client.ts';
	import type { Customer, PeriodicReview, ReviewAnswer, ReviewSet, ReviewSetList } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { language } from '#lib/i18n/language.svelte.ts';
	import { compactNumbers, formatDay, isoDate, overdueTypes } from '#lib/reports.ts';
	import { can } from '#lib/session.svelte.ts';

	const items: ReviewAnswer['item'][] = ['process_flow', 'claims_findings', 'occurrence_detection', 'special_characteristics', 'control_methods', 'references', 'reaction_plan', 'pfmea_cp_sync'];
	type Tab = 'due' | 'year' | 'all';

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let engineer = $state('');
	let tab = $state<Tab>('due');
	let showEol = $state(false);
	let showEvery = $state(false);
	let overview = $state<ReviewSetList | null>(null);
	let waiting = $state<PeriodicReview[]>([]);
	let recent = $state<PeriodicReview[]>([]);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let busy = $state(false);

	// Panel "Start review" untuk satu set.
	let open = $state<ReviewSet | null>(null);
	let outcome = $state<'changes' | 'no_change'>('changes');
	let reviewedOn = $state('');
	let participants = $state('');
	let conclusion = $state('');
	let answers = $state<Record<string, ReviewAnswer['answer'] | ''>>({});

	let canRecord = $derived(can('E-3'));
	let canConfirm = $derived(can('V-1'));
	const shownRows = 6;

	let engineers = $derived([...new Set((overview?.items ?? []).map((s) => s.engineer).filter((e) => e !== ''))].sort());
	let rows = $derived(
		(overview?.items ?? []).filter((s) => {
			if (engineer !== '' && s.engineer !== engineer) return false;
			if (s.status === 'eol') return showEol;
			if (tab === 'due') return s.status === 'overdue' || s.status === 'due_soon';
			if (tab === 'year') return s.status !== 'ok';
			return true;
		})
	);
	let visible = $derived(showEvery ? rows : rows.slice(0, shownRows));

	function day(iso: string): string {
		return formatDay(iso, language.current);
	}

	function setName(s: { customer_code: string; sequence: number }): string {
		return `${s.customer_code.toUpperCase()} · ${String(s.sequence).padStart(4, '0')}`;
	}

	function modelText(s: ReviewSet): string {
		return s.models.join(', ') || setName(s);
	}

	// errorText memberi pesan masalah pertama dari server, atau pesan umum.
	function errorText(error: Parameters<typeof failureText>[0]): string {
		const details = error.body?.details ?? [];
		return details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	// dueText memberi teks lencana jatuh tempo.
	function dueText(s: ReviewSet): string {
		if (s.status === 'eol') return t('reviewEol');
		if (s.days_left < 0) return t('reviewOverdue', { days: String(-s.days_left) });
		return t('reviewDueIn', { days: String(s.days_left) });
	}

	// whichText menyebut dokumen yang lewat jatuh tempo, atau tanggal jatuh tempo.
	function whichText(s: ReviewSet): string {
		const late = overdueTypes(s.documents);
		if (late.length === 0) return t('reviewDueOnDate', { date: day(s.due_on) });
		if (late.length === s.documents.length && late.length > 1) return t('reviewAllDocuments');
		return late.map((k) => t(`reviewType_${k}` as MessageKey)).join(t('reviewAnd')) + t('reviewOnly');
	}

	function docOf(s: ReviewSet, key: string) {
		return s.documents.find((d) => d.document_type === key);
	}

	async function load() {
		const [sets, openReviews, all] = await Promise.all([listReviewSets({ customerId, all: true, eol: true }), listPeriodicReviews(true), listPeriodicReviews(false)]);
		if (!sets.ok) {
			loadError = failureText(sets.error);
			return;
		}
		loadError = '';
		overview = sets.data;
		waiting = openReviews.ok ? openReviews.data.items : [];
		recent = all.ok ? all.data.items.filter((r) => r.status === 'confirmed').slice(0, 20) : [];
	}

	function start(s: ReviewSet) {
		open = s;
		outcome = 'changes';
		reviewedOn = isoDate(new Date());
		participants = '';
		conclusion = '';
		answers = Object.fromEntries(items.map((i) => [i, '']));
		message = '';
		failure = '';
		queueMicrotask(() => document.getElementById('start-review')?.scrollIntoView({ block: 'start' }));
	}

	// changeLink membuka formulir perubahan dengan customer, pemicu, dan dokumen set itu.
	function changeLink(s: ReviewSet): string {
		const query = new URLSearchParams({ customer: s.customer_id, trigger: 'Periodic review', documents: s.documents.map((d) => d.id).join(',') });
		return '/changes?' + query.toString();
	}

	async function record(event: SubmitEvent) {
		event.preventDefault();
		if (!open) return;
		if (outcome === 'changes') {
			window.location.assign(changeLink(open));
			return;
		}
		if (items.some((i) => answers[i] === '')) {
			failure = t('reviewAnswerAll');
			return;
		}
		busy = true;
		const result = await recordPeriodicReview({
			document_id: open.documents[0].id,
			reviewed_on: reviewedOn,
			participants,
			conclusion,
			checklist: items.map((i) => ({ item: i, answer: answers[i] as ReviewAnswer['answer'] }))
		});
		busy = false;
		if (!result.ok) {
			failure = errorText(result.error);
			return;
		}
		message = t('reviewRecorded', { ref: result.data.ref, set: setName(result.data) });
		open = null;
		await load();
	}

	async function confirm(r: PeriodicReview) {
		message = '';
		failure = '';
		busy = true;
		const result = await confirmPeriodicReview(r.id);
		busy = false;
		if (!result.ok) {
			failure = errorText(result.error);
			return;
		}
		message = t('reviewConfirmed', { ref: r.ref, set: setName(r) });
		await load();
	}

	onMount(async () => {
		const result = await listCustomers();
		customers = result.ok ? result.data.items : [];
		await load();
	});
</script>

<div class="head">
	<div>
		<h1>{t('reviewTitle')}</h1>
		<p class="hint">{t('reviewHint')}</p>
	</div>
	<a class="button" href={reviewExportUrl({ customerId, all: true, eol: showEol })} download>{t('reviewDownload')}</a>
</div>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}
{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

{#if overview}
	<div class="stats">
		<section class="stat" data-tone="danger" aria-labelledby="stat-overdue">
			<h2 id="stat-overdue">{t('reviewStatOverdue')}</h2>
			<p class="big" data-testid="review-overdue">{overview.overdue}</p>
			<p>{t('reviewStatOverdueHint', { months: String(overview.period_months) })}</p>
		</section>
		<section class="stat" data-tone="warning" aria-labelledby="stat-soon">
			<h2 id="stat-soon">{t('reviewStatDueSoon', { days: String(overview.warning_days) })}</h2>
			<p class="big">{overview.due_soon}</p>
			<p>{t('reviewStatDueSoonHint')}</p>
		</section>
		<section class="stat" aria-labelledby="stat-year">
			<h2 id="stat-year">{t('reviewStatYear')}</h2>
			<p class="big">{overview.due_year}</p>
			<p>{t('reviewStatYearHint')}</p>
		</section>
		<section class="stat" aria-labelledby="stat-done">
			<h2 id="stat-done">{t('reviewStatDone')}</h2>
			<p class="big">{overview.reviewed_revised + overview.reviewed_no_change}</p>
			<p>{t('reviewStatDoneHint', { revised: String(overview.reviewed_revised), unchanged: String(overview.reviewed_no_change) })}</p>
		</section>
	</div>

	<div class="filters">
		<div class="segmented" role="group" aria-label={t('reviewShow')}>
			<button type="button" aria-pressed={tab === 'due'} onclick={() => (tab = 'due')}>{t('reviewTabDue', { count: String(overview.overdue + overview.due_soon) })}</button>
			<button type="button" aria-pressed={tab === 'year'} onclick={() => (tab = 'year')}>{t('reviewTabYear')}</button>
			<button type="button" aria-pressed={tab === 'all'} onclick={() => (tab = 'all')}>{t('reviewTabAll')}</button>
		</div>
		<label class="visually-hidden" for="review-customer">{t('documentCustomer')}</label>
		<select id="review-customer" bind:value={customerId} onchange={load}>
			<option value="">{t('dashboardAllCustomers')}</option>
			{#each customers as c (c.id)}
				<option value={c.id}>{c.code} · {c.name}</option>
			{/each}
		</select>
		<label class="visually-hidden" for="review-engineer">{t('reviewEngineer')}</label>
		<select id="review-engineer" bind:value={engineer}>
			<option value="">{t('reviewAllEngineers')}</option>
			{#each engineers as e (e)}<option value={e}>{e}</option>{/each}
		</select>
		<label class="check"><input type="checkbox" bind:checked={showEol} /> {t('reviewShowEol')}</label>
	</div>

	<div class="table-card">
		{#if rows.length === 0}
			<p class="empty">{t('reviewNone')}</p>
		{:else}
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('reviewModel')}</th>
							<th>PFD</th>
							<th>PFMEA</th>
							<th>{t('reviewControlPlan')}</th>
							<th>{t('reviewDueColumn')}</th>
							<th>{t('reviewEngineer')}</th>
							<th><span class="visually-hidden">{t('reviewAction')}</span></th>
						</tr>
					</thead>
					<tbody>
						{#each visible as s (s.customer_code + s.sequence)}
							<tr data-review={s.status}>
								<td>
									<strong>{modelText(s)}</strong>
									<div class="mono sub">
										<a href="/documents/released/view?id={s.documents[0].id}">{compactNumbers(s.documents.map((d) => d.number))}</a>
										· {s.customer_code.toUpperCase()}
									</div>
								</td>
								{#each ['pfd', 'pfmea', 'cp'] as key (key)}
									{@const d = docOf(s, key)}
									<td class:late={d?.overdue}>
										{#if d}
											<div>rev {String(d.revision).padStart(2, '0')}</div>
											<div class="sub">{day(d.last_release)}</div>
										{:else}
											<span class="sub">—</span>
										{/if}
									</td>
								{/each}
								<td>
									<span class="badge" data-review={s.status}>{dueText(s)}</span>
									<div class="sub">{s.status === 'eol' ? '' : whichText(s)}</div>
								</td>
								<td>{s.engineer || '—'}</td>
								<td>
									{#if s.open_review_id}
										<span class="sub">{t('reviewWaitingConfirmation')}</span>
									{:else if canRecord && s.status !== 'eol'}
										<button type="button" class={s.status === 'overdue' ? 'primary' : ''} onclick={() => start(s)}>{t('reviewStart')}</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			{#if rows.length > shownRows && !showEvery}
				<p class="more">{t('reviewMore', { count: String(rows.length - shownRows) })} · <button type="button" class="link" onclick={() => (showEvery = true)}>{t('reviewShowAllRows')}</button></p>
			{/if}
		{/if}
	</div>

	<div class="lower">
		{#if open}
			<section class="card" id="start-review" aria-labelledby="start-title">
				<h2 id="start-title">{t('reviewStartTitle', { model: modelText(open) })}</h2>
				<p class="hint">{t('reviewStartHint')}</p>
				<form onsubmit={record} novalidate>
					<fieldset class="outcomes">
						<legend class="visually-hidden">{t('reviewOutcome')}</legend>
						<label class="outcome" class:chosen={outcome === 'changes'}>
							<input type="radio" name="outcome" value="changes" bind:group={outcome} />
							<span><strong>{t('reviewChangesNeeded')}</strong><span class="sub">{t('reviewChangesNeededHint')}</span></span>
						</label>
						<label class="outcome" class:chosen={outcome === 'no_change'}>
							<input type="radio" name="outcome" value="no_change" bind:group={outcome} />
							<span><strong>{t('reviewNoChange')}</strong><span class="sub">{t('reviewNoChangeHint')}</span></span>
						</label>
					</fieldset>
					{#if outcome === 'no_change'}
						<div class="grid">
							<div class="field">
								<label for="review-date">{t('reviewDate')}</label>
								<input id="review-date" type="date" bind:value={reviewedOn} />
							</div>
							<div class="field">
								<label for="review-participants">{t('reviewParticipants')}</label>
								<input id="review-participants" bind:value={participants} />
							</div>
						</div>
						<fieldset>
							<legend>{t('reviewChecklist')}</legend>
							{#each items as item (item)}
								<div class="item">
									<span>{t(`reviewItem_${item}` as MessageKey)}</span>
									<label><input type="radio" name="item-{item}" value="ok" bind:group={answers[item]} /> {t('reviewAnswerOk')}</label>
									<label><input type="radio" name="item-{item}" value="not_applicable" bind:group={answers[item]} /> {t('reviewAnswerNa')}</label>
								</div>
							{/each}
						</fieldset>
						<div class="field">
							<label for="review-conclusion">{t('reviewConclusion')}</label>
							<textarea id="review-conclusion" rows="3" placeholder={t('reviewConclusionPlaceholder')} bind:value={conclusion}></textarea>
						</div>
					{/if}
					<div class="actions">
						<button type="submit" class="primary" disabled={busy}>{outcome === 'changes' ? t('reviewOpenChange') : t('reviewSave')}</button>
						<button type="button" onclick={() => (open = null)}>{t('reviewCancel')}</button>
					</div>
				</form>
			</section>
		{/if}
		<aside class="card clock" aria-labelledby="clock-title">
			<h2 id="clock-title">{t('reviewClockTitle')}</h2>
			<p>{t('reviewClockHint')}</p>
			<table>
				<thead>
					<tr><th>{t('reviewClockWhen')}</th><th>{t('reviewClockShown')}</th><th>{t('reviewClockWhere')}</th></tr>
				</thead>
				<tbody>
					<tr><td>{t('reviewClockYear')}</td><td><span class="badge" data-review="due_year">{t('reviewTabYear')}</span></td><td>{t('reviewClockYearWhere')}</td></tr>
					<tr><td>{t('reviewClockSoon', { days: String(overview.warning_days) })}</td><td><span class="badge" data-review="due_soon">{t('reviewDueSoonBadge')}</span></td><td>{t('reviewClockSoonWhere')}</td></tr>
					<tr><td>{t('reviewClockPassed')}</td><td><span class="badge" data-review="overdue">{t('reviewOverdueBadge')}</span></td><td>{t('reviewClockPassedWhere')}</td></tr>
				</tbody>
			</table>
			<p class="sub">{t('reviewClockNote')}</p>
		</aside>
	</div>
{/if}

<section>
	<h2>{t('reviewWaitingTitle')}</h2>
	{#if waiting.length === 0}
		<p>{t('reviewWaitingNone')}</p>
	{:else}
		{#each waiting as r (r.id)}
			<article class="card" aria-label={r.ref}>
				<h3>{r.ref} · {setName(r)}</h3>
				<p>{t('reviewRecordedBy', { by: r.recorded_by, on: day(r.reviewed_on) })}</p>
				<p>{t('reviewParticipants')}: {r.participants}</p>
				<p>{r.conclusion}</p>
				<p class="hint">{r.revisions.map((v) => `${v.number} rev ${v.revision}`).join(', ')}</p>
				{#if canConfirm}
					<button type="button" class="primary" disabled={busy} onclick={() => confirm(r)}>{t('reviewConfirm')}</button>
				{/if}
			</article>
		{/each}
	{/if}
</section>

<section>
	<h2>{t('reviewHistoryTitle')}</h2>
	{#if recent.length === 0}
		<p>{t('reviewHistoryNone')}</p>
	{:else}
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t('reviewRef')}</th>
						<th>{t('reviewSet')}</th>
						<th>{t('reviewDate')}</th>
						<th>{t('reviewRecordedByColumn')}</th>
						<th>{t('reviewConfirmedByColumn')}</th>
					</tr>
				</thead>
				<tbody>
					{#each recent as r (r.id)}
						<tr>
							<td>{r.ref}</td>
							<td>{setName(r)}</td>
							<td>{day(r.reviewed_on)}</td>
							<td>{r.recorded_by}</td>
							<td>{r.confirmed_by} · {r.confirmed_at ? formatDateTime(r.confirmed_at) : ''}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>

<style>
	.head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: flex-start;
		gap: 16px;
	}

	.head > div {
		flex: 1 1 420px;
	}

	.stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 16px;
		margin: 1rem 0 1.25rem;
	}

	.stat {
		padding: 18px 22px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
	}

	.stat h2 {
		margin: 0;
		font-size: 16px;
	}

	.stat p {
		margin: 0;
		color: var(--text-3);
	}

	.stat .big {
		margin: 4px 0;
		font-size: 34px;
		font-weight: 600;
		color: var(--ink);
	}

	.stat[data-tone='danger'] {
		background: var(--danger-weak);
		border-color: var(--danger-ring);
	}

	.stat[data-tone='danger'] h2,
	.stat[data-tone='danger'] .big {
		color: var(--danger);
	}

	.stat[data-tone='warning'] {
		background: var(--warning-weak);
		border-color: #fedf89;
	}

	.stat[data-tone='warning'] h2,
	.stat[data-tone='warning'] .big {
		color: var(--warning);
	}

	.filters {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px;
		margin-bottom: 12px;
	}

	.filters select {
		width: auto;
		min-width: 170px;
	}

	.check {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.table-card {
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
		overflow: hidden;
	}

	.scroll {
		overflow-x: auto;
	}

	.scroll table {
		width: 100%;
	}

	.empty,
	.more {
		margin: 0;
		padding: 14px 18px;
		color: var(--text-3);
	}

	tr[data-review='overdue'] {
		background: #fffbfa;
	}

	td.late {
		color: var(--danger);
		font-weight: 600;
	}

	.sub {
		display: block;
		color: var(--text-3);
		font-size: 14px;
	}

	td.late .sub {
		color: var(--danger);
	}

	.lower {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
		gap: 16px;
		margin: 1.25rem 0;
	}

	.card {
		padding: 20px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
		min-width: 0;
	}

	.clock {
		background: var(--surface-2);
	}

	.clock table {
		width: 100%;
	}

	.outcomes {
		display: flex;
		flex-direction: column;
		gap: 10px;
		border: none;
		padding: 0;
		margin: 0 0 12px;
	}

	.outcome {
		display: flex;
		gap: 12px;
		align-items: flex-start;
		padding: 14px 16px;
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
		cursor: pointer;
	}

	.outcome.chosen {
		border-color: var(--accent);
		background: var(--accent-weak);
	}

	.outcome input {
		margin-top: 4px;
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 16px;
	}

	.item {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
		align-items: center;
		padding: 6px 0;
		border-bottom: 1px solid var(--border);
	}

	.item > span {
		flex: 1 1 240px;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-top: 12px;
	}
</style>
