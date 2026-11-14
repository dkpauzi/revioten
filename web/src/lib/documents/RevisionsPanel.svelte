<script lang="ts">
	// Revisi dokumen (M5.4, M5.5): daftar revisi dengan status, pencatat rilis,
	// dan SHA-256 snapshot. Pemegang E-7 (Dokumen kontrol) mencatat rilis draft
	// yang terbuka; isinya dibekukan dan tidak bisa diubah lagi (R-REV-02).
	// Sejak M6.7 rilis butuh scan bertanda tangan yang diunggah lebih dulu.
	import { listDocumentRevisions, releaseDocument, signedScanUrl, uploadSignedScan } from '#lib/api/client.ts';
	import DiffPanel from '#lib/documents/DiffPanel.svelte';
	import type { Document, DocumentRevision } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let { doc, onreleased }: { doc: Document; onreleased: (doc: Document) => void } = $props();

	let items = $state<DocumentRevision[]>([]);
	let message = $state('');
	let failure = $state('');
	let canRelease = $derived(can('E-7') && doc.status === 'draft');
	let draft = $derived(items.find((r) => r.status === 'draft'));
	let scanInput = $state<HTMLInputElement | null>(null);

	// Daftar dimuat ulang setiap kali revisi atau status dokumen berubah.
	$effect(() => {
		void doc.revision;
		void doc.status;
		load(doc.id);
	});

	async function load(id: string) {
		const result = await listDocumentRevisions(id);
		items = result.ok ? result.data.items : [];
	}

	async function upload(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		const file = scanInput?.files?.[0];
		if (!file) {
			failure = t('scanChoose');
			return;
		}
		const result = await uploadSignedScan(doc.id, file);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		if (scanInput) scanInput.value = '';
		message = t('scanUploaded', { file: result.data.file_name });
		await load(doc.id);
	}

	async function release() {
		message = '';
		failure = '';
		if (!confirm(t('revisionReleaseConfirm', { number: doc.number, revision: String(doc.revision) }))) {
			return;
		}
		const result = await releaseDocument(doc.id);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		message = t('revisionReleased', { number: result.data.number, revision: String(result.data.revision) });
		onreleased(result.data);
	}
</script>

<section>
	<h2>{t('revisions')}</h2>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<table>
		<thead>
			<tr>
				<th>{t('changeRevision')}</th>
				<th>{t('numberColumnStatus')}</th>
				<th>{t('revisionCreatedBy')}</th>
				<th>{t('revisionReleasedBy')}</th>
				<th>SHA-256</th>
				<th>{t('scanTitle')}</th>
				<th>{t('revisionChanges')}</th>
			</tr>
		</thead>
		<tbody>
			{#each items as r (r.id)}
				<tr>
					<td>{r.revision}</td>
					<td>
						<span class="badge" data-status={r.status}>{t(r.status === 'released' ? 'documentStatusReleased' : 'versionStatusDraft')}</span>
					</td>
					<td>{r.created_by} · {formatDateTime(r.created_at)}</td>
					<td>{r.released_by && r.released_at ? `${r.released_by} · ${formatDateTime(r.released_at)}` : ''}</td>
					<td class="mono hash" title={r.sha256 ?? ''}>{r.sha256 ? `${r.sha256.slice(0, 12)}…` : ''}</td>
					<td class="mono">{r.changes.join(', ')}</td>
					<td>{#if r.scan_file_name}<a href={signedScanUrl(doc.id, r.revision)} download={r.scan_file_name}>{r.scan_file_name}</a>{/if}</td>
				</tr>
			{/each}
		</tbody>
	</table>
	{#if canRelease}
		<form class="row" onsubmit={upload} novalidate>
			<div class="field">
				<label for="scan-file">{t('scanUpload')}</label>
				<input id="scan-file" type="file" accept=".pdf,.png,.jpg,.jpeg" bind:this={scanInput} />
			</div>
			<button type="submit">{t('scanUploadButton')}</button>
		</form>
		<p class="hint">{t('revisionReleaseHint')}</p>
		<button type="button" class="primary" disabled={!draft?.scan_file_name} onclick={release}>{t('revisionRelease', { revision: String(doc.revision) })}</button>
	{/if}

	<h3>{t('diffTitle')}</h3>
	{#if items.some((r) => r.status === 'released')}
		<!-- Dibuat ulang setelah rilis atau draft baru, supaya pilihan bawaan ikut. -->
		{#key `${doc.status}-${doc.revision}-${items.length}`}
			<DiffPanel {doc} revisions={items} />
		{/key}
	{:else}
		<p class="hint">{t('diffNoRelease')}</p>
	{/if}
</section>

<style>
	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1rem;
		align-items: flex-start;
	}

	.row > button {
		margin-top: var(--label-space);
	}

	.hash {
		color: var(--text-3);
	}
</style>
