<script lang="ts">
	// Halaman satu dokumen (M4.1, M4.2) sebagai workbook (D-145, C1): kepala
	// dengan satu langkah berikutnya, toolbar berkelompok yang membuka panel di
	// laci samping, lembar dokumen, dan tab sheet di bawah (Cover, PFD, PFMEA,
	// CP, Check). Kolom terisi otomatis (nomor operasi, proses) diambil dari
	// atribut baris (D-122, D-123); kolom yang dihitung tidak diketik.
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { addDocumentRow, addDocumentRows, deleteDocumentRow, fillDocumentReferences, reorderDocumentRows, getDocument, getFormatVersion, getPagePreview, getPfdComposition, listProcesses, setDocumentHeader, updateDocumentRow, type ApiFailure } from '#lib/api/client.ts';
	import type { Document, DocumentRow, DocumentRowInput, FormatVersion, PagePreview, Process } from '#lib/api/gen/index.ts';
	import ComposePanel from '#lib/documents/ComposePanel.svelte';
	import ComparePanel from '#lib/documents/ComparePanel.svelte';
	import PfdComposePanel from '#lib/documents/PfdComposePanel.svelte';
	import { warnings } from '#lib/documents/pfdcompose.ts';
	import CopyPanel from '#lib/documents/CopyPanel.svelte';
	import OrderPanel from '#lib/documents/OrderPanel.svelte';
	import PastePanel from '#lib/documents/PastePanel.svelte';
	import RevisionsPanel from '#lib/documents/RevisionsPanel.svelte';
	import SignaturePanel from '#lib/documents/SignaturePanel.svelte';
	import SubmissionPanel from '#lib/documents/SubmissionPanel.svelte';
	import PrintPanel from '#lib/documents/PrintPanel.svelte';
	import TemplateLinksPanel from '#lib/documents/TemplateLinksPanel.svelte';
	import MigrationPanel from '#lib/documents/MigrationPanel.svelte';
	import ExcelComparePanel from '#lib/documents/ExcelComparePanel.svelte';
	import LinkPanel from '#lib/documents/LinkPanel.svelte';
	import SectionBar from '#lib/documents/SectionBar.svelte';
	import Drawer from '#lib/documents/Drawer.svelte';
	import SheetTabs from '#lib/documents/SheetTabs.svelte';
	import { drawerTitle, nextStep, type DrawerId } from '#lib/documents/frame.ts';
	import PreviewSheet from '#lib/documents/PreviewSheet.svelte';
	import Workbook from '#lib/documents/Workbook.svelte';
	import ReviewBanner from '#lib/ReviewBanner.svelte';
	import { rememberRecent } from '#lib/recent.ts';
	import { cleanContent, rowsOfSection, typedFields } from '#lib/documents.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let doc = $state<Document | null>(null);
	let format = $state<FormatVersion | null>(null);
	let processes = $state<Process[]>([]);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	// Dokumen tanpa draft terbuka hanya bisa dibaca (M5.4): isinya berubah lewat perubahan.
	// Draft yang sedang atau sudah ditandatangani digital terkunci (D-198).
	let canEdit = $derived(can('E-2') && doc?.status !== 'released' && !doc?.signing);

	let typedHeader = $derived(format ? typedFields(format.fields, 'header') : []);

	// drawer: panel yang sedang terbuka di laci samping (C1), misalnya
	// perakitan (M4.5), urutan (M4.6), cetak, atau revisi. Satu per satu.
	let drawer = $state<DrawerId | null>(null);
	// sheet: 'cover' untuk sheet Cover; kosong untuk lembar dokumen.
	let sheet = $derived(page.url.searchParams.get('sheet') ?? '');
	// Pratinjau lembar (C2, C3): baca saja, per halaman seperti cetakan. Rencana
	// halamannya dari server, sama dengan PDF dan Excel. Dimuat ulang setiap
	// kali tombol Preview ditekan, supaya baris terbaru ikut.
	let previewing = $state(false);
	let preview = $state<PagePreview | null>(null);
	let previewError = $state('');

	async function loadPreview() {
		if (!doc) return;
		previewError = '';
		preview = null;
		const result = await getPagePreview(doc.id);
		if (!result.ok) {
			previewError = failureText(result.error);
			return;
		}
		preview = result.data;
		// Pemeriksaan dengan PFD (C4) untuk PFMEA dan CP yang dirakit dari PFD.
		previewChecks = [];
		if (setRole === 'follows_pfd') {
			const steps = await getPfdComposition(doc.id);
			if (steps.ok) {
				const names = Object.fromEntries(steps.data.documents.map((d) => [d.document_type_key, d.document_type_name]));
				previewChecks = warnings(steps.data)
					.filter((w) => w.documentType === doc!.document_type.key)
					.map((w) => t(w.kind === 'missing' ? 'pfdMissing' : w.kind === 'repeated' ? 'pfdRepeated' : 'pfdExtra', { operation: w.operation, process: w.process, document: names[w.documentType] ?? w.documentType }));
			}
		}
	}
	let previewChecks = $state<string[]>([]);

	function togglePreview() {
		reset();
		previewing = !previewing;
		if (previewing) {
			loadPreview();
		}
	}

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
			drawer = null;
			preview = null;
			previewing = false;
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
		rememberRecent({ id: d.id, number: d.number, title: d.models.map((m) => m.name).join(', '), status: d.status === 'draft' ? 'editing' : 'released', revision: d.revision, view: 'draft' });
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
		drawer = null;
		message = t('pasteDone', { rows: String(rows.length) });
		return true;
	}

	// composed menerima dokumen sesudah perakitan (M4.5).
	function composed(d: Document, process: string) {
		reset();
		use(d);
		drawer = null;
		message = t('composeDone', { process });
	}

	// composedFromPfd menerima dokumen sesudah dirakit dari langkah PFD (C4).
	function composedFromPfd(d: Document, steps: number) {
		reset();
		use(d);
		drawer = null;
		message = t('pfdDone', { count: String(steps) });
	}

	// Peran dokumen di set (C4, D-211): PFD diisi langsung, PFMEA dan CP satu
	// set dengan PFD dirakit per langkah PFD.
	let pfdMember = $derived(doc?.set.find((m) => m.document_type.key === 'pfd' && m.id !== doc?.id));
	let setRole = $derived<'pfd' | 'follows_pfd' | undefined>(
		doc?.document_type.key === 'pfd' ? 'pfd' : pfdMember ? 'follows_pfd' : undefined
	);
	// Jumlah langkah PFD, dibaca hanya untuk dokumen kosong (langkah berikutnya).
	let pfdSteps = $state<number | undefined>(undefined);
	$effect(() => {
		const id = doc?.id;
		if (!id || setRole !== 'follows_pfd' || doc!.rows.length > 0 || !canEdit) {
			return;
		}
		untrack(async () => {
			const result = await getPfdComposition(id);
			pfdSteps = result.ok ? result.data.steps.length : undefined;
		});
	});

	// copied menerima dokumen sesudah baris disalin dari model lain (M4.7).
	function copied(d: Document, source: string) {
		reset();
		use(d);
		drawer = null;
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

	// Langkah berikutnya di kepala workbook (D-145): satu aksi utama menurut status.
	let step = $derived(
		doc
			? nextStep({
					status: doc.status === 'released' ? 'released' : 'draft',
					rows: doc.rows.length,
					hasRoute: Boolean(doc.route),
					signing: doc.signing,
					migrated: doc.origin === 'migrated',
					verified: doc.baseline?.verified_at !== undefined,
					canEdit,
					canPropose: can('E-3'),
					canRecord: can('E-7'),
					setRole,
					pfdSteps,
					pfdHref: pfdMember ? `/documents/view?id=${pfdMember.id}` : undefined
				})
			: null
	);

	// toggle membuka laci itu, atau menutupnya bila sedang terbuka.
	function toggle(id: DrawerId) {
		reset();
		drawer = drawer === id ? null : id;
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
		drawer = null;
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
	{@const linkable = canEdit && doc.rows.some((r) => r.process_id && !r.template_point_id)}
	{@const isPfd = doc.document_type.key === 'pfd'}
	<p class="back"><a href="/documents">{t('documentBack')}</a></p>
	<!-- Kepala workbook (D-145): nomor, model, form set, status, dan satu langkah berikutnya. -->
	<header class="wb-head">
		<div>
			<h1 class="mono">{doc.number}</h1>
			<p class="wb-sub">
				<span>{doc.models.map((m) => m.name).join(', ')} · {doc.document_type.name}</span>
				<span class="badge form-set">{doc.form_set_name}</span>
				<span class="badge" data-status={doc.status}>
					{t(doc.status !== 'released' ? 'versionStatusDraft' : doc.released_migrated ? 'documentStatusReleasedMigrated' : 'documentStatusReleased')}
				</span>
				<span>{t(doc.status === 'draft' ? 'documentDraftRevision' : 'documentReleasedRevision')} {doc.revision}</span>
			</p>
		</div>
		{#if step}
			<section class="next" aria-labelledby="next-h">
				<h2 id="next-h">{t('nextStepTitle')}</h2>
				<p>{t(step.text)}</p>
				{#if step.open}
					{@const target = step.open}
					<button type="button" class="primary" onclick={() => toggle(target)}>{t(drawerTitle[target])}</button>
				{:else if step.href}
					<a class="button primary" href={step.href}>{t(step.label ?? 'nextChangeOpen')}</a>
				{/if}
			</section>
		{/if}
	</header>
	<ReviewBanner documentId={doc.id} />

	<!-- Tombol toolbar yang membuka laci: teks pendek seperti pita Excel; nama lengkapnya
	     (judul panel) menjadi nama untuk pembaca layar dan tooltip. Nama lengkap ditulis sebagai
	     teks tersembunyi, bukan aria-label, supaya tidak tertukar dengan label field di panel. -->
	{#snippet drawerButton(id: DrawerId, short: MessageKey)}
		<button type="button" title={t(drawerTitle[id])} aria-pressed={drawer === id} onclick={() => toggle(id)}><span aria-hidden="true">{t(short)}</span><span class="visually-hidden">{t(drawerTitle[id])}</span></button>
	{/snippet}
	<!-- Toolbar berkelompok (D-145): setiap tombol membuka panel di laci samping atau langsung bekerja. -->
	<div class="wb-toolbar" role="toolbar" aria-label={t('wbToolbar')}>
		{#if canEdit && sheet !== 'cover'}
			<div class="wb-group" role="group" aria-labelledby="wb-rows">
				<div class="wb-buttons">
					<button type="button" onclick={addRow}>{t('documentRowAdd')}</button>
					{@render drawerButton('paste', 'wbShortPaste')}
					{@render drawerButton('order', 'wbShortOrder')}
					{#if hasRelevant && doc.rows.length > 0}<button type="button" title={t('refFill')} onclick={fillReferences}><span aria-hidden="true">{t('wbShortFill')}</span><span class="visually-hidden">{t('refFill')}</span></button>{/if}
				</div>
				<span class="wb-label" id="wb-rows">{t('wbGroupRows')}</span>
			</div>
		{/if}
		{#if canEdit || hasLinkedRows}
			<div class="wb-group" role="group" aria-labelledby="wb-template">
				<div class="wb-buttons">
					<!-- PFD diisi langsung tanpa template; PFMEA dan CP satu set dengan PFD dirakit per langkah PFD (C4, D-211). -->
					{#if canEdit && setRole === 'follows_pfd'}{@render drawerButton('pfd', 'wbShortPfd')}{/if}
					{#if canEdit && setRole !== 'pfd'}{@render drawerButton('compose', 'wbShortCompose')}{/if}
					<!-- Menyalin hanya untuk dokumen yang masih kosong (R-CMP-05). -->
					{#if canEdit && doc.rows.length === 0}{@render drawerButton('copy', 'wbShortCopy')}{/if}
					{#if hasLinkedRows}
						{@render drawerButton('compare', 'wbShortCompare')}
						{@render drawerButton('templates', 'wbShortTemplates')}
					{/if}
				</div>
				<span class="wb-label" id="wb-template">{t('wbGroupTemplate')}</span>
			</div>
		{/if}
		<div class="wb-group" role="group" aria-labelledby="wb-review">
			<div class="wb-buttons">
				{#if isPfd && sheet !== 'cover'}
					<button type="button" aria-pressed={showDiagram} title={showDiagram ? t('diagramClose') : t('diagramOpen')} onclick={() => (showDiagram = !showDiagram)}><span aria-hidden="true">{t('wbShortDiagram')}</span><span class="visually-hidden">{showDiagram ? t('diagramClose') : t('diagramOpen')}</span></button>
				{/if}
				{#if sheet !== 'cover'}
					<button type="button" aria-pressed={previewing} title={t('previewOpen')} onclick={togglePreview}><span aria-hidden="true">{t('wbShortPreview')}</span><span class="visually-hidden">{t('previewOpen')}</span></button>
				{/if}
				{@render drawerButton('revisions', 'wbShortRevisions')}
			</div>
			<span class="wb-label" id="wb-review">{t('wbGroupReview')}</span>
		</div>
		<div class="wb-group" role="group" aria-labelledby="wb-output">
			<div class="wb-buttons">
				{@render drawerButton('print', 'wbShortPrint')}
				{#if doc.status === 'draft' || doc.signing}
					{@render drawerButton('signature', 'wbShortSign')}
				{/if}
				{@render drawerButton('submission', 'wbShortSubmission')}
			</div>
			<span class="wb-label" id="wb-output">{t('wbGroupOutput')}</span>
		</div>
		{#if doc.origin === 'migrated' || linkable}
			<div class="wb-group" role="group" aria-labelledby="wb-legacy">
				<div class="wb-buttons">
					{#if doc.baseline}{@render drawerButton('migration', 'wbShortBaseline')}{/if}
					{#if doc.origin === 'migrated'}{@render drawerButton('excel', 'wbShortExcel')}{/if}
					<!-- Penautan (M7.4): hanya bila ada baris berproses yang belum tertaut. -->
					{#if linkable}{@render drawerButton('link', 'wbShortLink')}{/if}
				</div>
				<span class="wb-label" id="wb-legacy">{t('wbGroupLegacy')}</span>
			</div>
		{/if}
	</div>

	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<!-- Lembar dan laci berdampingan (C1). -->
	<div class="wb-body">
	<div class="wb-main">
	{#if sheet === 'cover'}
		<!-- Sheet Cover (C1): data yang sama untuk setiap sheet satu set. Bentuk sampul form menyusul di C4. -->
		<section aria-labelledby="cover-h">
			<h2 id="cover-h">{t('coverTitle')}</h2>
			<p class="hint">{t('coverIntro')}</p>
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
				<dt>{t(doc.status === 'draft' ? 'documentDraftRevision' : 'documentReleasedRevision')}</dt>
				<dd>{doc.revision}</dd>
				{#if doc.status === 'draft' && doc.released_revision !== undefined}
					<dt>{t('documentReleasedRevision')}</dt>
					<dd>{doc.released_revision}</dd>
				{/if}
			</dl>
			{#if doc.status === 'released'}<p class="hint">{t('documentNoDraft')}</p>{/if}
		</section>
	{:else if previewing}
		<!-- Pratinjau lembar (C2, C3): baca saja, per halaman, sama dengan PDF dan Excel. -->
		<section aria-labelledby="preview-h">
			<h2 id="preview-h">{t('sheetPreviewTitle')}</h2>
			{#if previewError}
				<p class="error" role="alert">{previewError}</p>
			{:else if !preview}
				<p>{t('formatLoading')}</p>
			{:else}
				{#if previewChecks.length > 0}
					<div class="preview-checks" data-testid="preview-pfd-check">
						<h3>{t('pfdCheckTitle')}</h3>
						<ul>{#each previewChecks as text, i (i)}<li>{text}</li>{/each}</ul>
					</div>
				{/if}
				<PreviewSheet plan={preview} />
			{/if}
		</section>
	{:else}
		{#if doc.status === 'released'}
			<p class="hint">{t('documentNoDraft')}</p>
		{/if}
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
			<h2>{t('documentRowsTitle')}</h2>
			<SectionBar {doc} current={section} {canEdit} onselect={(key) => (sectionKey = key)} onchanged={sectionChanged} />
			{#if section}<h3 class="section-title">{section.name}{section.part_numbers.length > 0 ? ` · ${section.part_numbers.map((p) => p.number).join(', ')}` : ''}</h3>{/if}
			{#if canEdit}<p class="hint">{t('documentGridHint')}</p>{/if}
			{#key doc.id + sectionId}
				<Workbook doc={shown ?? doc} {format} {processes} {canEdit} onsave={saveCell} ondelete={removeRow} />
			{/key}
			{#if isPfd && showDiagram}
				<h3>{t('diagramTitle')}</h3>
				<p class="hint">{t('diagramHint')}</p>
				<!-- Dimuat hanya saat dibuka: Svelte Flow dan ELK cukup besar (M4.10). -->
				{#await import('#lib/documents/PfdDiagram.svelte')}
					<p>{t('formatLoading')}</p>
				{:then diagram}
					<diagram.default rows={shown?.rows ?? doc.rows} {format} />
				{/await}
			{/if}
		</section>
	{/if}

	</div>
	{#if drawer}
		{@const open = drawer}
		<Drawer title={t(drawerTitle[open])} onclose={() => (drawer = null)}>
			{#if open === 'compose' && canEdit}
				{#key doc.id + sectionId}
					<ComposePanel {doc} sectionKey={section?.key ?? ''} oncomposed={composed} oncancel={() => (drawer = null)} />
				{/key}
			{:else if open === 'pfd' && canEdit}
				{#key doc.id + doc.rows.length}
					<PfdComposePanel {doc} oncomposed={composedFromPfd} oncancel={() => (drawer = null)} />
				{/key}
			{:else if open === 'compare'}
				{#key compareRound}
					<ComparePanel {doc} candetach={canEdit} ondetached={detached} oncancel={() => (drawer = null)} />
				{/key}
			{:else if open === 'copy' && canEdit}
				<CopyPanel {doc} oncopied={copied} oncancel={() => (drawer = null)} />
			{:else if open === 'order' && canEdit}
				{#key sectionId}
					<OrderPanel doc={shown ?? doc} onsave={saveOrder} oncancel={() => (drawer = null)} />
				{/key}
			{:else if open === 'paste' && canEdit}
				<PastePanel {format} {processes} onimport={pasteRows} oncancel={() => (drawer = null)} />
			{:else if open === 'templates'}
				<TemplateLinksPanel {doc} />
			{:else if open === 'revisions'}
				<RevisionsPanel {doc} onreleased={(d) => (doc = d)} />
			{:else if open === 'print'}
				<PrintPanel {doc} />
			{:else if open === 'signature'}
				<SignaturePanel {doc} onchanged={() => doc && load(doc.id)} />
			{:else if open === 'submission'}
				<SubmissionPanel {doc} />
			{:else if open === 'migration'}
				<MigrationPanel {doc} onchanged={(d) => (doc = d)} />
			{:else if open === 'excel'}
				<ExcelComparePanel {doc} />
			{:else if open === 'link'}
				<LinkPanel {doc} onlinked={(d) => (doc = d)} />
			{/if}
		</Drawer>
	{/if}
	</div>

	<SheetTabs set={doc.set} documentId={doc.id} current={sheet === 'cover' ? 'cover' : doc.id} />
{/if}

<style>
	.preview-checks {
		color: var(--warning);
		margin: 0 0 12px;
	}

	.preview-checks h3 {
		margin: 0 0 4px;
	}

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


	.back {
		margin: 0 0 8px;
	}

	/* Workbook memakai lebar layar penuh, seperti Excel. */
	:global(main.page:has(.wb-toolbar)) {
		max-width: none;
	}

	.wb-body {
		display: flex;
		align-items: flex-start;
		gap: 20px;
	}

	.wb-main {
		flex: 1 1 0;
		min-width: 0;
	}

	@media (max-width: 1023px) {
		.wb-body {
			flex-wrap: wrap;
		}

		.wb-main {
			flex-basis: 100%;
		}
	}

	.wb-head {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-start;
		justify-content: space-between;
		gap: 16px;
		margin-bottom: 16px;
	}

	.wb-head h1 {
		margin: 0 0 6px;
	}

	.wb-sub {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 12px;
		margin: 0;
		color: var(--text-3);
	}

	.form-set {
		background: var(--accent-weak);
		color: var(--accent-ink);
	}

	/* Kotak langkah berikutnya: satu aksi utama (D-145). */
	.next {
		max-width: 420px;
		padding: 14px 18px;
		border: 1px solid var(--accent-ring);
		border-radius: var(--radius-lg);
		background: var(--accent-weak);
	}

	.next h2 {
		margin: 0 0 4px;
		font-size: 0.95rem;
	}

	.next p {
		margin: 0 0 10px;
	}

	/* Toolbar berkelompok seperti pita Excel: tombol di atas, nama kelompok di bawah. */
	.wb-toolbar {
		display: flex;
		flex-wrap: nowrap;
		overflow-x: auto;
		gap: 0;
		margin: 0 0 20px;
		padding: 8px 4px;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}

	.wb-group {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		padding: 0 12px;
	}

	.wb-group + .wb-group {
		border-left: 1px solid var(--border);
	}

	.wb-buttons {
		display: flex;
		flex-wrap: nowrap;
		gap: 2px;
	}

	.wb-buttons button {
		white-space: nowrap;
	}

	.wb-buttons button {
		border: none;
		background: none;
		box-shadow: none;
		color: var(--ink);
		font-weight: 500;
		padding: 6px 10px;
	}

	.wb-buttons button:hover,
	.wb-buttons button[aria-pressed='true'] {
		background: var(--accent-weak);
		color: var(--accent-ink);
	}

	.wb-label {
		font-size: 0.8rem;
		color: var(--muted);
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
