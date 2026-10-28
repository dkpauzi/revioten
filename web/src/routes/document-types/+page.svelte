<script lang="ts">
	// Halaman jenis dokumen dan kode nomor (M1.6, UC-05, E-12): kode pabrik,
	// jenis dokumen dengan awalannya, dan kode khusus. Semua akun bisa melihat;
	// mengubah hanya untuk pemegang A-2. Server tetap memeriksa setiap operasi.
	import { onMount } from 'svelte';
	import {
		getNumberSettings,
		listDocumentTypes,
		listSpecialCodes,
		updateNumberSettings
	} from '#lib/api/client.ts';
	import type { DocumentType, SpecialCode } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import DocumentTypesSection from '#lib/master/DocumentTypesSection.svelte';
	import SpecialCodesSection from '#lib/master/SpecialCodesSection.svelte';
	import { can } from '#lib/session.svelte.ts';

	// Daftar dari server. null berarti belum selesai dimuat.
	let types = $state<DocumentType[] | null>(null);
	let codes = $state<SpecialCode[]>([]);
	let factoryCode = $state('');
	let loadError = $state('');
	let canEdit = $derived(can('A-2'));

	// Formulir kode pabrik.
	let newFactoryCode = $state('');
	let factoryError = $state('');
	let factoryMessage = $state('');
	let factoryFailure = $state('');

	async function load() {
		// Tiga permintaan dijalankan bersamaan, lalu ditunggu semuanya.
		const [settings, typeResult, codeResult] = await Promise.all([
			getNumberSettings(),
			listDocumentTypes(),
			listSpecialCodes()
		]);
		if (!settings.ok) {
			loadError = failureText(settings.error);
			return;
		}
		if (!typeResult.ok) {
			loadError = failureText(typeResult.error);
			return;
		}
		if (!codeResult.ok) {
			loadError = failureText(codeResult.error);
			return;
		}
		factoryCode = settings.data.factory_code;
		newFactoryCode = settings.data.factory_code;
		types = typeResult.data.items;
		codes = codeResult.data.items;
		loadError = '';
	}

	// saveFactoryCode mengganti kode pabrik. Awalan semua jenis dokumen ikut
	// berubah untuk nomor baru (R-NUM-02).
	async function saveFactoryCode(event: SubmitEvent) {
		event.preventDefault();
		factoryError = '';
		factoryMessage = '';
		factoryFailure = '';
		const result = await updateNumberSettings({ factory_code: newFactoryCode });
		if (!result.ok) {
			const detail = result.error.body?.details?.[0];
			if (detail) {
				factoryError = problemText(detail);
			} else {
				factoryFailure = failureText(result.error);
			}
			return;
		}
		factoryMessage = t('factoryCodeSaved', { code: result.data.factory_code });
		await load();
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('documentTypesTitle')}</h1>
<p>{t('documentTypesIntro')}</p>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if types === null}
	<p>{t('documentTypesLoading')}</p>
{:else}
	<section>
		<h2>{t('factoryCodeTitle')}</h2>
		{#if factoryMessage}<p role="status">{factoryMessage}</p>{/if}
		{#if factoryFailure}<p class="error" role="alert">{factoryFailure}</p>{/if}
		{#if canEdit}
			<form class="add" onsubmit={saveFactoryCode} novalidate>
				<div class="field">
					<label for="factory-code">{t('factoryCode')}</label>
					<input id="factory-code" bind:value={newFactoryCode} autocomplete="off" size="2" />
					{#if factoryError}<p class="error">{factoryError}</p>{/if}
				</div>
				<button type="submit">{t('factoryCodeSave')}</button>
			</form>
		{:else}
			<p>{t('factoryCode')}: <strong>{factoryCode}</strong></p>
		{/if}
	</section>

	<DocumentTypesSection {types} {canEdit} onsaved={load} />
	<SpecialCodesSection {codes} {canEdit} onsaved={load} />
{/if}

<style>
	section {
		margin-bottom: 2rem;
	}

	.add {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-end;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.error {
		color: #b00020;
		margin: 0;
	}
</style>
