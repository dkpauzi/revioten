<script lang="ts">
	// Halaman Form sets (form set langkah D, D-133, D-139). Semua akun melihat
	// (L-2). PPAP Engineer (E-11) menyusun form set dan memilih form set setiap
	// customer; penyetuju form (V-3) menerbitkan versinya. Server tetap
	// memeriksa setiap operasi, termasuk bahwa penyetuju bukan penyusunnya.
	import { onMount } from 'svelte';
	import {
		createFormSet,
		createFormSetVersion,
		listFormSets,
		listFormatTemplates,
		publishFormSetVersion,
		setCustomerFormSet,
		setFormSetHidden,
		setFormSetSheets,
		type ApiFailure
	} from '#lib/api/client.ts';
	import type { FormSet, FormSetList, FormSetVersion, FormatTemplate } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formSetNameOf, sheetKeys, sheetOptions, sheetText, visibleItems, type SheetKey } from '#lib/formSets.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';
	import Approvals from '#lib/templates/Approvals.svelte';
	import VersionBar from '#lib/templates/VersionBar.svelte';
	import { latestVersion } from '#lib/versions.ts';

	const sheetLabels: Record<SheetKey, MessageKey> = { pfd: 'formSetSheetPfd', pfmea: 'formSetSheetPfmea', cp: 'formSetSheetCp' };

	// Data dari server. null berarti belum selesai dimuat.
	let list = $state<FormSetList | null>(null);
	let templates = $state<FormatTemplate[]>([]);
	let loadError = $state('');
	let canEdit = $derived(can('E-11'));
	let canApprove = $derived(can('V-3'));
	// D-218: form set tersembunyi hanya tampil bila Show hidden dicentang.
	let showHidden = $state(false);
	let visibleSets = $derived(visibleItems(list?.items ?? [], showHidden));

	// Versi yang sedang dibuka per form set (id form set ke id versi).
	let shown = $state<Record<string, string>>({});
	// Isian sheet versi draft yang sedang diubah (id versi ke pilihan).
	let sheetDrafts = $state<Record<string, Record<SheetKey, string>>>({});

	let message = $state('');
	let failure = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	// Masalah formulir form set baru dipisah, supaya tidak tampil di formulir sheet lain.
	let createErrors = $state<Record<string, string>>({});

	// Formulir form set baru.
	let draft = $state<{ name: string } & Record<SheetKey, string>>({ name: '', pfd: '', pfmea: '', cp: '' });

	async function load() {
		const [setResult, templateResult] = await Promise.all([listFormSets(), listFormatTemplates()]);
		if (!setResult.ok) {
			loadError = failureText(setResult.error);
			return;
		}
		if (!templateResult.ok) {
			loadError = failureText(templateResult.error);
			return;
		}
		list = setResult.data;
		templates = templateResult.data.items;
		loadError = '';
	}

	// current mengembalikan versi yang sedang dibuka satu form set: pilihan
	// pengguna, atau versi terakhirnya.
	function current(fs: FormSet): FormSetVersion | null {
		return fs.versions.find((v) => v.id === shown[fs.id]) ?? latestVersion(fs.versions);
	}

	// sheetsOf mengembalikan isian sheet satu versi untuk formulir ubah: yang
	// sudah diubah pengguna, atau isi versi itu. Tidak mengubah state, supaya
	// aman dipanggil saat halaman digambar.
	function sheetsOf(v: FormSetVersion): Record<SheetKey, string> {
		return sheetDrafts[v.id] ?? { pfd: v.pfd.format_version_id, pfmea: v.pfmea.format_version_id, cp: v.cp.format_version_id };
	}

	// setSheet menyimpan satu pilihan sheet di isian versi itu.
	function setSheet(v: FormSetVersion, key: SheetKey, id: string) {
		sheetDrafts[v.id] = { ...sheetsOf(v), [key]: id };
	}

	function reset() {
		message = '';
		failure = '';
		fieldErrors = {};
		createErrors = {};
	}

	function showErrors(error: ApiFailure) {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'formSet')]));
		if (details.length === 0) {
			failure = failureText(error);
		}
	}

	// replace memakai form set dari server di daftar.
	function replace(fs: FormSet) {
		if (list) {
			list.items = list.items.map((x) => (x.id === fs.id ? fs : x));
		}
	}

	async function saveSheets(fs: FormSet, v: FormSetVersion) {
		reset();
		const s = sheetsOf(v);
		const result = await setFormSetSheets(v.id, { pfd_format_version_id: s.pfd, pfmea_format_version_id: s.pfmea, cp_format_version_id: s.cp });
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		delete sheetDrafts[v.id];
		replace(result.data);
		message = t('formSetSheetsSaved', { name: fs.name });
	}

	// publish menerbitkan versi draft (V-3). Konfirmasi sudah diminta VersionBar.
	async function publish(fs: FormSet, v: FormSetVersion) {
		reset();
		const result = await publishFormSetVersion(v.id);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		replace(result.data);
		// Dengan aturan "semua penyetuju" (D-139), persetujuan bisa tercatat
		// tanpa versi langsung terbit.
		const after = result.data.versions.find((x) => x.id === v.id);
		message =
			after?.status === 'published'
				? t('formSetPublished', { version: String(v.version), name: fs.name })
				: t('approvalRecorded');
	}

	async function newVersion(fs: FormSet) {
		reset();
		const result = await createFormSetVersion(fs.id);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		replace(result.data);
		const latest = latestVersion(result.data.versions);
		if (latest) {
			shown[fs.id] = latest.id;
			message = t('formSetVersionCreated', { version: String(latest.version), name: fs.name });
		}
	}

	async function chooseCustomerSet(customerId: string, code: string, formSetId: string) {
		reset();
		const result = await setCustomerFormSet(customerId, formSetId);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		await load();
		message = t('formSetCustomerSaved', { code });
	}

	// setHidden menyembunyikan atau menampilkan satu form set (E-11, D-218).
	async function setHidden(fs: FormSet, hidden: boolean) {
		reset();
		const result = await setFormSetHidden(fs.id, hidden);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		await load();
		message = t(hidden ? 'formSetHidden' : 'formSetShown', { name: result.data.name });
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		reset();
		const result = await createFormSet({
			name: draft.name,
			pfd_format_version_id: draft.pfd,
			pfmea_format_version_id: draft.pfmea,
			cp_format_version_id: draft.cp
		});
		if (!result.ok) {
			showErrors(result.error);
			createErrors = fieldErrors;
			fieldErrors = {};
			return;
		}
		draft = { name: '', pfd: '', pfmea: '', cp: '' };
		await load();
		message = t('formSetCreated', { name: result.data.name });
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('formSetsTitle')}</h1>
<p>{t('formSetsIntro')}</p>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if list === null}
	<p>{t('formSetsLoading')}</p>
{:else}
	{#if !canEdit}<p class="hint">{t('formSetsReadOnly')}</p>{/if}
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	{#each ['status', 'sheets', 'approver', 'hidden'] as field (field)}
		{#if fieldErrors[field]}<p class="error" role="alert">{fieldErrors[field]}</p>{/if}
	{/each}

	<p class="hint">{t('formSetHiddenHint')}</p>
	<label class="check"><input type="checkbox" bind:checked={showHidden} /> {t('showHidden')}</label>

	{#each visibleSets as fs (fs.id)}
		{@const v = current(fs)}
		<section aria-labelledby="set-{fs.id}">
			<h2 id="set-{fs.id}">
				{fs.name}
				{#if fs.default}<span class="badge">{t('formSetDefault')}</span>{/if}
				{#if fs.hidden}<span class="badge">{t('hiddenMark')}</span>{/if}
			</h2>
			{#if fs.default}<p class="hint">{t('formSetDefaultHint')}</p>{/if}
			{#if canEdit && !fs.default}
				<p>
					<button type="button" onclick={() => setHidden(fs, !fs.hidden)}>
						{t(fs.hidden ? 'formSetShowFor' : 'formSetHideFor', { name: fs.name })}
					</button>
				</p>
			{/if}
			{#if v}
				<VersionBar
					id="form-set-{fs.key}"
					versions={fs.versions}
					current={v}
					{canEdit}
					canPublish={canApprove}
					onselect={(id) => (shown[fs.id] = id)}
					onpublish={() => publish(fs, v)}
					onnew={() => newVersion(fs)}
				/>
				<Approvals approvals={v.approvals} pending={v.pending_approvers} />
				{#if v.status === 'draft' && (canEdit || canApprove)}
					<p class="hint">{t('formSetPublishHint')}</p>
				{/if}

				{#if v.status === 'draft' && canEdit}
					<form class="sheets" onsubmit={(e) => (e.preventDefault(), saveSheets(fs, v))} novalidate>
						{#each sheetKeys as key (key)}
							<div class="field">
								<label for="sheet-{v.id}-{key}">{t(sheetLabels[key])}</label>
								<select id="sheet-{v.id}-{key}" bind:value={() => sheetsOf(v)[key], (id) => setSheet(v, key, id)}>
									{#each sheetOptions(templates, key, sheetsOf(v)[key]) as o (o.id)}
										<option value={o.id}>{o.label}</option>
									{/each}
								</select>
								{#if fieldErrors[`${key}_format_version_id`]}<p class="error">{fieldErrors[`${key}_format_version_id`]}</p>{/if}
							</div>
						{/each}
						<button type="submit">{t('formSetSaveSheets')}</button>
					</form>
				{:else}
					<table>
						<tbody>
							{#each sheetKeys as key (key)}
								<tr>
									<th scope="row">{t(sheetLabels[key])}</th>
									<td>{sheetText(v[key])}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			{/if}
		</section>
	{/each}

	<section>
		<h2>{t('formSetCustomersTitle')}</h2>
		<p class="hint">{t('formSetCustomersIntro')}</p>
		<table>
			<thead>
				<tr>
					<th>{t('formSetCustomerColumn')}</th>
					<th>{t('formSetColumn')}</th>
				</tr>
			</thead>
			<tbody>
				{#each list.customer_defaults as c (c.customer_id)}
					<tr>
						<td>{c.customer_code} · {c.customer_name}</td>
						<td>
							{#if canEdit}
								<select
									aria-label={t('formSetCustomerChoose', { code: c.customer_code })}
									value={c.form_set_id ?? ''}
									onchange={(e) => chooseCustomerSet(c.customer_id, c.customer_code, e.currentTarget.value)}
								>
									<option value="">{t('formSetUseDefault', { name: formSetNameOf(list.items, undefined) })}</option>
									{#each visibleItems(list.items, false, c.form_set_id ?? '') as fs (fs.id)}
										<option value={fs.id}>{fs.name}</option>
									{/each}
								</select>
							{:else}
								{formSetNameOf(list.items, c.form_set_id)}
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>

	{#if canEdit}
		<section>
			<h2>{t('formSetCreateTitle')}</h2>
			<form class="sheets" onsubmit={create} novalidate>
				<div class="field">
					<label for="new-set-name">{t('formSetName')}</label>
					<input id="new-set-name" bind:value={draft.name} autocomplete="off" maxlength="200" />
					{#if createErrors.name}<p class="error">{createErrors.name}</p>{/if}
				</div>
				{#each sheetKeys as key (key)}
					<div class="field">
						<label for="new-set-{key}">{t(sheetLabels[key])}</label>
						<select id="new-set-{key}" bind:value={draft[key]}>
							<option value="">{t('formSetSheetRequired')}</option>
							{#each sheetOptions(templates, key) as o (o.id)}
								<option value={o.id}>{o.label}</option>
							{/each}
						</select>
						{#if createErrors[`${key}_format_version_id`]}<p class="error">{createErrors[`${key}_format_version_id`]}</p>{/if}
					</div>
				{/each}
				<button type="submit">{t('formSetCreate')}</button>
			</form>
		</section>
	{/if}
{/if}

<style>
	h2 .badge {
		margin-left: 8px;
		vertical-align: middle;
	}

	.sheets {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	.sheets > button {
		margin-top: var(--label-space);
	}

	.sheets select {
		max-width: 22rem;
	}

	th[scope='row'] {
		background: var(--surface-2);
	}
</style>
