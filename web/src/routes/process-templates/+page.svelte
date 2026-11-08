<script lang="ts">
	// Halaman process template atau pustaka proses (M3.2, UC-21, R-TPL-01,
	// R-TPL-04, D-64): sub-proses satu proses dan poinnya untuk PFD, PFMEA,
	// dan CP, masing-masing dengan ID tetap. Semua akun bisa melihat (L-2);
	// mengubah hanya untuk pemegang E-6 (Pengelola template, PPAP Engineer).
	// Server tetap memeriksa setiap operasi.
	//
	// Sejak M3.3 (D-129) poin punya cakupan dan varian, dan bagian pratinjau
	// menampilkan poin yang berlaku untuk satu model (R-TPL-03).
	import { onMount } from 'svelte';
	import {
		addSubprocess,
		addTemplatePoint,
		createProcessTemplate,
		getProcessTemplateVersion,
		listCustomers,
		listDocumentTypes,
		listModels,
		listProcessTemplates,
		listProcesses,
		resolveProcessTemplatePoints,
		type ApiFailure
	} from '#lib/api/client.ts';
	import type {
		Customer,
		DocumentType,
		Model,
		Process,
		ProcessTemplate,
		ProcessTemplateVersion,
		ResolvedPoints
	} from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import PointTable from '#lib/processTemplates/PointTable.svelte';
	import ScopePicker from '#lib/processTemplates/ScopePicker.svelte';
	import SubprocessTable from '#lib/processTemplates/SubprocessTable.svelte';
	import { pointTypes, pointsOf, scopeLevels, scopeText, templateOf, versionText, type ScopeLevel } from '#lib/processTemplates.ts';
	import { can } from '#lib/session.svelte.ts';

	// Data dari server. null berarti belum selesai dimuat.
	let processes = $state<Process[] | null>(null);
	let templates = $state<ProcessTemplate[]>([]);
	let types = $state<DocumentType[]>([]);
	let customers = $state<Customer[]>([]);
	let loadError = $state('');
	let canEdit = $derived(can('E-6'));

	// Proses yang dipilih, template dan versinya, dan sub-proses yang poinnya dilihat.
	let processId = $state('');
	let selectedProcess = $derived(processes?.find((p) => p.id === processId) ?? null);
	let template = $derived(templateOf(templates, processId));
	let version = $state<ProcessTemplateVersion | null>(null);
	let subprocessId = $state('');
	let subprocess = $derived(version?.subprocesses.find((s) => s.id === subprocessId) ?? null);

	let message = $state('');
	let failure = $state('');
	let fieldErrors = $state<Record<string, string>>({});

	// Isian formulir tambah.
	let newSubprocess = $state('');
	let newPoint = $state<{ documentTypeId: string; text: string; scopeLevel: ScopeLevel; scopeTarget: string }>({
		documentTypeId: '',
		text: '',
		scopeLevel: 'general',
		scopeTarget: ''
	});

	// Pratinjau poin untuk satu model (M3.3).
	let previewCustomerId = $state('');
	let previewModels = $state<Model[]>([]);
	let previewModelId = $state('');
	let preview = $state<ResolvedPoints | null>(null);
	let previewError = $state('');

	async function load() {
		const [processResult, templateResult, typeResult, customerResult] = await Promise.all([
			listProcesses(),
			listProcessTemplates(),
			listDocumentTypes(),
			listCustomers()
		]);
		// Daftar customer hanya untuk memilih cakupan dan pratinjau; tanpa daftar
		// ini halaman tetap bisa dipakai untuk poin umum.
		customers = customerResult.ok ? customerResult.data.items : [];
		if (!processResult.ok) {
			loadError = failureText(processResult.error);
			return;
		}
		if (!templateResult.ok) {
			loadError = failureText(templateResult.error);
			return;
		}
		if (!typeResult.ok) {
			loadError = failureText(typeResult.error);
			return;
		}
		processes = processResult.data.items;
		templates = templateResult.data.items;
		// Hanya PFD, PFMEA, dan CP yang punya poin (R-TPL-01).
		types = pointTypes(typeResult.data.items);
		loadError = '';
	}

	// show memuat versi pertama template proses yang dipilih. Versi berikutnya datang di M3.4.
	async function show() {
		version = null;
		subprocessId = '';
		reset();
		if (template === null) {
			return;
		}
		const result = await getProcessTemplateVersion(template.versions[0].id);
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		version = result.data;
	}

	function saved(v: ProcessTemplateVersion, text: string) {
		version = v;
		message = text;
		failure = '';
		fieldErrors = {};
		// Isi versi berubah, jadi pratinjau dimuat ulang bila sedang terbuka.
		if (previewModelId !== '') {
			showPreview();
		}
	}

	async function choosePreviewCustomer() {
		previewModelId = '';
		previewModels = [];
		preview = null;
		previewError = '';
		if (previewCustomerId === '') {
			return;
		}
		const result = await listModels(previewCustomerId);
		previewModels = result.ok ? result.data.items : [];
	}

	async function showPreview() {
		preview = null;
		previewError = '';
		if (version === null || previewModelId === '') {
			return;
		}
		const result = await resolveProcessTemplatePoints(version.id, previewModelId);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			previewError = details.length > 0 ? problemText(details[0], 'processTemplate') : failureText(result.error);
			return;
		}
		preview = result.data;
	}

	// showErrors menampilkan masalah per field, atau pesan umum bila tidak ada.
	function showErrors(error: ApiFailure) {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'processTemplate')]));
		if (details.length === 0) {
			failure = failureText(error);
		}
	}

	function reset() {
		message = '';
		failure = '';
		fieldErrors = {};
	}

	async function create() {
		reset();
		if (selectedProcess === null) {
			return;
		}
		const result = await createProcessTemplate(selectedProcess.id);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		await load();
		await show();
		message = t('processTemplateCreated', { process: result.data.process.name });
	}

	async function addSub(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (version === null) {
			return;
		}
		const name = newSubprocess;
		const result = await addSubprocess(version.id, { name });
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		saved(result.data, t('subprocessAdded', { name: name.trim() }));
		newSubprocess = '';
	}

	async function addPoint(event: SubmitEvent) {
		event.preventDefault();
		reset();
		if (version === null || subprocess === null) {
			return;
		}
		const result = await addTemplatePoint(version.id, {
			subprocess_id: subprocess.id,
			document_type_id: newPoint.documentTypeId,
			text: newPoint.text,
			scope_level: newPoint.scopeLevel,
			scope_target_id: newPoint.scopeLevel === 'general' ? undefined : newPoint.scopeTarget
		});
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		saved(result.data, t('pointAdded'));
		// Jenis dokumen dan cakupan tetap terpilih, supaya beberapa poin bisa ditambah berturut-turut.
		newPoint = { ...newPoint, text: '' };
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('processTemplateTitle')}</h1>
<p>{t('processTemplateIntro')}</p>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if processes === null}
	<p>{t('processTemplateLoading')}</p>
{:else if processes.length === 0}
	<p>{t('processTemplateNoProcesses')}</p>
{:else}
	{#if !canEdit}<p class="hint">{t('processTemplateReadOnly')}</p>{/if}
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<div class="field chooser">
		<label for="process-choose">{t('processTemplateChoose')}</label>
		<select id="process-choose" bind:value={processId} onchange={show}>
			<option value="">{t('processTemplateChooseOption')}</option>
			{#each processes as p (p.id)}
				<option value={p.id}>{p.name}</option>
			{/each}
		</select>
	</div>

	{#if selectedProcess && template === null}
		<section>
			<p>{t('processTemplateNone')}</p>
			{#if canEdit}
				<button type="button" onclick={create}>{t('processTemplateCreate')}</button>
				{#if fieldErrors.process_id}<p class="error">{fieldErrors.process_id}</p>{/if}
			{/if}
		</section>
	{/if}

	{#if selectedProcess && version}
		<section>
			<p>{versionText(version)}</p>
			<h2>{t('subprocessesTitle')}</h2>
			<SubprocessTable
				{version}
				{types}
				{canEdit}
				selectedId={subprocessId}
				onselect={(id) => {
					subprocessId = id;
					reset();
				}}
				onsaved={saved}
			/>
			{#if canEdit}
				<h3>{t('subprocessAddTitle')}</h3>
				<form class="row" onsubmit={addSub} novalidate>
					<div class="field">
						<label for="new-subprocess">{t('subprocessName')}</label>
						<input id="new-subprocess" bind:value={newSubprocess} autocomplete="off" maxlength="200" size="30" />
						{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
					</div>
					<button type="submit">{t('subprocessAdd')}</button>
				</form>
			{/if}
		</section>

		{#if subprocess}
			<section>
				<h2>{t('pointsTitle', { name: subprocess.name })}</h2>
				<p class="hint">{t('pointsIntro')}</p>
				{#each types as d (d.id)}
					<h3>{d.name}</h3>
					<PointTable {version} points={pointsOf(subprocess, d.id)} {customers} {canEdit} onsaved={saved} />
				{/each}
				{#if canEdit}
					<h3>{t('pointAddTitle', { name: subprocess.name })}</h3>
					<form class="row" onsubmit={addPoint} novalidate>
						<div class="field">
							<label for="new-point-type">{t('pointDocumentType')}</label>
							<select id="new-point-type" bind:value={newPoint.documentTypeId}>
								<option value="">{t('pointDocumentTypeChoose')}</option>
								{#each types as d (d.id)}
									<option value={d.id}>{d.name}</option>
								{/each}
							</select>
							{#if fieldErrors.document_type_id}<p class="error">{fieldErrors.document_type_id}</p>{/if}
						</div>
						<div class="field">
							<label for="new-point-text">{t('pointText')}</label>
							<input id="new-point-text" bind:value={newPoint.text} autocomplete="off" maxlength="500" size="50" />
							{#if fieldErrors.text}<p class="error">{fieldErrors.text}</p>{/if}
							{#if fieldErrors.subprocess_id}<p class="error">{fieldErrors.subprocess_id}</p>{/if}
						</div>
						<ScopePicker
							id="new-point-scope"
							{customers}
							levels={scopeLevels}
							bind:level={newPoint.scopeLevel}
							bind:targetId={newPoint.scopeTarget}
							errors={fieldErrors}
						/>
						<button type="submit">{t('pointAdd')}</button>
					</form>
				{/if}
			</section>
		{/if}

		<!-- Pratinjau: poin yang berlaku untuk satu model (M3.3, R-TPL-03). -->
		<section>
			<h2>{t('previewTitle')}</h2>
			<p class="hint">{t('previewIntro')}</p>
			<div class="row">
				<div class="field">
					<label for="preview-customer">{t('scopeLevelCustomer')}</label>
					<select id="preview-customer" bind:value={previewCustomerId} onchange={choosePreviewCustomer}>
						<option value="">{t('scopeCustomerChoose')}</option>
						{#each customers as c (c.id)}
							<option value={c.id}>{c.code} · {c.name}</option>
						{/each}
					</select>
				</div>
				<div class="field">
					<label for="preview-model">{t('scopeLevelModel')}</label>
					<select id="preview-model" bind:value={previewModelId} onchange={showPreview} disabled={previewCustomerId === ''}>
						<option value="">{t('scopeModelChoose')}</option>
						{#each previewModels as m (m.id)}
							<option value={m.id}>{m.name}{m.model_group ? ` · ${m.model_group.name}` : ''}</option>
						{/each}
					</select>
				</div>
			</div>
			{#if previewError}<p class="error" role="alert">{previewError}</p>{/if}
			{#if preview}
				{#if preview.subprocesses.length === 0}
					<p>{t('previewEmpty')}</p>
				{:else}
					<table>
						<caption>{preview.model.customer_code} · {preview.model.name}</caption>
						<thead>
							<tr>
								<th>{t('subprocessColumnName')}</th>
								<th>{t('pointDocumentType')}</th>
								<th>{t('pointColumnText')}</th>
								<th>{t('previewColumnFrom')}</th>
							</tr>
						</thead>
						<tbody>
							{#each preview.subprocesses as s (s.id)}
								{#each s.points as p (p.point_id)}
									<tr>
										<td>{s.name}</td>
										<td>{p.document_type.name}</td>
										<td>{p.text}</td>
										<td>{scopeText(p.scope)}</td>
									</tr>
								{/each}
							{/each}
						</tbody>
					</table>
				{/if}
			{/if}
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
</style>
