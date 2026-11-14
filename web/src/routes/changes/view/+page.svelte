<script lang="ts">
	// Halaman satu perubahan (M5.1): sumber, jenis, alasan, dan dokumen
	// terdampak. Selama perubahan masih diajukan, pemegang E-3 bisa menambah
	// atau melepas dokumen terdampak (yang terakhir tidak bisa dilepas).
	// Lampiran (M5.2): berkas pendukung diunggah, diunduh, dan dilepas di sini.
	// Verifikasi (M5.3): pemegang V-1 menerima atau menolak dengan catatan wajib.
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import {
		addChangeAttachment,
		addChangeDocument,
		changeAttachmentUrl,
		getChange,
		listDocuments,
		removeChangeAttachment,
		removeChangeDocument,
		verifyChange,
		type ApiFailure
	} from '#lib/api/client.ts';
	import type { Change, DocumentSummary } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import ImpactChecklist from '#lib/changes/ImpactChecklist.svelte';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let change = $state<Change | null>(null);
	let documents = $state<DocumentSummary[]>([]);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let documentId = $state('');
	let fileInput = $state<HTMLInputElement | null>(null);
	let uploading = $state(false);
	let note = $state('');
	let noteError = $state('');
	let canVerify = $derived(can('V-1') && change?.status === 'submitted');
	let canPropose = $derived(can('E-3'));
	let editable = $derived(canPropose && change?.status === 'submitted');
	// Daftar periksa dampak masih bisa diisi sesudah diterima (M5.7, ASUMSI E-104).
	let impactsEditable = $derived(canPropose && (change?.status === 'submitted' || change?.status === 'accepted'));
	let addable = $derived(documents.filter((d) => !change?.documents.some((x) => x.id === d.id)));

	let currentId = $derived(page.url.searchParams.get('id') ?? '');
	$effect(() => {
		const id = currentId;
		untrack(() => load(id));
	});

	async function load(id: string) {
		const result = await getChange(id);
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		change = result.data;
		const docs = await listDocuments(result.data.customer_id);
		documents = docs.ok ? docs.data.items : [];
	}

	function showError(error: ApiFailure) {
		const details = error.body?.details ?? [];
		failure = details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	async function add(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		if (!change || documentId === '') {
			return;
		}
		const result = await addChangeDocument(change.id, documentId);
		if (!result.ok) {
			showError(result.error);
			return;
		}
		const number = documents.find((d) => d.id === documentId)?.number ?? '';
		change = result.data;
		documentId = '';
		message = t('changeDocumentAdded', { number });
	}

	async function upload(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		const file = fileInput?.files?.[0];
		if (!change || !file) {
			failure = t('changeAttachmentChoose');
			return;
		}
		uploading = true;
		const result = await addChangeAttachment(change.id, file);
		uploading = false;
		if (!result.ok) {
			showError(result.error);
			return;
		}
		change = result.data;
		if (fileInput) {
			fileInput.value = '';
		}
		message = t('changeAttachmentAdded', { file: file.name });
	}

	async function removeAttachment(id: string, file: string) {
		message = '';
		failure = '';
		if (!change || !confirm(t('changeAttachmentRemoveConfirm', { file }))) {
			return;
		}
		const result = await removeChangeAttachment(change.id, id);
		if (!result.ok) {
			showError(result.error);
			return;
		}
		change = result.data;
		message = t('changeAttachmentRemoved', { file });
	}

	// fileSize menulis ukuran berkas dalam KB atau MB.
	function fileSize(bytes: number): string {
		return bytes < 1024 * 1024 ? `${Math.max(1, Math.round(bytes / 1024))} KB` : `${(bytes / 1024 / 1024).toFixed(1)} MB`;
	}

	async function verify(decision: 'accept' | 'reject') {
		message = '';
		failure = '';
		noteError = '';
		if (!change) {
			return;
		}
		if (note.trim() === '') {
			noteError = t('changeVerifyNoteRequired');
			return;
		}
		const result = await verifyChange(change.id, { decision, note: note.trim() });
		if (!result.ok) {
			showError(result.error);
			return;
		}
		change = result.data;
		note = '';
		message = t(decision === 'accept' ? 'changeAccepted' : 'changeRejected', { ref: change.ref });
	}

	async function remove(id: string, number: string) {
		message = '';
		failure = '';
		if (!change) {
			return;
		}
		const result = await removeChangeDocument(change.id, id);
		if (!result.ok) {
			showError(result.error);
			return;
		}
		change = result.data;
		message = t('changeDocumentRemoved', { number });
	}
</script>

{#if loadError}
	<h1>{t('changesTitle')}</h1>
	<p class="error" role="alert">{loadError}</p>
{:else if !change}
	<p>{t('formatLoading')}</p>
{:else}
	<p><a href="/changes">{t('changeBack')}</a></p>
	<h1 class="mono">{change.ref}</h1>
	<dl class="facts">
		<dt>{t('numberColumnStatus')}</dt>
		<dd><span class="badge" data-change-status={change.status}>{t(`changeStatus_${change.status}` as 'changeStatus_submitted')}</span></dd>
		<dt>{t('documentCustomer')}</dt>
		<dd>{change.customer_code}</dd>
		<dt>{t('changeMechanism')}</dt>
		<dd>{t(`changeMechanism_${change.mechanism}` as 'changeMechanism_manual')}</dd>
		{#if change.trigger}
			<dt>{t('changeTrigger')}</dt>
			<dd>{change.trigger.name}</dd>
		{/if}
		<dt>{t('changeType')}</dt>
		<dd>{change.change_type.name}</dd>
		{#if change.four_m}
			<dt>{t('changeFourM')}</dt>
			<dd>{change.four_m.name}</dd>
		{/if}
		{#if change.source_reference}
			<dt>{t('changeReference')}</dt>
			<dd>{change.source_reference}</dd>
		{/if}
		<dt>{t('changeReason')}</dt>
		<dd class="reason">{change.reason}</dd>
		<dt>{t('changeRequestedBy')}</dt>
		<dd>{change.requested_by} · {formatDateTime(change.created_at)}</dd>
		{#if change.verification}
			<dt>{t(change.status === 'accepted' ? 'changeAcceptedBy' : 'changeRejectedBy')}</dt>
			<dd>{change.verification.verified_by} · {formatDateTime(change.verification.verified_at)}</dd>
			<dt>{t('changeVerifyNote')}</dt>
			<dd class="reason">{change.verification.note}</dd>
		{/if}
	</dl>

	{#if canVerify}
		<section>
			<h2>{t('changeVerify')}</h2>
			<p class="hint">{t('changeVerifyHint')}</p>
			<div class="field">
				<label for="change-verify-note">{t('changeVerifyNote')}</label>
				<textarea id="change-verify-note" rows="3" maxlength="2000" bind:value={note} aria-invalid={noteError !== ''} aria-describedby={noteError ? 'change-verify-note-error' : undefined}></textarea>
				{#if noteError}<p class="error" id="change-verify-note-error">{noteError}</p>{/if}
			</div>
			<div class="row">
				<button type="button" class="primary" onclick={() => verify('accept')}>{t('changeAccept')}</button>
				<button type="button" onclick={() => verify('reject')}>{t('changeReject')}</button>
			</div>
		</section>
	{/if}

	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<section>
		<h2>{t('changeDocuments')}</h2>
		<table>
			<thead>
				<tr>
					<th>{t('documentNumber')}</th>
					<th>{t('numberColumnType')}</th>
					<th>{t('changeRevision')}</th>
					<th>{t('changeDraft')}</th>
					{#if editable}<th>{t('masterActions')}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each change.documents as d (d.id)}
					<tr>
						<td class="mono"><a href="/documents/view?id={d.id}">{d.number}</a></td>
						<td>{d.document_type.name}</td>
						<td>{d.revision}</td>
						<td>{d.draft_revision === undefined ? '' : t(d.merged ? 'changeDraftMerged' : 'changeDraftOpened', { revision: String(d.draft_revision) })}</td>
						{#if editable}
							<td>
								<button type="button" aria-label={t('changeDocumentRemoveFor', { number: d.number })} onclick={() => remove(d.id, d.number)}>{t('changeDocumentRemove')}</button>
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
		{#if editable && addable.length > 0}
			<form class="row" onsubmit={add} novalidate>
				<div class="field">
					<label for="change-add-document">{t('changeDocumentAdd')}</label>
					<select id="change-add-document" bind:value={documentId}>
						<option value="">{t('changeChoose')}</option>
						{#each addable as d (d.id)}<option value={d.id}>{d.number} · {d.document_type.name}</option>{/each}
					</select>
				</div>
				<button type="submit">{t('changeDocumentAddButton')}</button>
			</form>
		{/if}
	</section>

	{#key change.id}
		<ImpactChecklist {change} editable={impactsEditable} onsaved={(c) => (change = c)} />
	{/key}

	<section>
		<h2>{t('changeAttachments')}</h2>
		{#if change.attachments.length === 0}
			<p>{t('changeAttachmentsNone')}</p>
		{:else}
			<table>
				<thead>
					<tr>
						<th>{t('changeAttachmentFile')}</th>
						<th>{t('changeAttachmentSize')}</th>
						<th>{t('changeAttachmentUploadedBy')}</th>
						<th>SHA-256</th>
						{#if editable}<th>{t('masterActions')}</th>{/if}
					</tr>
				</thead>
				<tbody>
					{#each change.attachments as a (a.id)}
						<tr>
							<td><a href={changeAttachmentUrl(change.id, a.id)} download={a.file_name}>{a.file_name}</a></td>
							<td>{fileSize(a.size)}</td>
							<td>{a.uploaded_by} · {formatDateTime(a.uploaded_at)}</td>
							<td class="mono hash" title={a.sha256}>{a.sha256.slice(0, 12)}…</td>
							{#if editable}
								<td>
									<button type="button" aria-label={t('changeAttachmentRemoveFor', { file: a.file_name })} onclick={() => removeAttachment(a.id, a.file_name)}>{t('changeDocumentRemove')}</button>
								</td>
							{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
		{#if editable}
			<form class="row" onsubmit={upload} novalidate>
				<div class="field">
					<label for="change-attachment-file">{t('changeAttachmentAdd')}</label>
					<input id="change-attachment-file" type="file" bind:this={fileInput} accept=".pdf,.png,.jpg,.jpeg,.xlsx,.xls,.docx,.doc,.msg,.txt,.csv" />
					<p class="hint">{t('changeAttachmentHint')}</p>
				</div>
				<button type="submit" disabled={uploading}>{t('changeAttachmentUpload')}</button>
			</form>
		{/if}
	</section>
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
		gap: 1rem;
		align-items: flex-start;
	}

	.row > button {
		margin-top: var(--label-space);
	}

	.hash {
		color: var(--text-3);
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
