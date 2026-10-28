<script lang="ts">
	// Halaman jenis perubahan, sumber perubahan, dan kategori 4M (M1.6, UC-05).
	// Semua akun bisa melihat; mengubah hanya untuk pemegang A-2. Server tetap
	// memeriksa setiap operasi.
	import { onMount } from 'svelte';
	import {
		listChangeSources,
		listChangeTypes,
		listFourMCategories,
		updateChangeType,
		updateFourMCategory
	} from '#lib/api/client.ts';
	import type { ChangeSource, ChangeType, FourMCategory } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import ChangeSourcesSection from '#lib/master/ChangeSourcesSection.svelte';
	import NameListSection from '#lib/master/NameListSection.svelte';
	import { can } from '#lib/session.svelte.ts';

	// Daftar dari server. null berarti belum selesai dimuat.
	let types = $state<ChangeType[] | null>(null);
	let sources = $state<ChangeSource[]>([]);
	let categories = $state<FourMCategory[]>([]);
	let loadError = $state('');
	let canEdit = $derived(can('A-2'));

	async function load() {
		const [typeResult, sourceResult, categoryResult] = await Promise.all([
			listChangeTypes(),
			listChangeSources(),
			listFourMCategories()
		]);
		if (!typeResult.ok) {
			loadError = failureText(typeResult.error);
			return;
		}
		if (!sourceResult.ok) {
			loadError = failureText(sourceResult.error);
			return;
		}
		if (!categoryResult.ok) {
			loadError = failureText(categoryResult.error);
			return;
		}
		types = typeResult.data.items;
		sources = sourceResult.data.items;
		categories = categoryResult.data.items;
		loadError = '';
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('changeTypesTitle')}</h1>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if types === null}
	<p>{t('changeTypesLoading')}</p>
{:else}
	<NameListSection
		title={t('changeTypesListTitle')}
		items={types}
		{canEdit}
		rename={(id, name) => updateChangeType(id, name)}
		onsaved={load}
	/>
	<ChangeSourcesSection {sources} {canEdit} onsaved={load} />
	<NameListSection
		title={t('fourMTitle')}
		items={categories}
		{canEdit}
		rename={(id, name) => updateFourMCategory(id, name)}
		onsaved={load}
	/>
{/if}

<style>
	.error {
		color: #b00020;
		margin: 0;
	}
</style>
