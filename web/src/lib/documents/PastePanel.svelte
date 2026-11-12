<script lang="ts">
	// Tempel dari Excel (M4.4, R-MIG-02): penyusun menempel blok sel, melihat
	// pratinjau, mengatur kolom tujuan, lalu mengimpor. Sel janggal ditandai
	// tetapi barisnya tetap masuk (ASUMSI E-91). Belum ada yang disimpan
	// sebelum tombol Import ditekan.
	import type { DocumentRowInput, FormatVersion, Process } from '#lib/api/gen/index.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { buildRows, guessMapping, parseClipboard, pasteTargets, type PasteReason } from '#lib/paste.ts';

	let {
		format,
		processes,
		onimport,
		oncancel
	}: {
		format: FormatVersion;
		processes: Process[];
		// onimport menyimpan baris; true bila berhasil.
		onimport: (rows: DocumentRowInput[]) => Promise<boolean>;
		oncancel: () => void;
	} = $props();

	let text = $state('');
	let hasHeader = $state(false);
	let problem = $state('');
	let busy = $state(false);
	// grid adalah sel tempelan (tanpa baris judul); null sebelum pratinjau.
	let grid = $state<string[][] | null>(null);
	let mapping = $state<string[]>([]);

	let targets = $derived(pasteTargets(format));
	let built = $derived(grid ? buildRows(grid, mapping, format, processes) : null);
	// Dua kolom tempelan tidak boleh menuju kolom dokumen yang sama.
	let duplicate = $derived(mapping.some((m, i) => m !== '' && mapping.indexOf(m) !== i));

	const reasonKeys: Record<PasteReason, MessageKey> = {
		rating: 'pasteReasonRating',
		date: 'pasteReasonDate',
		option: 'pasteReasonOption',
		symbol: 'pasteReasonSymbol',
		process: 'pasteReasonProcess',
		too_long: 'pasteReasonTooLong'
	};

	function reasonOf(row: number, column: number): string {
		const flag = built?.flags.find((f) => f.row === row && f.column === column);
		return flag ? t(reasonKeys[flag.reason]) : '';
	}

	function preview(event: SubmitEvent) {
		event.preventDefault();
		problem = '';
		const cells = parseClipboard(text);
		const body = hasHeader ? cells.slice(1) : cells;
		if (body.length === 0) {
			grid = null;
			problem = t('pasteEmpty');
			return;
		}
		const width = Math.max(...cells.map((r) => r.length));
		mapping = guessMapping(hasHeader ? cells[0] : null, targets, width);
		// Setiap baris dibuat selebar baris terpanjang, supaya tabel pratinjau rata.
		grid = body.map((r) => [...r, ...Array(width - r.length).fill('')]);
	}

	async function run() {
		if (!built || duplicate) {
			return;
		}
		busy = true;
		const ok = await onimport(built.rows);
		busy = false;
		if (ok) {
			text = '';
			grid = null;
		}
	}
</script>

<h2>{t('pasteTitle')}</h2>
<p class="hint">{t('pasteHint')}</p>
<form onsubmit={preview} novalidate>
	<div class="field">
		<label for="paste-text">{t('pasteText')}</label>
		<textarea id="paste-text" rows="6" bind:value={text}></textarea>
	</div>
	<p>
		<label class="check"><input type="checkbox" bind:checked={hasHeader} /> {t('pasteHasHeader')}</label>
	</p>
	{#if problem}<p class="error" role="alert">{problem}</p>{/if}
	<p class="actions">
		<button type="submit">{t('pastePreview')}</button>
		<button type="button" onclick={oncancel}>{t('pasteCancel')}</button>
	</p>
</form>

{#if grid && built}
	<h3>{t('pastePreviewTitle')}</h3>
	<p data-testid="paste-summary">
		{built.flags.length === 0
			? t('pasteSummaryClean', { rows: String(built.rows.length) })
			: t('pasteSummary', { rows: String(built.rows.length), flags: String(built.flags.length) })}
	</p>
	{#if duplicate}<p class="error" role="alert">{t('pasteDuplicate')}</p>{/if}
	<table class="preview">
		<thead>
			<tr>
				<th>#</th>
				{#each mapping as _, c (c)}
					<th>
						<select aria-label={t('pasteColumn', { n: String(c + 1) })} bind:value={mapping[c]}>
							<option value="">{t('pasteSkip')}</option>
							{#each targets as target (target.id)}<option value={target.id}>{target.label}</option>{/each}
						</select>
					</th>
				{/each}
			</tr>
		</thead>
		<tbody>
			{#each grid as cells, r (r)}
				<tr>
					<td>{r + 1}</td>
					{#each cells as cell, c (c)}
						{@const reason = reasonOf(r, c)}
						<td class:flag={reason !== ''} class:skipped={mapping[c] === ''}>
							{cell}
							{#if reason}<small>{reason}</small>{/if}
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>
	<p class="actions">
		<button type="button" class="primary" onclick={run} disabled={busy || duplicate}>{t('pasteImport', { rows: String(built.rows.length) })}</button>
	</p>
{/if}

<style>
	textarea {
		width: 100%;
		font-family: 'Geist Mono Variable', monospace;
		font-size: 13px;
	}

	.check {
		display: inline-flex;
		gap: 8px;
		align-items: center;
	}

	.actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	.preview td {
		white-space: pre-wrap;
		vertical-align: top;
	}

	.preview select {
		min-width: 9rem;
	}

	.preview td.skipped {
		color: var(--text-3);
	}

	.preview td.flag {
		background: var(--warning-weak);
	}

	.preview td.flag small {
		display: block;
		color: var(--warning);
	}
</style>
