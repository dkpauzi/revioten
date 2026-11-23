<script lang="ts">
	// Halaman Customers (M0.5, diperluas di M1.4): Project Group, customer
	// beserta alokasinya (D-54), status aktif (E-09), dan riwayat kode (E-11).
	// Semua akun bisa melihat; mengubah hanya untuk pemegang E-11 (E-10).
	import { onMount } from 'svelte';
	import {
		changeCustomerCode,
		createCustomer,
		listCustomers,
		listProjectGroups,
		setCustomerSignatureMode,
		setCustomerSubmissionRequired,
		updateCustomer
	} from '#lib/api/client.ts';
	import type { ApiFailure } from '#lib/api/client.ts';
	import type { Customer, ProjectGroup } from '#lib/api/gen/index.ts';
	import { failureText, problemText, type ProblemScope } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import ProjectGroupsSection from '#lib/master/ProjectGroupsSection.svelte';
	import { can } from '#lib/session.svelte.ts';

	// Daftar dari server. null berarti belum selesai dimuat.
	let customers = $state<Customer[] | null>(null);
	let groups = $state<ProjectGroup[]>([]);
	let loadError = $state('');
	// Menu dan tombol hanya untuk pemegang E-11; server tetap memeriksa setiap operasi.
	let canEdit = $derived(can('E-11'));

	// Isi formulir tambah customer. bind:value menghubungkan variabel ini dengan input.
	let code = $state('');
	let name = $state('');
	let projectGroupId = $state('');
	let saving = $state(false);
	// Pesan per field dari server, misalnya { code: 'Use two letters A to Z.' }.
	let fieldErrors = $state<Record<string, string>>({});
	let formError = $state('');
	let savedMessage = $state('');

	// Baris yang sedang diubah: 'edit' untuk nama, Project Group, dan status;
	// 'code' untuk ganti kode.
	let editing = $state<{ id: string; mode: 'edit' | 'code' } | null>(null);
	let editName = $state('');
	let editGroupId = $state('');
	let editActive = $state(true);
	let newCode = $state('');
	let rowErrors = $state<Record<string, string>>({});
	let rowMessage = $state('');
	let rowFailure = $state('');

	// Bawaan "perlu dikirim ke customer" (D-207).
	async function setSubmission(customer: Customer, required: boolean) {
		rowMessage = '';
		rowFailure = '';
		const result = await setCustomerSubmissionRequired(customer.id, required);
		if (!result.ok) {
			rowFailure = failureText(result.error);
			return;
		}
		customers = (customers ?? []).map((c) => (c.id === customer.id ? result.data : c));
		rowMessage = t('customerSubmissionSaved', { code: customer.code, value: t(required ? 'submissionRequired' : 'submissionNotRequired') });
	}

	// Cara tanda tangan customer (D-198): digital boleh, atau wajib basah.
	async function setMode(customer: Customer, mode: 'digital' | 'wet_only') {
		rowMessage = '';
		rowFailure = '';
		const result = await setCustomerSignatureMode(customer.id, mode);
		if (!result.ok) {
			rowFailure = failureText(result.error);
			return;
		}
		customers = (customers ?? []).map((c) => (c.id === customer.id ? result.data : c));
		rowMessage = t('customerSignatureModeSaved', {
			code: customer.code,
			mode: t(mode === 'digital' ? 'signatureModeDigital' : 'signatureModeWetOnly')
		});
	}

	async function load() {
		// Dua permintaan dijalankan bersamaan, lalu ditunggu keduanya.
		const [groupResult, customerResult] = await Promise.all([listProjectGroups(), listCustomers()]);
		if (!groupResult.ok) {
			loadError = failureText(groupResult.error);
			return;
		}
		if (!customerResult.ok) {
			loadError = failureText(customerResult.error);
			return;
		}
		groups = groupResult.data.items;
		customers = customerResult.data.items;
		loadError = '';
	}

	// errorsOf mengubah masalah per field dari server menjadi teks layar.
	function errorsOf(error: ApiFailure, scope?: ProblemScope): Record<string, string> {
		const details = error.body?.details ?? [];
		return Object.fromEntries(details.map((d) => [d.field, problemText(d, scope)]));
	}

	async function save(event: SubmitEvent) {
		// Tanpa ini browser mengirim formulir dengan cara lama dan memuat ulang halaman.
		event.preventDefault();
		saving = true;
		fieldErrors = {};
		formError = '';
		savedMessage = '';

		// Input dikirim apa adanya; server yang merapikan dan memeriksa (R-API-01).
		const result = await createCustomer({ code, name, project_group_id: projectGroupId });
		saving = false;

		if (result.ok) {
			savedMessage = t('customerSaved', { code: result.data.code });
			code = '';
			name = '';
			await load();
			return;
		}
		fieldErrors = errorsOf(result.error);
		if (Object.keys(fieldErrors).length === 0) {
			formError = failureText(result.error);
		}
	}

	function startEdit(customer: Customer) {
		editing = { id: customer.id, mode: 'edit' };
		editName = customer.name;
		editGroupId = customer.project_group?.id ?? '';
		editActive = customer.active;
		rowErrors = {};
	}

	function startCodeChange(customer: Customer) {
		editing = { id: customer.id, mode: 'code' };
		newCode = '';
		rowErrors = {};
	}

	// saveRow menyimpan baris yang sedang diubah: isian customer atau kode baru.
	async function saveRow(event: SubmitEvent) {
		event.preventDefault();
		const current = customers?.find((c) => c.id === editing?.id);
		if (!editing || !current) {
			return;
		}
		rowErrors = {};
		rowMessage = '';
		rowFailure = '';
		const changingCode = editing.mode === 'code';
		const result = changingCode
			? await changeCustomerCode(current.id, newCode)
			: await updateCustomer(current.id, {
					name: editName,
					project_group_id: editGroupId,
					active: editActive
				});
		if (!result.ok) {
			rowErrors = errorsOf(result.error, changingCode ? 'customerCode' : undefined);
			if (Object.keys(rowErrors).length === 0) {
				rowFailure = failureText(result.error);
			}
			return;
		}
		rowMessage = changingCode
			? t('customerCodeChanged', { before: current.code, after: result.data.code })
			: t('customerUpdated', { code: result.data.code });
		editing = null;
		await load();
	}

	// previousCodes menulis kode lama customer, yaitu semua kode kecuali yang berlaku.
	function previousCodes(customer: Customer): string {
		return customer.codes
			.slice(0, -1)
			.map((p) => p.code)
			.join(', ');
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('customersTitle')}</h1>
{#if can('E-11')}<p><a href="/customers/import">{t('customersImportLink')}</a></p>{/if}

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if customers === null}
	<p>{t('customersLoading')}</p>
{:else}
	<ProjectGroupsSection {groups} {canEdit} onsaved={load} />

	<!-- Formulir hanya untuk pemegang hak E-11 (E-10); server tetap memeriksa setiap operasi. -->
	{#if canEdit}
		<section>
			<h2>{t('customerAddTitle')}</h2>
			<!-- novalidate: pemeriksaan input dilakukan server, supaya aturannya hanya satu. -->
			<form onsubmit={save} novalidate>
				<div class="field">
					<label for="customer-code">{t('customerCode')}</label>
					<input
						id="customer-code"
						name="code"
						bind:value={code}
						autocomplete="off"
						aria-describedby="customer-code-hint"
						aria-invalid={fieldErrors.code ? 'true' : undefined}
					/>
					<small id="customer-code-hint">{t('customerCodeHint')}</small>
					{#if fieldErrors.code}
						<p class="error">{fieldErrors.code}</p>
					{/if}
				</div>

				<div class="field">
					<label for="customer-name">{t('customerName')}</label>
					<input
						id="customer-name"
						name="name"
						bind:value={name}
						autocomplete="off"
						aria-invalid={fieldErrors.name ? 'true' : undefined}
					/>
					{#if fieldErrors.name}
						<p class="error">{fieldErrors.name}</p>
					{/if}
				</div>

				<div class="field">
					<label for="customer-group">{t('customerProjectGroup')}</label>
					<select id="customer-group" bind:value={projectGroupId}>
						<option value="">{t('customerProjectGroupChoose')}</option>
						{#each groups as group (group.id)}
							<option value={group.id}>{group.code} · {group.name}</option>
						{/each}
					</select>
					{#if fieldErrors.project_group_id}
						<p class="error">{fieldErrors.project_group_id}</p>
					{/if}
				</div>

				<button type="submit" disabled={saving}>
					{saving ? t('customerSaving') : t('customerSave')}
				</button>

				{#if formError}
					<p class="error" role="alert">{formError}</p>
				{/if}
				{#if savedMessage}
					<p role="status">{savedMessage}</p>
				{/if}
			</form>
		</section>
	{/if}

	<section>
		<h2>{t('customersListTitle')}</h2>
		{#if rowMessage}
			<p role="status">{rowMessage}</p>
		{/if}
		{#if rowFailure}
			<p class="error" role="alert">{rowFailure}</p>
		{/if}
		{#if customers.length === 0}
			<p>{t('customersEmpty')}</p>
		{:else}
			<!-- Satu formulir untuk baris yang sedang diubah, supaya tombol Enter menyimpan. -->
			<form onsubmit={saveRow} novalidate>
				<table>
					<thead>
						<tr>
							<th>{t('customerCode')}</th>
							<th>{t('customerName')}</th>
							<th>{t('customerProjectGroup')}</th>
							<th>{t('customerStatus')}</th>
							<th>{t('customerSignatureMode')}</th>
							<th>{t('customerSubmission')}</th>
							<th>{t('customerPreviousCodes')}</th>
							<th>{t('customerCreated')}</th>
							{#if canEdit}
								<th>{t('customerActions')}</th>
							{/if}
						</tr>
					</thead>
					<tbody>
						<!-- (customer.id) adalah kunci baris, supaya Svelte tahu baris mana yang berubah. -->
						{#each customers as customer (customer.id)}
							<tr>
								{#if editing?.id === customer.id && editing.mode === 'code'}
									<td>
										<input
											aria-label={t('customerNewCode', { code: customer.code })}
											bind:value={newCode}
											autocomplete="off"
										/>
										<small>{t('customerChangeCodeHint')}</small>
										{#if rowErrors.code}<p class="error">{rowErrors.code}</p>{/if}
									</td>
								{:else}
									<td>{customer.code}</td>
								{/if}
								{#if editing?.id === customer.id && editing.mode === 'edit'}
									<td>
										<input aria-label={t('customerName')} bind:value={editName} autocomplete="off" />
										{#if rowErrors.name}<p class="error">{rowErrors.name}</p>{/if}
									</td>
									<td>
										<select aria-label={t('customerProjectGroup')} bind:value={editGroupId}>
											<option value="">{t('customerProjectGroupChoose')}</option>
											{#each groups as group (group.id)}
												<option value={group.id}>{group.code} · {group.name}</option>
											{/each}
										</select>
										{#if rowErrors.project_group_id}
											<p class="error">{rowErrors.project_group_id}</p>
										{/if}
									</td>
									<td>
										<label>
											<input type="checkbox" bind:checked={editActive} />
											{t('customerActive')}
										</label>
									</td>
								{:else}
									<td>{customer.name}</td>
									<td>{customer.project_group?.code ?? t('customerProjectGroupNone')}</td>
									<td>{customer.active ? t('customerStatusActive') : t('customerStatusInactive')}</td>
								{/if}
								<td>
									{#if canEdit}
										<select
											aria-label={t('customerSignatureModeFor', { code: customer.code })}
											value={customer.signature_mode ?? 'digital'}
											onchange={(event) => setMode(customer, event.currentTarget.value as 'digital' | 'wet_only')}
										>
											<option value="digital">{t('signatureModeDigital')}</option>
											<option value="wet_only">{t('signatureModeWetOnly')}</option>
										</select>
									{:else}
										{t(customer.signature_mode === 'wet_only' ? 'signatureModeWetOnly' : 'signatureModeDigital')}
									{/if}
								</td>
								<td>
									{#if canEdit}
										<select
											aria-label={t('customerSubmissionFor', { code: customer.code })}
											value={customer.submission_required === false ? 'no' : 'yes'}
											onchange={(event) => setSubmission(customer, event.currentTarget.value === 'yes')}
										>
											<option value="yes">{t('submissionRequired')}</option>
											<option value="no">{t('submissionNotRequired')}</option>
										</select>
									{:else}
										{t(customer.submission_required === false ? 'submissionNotRequired' : 'submissionRequired')}
									{/if}
								</td>
								<td>{previousCodes(customer)}</td>
								<td>{formatDateTime(customer.created_at)}</td>
								{#if canEdit}
									<td>
										{#if editing?.id === customer.id}
											<button type="submit">{t('masterSave')}</button>
											<button type="button" onclick={() => (editing = null)}>{t('masterCancel')}</button>
										{:else}
											<button type="button" onclick={() => startEdit(customer)}>{t('customerEdit')}</button>
											<button type="button" onclick={() => startCodeChange(customer)}>
												{t('customerChangeCode')}
											</button>
										{/if}
									</td>
								{/if}
							</tr>
						{/each}
					</tbody>
				</table>
			</form>
		{/if}
	</section>
{/if}

<style>
	section {
		margin-bottom: 2rem;
	}

	.field {
		margin-bottom: 1rem;
		max-width: 28rem;
	}
</style>
