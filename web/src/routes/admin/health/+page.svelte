<script lang="ts">
	// Admin → Web health (fase 4, D-205): rincian dasbor admin. Tanda tangan
	// yang tertahan (yang paling lama dulu), error server, dan laporan masalah
	// dari pengguna beserta statusnya.
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { listErrorReports, listProblemReports, listWaitingSignatures, updateProblemReport } from '#lib/api/client.ts';
	import type { ErrorReport, ProblemReport, WaitingSignature } from '#lib/api/gen/index.ts';
	import { waited } from '#lib/admin/health.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';

	type Tab = 'signatures' | 'errors' | 'problems';

	let tab = $state<Tab>('signatures');
	let signatures = $state<WaitingSignature[]>([]);
	let errors = $state<ErrorReport[]>([]);
	let problems = $state<ProblemReport[]>([]);
	let edits = $state<Record<string, { status: ProblemReport['status']; note: string }>>({});
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');

	onMount(async () => {
		const wanted = page.url.searchParams.get('tab');
		if (wanted === 'errors' || wanted === 'problems' || wanted === 'signatures') tab = wanted;
		const [s, e, p] = await Promise.all([listWaitingSignatures(), listErrorReports(), listProblemReports()]);
		for (const r of [s, e, p]) {
			if (!r.ok) {
				loadError = failureText(r.error);
				return;
			}
		}
		if (s.ok) signatures = s.data.items;
		if (e.ok) errors = e.data.items;
		if (p.ok) setProblems(p.data.items);
	});

	function setProblems(list: ProblemReport[]) {
		problems = list;
		edits = Object.fromEntries(list.map((r) => [r.id, { status: r.status, note: r.admin_note ?? '' }]));
	}

	async function save(report: ProblemReport) {
		message = '';
		failure = '';
		const edit = edits[report.id];
		const result = await updateProblemReport(report.id, edit.status, edit.note);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		const fresh = await listProblemReports();
		if (fresh.ok) setProblems(fresh.data.items);
		message = t('adminSaved');
	}
</script>

<h1>{t('navAdminHealth')}</h1>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}
{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

<div class="segmented tabs" role="group" aria-label={t('navAdminHealth')}>
	<button type="button" aria-pressed={tab === 'signatures'} onclick={() => (tab = 'signatures')}>
		{t('adminHealthTabSignatures', { count: String(signatures.length) })}
	</button>
	<button type="button" aria-pressed={tab === 'errors'} onclick={() => (tab = 'errors')}>
		{t('adminHealthTabErrors', { count: String(errors.length) })}
	</button>
	<button type="button" aria-pressed={tab === 'problems'} onclick={() => (tab = 'problems')}>
		{t('adminHealthTabProblems', { count: String(problems.filter((p) => p.status !== 'done').length) })}
	</button>
</div>

<div class="scroll">
	{#if tab === 'signatures'}
		{#if signatures.length === 0}
			<p class="hint">{t('adminNothing')}</p>
		{:else}
			<table>
				<thead><tr><th>{t('adminDocument')}</th><th>{t('adminSlot')}</th><th>{t('adminWaitingFor')}</th><th>{t('signaturesSent')}</th><th>{t('adminSince')}</th><th>{t('adminWaited')}</th></tr></thead>
				<tbody>
					{#each signatures as w (w.request_id)}
						<tr>
							<td><span class="mono">{w.number}</span> · {w.revision} · {w.customer_code}</td>
							<td>{w.slot}</td>
							<td>{w.signer}</td>
							<td>{w.sent_by} · {formatDateTime(w.sent_at)}</td>
							<td>{formatDateTime(w.waiting_since)}</td>
							<td class="num">{waited(w.hours)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	{:else if tab === 'errors'}
		{#if errors.length === 0}
			<p class="hint">{t('adminNothing')}</p>
		{:else}
			<table>
				<thead><tr><th>{t('adminWhen')}</th><th>{t('adminRequestId')}</th><th>{t('adminAddress')}</th><th>{t('adminAccount')}</th><th>{t('adminMessage')}</th></tr></thead>
				<tbody>
					{#each errors as e (e.id)}
						<tr>
							<td>{formatDateTime(e.occurred_at)}</td>
							<td class="mono">{e.request_id}</td>
							<td>{e.method} {e.path}</td>
							<td>{e.login ?? ''}</td>
							<td class="message">{e.message}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	{:else if problems.length === 0}
		<p class="hint">{t('adminNothing')}</p>
	{:else}
		<table>
			<thead><tr><th>{t('adminWhen')}</th><th>{t('adminReportedBy')}</th><th>{t('adminPage')}</th><th>{t('adminDescription')}</th><th>{t('adminRequestId')}</th><th>{t('adminStatus')}</th><th>{t('adminNote')}</th><th></th></tr></thead>
			<tbody>
				{#each problems as p (p.id)}
					<tr>
						<td>{formatDateTime(p.created_at)}</td>
						<td>{p.reported_by} ({p.login})</td>
						<td>{p.page}</td>
						<td class="message">{p.description}</td>
						<td class="mono">{p.last_request_id ?? ''}</td>
						<td>
							<select aria-label={`${t('adminStatus')}: ${p.description}`} bind:value={edits[p.id].status}>
								{#each ['new', 'checking', 'done'] as s (s)}
									<option value={s}>{t(`problemStatus_${s}` as MessageKey)}</option>
								{/each}
							</select>
						</td>
						<td><input aria-label={`${t('adminNote')}: ${p.description}`} bind:value={edits[p.id].note} maxlength="2000" /></td>
						<td><button type="button" onclick={() => save(p)}>{t('adminSave')}</button></td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>

<style>
	.tabs {
		display: flex;
		flex-wrap: wrap;
		max-width: 100%;
	}

	.scroll {
		overflow-x: auto;
		margin-top: 1rem;
	}

	.scroll table {
		width: 100%;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	.num {
		text-align: right;
	}

	.message {
		max-width: 28rem;
		overflow-wrap: anywhere;
	}
</style>
