<script lang="ts">
	import { onMount } from 'svelte';
	import { createCustomer, listCustomers } from '#lib/api/client.ts';
	import type { Customer } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';

	// Daftar customer dari server. null berarti belum selesai dimuat.
	let customers = $state<Customer[] | null>(null);
	let loadError = $state('');

	// Isi formulir. bind:value di bawah menghubungkan variabel ini dengan input.
	let code = $state('');
	let name = $state('');
	let saving = $state(false);
	// Pesan per field dari server, misalnya { code: 'Use two letters A to Z.' }.
	let fieldErrors = $state<Record<string, string>>({});
	let formError = $state('');
	let savedMessage = $state('');

	async function load() {
		const result = await listCustomers();
		if (result.ok) {
			customers = result.data.items;
			loadError = '';
		} else {
			loadError = failureText(result.error);
		}
	}

	async function save(event: SubmitEvent) {
		// Tanpa ini browser mengirim formulir dengan cara lama dan memuat ulang halaman.
		event.preventDefault();
		saving = true;
		fieldErrors = {};
		formError = '';
		savedMessage = '';

		// Input dikirim apa adanya; server yang merapikan dan memeriksa (R-API-01).
		const result = await createCustomer({ code, name });
		saving = false;

		if (result.ok) {
			savedMessage = t('customerSaved', { code: result.data.code });
			code = '';
			name = '';
			await load();
			return;
		}
		const details = result.error.body?.details ?? [];
		if (details.length > 0) {
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
		} else {
			formError = failureText(result.error);
		}
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('customersTitle')}</h1>

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

<section>
	<h2>{t('customersListTitle')}</h2>
	{#if loadError}
		<p class="error" role="alert">{loadError}</p>
	{:else if customers === null}
		<p>{t('customersLoading')}</p>
	{:else if customers.length === 0}
		<p>{t('customersEmpty')}</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>{t('customerCode')}</th>
					<th>{t('customerName')}</th>
					<th>{t('customerCreated')}</th>
				</tr>
			</thead>
			<tbody>
				<!-- (customer.id) adalah kunci baris, supaya Svelte tahu baris mana yang berubah. -->
				{#each customers as customer (customer.id)}
					<tr>
						<td>{customer.code}</td>
						<td>{customer.name}</td>
						<td>{formatDateTime(customer.created_at)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</section>

<style>
	section {
		margin-bottom: 2rem;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		margin-bottom: 1rem;
		max-width: 28rem;
	}

	.error {
		color: #b00020;
		margin: 0;
	}

	table {
		border-collapse: collapse;
	}

	th,
	td {
		border: 1px solid #ccc;
		padding: 0.25rem 0.75rem;
		text-align: left;
	}
</style>
