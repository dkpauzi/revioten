<script lang="ts">
	// Nama perusahaan dan logo cetakan (bagian C, E-144). Semua akun melihat;
	// hanya admin (A-3) yang mengubah. Server tetap memeriksa jenis berkas,
	// ukuran, dan haknya. Sejak D-223 admin wajib menjawab apakah dokumen
	// direvisi karena logo baru; bila ya, PPAP Engineer diminta mengajukan
	// perubahan logo per customer (halaman Changes).
	import { onMount } from 'svelte';
	import { getCompany, resetCompanyLogo, setCompanyLogo, setCompanyName, type ApiResult } from '#lib/api/client.ts';
	import type { Company } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let company = $state<Company | null>(null);
	let name = $state('');
	let file = $state<File | null>(null);
	let message = $state('');
	let failure = $state('');
	// Jawaban "revisi dokumen?": kosong sampai admin memilih (D-223).
	let revise = $state<'' | 'yes' | 'no'>('');
	let canEdit = $derived(can('A-3'));
	// Alamat logo berubah setiap kali logo diganti, supaya browser tidak memakai salinan lama.
	let logoUrl = $derived('/api/v1/company/logo?v=' + (company?.logo_sha256 ?? 'default'));

	function done(result: ApiResult<Company>, text: string): boolean {
		if (!result.ok) {
			failure = failureText(result.error);
			return false;
		}
		company = result.data;
		name = result.data.name;
		message = text;
		return true;
	}

	async function load() {
		const result = await getCompany();
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		company = result.data;
		name = result.data.name;
	}

	async function saveName(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		done(await setCompanyName(name), t('companyNameSaved'));
	}

	async function upload(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		if (!file) {
			failure = t('companyLogoChoose');
			return;
		}
		if (!revise) {
			failure = t('companyLogoReviseChoose');
			return;
		}
		if (done(await setCompanyLogo(file, revise === 'yes'), revisedText(t('companyLogoUploaded')))) {
			file = null;
			revise = '';
		}
	}

	async function reset() {
		message = '';
		failure = '';
		if (!revise) {
			failure = t('companyLogoReviseChoose');
			return;
		}
		if (done(await resetCompanyLogo(revise === 'yes'), revisedText(t('companyLogoResetDone')))) {
			revise = '';
		}
	}

	// Pesan hasil menyebut langkah berikutnya bila dokumen direvisi.
	function revisedText(text: string): string {
		return revise === 'yes' ? text + ' ' + t('companyLogoReviseNext') : text;
	}

	onMount(() => {
		load();
	});
</script>

<section aria-labelledby="company-title">
	<h2 id="company-title">{t('companyTitle')}</h2>
	<p class="hint">{t('companyHint')}</p>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	{#if company}
		<form class="row" onsubmit={saveName} novalidate>
			<div class="field">
				<label for="company-name">{t('companyName')}</label>
				<input id="company-name" bind:value={name} maxlength="200" size="40" autocomplete="off" disabled={!canEdit} />
			</div>
			{#if canEdit}<button type="submit">{t('companyNameSave')}</button>{/if}
		</form>
		<p class="logo"><img src={logoUrl} alt={t('companyLogoAlt')} /></p>
		<p class="hint">
			{company.custom_logo ? t('companyLogoCustom', { hash: (company.logo_sha256 ?? '').slice(0, 16) }) : t('companyLogoDefault')}
		</p>
		{#if canEdit}
			<form class="row" onsubmit={upload} novalidate>
				<div class="field">
					<label for="company-logo">{t('companyLogoFile')}</label>
					<input id="company-logo" type="file" accept="image/png" onchange={(e) => (file = e.currentTarget.files?.[0] ?? null)} />
				</div>
				<fieldset class="revise">
					<legend>{t('companyLogoRevise')}</legend>
					<label class="check"><input type="radio" name="logo-revise" value="yes" bind:group={revise} /> {t('companyLogoReviseYes')}</label>
					<label class="check"><input type="radio" name="logo-revise" value="no" bind:group={revise} /> {t('companyLogoReviseNo')}</label>
				</fieldset>
				<button type="submit">{t('companyLogoUpload')}</button>
				{#if company.custom_logo}<button type="button" onclick={reset}>{t('companyLogoReset')}</button>{/if}
			</form>
		{/if}
	{/if}
</section>

<style>
	.logo img {
		max-height: 18mm;
		max-width: 100%;
		border: 1px solid var(--border);
		padding: 4px;
		background: #ffffff;
	}
</style>
