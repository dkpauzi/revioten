<script lang="ts">
	// Halaman model dan part number (M1.4, UC-03): pilih customer, lalu lihat
	// group model, model, dan part number-nya. Semua akun bisa melihat; mengubah
	// hanya untuk pemegang E-11 (E-10).
	import { onMount } from 'svelte';
	import { listCustomers, listModelGroups, listModels, listPartNumbers } from '#lib/api/client.ts';
	import type { Customer, Model, ModelGroup, PartNumber } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import ModelGroupsSection from '#lib/master/ModelGroupsSection.svelte';
	import ModelsSection from '#lib/master/ModelsSection.svelte';
	import PartNumbersSection from '#lib/master/PartNumbersSection.svelte';
	import { can } from '#lib/session.svelte.ts';

	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let groups = $state<ModelGroup[]>([]);
	let models = $state<Model[]>([]);
	let parts = $state<PartNumber[]>([]);
	// loaded: master customer yang dipilih sudah dimuat.
	let loaded = $state(false);
	let loadError = $state('');

	let selected = $derived(customers.find((c) => c.id === customerId));
	// Customer nonaktif tidak mendapat master baru (E-09), jadi formulirnya tidak ditawarkan.
	let canEdit = $derived(can('E-11') && (selected?.active ?? false));

	async function loadCustomers() {
		const result = await listCustomers();
		if (result.ok) {
			customers = result.data.items;
		} else {
			loadError = failureText(result.error);
		}
	}

	// choose memilih customer lalu memuat master-nya.
	function choose(id: string) {
		customerId = id;
		loaded = false;
		loadMasters();
	}

	// loadMasters memuat ketiga daftar milik customer yang dipilih, bersamaan.
	async function loadMasters() {
		if (customerId === '') {
			return;
		}
		const [groupResult, modelResult, partResult] = await Promise.all([
			listModelGroups(customerId),
			listModels(customerId),
			listPartNumbers(customerId)
		]);
		if (!groupResult.ok) {
			loadError = failureText(groupResult.error);
			return;
		}
		if (!modelResult.ok) {
			loadError = failureText(modelResult.error);
			return;
		}
		if (!partResult.ok) {
			loadError = failureText(partResult.error);
			return;
		}
		groups = groupResult.data.items;
		models = modelResult.data.items;
		parts = partResult.data.items;
		loadError = '';
		loaded = true;
	}

	onMount(() => {
		loadCustomers();
	});
</script>

<h1>{t('modelsTitle')}</h1>

<div class="field">
	<label for="models-customer">{t('modelsCustomer')}</label>
	<select
		id="models-customer"
		value={customerId}
		onchange={(event) => choose(event.currentTarget.value)}
	>
		<option value="">{t('modelsCustomerChoose')}</option>
		{#each customers as customer (customer.id)}
			<option value={customer.id}>
				{customer.code} · {customer.name}{customer.active ? '' : ` (${t('customerStatusInactive')})`}
			</option>
		{/each}
	</select>
</div>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if customerId === ''}
	<p>{t('modelsChooseCustomer')}</p>
{:else if !loaded}
	<p>{t('modelsLoading')}</p>
{:else}
	<ModelGroupsSection {customerId} {groups} {canEdit} onsaved={loadMasters} />
	<ModelsSection {customerId} {models} {groups} {canEdit} onsaved={loadMasters} />
	<PartNumbersSection {parts} {models} {canEdit} onsaved={loadMasters} />
{/if}

<style>
	.field {
		margin-bottom: 1.5rem;
		max-width: 28rem;
	}
</style>
