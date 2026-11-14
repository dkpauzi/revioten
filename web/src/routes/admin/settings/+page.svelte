<script lang="ts">
	// Halaman pengaturan sistem (M1.8, UC-07): saklar tanggal tanda tangan per
	// slot, yang saat ini mati (D-52, R-APR-14), dan aturan persetujuan form
	// set dan tabel AP (D-139, bawaan semua penyetuju).
	// Mengubahnya butuh hak A-3; server tetap memeriksa setiap operasi.
	import { onMount } from 'svelte';
	import { getSystemSettings, updateSystemSettings } from '#lib/api/client.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let loaded = $state(false);
	let signDates = $state(false);
	// Aturan persetujuan form set dan tabel AP (D-139).
	let approvalMode = $state<'all' | 'any'>('all');
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let canEdit = $derived(can('A-3'));

	// Saklar QR di cetakan (D-62); bawaannya mati.
	let qrEnabled = $state(false);

	async function load() {
		const result = await getSystemSettings();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		signDates = result.data.sign_dates_enabled;
		approvalMode = result.data.form_approval_mode;
		qrEnabled = result.data.qr_enabled;
		loaded = true;
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		const result = await updateSystemSettings({ sign_dates_enabled: signDates, form_approval_mode: approvalMode, qr_enabled: qrEnabled });
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		signDates = result.data.sign_dates_enabled;
		approvalMode = result.data.form_approval_mode;
		qrEnabled = result.data.qr_enabled;
		message = t('settingsSaved');
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('settingsTitle')}</h1>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if !loaded}
	<p>{t('settingsLoading')}</p>
{:else}
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<form onsubmit={save} novalidate>
		<label>
			<input type="checkbox" bind:checked={signDates} disabled={!canEdit} aria-describedby="sign-dates-hint" />
			{t('settingsSignDates')}
		</label>
		<p id="sign-dates-hint"><small>{t('settingsSignDatesHint')}</small></p>
		<label>
			<input type="checkbox" bind:checked={qrEnabled} disabled={!canEdit} aria-describedby="qr-hint" />
			{t('settingsQr')}
		</label>
		<p id="qr-hint"><small>{t('settingsQrHint')}</small></p>
		<fieldset>
			<legend>{t('settingsFormApproval')}</legend>
			<label><input type="radio" bind:group={approvalMode} value="all" disabled={!canEdit} /> {t('settingsFormApprovalAll')}</label>
			<label><input type="radio" bind:group={approvalMode} value="any" disabled={!canEdit} /> {t('settingsFormApprovalAny')}</label>
			<p><small>{t('settingsFormApprovalHint')}</small></p>
		</fieldset>
		{#if canEdit}
			<button type="submit">{t('settingsSave')}</button>
		{/if}
	</form>
{/if}
