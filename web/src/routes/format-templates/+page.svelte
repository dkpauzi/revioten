<script lang="ts">
	// Halaman format template (M3.1, UC-20, R-TPL-01, R-TPL-02, R-TPL-11):
	// field header dan kolom ber-ID tetap, metode risiko, tabel skala, simbol,
	// dan aturan sorot. Semua akun bisa melihat (L-2); mengubah hanya untuk
	// pemegang A-2. Server tetap memeriksa setiap operasi.
	//
	// Sejak M3.4 (R-TPL-05) versi berjalan dari draft ke terbit. Versi terbit
	// hanya bisa dibaca; perubahan berikutnya masuk ke versi baru.
	import { onMount } from 'svelte';
	import {
		addFormatField,
		createFormatTemplate,
		createFormatVersion,
		getFormatVersion,
		listCustomers,
		listDocumentTypes,
		listFormatTemplates,
		publishFormatVersion,
		renameFormatTemplate,
		saveFormatSymbols,
		setFormatRatingCriteria,
		updateFormatHighlights,
		type ApiFailure
	} from '#lib/api/client.ts';
	import type {
		Customer,
		DocumentType,
		FormatField,
		FormatHighlights,
		FormatTemplate,
		FormatVersion,
		NewFormatField,
		NewFormatTemplate
	} from '#lib/api/gen/index.ts';
	import { failureText, problemText, type ProblemScope } from '#lib/errors.ts';
	import ApTable from '#lib/formats/ApTable.svelte';
	import FieldTable from '#lib/formats/FieldTable.svelte';
	import {
		automaticSymbolNames,
		kindName,
		mergeSymbols,
		methodologyText,
		ratingsOf,
		riskText,
		scaleText,
		symbolLines,
		templateLabel
	} from '#lib/formats.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';
	import VersionBar from '#lib/templates/VersionBar.svelte';
	import { isDraft, latestVersion } from '#lib/versions.ts';

	const kinds: FormatField['kind'][] = ['text', 'date', 'select', 'symbol', 'rating', 'rpn', 'operation_number', 'process', 'process_flow'];
	const scales = ['severity', 'occurrence', 'detection'] as const;

	// Data dari server. null berarti belum selesai dimuat.
	let templates = $state<FormatTemplate[] | null>(null);
	let documentTypes = $state<DocumentType[]>([]);
	let customers = $state<Customer[]>([]);
	let loadError = $state('');
	let canEdit = $derived(can('A-2'));

	// Template yang dipilih dan versinya.
	let selectedId = $state('');
	let selected = $derived(templates?.find((ft) => ft.id === selectedId) ?? null);
	let version = $state<FormatVersion | null>(null);
	// Isi versi hanya bisa diubah oleh pemegang A-2, dan hanya selama versinya
	// draft (M3.4). Nama template bukan bagian dari versi, jadi tetap memakai canEdit.
	let editable = $derived(canEdit && isDraft(version));

	let message = $state('');
	let failure = $state('');
	let fieldErrors = $state<Record<string, string>>({});

	// Isian per formulir.
	let newName = $state('');
	let field = $state<{ area: FormatField['area']; kind: string; label: string; required: boolean; scale: string; set: number; options: string }>(
		{ area: 'row', kind: '', label: '', required: false, scale: '', set: 1, options: '' }
	);
	let criteria = $state<Record<string, string>>({});
	let symbolsText = $state('');
	let rpnAbove = $state(0);
	let severityAtLeast = $state(0);
	let draft = $state<{ name: string; documentTypeId: string; methodology: string; customerId: string; riskMethod: string; copyFrom: string }>(
		{ name: '', documentTypeId: '', methodology: '', customerId: '', riskMethod: '', copyFrom: '' }
	);

	// Versi yang bisa disalin: versi terakhir setiap template dengan jenis dokumen yang dipilih.
	let copySources = $derived((templates ?? []).filter((ft) => ft.document_type.id === draft.documentTypeId));

	async function load() {
		const [templateResult, typeResult, customerResult] = await Promise.all([
			listFormatTemplates(),
			listDocumentTypes(),
			listCustomers()
		]);
		if (!templateResult.ok) {
			loadError = failureText(templateResult.error);
			return;
		}
		if (!typeResult.ok) {
			loadError = failureText(typeResult.error);
			return;
		}
		if (!customerResult.ok) {
			loadError = failureText(customerResult.error);
			return;
		}
		templates = templateResult.data.items;
		// Hanya PFD, PFMEA, dan CP yang punya format template (R-TPL-01).
		documentTypes = typeResult.data.items.filter((d) => d.key !== undefined);
		customers = customerResult.data.items.filter((c) => c.active);
		loadError = '';
	}

	// show memuat versi terakhir template yang dipilih: draft yang sedang
	// dikerjakan, atau versi terbit terbaru (M3.4).
	async function show() {
		version = null;
		message = '';
		failure = '';
		const latest = selected === null ? null : latestVersion(selected.versions);
		if (selected === null || latest === null) {
			return;
		}
		newName = selected.name;
		await showVersion(latest.id);
	}

	// showVersion memuat satu versi, misalnya versi lama yang dipilih dari daftar.
	async function showVersion(versionId: string) {
		reset();
		const result = await getFormatVersion(versionId);
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		use(result.data);
	}

	// publish menerbitkan versi draft yang sedang dibuka (A-2). Daftar template
	// dimuat ulang supaya status versinya ikut berubah.
	async function publish() {
		reset();
		if (version === null) {
			return;
		}
		const result = await publishFormatVersion(version.id);
		if (!result.ok) {
			showErrors(result.error, 'formatTemplate');
			return;
		}
		await load();
		saved(result.data, t('versionPublished', { version: String(result.data.version) }));
	}

	// newVersion membuat versi draft berikutnya, salinan versi terbit terakhir (A-2).
	async function newVersion() {
		reset();
		if (selected === null || version === null) {
			return;
		}
		const from = latestVersion(selected.versions)?.version ?? version.version;
		const result = await createFormatVersion(selected.id);
		if (!result.ok) {
			showErrors(result.error, 'formatTemplate');
			return;
		}
		await load();
		saved(result.data, t('versionCreated', { version: String(result.data.version), from: String(from) }));
	}

	// use memakai versi dari server dan mengisi ulang formulir yang bergantung padanya.
	function use(v: FormatVersion) {
		version = v;
		criteria = Object.fromEntries(v.ratings.map((r) => [`${r.scale}-${r.value}`, r.criteria]));
		symbolsText = symbolLines(v.symbols);
		rpnAbove = v.highlight_rpn_above;
		severityAtLeast = v.highlight_severity_at_least;
	}

	function saved(v: FormatVersion, text: string) {
		use(v);
		message = text;
		failure = '';
	}

	// showErrors menampilkan masalah per field, atau pesan umum bila tidak ada.
	function showErrors(error: ApiFailure, scope?: ProblemScope) {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, scope)]));
		if (details.length === 0) {
			failure = failureText(error);
		}
	}

	function reset() {
		message = '';
		failure = '';
		fieldErrors = {};
	}

	async function rename(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (selected === null) {
			return;
		}
		const result = await renameFormatTemplate(selected.id, newName);
		if (!result.ok) {
			showErrors(result.error, 'formatTemplate');
			return;
		}
		message = t('formatSaved', { name: result.data.name });
		await load();
	}

	async function addField(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (version === null) {
			return;
		}
		const input: NewFormatField = {
			area: field.area,
			kind: field.kind as FormatField['kind'],
			label: field.label,
			required: field.required
		};
		if (field.kind === 'rating') {
			input.rating_scale = (field.scale || undefined) as NewFormatField['rating_scale'];
		}
		if (field.kind === 'rating' || field.kind === 'rpn') {
			input.rating_set = field.set;
		}
		if (field.kind === 'select') {
			input.options = field.options
				.split('\n')
				.map((line) => line.trim())
				.filter((line) => line !== '')
				.map((label) => ({ label }));
		}
		const result = await addFormatField(version.id, input);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		saved(result.data, t('formatFieldAdded', { label: input.label }));
		field = { area: field.area, kind: '', label: '', required: false, scale: '', set: 1, options: '' };
	}

	async function saveRatings(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (version === null) {
			return;
		}
		const items = version.ratings.map((r) => ({ scale: r.scale, value: r.value, criteria: criteria[`${r.scale}-${r.value}`] ?? '' }));
		const result = await setFormatRatingCriteria(version.id, items);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		saved(result.data, t('formatRatingsSaved'));
	}

	async function saveSymbols(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (version === null) {
			return;
		}
		const result = await saveFormatSymbols(version.id, mergeSymbols(version.symbols, symbolsText));
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		saved(result.data, t('formatSymbolsSaved'));
	}

	async function saveHighlights(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (version === null) {
			return;
		}
		// Aturan RPN hanya untuk metode RPN; kotak yang dikosongkan berarti 0 (mati).
		const input: FormatHighlights = { highlight_severity_at_least: severityAtLeast ?? 0 };
		if (version.risk_method === 'rpn') {
			input.highlight_rpn_above = rpnAbove ?? 0;
		}
		const result = await updateFormatHighlights(version.id, input);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		saved(result.data, t('formatHighlightsSaved'));
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		reset();
		const input: NewFormatTemplate = { name: draft.name, document_type_id: draft.documentTypeId };
		if (draft.methodology) {
			input.methodology = draft.methodology as NewFormatTemplate['methodology'];
		}
		if (draft.customerId) {
			input.customer_id = draft.customerId;
		}
		if (draft.copyFrom) {
			input.copy_from_version_id = draft.copyFrom;
		} else if (draft.riskMethod) {
			input.risk_method = draft.riskMethod as NewFormatTemplate['risk_method'];
		}
		const result = await createFormatTemplate(input);
		if (!result.ok) {
			showErrors(result.error, 'formatTemplate');
			return;
		}
		draft = { name: '', documentTypeId: '', methodology: '', customerId: '', riskMethod: '', copyFrom: '' };
		await load();
		// Template baru langsung dibuka, supaya field-nya bisa ditambah.
		selectedId = result.data.id;
		await show();
		message = t('formatCreated', { name: result.data.name });
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('formatTitle')}</h1>
<p>{t('formatIntro')}</p>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if templates === null}
	<p>{t('formatLoading')}</p>
{:else}
	{#if !canEdit}<p class="hint">{t('formatReadOnly')}</p>{/if}
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<div class="field chooser">
		<label for="format-choose">{t('formatChoose')}</label>
		<select id="format-choose" bind:value={selectedId} onchange={show}>
			<option value="">{t('formatChooseOption')}</option>
			{#each templates as ft (ft.id)}
				<option value={ft.id}>{templateLabel(ft)}</option>
			{/each}
		</select>
	</div>

	{#if selected && version}
		<section>
			<dl>
				<dt>{t('formatDocumentType')}</dt>
				<dd>{selected.document_type.name}</dd>
				<dt>{t('formatMethodology')}</dt>
				<dd>{methodologyText(selected.methodology) || t('formatMethodologyNone')}</dd>
				<dt>{t('formatCustomer')}</dt>
				<dd>{selected.customer_code ?? t('formatCustomerAll')}</dd>
				<dt>{t('formatRiskMethod')}</dt>
				<dd>{riskText(version.risk_method)}</dd>
			</dl>
			<VersionBar
				id="format-template"
				versions={selected.versions}
				current={version}
				{canEdit}
				onselect={showVersion}
				onpublish={publish}
				onnew={newVersion}
			/>
			{#if fieldErrors.status}<p class="error" role="alert">{fieldErrors.status}</p>{/if}
			{#if canEdit}
				<form class="row" onsubmit={rename} novalidate>
					<div class="field">
						<label for="format-name">{t('formatName')}</label>
						<input id="format-name" bind:value={newName} autocomplete="off" maxlength="200" size="40" />
						{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
					</div>
					<button type="submit">{t('formatRename')}</button>
				</form>
			{/if}
		</section>

		<section>
			<h2>{t('formatHeaderFields')}</h2>
			<FieldTable {version} area="header" canEdit={editable} onsaved={saved} />
			<h2>{t('formatColumns')}</h2>
			<FieldTable {version} area="row" canEdit={editable} onsaved={saved} />
		</section>

		{#if editable}
			<section>
				<h2>{t('formatAddFieldTitle')}</h2>
				<form class="row" onsubmit={addField} novalidate>
					<div class="field">
						<label for="new-field-area">{t('formatArea')}</label>
						<select id="new-field-area" bind:value={field.area}>
							<option value="row">{t('formatAreaRow')}</option>
							<option value="header">{t('formatAreaHeader')}</option>
						</select>
					</div>
					<div class="field">
						<label for="new-field-kind">{t('formatKind')}</label>
						<select id="new-field-kind" bind:value={field.kind}>
							<option value="">{t('formatKindChoose')}</option>
							{#each kinds as kind (kind)}
								<option value={kind}>{kindName(kind)}</option>
							{/each}
						</select>
						{#if fieldErrors.kind}<p class="error">{fieldErrors.kind}</p>{/if}
					</div>
					<div class="field">
						<label for="new-field-label">{t('formatColumnLabel')}</label>
						<input id="new-field-label" bind:value={field.label} autocomplete="off" maxlength="100" />
						{#if fieldErrors.label}<p class="error">{fieldErrors.label}</p>{/if}
					</div>
					{#if field.kind === 'rating'}
						<div class="field">
							<label for="new-field-scale">{t('formatRatingScale')}</label>
							<select id="new-field-scale" bind:value={field.scale}>
								<option value=""></option>
								{#each scales as scale (scale)}
									<option value={scale}>{scaleText(scale)}</option>
								{/each}
							</select>
							{#if fieldErrors.rating_scale}<p class="error">{fieldErrors.rating_scale}</p>{/if}
						</div>
					{/if}
					{#if field.kind === 'rating' || field.kind === 'rpn'}
						<div class="field">
							<label for="new-field-set">{t('formatRatingSet')}</label>
							<select id="new-field-set" bind:value={field.set}>
								<option value={1}>{t('formatSetCurrent')}</option>
								<option value={2}>{t('formatSetResult')}</option>
							</select>
							{#if fieldErrors.rating_set}<p class="error">{fieldErrors.rating_set}</p>{/if}
						</div>
					{/if}
					{#if field.kind === 'select'}
						<div class="field">
							<label for="new-field-options">{t('formatOptions')}</label>
							<textarea id="new-field-options" bind:value={field.options} rows="4"></textarea>
							{#if fieldErrors.options}<p class="error">{fieldErrors.options}</p>{/if}
						</div>
					{/if}
					<label class="check"><input type="checkbox" bind:checked={field.required} /> {t('formatRequired')}</label>
					<button type="submit">{t('formatAddField')}</button>
				</form>
			</section>
		{/if}

		{#if version.risk_method !== 'none'}
			<section>
				<h2>{t('formatHighlightsTitle')}</h2>
				<p class="hint">{t('formatHighlightsIntro')}</p>
				<form class="row" onsubmit={saveHighlights} novalidate>
					{#if version.risk_method === 'rpn'}
						<div class="field">
							<label for="highlight-rpn">{t('formatHighlightRpn')}</label>
							<input id="highlight-rpn" type="number" min="0" max="999" bind:value={rpnAbove} disabled={!editable} />
							{#if fieldErrors.highlight_rpn_above}<p class="error">{fieldErrors.highlight_rpn_above}</p>{/if}
						</div>
					{/if}
					<div class="field">
						<label for="highlight-severity">{t('formatHighlightSeverity')}</label>
						<input id="highlight-severity" type="number" min="0" max="10" bind:value={severityAtLeast} disabled={!editable} />
						{#if fieldErrors.highlight_severity_at_least}<p class="error">{fieldErrors.highlight_severity_at_least}</p>{/if}
					</div>
					{#if editable}<button type="submit">{t('formatHighlightsSave')}</button>{/if}
				</form>
			</section>

			<section>
				<h2>{t('formatRatingsTitle')}</h2>
				<p class="hint">{t('formatRatingsIntro')}</p>
				<form onsubmit={saveRatings} novalidate>
					<div class="scales">
						{#each scales as scale (scale)}
							<table>
								<caption>{scaleText(scale)}</caption>
								<thead>
									<tr>
										<th>{t('formatRatingValue')}</th>
										<th>{t('formatRatingCriteria')}</th>
									</tr>
								</thead>
								<tbody>
									{#each ratingsOf(version, scale) as r (r.value)}
										<tr>
											<td>{r.value}</td>
											<td>
												<input
													aria-label={t('formatRatingCriteriaFor', { scale: scaleText(scale), value: String(r.value) })}
													bind:value={criteria[`${scale}-${r.value}`]}
													maxlength="500"
													size="30"
													disabled={!editable}
												/>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						{/each}
					</div>
					{#if fieldErrors.ratings}<p class="error">{fieldErrors.ratings}</p>{/if}
					{#if editable}<button type="submit">{t('formatRatingsSave')}</button>{/if}
				</form>
			</section>
		{/if}

		{#if version.risk_method === 'ap'}
			<!-- Tabel Action Priority (form set langkah A, D-134). -->
			<section>
				<h2>{t('apTableTitle')}</h2>
				<p class="hint">{t('apTableIntro')}</p>
				<ApTable rules={version.ap_rules} />
			</section>
		{/if}

		<section>
			<h2>{t('formatSymbolsTitle')}</h2>
			<p class="hint">{t('formatSymbolsIntro')}</p>
			{#if automaticSymbolNames(version.symbols)}
				<p class="hint">{t('formatSymbolsAutomatic', { names: automaticSymbolNames(version.symbols) })}</p>
			{/if}
			<form class="row" onsubmit={saveSymbols} novalidate>
				<div class="field">
					<label for="format-symbols">{t('formatSymbols')}</label>
					<textarea id="format-symbols" bind:value={symbolsText} rows="5" disabled={!editable}></textarea>
					{#if fieldErrors.symbols}<p class="error">{fieldErrors.symbols}</p>{/if}
				</div>
				{#if editable}<button type="submit">{t('formatSymbolsSave')}</button>{/if}
			</form>
		</section>
	{/if}

	{#if canEdit}
		<section>
			<h2>{t('formatCreateTitle')}</h2>
			<form class="row" onsubmit={create} novalidate>
				<div class="field">
					<label for="create-name">{t('formatName')}</label>
					<input id="create-name" bind:value={draft.name} autocomplete="off" maxlength="200" />
					{#if !selected && fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
				</div>
				<div class="field">
					<label for="create-type">{t('formatDocumentType')}</label>
					<select id="create-type" bind:value={draft.documentTypeId} onchange={() => (draft.copyFrom = '')}>
						<option value=""></option>
						{#each documentTypes as d (d.id)}
							<option value={d.id}>{d.name}</option>
						{/each}
					</select>
					{#if fieldErrors.document_type_id}<p class="error">{fieldErrors.document_type_id}</p>{/if}
				</div>
				<div class="field">
					<label for="create-methodology">{t('formatMethodology')}</label>
					<select id="create-methodology" bind:value={draft.methodology}>
						<option value="">{t('formatMethodologyNone')}</option>
						<option value="aiag_4th">{t('formatMethodAiag4th')}</option>
						<option value="aiag_vda">{t('formatMethodAiagVda')}</option>
						<option value="sqam">{t('formatMethodSqam')}</option>
					</select>
				</div>
				<div class="field">
					<label for="create-customer">{t('formatCustomer')}</label>
					<select id="create-customer" bind:value={draft.customerId}>
						<option value="">{t('formatCustomerAll')}</option>
						{#each customers as c (c.id)}
							<option value={c.id}>{c.code} · {c.name}</option>
						{/each}
					</select>
					{#if fieldErrors.customer_id}<p class="error">{fieldErrors.customer_id}</p>{/if}
				</div>
				<div class="field">
					<label for="create-copy">{t('formatCopyFrom')}</label>
					<select id="create-copy" bind:value={draft.copyFrom}>
						<option value="">{t('formatCopyNone')}</option>
						{#each copySources as ft (ft.id)}
							<option value={latestVersion(ft.versions)?.id}>{templateLabel(ft)}</option>
						{/each}
					</select>
					{#if fieldErrors.copy_from_version_id}<p class="error">{fieldErrors.copy_from_version_id}</p>{/if}
				</div>
				{#if !draft.copyFrom}
					<div class="field">
						<label for="create-risk">{t('formatRiskMethod')}</label>
						<select id="create-risk" bind:value={draft.riskMethod}>
							<option value="">{t('formatRiskChoose')}</option>
							<option value="rpn">{t('formatRiskRpn')}</option>
							<option value="ap">{t('formatRiskAp')}</option>
							<option value="none">{t('formatRiskNone')}</option>
						</select>
						{#if fieldErrors.risk_method}<p class="error">{fieldErrors.risk_method}</p>{/if}
					</div>
				{/if}
				<button type="submit">{t('formatCreate')}</button>
			</form>
		</section>
	{/if}
{/if}

<style>
	section {
		margin-bottom: 2rem;
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	/* Field dirapikan ke atas; tombol diturunkan setinggi label supaya sejajar
	   dengan kotak isian, juga bila ada teks bantuan atau pesan masalah. */
	.row > button {
		margin-top: var(--label-space);
	}

	.field {
		margin-bottom: 1rem;
	}

	.chooser {
		align-items: flex-start;
	}

	dl {
		display: grid;
		grid-template-columns: max-content auto;
		gap: 0.25rem 1rem;
	}

	dd {
		margin: 0;
	}

	.scales {
		display: flex;
		flex-wrap: wrap;
		gap: 1.5rem;
		align-items: flex-start;
	}

	.check {
		display: flex;
		gap: 0.25rem;
		align-items: center;
		margin-bottom: 1rem;
	}
</style>
