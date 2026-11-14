<script lang="ts">
	// Halaman satu dokumen (M4.1, M4.2): header, dan baris di workbook mirip
	// Excel (D-145) dengan tab sheet untuk dokumen satu set. Kolom terisi
	// otomatis (nomor operasi, proses) diambil dari atribut baris (D-122,
	// D-123); kolom yang dihitung tidak diketik.
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { addDocumentRow, addDocumentRows, deleteDocumentRow, fillDocumentReferences, reorderDocumentRows, getDocument, getFormatVersion, listProcesses, setDocumentHeader, updateDocumentRow, type ApiFailure } from '#lib/api/client.ts';
	import type { Document, DocumentRow, DocumentRowInput, FormatVersion, Process } from '#lib/api/gen/index.ts';
	import ComposePanel from '#lib/documents/ComposePanel.svelte';
	import ComparePanel from '#lib/documents/ComparePanel.svelte';
	import CopyPanel from '#lib/documents/CopyPanel.svelte';
	import OrderPanel from '#lib/documents/OrderPanel.svelte';
	import PastePanel from '#lib/documents/PastePanel.svelte';
	import RevisionsPanel from '#lib/documents/RevisionsPanel.svelte';
	import PrintPanel from '#lib/documents/PrintPanel.svelte';
	import TemplateLinksPanel from '#lib/documents/TemplateLinksPanel.svelte';
	import SectionBar from '#lib/documents/SectionBar.svelte';
	import Workbook from '#lib/documents/Workbook.svelte';
	import { cleanContent, rowsOfSection, typedFields } from '#lib/documents.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let doc = $state<Document | null>(null);
	let format = $state<FormatVersion | null>(null);
	let processes = $state<Process[]>([]);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	// Dokumen tanpa draft terbuka hanya bisa dibaca (M5.4): isinya berubah lewat perubahan.
	let canEdit = $derived(can('E-2') && doc?.status !== 'released');

	let typedHeader = $derived(format ? typedFields(format.fields, 'header') : []);

	// pasting: panel Tempel dari Excel sedang dibuka (M4.4).
	let pasting = $state(false);
	// composing: panel perakitan dari process template sedang dibuka (M4.5).
	let composing = $state(false);
	// ordering: panel urutan dan nomor operasi sedang dibuka (M4.6).
	let ordering = $state(false);
	// copying: panel salin dari model lain sedang dibuka (M4.7).
	let copying = $state(false);
	// comparing: tampilan banding dengan template sedang dibuka (M4.8).
	let comparing = $state(false);
	// compareRound naik setiap tautan diputus, supaya panel banding memuat ulang perbandingannya.
	let compareRound = $state(0);
	// showDiagram: diagram PFD sedang ditampilkan (M4.10). Hanya untuk dokumen PFD.
	let showDiagram = $state(false);
	// Tombol banding hanya untuk dokumen yang punya baris tertaut ke template.
	let hasLinkedRows = $derived(doc?.rows.some((r) => r.template_point_id) ?? false);

	// Bagian per PCB yang dipilih (D-155), menurut kuncinya, supaya pilihan
	// tetap saat berpindah tab PFD, PFMEA, dan CP. Tanpa pilihan: bagian pertama.
	let sectionKey = $state('');
	let section = $derived(doc?.sections.find((s) => s.key === sectionKey) ?? doc?.sections[0] ?? null);
	let sectionId = $derived(section?.id ?? '');
	// Dokumen yang ditampilkan: hanya baris bagian yang dipilih, bernomor mulai 1.
	let shown = $derived(doc ? { ...doc, rows: rowsOfSection(doc.rows, sectionId) } : null);

	// sectionChanged menerima dokumen sesudah bagian ditambah, diubah, atau dihapus.
	function sectionChanged(d: Document, text: string, key: string) {
		reset();
		use(d);
		sectionKey = key;
		message = text;
	}

	// Isian header.
	let header = $state<Record<string, string>>({});

	// Dokumen mengikuti ?id= di alamat. Tab sheet adalah tautan biasa ke
	// dokumen lain di set yang sama; SvelteKit mengganti alamat tanpa memuat
	// ulang halaman, lalu efek ini memuat dokumennya (juga saat tombol Back).
	let currentId = $derived(page.url.searchParams.get('id') ?? '');
	$effect(() => {
		const id = currentId;
		untrack(() => {
			reset();
			doc = null;
			pasting = false;
			composing = false;
			ordering = false;
			copying = false;
			comparing = false;
			load(id);
		});
	});

	async function load(id: string) {
		const result = await getDocument(id);
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		const [formatResult, processResult] = await Promise.all([getFormatVersion(result.data.format_version_id), listProcesses()]);
		if (!formatResult.ok) {
			loadError = failureText(formatResult.error);
			return;
		}
		format = formatResult.data;
		processes = processResult.ok ? processResult.data.items : [];
		use(result.data);
	}

	// use memakai dokumen terbaru dari server. Isian header hanya diganti bila
	// header di server memang berubah, supaya ketikan yang belum disimpan tidak
	// hilang saat baris ditambah atau dimuat ulang.
	function use(d: Document) {
		const sameHeader = doc !== null && doc.id === d.id && doc.header_version === d.header_version;
		doc = d;
		if (!sameHeader) {
			header = Object.fromEntries(Object.entries(d.header).map(([k, v]) => [k, String(v ?? '')]));
		}
	}

	// changedByOther: server menolak karena isinya sudah diubah orang lain
	// sejak dibaca (penguncian optimistik, M4.3, R-DOC-07).
	function changedByOther(error: ApiFailure): boolean {
		return error.status === 409 && (error.body?.details ?? []).some((d) => d.code === 'changed');
	}

	// reloadAfterConflict memuat isi terbaru dan menampilkan pesan konflik.
	async function reloadAfterConflict(text: string) {
		if (!doc) {
			return;
		}
		const latest = await getDocument(doc.id);
		if (latest.ok) {
			use(latest.data);
		}
		failure = text;
	}

	function showErrors(error: ApiFailure) {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
		failure = details.length === 0 ? failureText(error) : t('documentFixFields');
	}

	function reset() {
		message = '';
		failure = '';
		fieldErrors = {};
	}

	async function saveHeader(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (!doc) {
			return;
		}
		const result = await setDocumentHeader(doc.id, cleanContent(typedHeader, header), doc.header_version);
		if (!result.ok) {
			if (changedByOther(result.error)) {
				await reloadAfterConflict(t('documentHeaderChanged'));
				return;
			}
			showErrors(result.error);
			return;
		}
		use(result.data);
		message = t('documentHeaderSaved');
	}

	// saveCell menyimpan satu baris dari tabel workbook (M4.2). Dokumen tidak
	// dimuat ulang, supaya posisi kursor di tabel tidak hilang. Jawabannya:
	// baris yang tersimpan (dengan versi barunya), 'reloaded' bila baris sudah
	// diubah orang lain dan isi terbaru dimuat (M4.3), atau null bila ditolak.
	async function saveCell(rowId: string, input: DocumentRowInput): Promise<DocumentRow | 'reloaded' | null> {
		reset();
		if (!doc) {
			return null;
		}
		const result = await updateDocumentRow(doc.id, rowId, input);
		if (!result.ok) {
			if (changedByOther(result.error)) {
				await reloadAfterConflict(t('documentRowChanged'));
				return 'reloaded';
			}
			showErrors(result.error);
			return null;
		}
		message = t('documentRowSaved');
		return result.data.rows.find((r) => r.id === rowId) ?? null;
	}

	// addRow menambah baris kosong di akhir; isinya diketik langsung di tabel.
	async function addRow() {
		reset();
		if (!doc) {
			return;
		}
		const result = await addDocumentRow(doc.id, { content: {}, section_id: sectionId || undefined });
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		use(result.data);
		message = t('documentRowAdded');
	}

	// pasteRows menambah blok baris dari Excel dalam satu permintaan (M4.4).
	async function pasteRows(rows: DocumentRowInput[]): Promise<boolean> {
		reset();
		if (!doc) {
			return false;
		}
		const result = await addDocumentRows(doc.id, rows.map((r) => ({ ...r, section_id: sectionId || undefined })));
		if (!result.ok) {
			showErrors(result.error);
			return false;
		}
		use(result.data);
		pasting = false;
		message = t('pasteDone', { rows: String(rows.length) });
		return true;
	}

	// composed menerima dokumen sesudah perakitan (M4.5).
	function composed(d: Document, process: string) {
		reset();
		use(d);
		composing = false;
		message = t('composeDone', { process });
	}

	// copied menerima dokumen sesudah baris disalin dari model lain (M4.7).
	function copied(d: Document, source: string) {
		reset();
		use(d);
		copying = false;
		message = t('copyDone', { number: source });
	}

	// detached menerima dokumen sesudah tautan satu baris diputus (M4.8).
	// Panel banding dimuat ulang supaya keadaannya terbaru.
	function detached(d: Document, position: number) {
		reset();
		use(d);
		compareRound++;
		message = t('compareDetachDone', { position: String(position) });
	}

	// Tombol isi dokumen terkait hanya untuk form yang punya kolom Relevant Document (D-130).
	let hasRelevant = $derived(format?.fields.some((f) => f.area === 'row' && !f.hidden && f.key === 'relevant_document') ?? false);

	async function fillReferences() {
		reset();
		if (!doc) {
			return;
		}
		const result = await fillDocumentReferences(doc.id);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		use(result.data);
		message = t('refFillDone');
	}

	// saveOrder menyimpan urutan baris dan, bila diminta, nomor operasi otomatis (M4.6).
	async function saveOrder(rowIds: string[], renumber: boolean): Promise<boolean> {
		reset();
		if (!doc) {
			return false;
		}
		const result = await reorderDocumentRows(doc.id, rowIds, renumber, sectionId);
		if (!result.ok) {
			showErrors(result.error);
			return false;
		}
		use(result.data);
		ordering = false;
		message = t('orderSaved');
		return true;
	}

	async function removeRow(rowId: string) {
		reset();
		if (!doc) {
			return;
		}
		const position = shown?.rows.find((r) => r.id === rowId)?.position ?? 0;
		if (!confirm(t('documentRowDeleteConfirm', { position: String(position) }))) {
			return;
		}
		const result = await deleteDocumentRow(doc.id, rowId);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		use(result.data);
		message = t('documentRowRemoved', { position: String(position) });
	}
</script>

{#if loadError}
	<h1>{t('documentsTitle')}</h1>
	<p class="error" role="alert">{loadError}</p>
{:else if !doc || !format}
	<p>{t('formatLoading')}</p>
{:else}
	<p><a href="/documents">{t('documentBack')}</a></p>
	<h1 class="mono">{doc.number}</h1>
	<dl class="facts">
		<dt>{t('numberColumnType')}</dt>
		<dd>{doc.document_type.name}</dd>
		<dt>{t('documentCustomer')}</dt>
		<dd>{doc.customer_code}</dd>
		<dt>{t('documentModels')}</dt>
		<dd>{doc.models.map((m) => m.name).join(', ')}</dd>
		{#if doc.part_numbers.length > 0}
			<dt>{t('documentPartNumbers')}</dt>
			<dd>{doc.part_numbers.map((p) => p.number).join(', ')}</dd>
		{/if}
		<dt>{t('documentFormSet')}</dt>
		<dd>{doc.form_set_name}</dd>
		<dt>{t('documentRoute')}</dt>
		<dd>{doc.route?.name ?? t('documentNoRoute')}{doc.route_reason ? ` · ${doc.route_reason}` : ''}</dd>
		<dt>{t('numberColumnStatus')}</dt>
		<dd>
			<span class="badge" data-status={doc.status}>{t(doc.status === 'released' ? 'documentStatusReleased' : 'versionStatusDraft')}</span>
		</dd>
		<dt>{t(doc.status === 'draft' ? 'documentDraftRevision' : 'documentReleasedRevision')}</dt>
		<dd>{doc.revision}</dd>
		{#if doc.status === 'draft' && doc.released_revision !== undefined}
			<dt>{t('documentReleasedRevision')}</dt>
			<dd>{doc.released_revision}</dd>
		{/if}
	</dl>
	{#if doc.status === 'released'}
		<p class="hint">{t('documentNoDraft')}</p>
	{/if}

	<RevisionsPanel {doc} onreleased={(d) => (doc = d)} />
	<PrintPanel {doc} />
	<TemplateLinksPanel {doc} />

	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if typedHeader.length > 0}
		<section>
			<h2>{t('formatHeaderFields')}</h2>
			<form class="grid" onsubmit={saveHeader} novalidate>
				{#each typedHeader as f (f.id)}
					<div class="field">
						<label for="header-{f.id}">{f.label}</label>
						{#if f.kind === 'select'}
							<select id="header-{f.id}" bind:value={header[f.id]} disabled={!canEdit}>
								<option value=""></option>
								{#each f.options ?? [] as o (o.key)}<option value={o.key}>{o.label}</option>{/each}
							</select>
						{:else}
							<input id="header-{f.id}" type={f.kind === 'date' ? 'date' : 'text'} bind:value={header[f.id]} disabled={!canEdit} />
						{/if}
						{#if fieldErrors[f.id]}<p class="error">{fieldErrors[f.id]}</p>{/if}
					</div>
				{/each}
				{#if canEdit}<button type="submit">{t('documentHeaderSave')}</button>{/if}
			</form>
		</section>
	{/if}

	<section>
		<!-- Tab sheet workbook (D-145): dokumen satu set PFD, PFMEA, dan CP, lalu tab Check (M4.9). -->
		<nav class="sheets" aria-label={t('documentSheets')}>
			{#each doc.set as m (m.id)}
				<a href="/documents/view?id={m.id}" data-sveltekit-noscroll aria-current={m.id === doc.id ? 'page' : undefined}>{m.document_type.name}</a>
			{/each}
			<!-- Tab Check (M4.9): hanya untuk set yang punya PFMEA dan CP. -->
			{#if doc.set.some((m) => m.document_type.key === 'pfmea') && doc.set.some((m) => m.document_type.key === 'cp')}
				<a href="/documents/check?id={doc.id}">{t('checkTab')}</a>
			{/if}
		</nav>
		<h2>{t('documentRowsTitle')}</h2>
		<SectionBar {doc} current={section} {canEdit} onselect={(key) => (sectionKey = key)} onchanged={sectionChanged} />
		{#if section}<h3 class="section-title">{section.name}{section.part_numbers.length > 0 ? ` · ${section.part_numbers.map((p) => p.number).join(', ')}` : ''}</h3>{/if}
		{#if canEdit}<p class="hint">{t('documentGridHint')}</p>{/if}
		{#key doc.id + sectionId}
			<Workbook doc={shown ?? doc} {format} {processes} {canEdit} onsave={saveCell} ondelete={removeRow} />
		{/key}
		{#if doc.document_type.key === 'pfd'}
			<p><button type="button" aria-expanded={showDiagram} onclick={() => (showDiagram = !showDiagram)}>{showDiagram ? t('diagramClose') : t('diagramOpen')}</button></p>
			{#if showDiagram}
				<h3>{t('diagramTitle')}</h3>
				<p class="hint">{t('diagramHint')}</p>
				<!-- Dimuat hanya saat dibuka: Svelte Flow dan ELK cukup besar (M4.10). -->
				{#await import('#lib/documents/PfdDiagram.svelte')}
					<p>{t('formatLoading')}</p>
				{:then diagram}
					<diagram.default rows={shown?.rows ?? doc.rows} {format} />
				{/await}
			{/if}
		{/if}
		{#if hasLinkedRows && !comparing}
			<p><button type="button" onclick={() => (comparing = true)}>{t('compareOpen')}</button></p>
		{/if}
		{#if canEdit}
			<p class="row-actions">
				<button type="button" onclick={addRow}>{t('documentRowAdd')}</button>
				{#if !pasting}<button type="button" onclick={() => (pasting = true)}>{t('pasteOpen')}</button>{/if}
				{#if !composing}<button type="button" onclick={() => (composing = true)}>{t('composeOpen')}</button>{/if}
				{#if !ordering}<button type="button" onclick={() => (ordering = true)}>{t('orderOpen')}</button>{/if}
				{#if hasRelevant && doc.rows.length > 0}<button type="button" onclick={fillReferences}>{t('refFill')}</button>{/if}
				<!-- Menyalin hanya untuk dokumen yang masih kosong (R-CMP-05). -->
				{#if !copying && doc.rows.length === 0}<button type="button" onclick={() => (copying = true)}>{t('copyOpen')}</button>{/if}
			</p>
		{/if}
	</section>

	{#if canEdit && composing}
		<section>
			{#key doc.id + sectionId}
				<ComposePanel {doc} sectionKey={section?.key ?? ''} oncomposed={composed} oncancel={() => (composing = false)} />
			{/key}
		</section>
	{/if}

	{#if comparing}
		<section>
			{#key compareRound}
				<ComparePanel {doc} candetach={canEdit} ondetached={detached} oncancel={() => (comparing = false)} />
			{/key}
		</section>
	{/if}

	{#if canEdit && copying}
		<section>
			<CopyPanel {doc} oncopied={copied} oncancel={() => (copying = false)} />
		</section>
	{/if}

	{#if canEdit && ordering}
		<section>
			{#key sectionId}
				<OrderPanel doc={shown ?? doc} onsave={saveOrder} oncancel={() => (ordering = false)} />
			{/key}
		</section>
	{/if}

	{#if canEdit && pasting}
		<section>
			<PastePanel {format} {processes} onimport={pasteRows} oncancel={() => (pasting = false)} />
		</section>
	{/if}{/if}

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

	.grid {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	.grid .field {
		min-width: 12rem;
	}

	.grid > button {
		margin-top: var(--label-space);
	}

	.row-actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	.sheets {
		display: flex;
		gap: 4px;
		margin-bottom: 1rem;
		border-bottom: 1px solid var(--border);
	}

	.sheets a {
		padding: 8px 16px;
		border: 1px solid transparent;
		border-bottom: none;
		border-radius: var(--radius) var(--radius) 0 0;
		color: var(--text-2);
	}

	.sheets a[aria-current='page'] {
		background: var(--surface);
		border-color: var(--border);
		color: var(--accent-ink);
		font-weight: 600;
		margin-bottom: -1px;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
