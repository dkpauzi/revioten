<script lang="ts">
	// Cetak (Tahap 6): pilih revisi dan tujuan, lalu Excel atau PDF. Draft
	// dicetak sebagai draft atau untuk tanda tangan (E-5, mengisi tanggal
	// rilis, D-59); revisi terbit sebagai referensi (L-3) atau salinan
	// terkendali bernomor (E-9). Setiap cetakan tercatat (M6.6).
	import { listDocumentPrints, listDocumentRevisions, printDocument, saveFile } from '#lib/api/client.ts';
	import type { Document, DocumentRevision, PrintLog } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let { doc }: { doc: Document } = $props();

	let revisions = $state<DocumentRevision[]>([]);
	let logs = $state<PrintLog[]>([]);
	let choice = $state('');
	let purpose = $state('draft');
	let busy = $state(false);
	let message = $state('');
	let failure = $state('');

	// selected kosong selama daftar revisi belum dimuat; tombol cetak menunggu
	// sampai ada, supaya tidak mengirim revisi yang salah.
	let selected = $derived(revisions.find((r) => String(r.revision) === choice));
	let released = $derived(selected?.status === 'released');
	let purposes = $derived(
		released
			? [
					{ key: 'reference', allowed: can('L-3') },
					{ key: 'controlled', allowed: can('E-9') }
				].filter((p) => p.allowed)
			: [
					{ key: 'draft', allowed: can('L-4') },
					{ key: 'signature', allowed: can('E-5') }
				].filter((p) => p.allowed)
	);

	$effect(() => {
		void doc.revision;
		void doc.status;
		load(doc.id);
	});

	async function load(id: string) {
		const [r, l] = await Promise.all([listDocumentRevisions(id), listDocumentPrints(id)]);
		revisions = r.ok ? r.data.items : [];
		logs = l.ok ? l.data.items : [];
		choice = String(doc.revision);
		const first = revisions.find((x) => x.revision === doc.revision);
		purpose = first?.status === 'released' ? 'reference' : 'draft';
	}

	function choose() {
		purpose = selected?.status === 'released' ? 'reference' : 'draft';
	}

	async function print(format: 'xlsx' | 'pdf') {
		message = '';
		failure = '';
		busy = true;
		const result = await printDocument(doc.id, { purpose, format, revision: Number(choice) });
		busy = false;
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		saveFile(result.data);
		message = t('printDone', { file: result.data.name });
		const l = await listDocumentPrints(doc.id);
		logs = l.ok ? l.data.items : logs;
	}
</script>

<section>
	<h2>{t('printTitle')}</h2>
	<p class="hint">{t('printHint')}</p>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<div class="row">
		<div class="field">
			<label for="print-revision">{t('changeRevision')}</label>
			<select id="print-revision" bind:value={choice} onchange={choose}>
				{#each revisions as r (r.id)}
					<option value={String(r.revision)}>{r.revision} · {t(r.status === 'released' ? 'documentStatusReleased' : 'versionStatusDraft')}</option>
				{/each}
			</select>
		</div>
		<div class="field">
			<label for="print-purpose">{t('printPurpose')}</label>
			<select id="print-purpose" bind:value={purpose}>
				{#each purposes as p (p.key)}<option value={p.key}>{t(`printPurpose_${p.key}` as 'printPurpose_draft')}</option>{/each}
			</select>
		</div>
		<button type="button" disabled={busy || !selected || purposes.length === 0} onclick={() => print('xlsx')}>{t('printExcel')}</button>
		<button type="button" disabled={busy || !selected || purposes.length === 0} onclick={() => print('pdf')}>{t('printPdf')}</button>
	</div>

	<h3>{t('printLog')}</h3>
	{#if logs.length === 0}
		<p>{t('printLogNone')}</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>{t('changeRevision')}</th>
					<th>{t('printPurpose')}</th>
					<th>{t('printFormat')}</th>
					<th>{t('printCopy')}</th>
					<th>{t('printBy')}</th>
				</tr>
			</thead>
			<tbody>
				{#each logs as l (l.id)}
					<tr>
						<td>{l.revision}</td>
						<td>{t(`printPurpose_${l.purpose}` as 'printPurpose_draft')}</td>
						<td>{l.format === 'pdf' ? 'PDF' : 'Excel'}</td>
						<td>{l.copy_number ?? ''}</td>
						<td>{l.printed_by} · {formatDateTime(l.printed_at)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</section>

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1rem;
		align-items: flex-start;
	}

	.row > button {
		margin-top: var(--label-space);
	}
</style>
