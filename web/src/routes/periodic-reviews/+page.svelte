<script lang="ts">
	// Review berkala 3 tahun (D-138, D-196). Set dokumen (PFD, PFMEA, CP satu
	// nomor berjalan) yang tidak direvisi selama satu siklus muncul di sini:
	// kuning dalam masa peringatan, merah bila lewat jatuh tempo. Dua hasil:
	//   - dengan perubahan: permintaan perubahan biasa, pemicu Periodic review;
	//   - tanpa perubahan: catatan review dengan daftar periksa, dikonfirmasi
	//     orang kedua (V-1), lalu jam review mulai lagi.
	import { onMount } from 'svelte';
	import { confirmPeriodicReview, listCustomers, listPeriodicReviews, listReviewSets, recordPeriodicReview } from '#lib/api/client.ts';
	import type { Customer, PeriodicReview, ReviewAnswer, ReviewSet, ReviewSetList } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	const items: ReviewAnswer['item'][] = ['process_flow', 'claims_findings', 'occurrence_detection', 'special_characteristics', 'control_methods', 'references', 'reaction_plan', 'pfmea_cp_sync'];

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let showAll = $state(false);
	let overview = $state<ReviewSetList | null>(null);
	let waiting = $state<PeriodicReview[]>([]);
	let recent = $state<PeriodicReview[]>([]);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let busy = $state(false);

	// Formulir "Reviewed, no change" untuk satu set.
	let open = $state<ReviewSet | null>(null);
	let reviewedOn = $state('');
	let participants = $state('');
	let conclusion = $state('');
	let answers = $state<Record<string, ReviewAnswer['answer'] | ''>>({});

	let canRecord = $derived(can('E-3'));
	let canConfirm = $derived(can('V-1'));

	// errorText memberi pesan masalah pertama dari server, atau pesan umum.
	function errorText(error: Parameters<typeof failureText>[0]): string {
		const details = error.body?.details ?? [];
		return details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	function setName(s: { customer_code: string; sequence: number }): string {
		return `${s.customer_code.toUpperCase()} · ${String(s.sequence).padStart(4, '0')}`;
	}

	function statusText(s: ReviewSet): string {
		if (s.status === 'overdue') return t('reviewOverdue', { days: String(-s.days_left) });
		if (s.status === 'due_soon') return t('reviewDueIn', { days: String(s.days_left) });
		return t('reviewOk');
	}

	async function load() {
		const [sets, open, all] = await Promise.all([listReviewSets({ customerId, all: showAll }), listPeriodicReviews(true), listPeriodicReviews(false)]);
		if (!sets.ok) {
			loadError = failureText(sets.error);
			return;
		}
		loadError = '';
		overview = sets.data;
		waiting = open.ok ? open.data.items : [];
		recent = all.ok ? all.data.items.filter((r) => r.status === 'confirmed').slice(0, 20) : [];
	}

	function today(): string {
		const d = new Date();
		return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
	}

	function start(s: ReviewSet) {
		open = s;
		reviewedOn = today();
		participants = '';
		conclusion = '';
		answers = Object.fromEntries(items.map((i) => [i, '']));
		message = '';
		failure = '';
	}

	async function record(event: SubmitEvent) {
		event.preventDefault();
		if (!open) return;
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

	// changeLink membuka formulir perubahan dengan customer dan dokumen set itu.
	function changeLink(s: ReviewSet): string {
		const query = new URLSearchParams({ customer: s.customer_id, trigger: 'Periodic review', documents: s.documents.map((d) => d.id).join(',') });
		return '/changes?' + query.toString();
	}

	onMount(async () => {
		const result = await listCustomers();
		customers = result.ok ? result.data.items : [];
		await load();
	});
</script>

<h1>{t('reviewTitle')}</h1>
<p class="hint">{t('reviewHint')}</p>
{#if overview}
	<p class="hint">{t('reviewRule', { months: String(overview.period_months), days: String(overview.warning_days) })}</p>
{/if}

<div class="filters">
	<div class="field">
		<label for="review-customer">{t('documentCustomer')}</label>
		<select id="review-customer" bind:value={customerId} onchange={load}>
			<option value="">{t('dashboardAllCustomers')}</option>
			{#each customers as c (c.id)}
				<option value={c.id}>{c.code} · {c.name}</option>
			{/each}
		</select>
	</div>
	<label class="check"><input type="checkbox" bind:checked={showAll} onchange={load} /> {t('reviewShowAll')}</label>
</div>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}
{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

{#if overview}
	<p class="counts">
		<span class="badge" data-review="overdue">{t('reviewCountOverdue', { count: String(overview.overdue) })}</span>
		<span class="badge" data-review="due_soon">{t('reviewCountDueSoon', { count: String(overview.due_soon) })}</span>
	</p>
	{#if overview.items.length === 0}
		<p>{t('reviewNone')}</p>
	{:else}
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t('reviewSet')}</th>
						<th>{t('reviewDocuments')}</th>
						<th>{t('documentModels')}</th>
						<th>{t('reviewLastRelease')}</th>
						<th>{t('reviewDueOn')}</th>
						<th>{t('numberColumnStatus')}</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each overview.items as s (s.customer_code + s.sequence)}
						<tr>
							<td>{setName(s)}</td>
							<td>
								{#each s.documents as d (d.id)}
									<div><a href="/documents/released/view?id={d.id}">{d.number}</a> · rev {d.revision}</div>
								{/each}
							</td>
							<td>{s.models.join(', ')}</td>
							<td>{s.last_review && s.last_review > s.last_release ? `${s.last_review} (${t('reviewLastReview')})` : s.last_release}</td>
							<td>{s.due_on}</td>
							<td><span class="badge" data-review={s.status}>{statusText(s)}</span></td>
							<td class="actions">
								{#if s.open_review_id}
									<span class="hint">{t('reviewWaitingConfirmation')}</span>
								{:else if canRecord}
									<a class="button" href={changeLink(s)}>{t('reviewWithChanges')}</a>
									<button type="button" onclick={() => start(s)}>{t('reviewNoChange')}</button>
								{/if}
								{#if s.draft_open}<span class="hint">{t('reviewDraftOpen')}</span>{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{/if}

{#if open}
	<section class="card">
		<h2>{t('reviewFormTitle', { set: setName(open) })}</h2>
		<p class="hint">{t('reviewFormHint')}</p>
		<form onsubmit={record} novalidate>
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
				<p class="hint">{t('reviewNeedsChangeHint')}</p>
			</fieldset>
			<div class="field">
				<label for="review-conclusion">{t('reviewConclusion')}</label>
				<textarea id="review-conclusion" rows="3" bind:value={conclusion}></textarea>
			</div>
			<button type="submit" class="primary" disabled={busy}>{t('reviewSave')}</button>
			<button type="button" onclick={() => (open = null)}>{t('reviewCancel')}</button>
		</form>
	</section>
{/if}

<section>
	<h2>{t('reviewWaitingTitle')}</h2>
	{#if waiting.length === 0}
		<p>{t('reviewWaitingNone')}</p>
	{:else}
		{#each waiting as r (r.id)}
			<article class="card" aria-label={r.ref}>
				<h3>{r.ref} · {setName(r)}</h3>
				<p>{t('reviewRecordedBy', { by: r.recorded_by, on: r.reviewed_on })}</p>
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
							<td>{r.reviewed_on}</td>
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
	.filters {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 16px;
		margin-bottom: 1rem;
	}

	.filters .field {
		min-width: 220px;
	}

	.check {
		display: flex;
		align-items: center;
		gap: 6px;
		padding-bottom: 10px;
	}

	.counts {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	.scroll {
		overflow-x: auto;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		align-items: center;
	}

	.card {
		margin: 1rem 0;
		padding: 18px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
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
		flex: 1 1 260px;
	}
</style>
