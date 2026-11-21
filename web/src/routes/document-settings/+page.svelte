<script lang="ts">
	// Pengaturan dokumen (D-209): saklar yang mengatur isi dokumen dan cetakan,
	// diubah PPAP Engineer (E-11). Pengaman sistem (mode migrasi, aturan
	// penyetujuan form) tetap di Admin → Settings (A-3).
	// Semua akun bisa melihat; server tetap memeriksa setiap perubahan.
	import { onMount } from 'svelte';
	import { getSystemSettings, updateSystemSettings } from '#lib/api/client.ts';
	import type { SystemSettings } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let loaded = $state(false);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let canEdit = $derived(can('E-11'));

	// Tanggal di setiap slot tanda tangan (D-52); bawaannya mati.
	let signDates = $state(false);
	// QR di cetakan (D-62); bawaannya mati.
	let qrEnabled = $state(false);
	// Nomor form Form Control Submission (D-207); kosong berarti tidak dicetak.
	let formNumber = $state('');

	function show(data: SystemSettings) {
		signDates = data.sign_dates_enabled;
		qrEnabled = data.qr_enabled;
		formNumber = data.submission_form_number ?? '';
	}

	async function load() {
		const result = await getSystemSettings();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		show(result.data);
		loaded = true;
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		// Hanya saklar halaman ini yang dikirim, supaya cukup hak E-11.
		const result = await updateSystemSettings({
			sign_dates_enabled: signDates,
			qr_enabled: qrEnabled,
			submission_form_number: formNumber
		});
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		show(result.data);
		message = t('settingsSaved');
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('docSettingsTitle')}</h1>
<p>{t('docSettingsIntro')}</p>

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
		<div class="field">
			<label for="form-number">{t('settingsSubmissionFormNumber')}</label>
			<input id="form-number" bind:value={formNumber} maxlength="100" disabled={!canEdit} aria-describedby="form-number-hint" autocomplete="off" />
			<p id="form-number-hint"><small>{t('settingsSubmissionFormNumberHint')}</small></p>
		</div>
		{#if canEdit}
			<button type="submit">{t('settingsSave')}</button>
		{/if}
	</form>
{/if}
