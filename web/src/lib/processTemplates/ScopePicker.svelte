<script lang="ts">
	// Pemilih cakupan poin atau varian (M3.3, D-129): umum, customer, group
	// model, atau model. Untuk group model dan model, customer dipilih dulu,
	// lalu group atau modelnya dimuat dari server. Server tetap memeriksa
	// bahwa sasarannya ada dan, untuk varian, berada di dalam poin asalnya.
	import { listModelGroups, listModels } from '#lib/api/client.ts';
	import type { Customer, Model, ModelGroup } from '#lib/api/gen/index.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import type { ScopeLevel } from '#lib/processTemplates.ts';

	let {
		id,
		customers,
		levels,
		level = $bindable(),
		targetId = $bindable(),
		errors = {}
	}: {
		id: string; // awalan id elemen, supaya label unik di halaman
		customers: Customer[];
		levels: ScopeLevel[]; // tingkat yang boleh dipilih
		level: ScopeLevel;
		targetId: string;
		errors?: Record<string, string>;
	} = $props();

	const levelKeys: Record<ScopeLevel, MessageKey> = {
		general: 'scopeGeneral',
		customer: 'scopeLevelCustomer',
		model_group: 'scopeLevelModelGroup',
		model: 'scopeLevelModel'
	};

	let customerId = $state('');
	let groups = $state<ModelGroup[]>([]);
	let models = $state<Model[]>([]);
	let loading = $state(false);

	// loadTargets memuat group model atau model milik customer yang dipilih.
	async function loadTargets() {
		groups = [];
		models = [];
		if (customerId === '' || level === 'general' || level === 'customer') {
			return;
		}
		loading = true;
		if (level === 'model_group') {
			const result = await listModelGroups(customerId);
			groups = result.ok ? result.data.items : [];
		} else {
			const result = await listModels(customerId);
			models = result.ok ? result.data.items : [];
		}
		loading = false;
	}

	// Ganti tingkat atau customer: sasaran dipilih ulang. Untuk tingkat
	// customer, sasarannya customer itu sendiri.
	function changed() {
		targetId = level === 'customer' ? customerId : '';
		loadTargets();
	}
</script>

<div class="field">
	<label for="{id}-level">{t('pointScope')}</label>
	<select id="{id}-level" bind:value={level} onchange={changed}>
		{#each levels as l (l)}
			<option value={l}>{t(levelKeys[l])}</option>
		{/each}
	</select>
	{#if errors.scope_level}<p class="error">{errors.scope_level}</p>{/if}
</div>

{#if level !== 'general'}
	<div class="field">
		<label for="{id}-customer">{t('scopeLevelCustomer')}</label>
		<select id="{id}-customer" bind:value={customerId} onchange={changed}>
			<option value="">{t('scopeCustomerChoose')}</option>
			{#each customers as c (c.id)}
				<option value={c.id}>{c.code} · {c.name}</option>
			{/each}
		</select>
		{#if level === 'customer' && errors.scope_target_id}<p class="error">{errors.scope_target_id}</p>{/if}
	</div>
{/if}

{#if level === 'model_group' || level === 'model'}
	<div class="field">
		<label for="{id}-target">{t(levelKeys[level])}</label>
		<select id="{id}-target" bind:value={targetId} disabled={customerId === '' || loading}>
			<option value="">{loading ? t('scopeLoading') : t(level === 'model' ? 'scopeModelChoose' : 'scopeModelGroupChoose')}</option>
			{#if level === 'model'}
				{#each models as m (m.id)}
					<option value={m.id}>{m.name}{m.model_group ? ` · ${m.model_group.name}` : ''}</option>
				{/each}
			{:else}
				{#each groups as g (g.id)}
					<option value={g.id}>{g.name}</option>
				{/each}
			{/if}
		</select>
		{#if errors.scope_target_id}<p class="error">{errors.scope_target_id}</p>{/if}
	</div>
{/if}
