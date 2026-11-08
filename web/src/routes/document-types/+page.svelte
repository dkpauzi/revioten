<script lang="ts">
	// Halaman jenis dokumen dan kode nomor (M1.6, UC-05, E-12): pengaturan nomor
	// (kode pabrik, pola nomor, dan digit revisi, M2.1), jenis dokumen dengan
	// awalannya, dan kode khusus. Semua akun bisa melihat; mengubah hanya untuk
	// pemegang A-2. Server tetap memeriksa setiap operasi.
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
	let pattern = $state('');
	let revisionDigits = $state(2);
	let loadError = $state('');
	let canEdit = $derived(can('A-2'));

	// Formulir pengaturan nomor.
	let newFactoryCode = $state('');
	let newPattern = $state('');
	// Kotak angka memberi null bila dikosongkan; null berarti tidak diubah.
	let newRevisionDigits = $state<number | null>(2);
	let settingsErrors = $state<Record<string, string>>({});
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
		pattern = settings.data.pattern;
		newPattern = settings.data.pattern;
		revisionDigits = settings.data.revision_digits;
		newRevisionDigits = settings.data.revision_digits;
		types = typeResult.data.items;
		codes = codeResult.data.items;
		loadError = '';
	}

	// saveFactoryCode menyimpan kode pabrik, pola nomor, dan digit revisi.
	// Perubahan hanya berlaku untuk nomor baru (R-NUM-02).
	async function saveFactoryCode(event: SubmitEvent) {
		event.preventDefault();
		settingsErrors = {};
		factoryMessage = '';
		factoryFailure = '';
		const result = await updateNumberSettings({
			factory_code: newFactoryCode,
			pattern: newPattern,
			revision_digits: newRevisionDigits ?? undefined
		});
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			settingsErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
			if (details.length === 0) {
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
					{#if settingsErrors.factory_code}<p class="error">{settingsErrors.factory_code}</p>{/if}
				</div>
				<div class="field">
					<label for="number-pattern">{t('numberPattern')}</label>
					<input id="number-pattern" bind:value={newPattern} autocomplete="off" size="30" />
					{#if settingsErrors.pattern}<p class="error">{settingsErrors.pattern}</p>{/if}
				</div>
				<div class="field">
					<label for="revision-digits">{t('numberRevisionDigits')}</label>
					<input id="revision-digits" type="number" min="1" max="3" bind:value={newRevisionDigits} />
					{#if settingsErrors.revision_digits}<p class="error">{settingsErrors.revision_digits}</p>{/if}
				</div>
				<button type="submit">{t('factoryCodeSave')}</button>
			</form>
			<p class="hint">{t('numberPatternHint')}</p>
		{:else}
			<p>{t('factoryCode')}: <strong>{factoryCode}</strong></p>
			<p>{t('numberPattern')}: <strong>{pattern}</strong></p>
			<p>{t('numberRevisionDigits')}: <strong>{revisionDigits}</strong></p>
		{/if}
		<p class="hint">{t('numberSequenceFormat')}</p>
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
		align-items: flex-start;
	}

	/* Field dirapikan ke atas; tombol diturunkan setinggi label supaya sejajar
	   dengan kotak isian, juga bila ada teks bantuan atau pesan masalah. */
	.add > button {
		margin-top: var(--label-space);
	}
</style>
