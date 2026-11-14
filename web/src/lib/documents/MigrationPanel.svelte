<script lang="ts">
	// Baseline migrasi (Tahap 7): data hardcopy dokumen lama, riwayat revisi
	// dari tabel lama sebagai teks (M7.2), dan verifikasi terhadap hardcopy
	// bertanda tangan oleh petugas migrasi lain (M7.5). Semua butuh E-10 dan
	// mode migrasi yang menyala; server tetap memeriksa setiap operasi.
	import { setDocumentHistory, verifyMigration } from '#lib/api/client.ts';
	import type { Document, HistoryEntry } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let { doc, onchanged }: { doc: Document; onchanged: (doc: Document) => void } = $props();

	let baseline = $derived(doc.baseline);
	let open = $derived(baseline !== undefined && baseline.verified_at === undefined && doc.status === 'draft');
	let canMigrate = $derived(can('E-10') && open);

	let entries = $state<{ revision: string; date: string; description: string }[]>([]);
	// Input bertipe number memberi angka, atau null bila kosong.
	let effort = $state<number | string | null>(null);
	let message = $state('');
	let failure = $state('');

	// Isian riwayat mengikuti dokumen setiap kali dokumen dimuat ulang.
	$effect(() => {
		entries = (doc.baseline?.history ?? []).map((h: HistoryEntry) => ({ revision: h.revision, date: h.date ?? '', description: h.description }));
	});

	function fail(error: Parameters<typeof failureText>[0]) {
		const details = error.body?.details ?? [];
		failure = details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	async function saveHistory(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		const result = await setDocumentHistory(
			doc.id,
			entries.map((e) => ({ revision: e.revision, date: e.date || undefined, description: e.description }))
		);
		if (!result.ok) {
			fail(result.error);
			return;
		}
		message = t('historySaved');
		onchanged(result.data);
	}

	async function verify() {
		message = '';
		failure = '';
		if (!confirm(t('migrationVerifyConfirm', { number: doc.number }))) {
			return;
		}
		const minutes = effort === null || String(effort).trim() === '' ? undefined : Number(effort);
		const result = await verifyMigration(doc.id, minutes);
		if (!result.ok) {
			fail(result.error);
			return;
		}
		message = t('migrationVerified', { number: result.data.number });
		onchanged(result.data);
	}
</script>

{#if baseline}
	<section>
		<h2>{t('migrationTitle')}</h2>
		{#if open}<p class="hint">{t('migrationBaselineHint')}</p>{/if}
		{#if message}<p role="status">{message}</p>{/if}
		{#if failure}<p class="error" role="alert">{failure}</p>{/if}
		<dl class="facts">
			<dt>{t('migrateRevision')}</dt>
			<dd>{baseline.revision}</dd>
			<dt>{t('migrateEffectiveDate')}</dt>
			<dd>{baseline.effective_date}</dd>
			<dt>{t('migratePreparedBy')}</dt>
			<dd>{baseline.prepared_by}</dd>
			<dt>{t('migrateApprovedBy')}</dt>
			<dd>{baseline.approved_by}</dd>
			<dt>{t('migrationEnteredBy')}</dt>
			<dd>{baseline.created_by} · {formatDateTime(baseline.created_at)}</dd>
			{#if baseline.verified_by && baseline.verified_at}
				<dt>{t('migrationVerifiedBy')}</dt>
				<dd>{baseline.verified_by} · {formatDateTime(baseline.verified_at)}</dd>
			{/if}
			{#if baseline.effort_minutes !== undefined}
				<dt>{t('migrationEffort')}</dt>
				<dd>{baseline.effort_minutes}</dd>
			{/if}
		</dl>

		<h3>{t('historyTitle')}</h3>
		<p class="hint">{t('historyHint')}</p>
		{#if canMigrate}
			<form onsubmit={saveHistory} novalidate>
				<div class="scroll">
					<table>
						<thead>
							<tr>
								<th>{t('historyRevision')}</th>
								<th>{t('historyDate')}</th>
								<th>{t('historyDescription')}</th>
								<th></th>
							</tr>
						</thead>
						<tbody>
							{#each entries as e, i (i)}
								<tr>
									<td><input type="text" class="short" aria-label="{t('historyRevision')} {i + 1}" bind:value={e.revision} /></td>
									<td><input type="date" aria-label="{t('historyDate')} {i + 1}" bind:value={e.date} /></td>
									<td><input type="text" aria-label="{t('historyDescription')} {i + 1}" bind:value={e.description} /></td>
									<td><button type="button" onclick={() => (entries = entries.filter((_, at) => at !== i))}>{t('historyRemove')}</button></td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<p class="actions">
					<button type="button" onclick={() => (entries = [...entries, { revision: '', date: '', description: '' }])}>{t('historyAdd')}</button>
					<button type="submit">{t('historySave')}</button>
				</p>
			</form>

			<h3>{t('migrationVerify')}</h3>
			<div class="field effort">
				<label for="migration-effort">{t('migrationEffort')}</label>
				<input id="migration-effort" type="number" min="1" bind:value={effort} aria-describedby="migration-effort-hint" />
				<p id="migration-effort-hint" class="hint">{t('migrationEffortHint')}</p>
			</div>
			<button type="button" class="primary" onclick={verify}>{t('migrationVerify')}</button>
		{:else if baseline.history.length === 0}
			<p>{t('historyNone')}</p>
		{:else}
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('historyRevision')}</th>
							<th>{t('historyDate')}</th>
							<th>{t('historyDescription')}</th>
						</tr>
					</thead>
					<tbody>
						{#each baseline.history as h, i (i)}
							<tr>
								<td>{h.revision}</td>
								<td>{h.date ?? ''}</td>
								<td>{h.description}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
{/if}

<style>
	.scroll {
		overflow-x: auto;
	}

	.short {
		width: 6rem;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.effort {
		max-width: 20rem;
	}
</style>
