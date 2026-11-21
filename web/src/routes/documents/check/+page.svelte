<script lang="ts">
	// Tab Check (M4.9, R-DOC-06, D-65): PFMEA dan CP satu set berdampingan,
	// per proses. Konsistensi keduanya diperiksa orang; pemegang V-2 (untuk
	// saat ini PPAP Engineer) memberi tanda "dicek oleh, tanggal" per proses.
	// Tanda berlaku untuk revisi kedua dokumen saat itu, dan ditandai perlu
	// dicek ulang bila baris proses itu berubah sesudahnya.
	//
	// Dengan ?released=1 halaman ini membaca PFMEA dan CP revisi yang berlaku
	// (D-195): untuk akun Viewer, tanpa draft, dan tanda dicek tidak bisa diubah.
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { getFormatVersion, getPfmeaCpComparison, getReleasedPfmeaCpComparison, markPfmeaCpChecked } from '#lib/api/client.ts';
	import type { DocumentRow, FormatField, FormatVersion, PfmeaCpComparison, PfmeaCpProcess } from '#lib/api/gen/index.ts';
	import { cellText } from '#lib/documents.ts';
	import { failureText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';
	import SheetTabs from '#lib/documents/SheetTabs.svelte';

	let comparison = $state<PfmeaCpComparison | null>(null);
	let formats = $state<{ pfmea: FormatVersion; cp: FormatVersion } | null>(null);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let busy = $state('');
	let currentId = $derived(page.url.searchParams.get('id') ?? '');
	let released = $derived(page.url.searchParams.get('released') === '1');
	let canCheck = $derived(can('V-2') && !released);
	// Alamat dokumen: halaman baca terbit atau workbook draft.
	let viewPath = $derived(released ? '/documents/released/view' : '/documents/view');
	$effect(() => {
		const id = currentId;
		const readOnly = released;
		untrack(() => load(id, readOnly));
	});

	async function load(id: string, readOnly: boolean) {
		comparison = null;
		loadError = '';
		const result = readOnly ? await getReleasedPfmeaCpComparison(id) : await getPfmeaCpComparison(id);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			loadError = details.some((d) => d.field === 'set') ? t('checkNoPair') : failureText(result.error);
			return;
		}
		const [pfmea, cp] = await Promise.all([getFormatVersion(result.data.pfmea.format_version_id), getFormatVersion(result.data.cp.format_version_id)]);
		if (!pfmea.ok) {
			loadError = failureText(pfmea.error);
			return;
		}
		if (!cp.ok) {
			loadError = failureText(cp.error);
			return;
		}
		formats = { pfmea: pfmea.data, cp: cp.data };
		comparison = result.data;
	}

	// shownFields memberi kolom baris yang ditampilkan: yang terlihat, kecuali
	// nomor operasi dan proses (sudah menjadi judul kelompok).
	function shownFields(format: FormatVersion): FormatField[] {
		return format.fields.filter((f) => f.area === 'row' && !f.hidden && f.kind !== 'process' && f.kind !== 'process_flow');
	}

	function rowsOf(ids: string[], rows: DocumentRow[]): DocumentRow[] {
		const byId = new Map(rows.map((r) => [r.id, r]));
		return ids.map((id) => byId.get(id)).filter((r): r is DocumentRow => r !== undefined);
	}

	async function mark(group: PfmeaCpProcess, checked: boolean) {
		if (!comparison || !group.process_id) {
			return;
		}
		message = '';
		failure = '';
		busy = group.process_id;
		const result = await markPfmeaCpChecked(currentId, group.process_id, checked, group.section_key ?? '');
		busy = '';
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		comparison = result.data;
		const name = group.process_name ?? '';
		message = checked ? t('checkMarked', { process: name }) : t('checkCleared', { process: name });
	}
</script>

{#if loadError}
	<h1>{t('checkTitle')}</h1>
	<p class="error" role="alert">{loadError}</p>
	<p><a href="{viewPath}?id={currentId}">{t('checkBack')}</a></p>
{:else if !comparison || !formats}
	<p>{t('formatLoading')}</p>
{:else}
	{@const pfmeaFields = shownFields(formats.pfmea)}
	{@const cpFields = shownFields(formats.cp)}
	<p><a href={released ? '/documents/released' : '/documents'}>{t(released ? 'releasedBack' : 'documentBack')}</a></p>
	<h1>{t('checkTitle')}</h1>
	<p class="hint">{t(released ? 'releasedCheckHint' : 'checkHint')}</p>

	<!-- Tab sheet workbook di bawah layar (D-145, C1): Cover, PFD, PFMEA, CP, lalu Check. -->
	<SheetTabs set={comparison.cp.set} documentId={currentId} current="check" {viewPath} {released} />

	<p class="pair">
		<span>{t('checkPfmea', { number: comparison.pfmea.number, revision: String(comparison.pfmea.revision) })}</span>
		<span>{t('checkCp', { number: comparison.cp.number, revision: String(comparison.cp.revision) })}</span>
	</p>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if comparison.processes.length === 0}
		<p>{t('checkEmpty')}</p>
	{/if}
	{#each comparison.processes as group, i ((group.section_key ?? '') + '/' + (group.process_id ?? '-'))}
		<!-- Judul bagian (D-155) tampil sekali, di atas proses pertama bagian itu. -->
		{#if group.section_key && group.section_key !== comparison.processes[i - 1]?.section_key}
			<h2 class="section-heading">{group.section_name}</h2>
		{/if}
		<section class="process" aria-label={(group.section_name ? group.section_name + ' · ' : '') + (group.process_name || t('checkNoProcess'))}>
			<header>
				<h2>{group.process_name || t('checkNoProcess')}</h2>
				{#if group.process_id}
					{#if group.check}
						<span class="badge" data-check={group.check.stale ? 'stale' : 'checked'} data-testid="check-mark">
							{group.check.stale
								? t('checkStale', { by: group.check.checked_by, at: formatDateTime(group.check.checked_at) })
								: t('checkDone', { by: group.check.checked_by, at: formatDateTime(group.check.checked_at) })}
						</span>
					{:else}
						<span class="badge" data-testid="check-mark">{t('checkNotYet')}</span>
					{/if}
					{#if canCheck}
						{#if !group.check || group.check.stale}
							<button
								type="button"
								class="primary"
								disabled={busy !== ''}
								aria-label={t('checkMarkFor', { process: group.process_name ?? '' })}
								onclick={() => mark(group, true)}>{t('checkMark')}</button
							>
						{/if}
						{#if group.check}
							<button
								type="button"
								disabled={busy !== ''}
								aria-label={t('checkClearFor', { process: group.process_name ?? '' })}
								onclick={() => mark(group, false)}>{t('checkClear')}</button
							>
						{/if}
					{/if}
				{:else}
					<span class="hint">{t('checkNoProcessHint')}</span>
				{/if}
			</header>
			<div class="sides">
				{#each [{ key: 'pfmea', name: 'PFMEA', ids: group.pfmea_row_ids, rows: comparison.pfmea.rows, fields: pfmeaFields }, { key: 'cp', name: 'CP', ids: group.cp_row_ids, rows: comparison.cp.rows, fields: cpFields }] as side (side.key)}
					<div class="side" data-testid="check-{side.key}">
						<h3>{side.name}</h3>
						{#if side.ids.length === 0}
							<p class="hint">{t('checkNoRows')}</p>
						{/if}
						{#each rowsOf(side.ids, side.rows) as row (row.id)}
							<dl class="row">
								{#each side.fields as f (f.id)}
									{@const text = cellText(f, row)}
									{#if text !== ''}
										<dt>{f.label}</dt>
										<dd>{text}</dd>
									{/if}
								{/each}
							</dl>
						{/each}
					</div>
				{/each}
			</div>
		</section>
	{/each}
{/if}

<style>

	.pair {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 24px;
		color: var(--text-2);
		font-family: 'Geist Mono Variable', monospace;
	}

	.section-heading {
		margin: 1.5rem 0 0.5rem;
		padding-bottom: 4px;
		border-bottom: 2px solid var(--border);
	}

	.process header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 12px;
		margin-bottom: 12px;
	}

	.process h2 {
		margin: 0;
		flex: 1 1 12rem;
	}

	.badge[data-check='checked'] {
		background: var(--success-weak);
		color: var(--success);
	}

	.badge[data-check='stale'] {
		background: var(--warning-weak);
		color: var(--warning);
	}

	.sides {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 24px;
	}

	/* Di HP dan tablet kecil kedua sisi bersusun (D-128). */
	@media (max-width: 899px) {
		.sides {
			grid-template-columns: 1fr;
		}
	}

	.side {
		min-width: 0;
	}

	.side h3 {
		margin-top: 0;
	}

	.row {
		display: grid;
		grid-template-columns: minmax(8rem, max-content) 1fr;
		gap: 2px 12px;
		margin: 0 0 12px;
		padding: 8px 12px;
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}

	.row dt {
		color: var(--text-3);
	}

	.row dd {
		margin: 0;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
</style>
