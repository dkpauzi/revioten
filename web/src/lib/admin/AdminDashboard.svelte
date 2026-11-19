<script lang="ts">
	// Dasbor admin (fase 4, D-205): Home untuk akun admin. Kesehatan web, bukan
	// isi dokumen: tanda tangan tertahan, error server, laporan masalah,
	// dokumen yang diproses, email, akun, backup, dan rantai audit log.
	import { onMount } from 'svelte';
	import { getAdminHealth } from '#lib/api/client.ts';
	import type { AdminHealth, ProcessedPeriod } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { waited } from '#lib/admin/health.ts';

	let report = $state<AdminHealth | null>(null);
	let loadError = $state('');

	onMount(async () => {
		const result = await getAdminHealth();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		report = result.data;
	});

	function bars(p: ProcessedPeriod) {
		const max = Math.max(1, ...p.counts.map((c) => c.count));
		return p.counts.map((c) => ({ ...c, width: `${(c.count / max) * 100}%` }));
	}

	function category(key: string): string {
		return t(`performanceCategory_${key}` as MessageKey);
	}
</script>

<h1>{t('adminHomeTitle')}</h1>
<p class="lead">{t('adminHomeLead')}</p>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

{#if report}
	<div class="cards">
		<a class="card stat" class:warn={report.signatures_waiting > 0} href="/admin/health?tab=signatures">
			<span class="label">{t('adminSignaturesWaiting')}</span>
			<strong>{report.signatures_waiting}</strong>
		</a>
		<a class="card stat" class:bad={report.errors_24h > 0} href="/admin/health?tab=errors">
			<span class="label">{t('adminErrors')}</span>
			<strong>{report.errors_24h}</strong>
			<small>{t('adminErrorsCount', { day: String(report.errors_24h), week: String(report.errors_7d) })}</small>
		</a>
		<a class="card stat" class:warn={report.problems_open > 0} href="/admin/health?tab=problems">
			<span class="label">{t('adminProblemsOpen')}</span>
			<strong>{report.problems_open}</strong>
		</a>
		<a class="card stat" class:bad={report.email_failed > 0} href="/admin/notifications">
			<span class="label">{t('adminEmail')}</span>
			<strong>{report.email_failed}</strong>
			<small>{t('adminEmailCount', { failed: String(report.email_failed), pending: String(report.email_pending) })}</small>
		</a>
		<a class="card stat" class:warn={report.accounts_locked > 0} href="/admin/accounts">
			<span class="label">{t('adminAccounts')}</span>
			<strong>{report.accounts_locked}</strong>
			<small>{t('adminAccountsCount', { locked: String(report.accounts_locked), disabled: String(report.accounts_disabled), failed: String(report.failed_logins_24h) })}</small>
		</a>
		<div class="card stat" class:warn={!report.backup_found}>
			<span class="label">{t('adminBackup')}</span>
			<strong class="small">
				{#if !report.backup_configured}{t('adminBackupNotConfigured')}{:else if !report.backup_found || !report.backup_at}{t('adminBackupNone')}{:else}{formatDateTime(report.backup_at)}{/if}
			</strong>
		</div>
		<a class="card stat" class:bad={!report.audit_valid} href="/admin/audit">
			<span class="label">{t('adminAudit')}</span>
			<strong class="small">{report.audit_valid ? t('adminAuditValid', { entries: String(report.audit_entries) }) : t('adminAuditBroken')}</strong>
		</a>
	</div>

	<section aria-labelledby="sig-h">
		<div class="head"><h2 id="sig-h">{t('adminTopSignatures')}</h2><a href="/admin/health?tab=signatures">{t('adminSeeAll')}</a></div>
		{#if report.top_signatures.length === 0}
			<p class="hint">{t('adminNothing')}</p>
		{:else}
			<div class="scroll">
				<table>
					<thead><tr><th>{t('adminDocument')}</th><th>{t('adminSlot')}</th><th>{t('adminWaitingFor')}</th><th>{t('adminSince')}</th><th>{t('adminWaited')}</th></tr></thead>
					<tbody>
						{#each report.top_signatures as w (w.request_id)}
							<tr>
								<td><span class="mono">{w.number}</span> · {w.revision} · {w.customer_code}</td>
								<td>{w.slot}</td>
								<td>{w.signer}</td>
								<td>{formatDateTime(w.waiting_since)}</td>
								<td class="num">{waited(w.hours)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<div class="pair">
		<section aria-labelledby="err-h">
			<div class="head"><h2 id="err-h">{t('adminLatestErrors')}</h2><a href="/admin/health?tab=errors">{t('adminSeeAll')}</a></div>
			{#if report.latest_errors.length === 0}
				<p class="hint">{t('adminNothing')}</p>
			{:else}
				<ul class="list">
					{#each report.latest_errors as e (e.id)}
						<li><span class="mono">{formatDateTime(e.occurred_at)}</span> · {e.method} {e.path} · <span class="mono">{e.request_id}</span></li>
					{/each}
				</ul>
			{/if}
		</section>
		<section aria-labelledby="prob-h">
			<div class="head"><h2 id="prob-h">{t('adminLatestProblems')}</h2><a href="/admin/health?tab=problems">{t('adminSeeAll')}</a></div>
			{#if report.latest_problems.length === 0}
				<p class="hint">{t('adminNothing')}</p>
			{:else}
				<ul class="list">
					{#each report.latest_problems as p (p.id)}
						<li>
							<span class="badge" data-problem={p.status}>{t(`problemStatus_${p.status}` as MessageKey)}</span>
							{p.reported_by} · {p.page} · {p.description}
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	</div>

	<section aria-labelledby="proc-h">
		<h2 id="proc-h">{t('adminProcessed')}</h2>
		<div class="pair">
			{#each [{ key: 'adminProcessedWeek', period: report.week }, { key: 'adminProcessedMonth', period: report.month }] as block (block.key)}
				<div>
					<h3>{t(block.key as MessageKey, { from: block.period.from, to: block.period.to, total: String(block.period.total) })}</h3>
					<dl class="bars">
						{#each bars(block.period) as c (c.category)}
							<dt>{category(c.category)}</dt>
							<dd><span class="bar" style="width: {c.width}"></span><span class="num">{c.count}</span></dd>
						{/each}
					</dl>
				</div>
			{/each}
		</div>
	</section>
{/if}

<style>
	.cards {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
		gap: 0.75rem;
		margin: 1rem 0 1.5rem;
	}

	.stat {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		padding: 0.75rem 1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: inherit;
		text-decoration: none;
	}

	a.stat:hover {
		border-color: var(--border-strong);
	}

	.stat strong {
		font-size: 1.75rem;
		line-height: 1.1;
	}

	.stat strong.small {
		font-size: 1rem;
	}

	.stat .label {
		color: var(--text-3);
		font-size: 0.85rem;
	}

	.stat.warn strong {
		color: var(--warning);
	}

	.stat.bad strong {
		color: var(--danger);
	}

	.head {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.pair {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr));
		gap: 1.5rem;
	}

	.list {
		padding-left: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		overflow-wrap: anywhere;
	}

	.scroll {
		overflow-x: auto;
	}

	.scroll table {
		width: 100%;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	.num {
		text-align: right;
		font-variant-numeric: tabular-nums;
	}

	.bars {
		display: grid;
		grid-template-columns: minmax(8rem, auto) 1fr;
		gap: 0.25rem 0.75rem;
		align-items: center;
	}

	.bars dd {
		margin: 0;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.bar {
		display: inline-block;
		height: 0.6rem;
		min-width: 2px;
		border-radius: 3px;
		background: var(--accent);
	}
</style>
