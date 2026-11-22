<script lang="ts">
	// Halaman pengaturan sistem (M1.8, UC-07). Sejak D-209 isinya hanya pengaman
	// yang tetap di admin: mode migrasi (R-MIG-06) dan aturan persetujuan form
	// set dan tabel AP (D-139, D-148). Tanggal di slot, QR, dan nomor form
	// submission pindah ke Master data → Document settings (E-11).
	// Mengubahnya butuh hak A-3; server tetap memeriksa setiap operasi.
	import { onMount } from 'svelte';
	import { getSystemSettings, updateSystemSettings } from '#lib/api/client.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';
	import CompanyLogo from '#lib/settings/CompanyLogo.svelte';

	let loaded = $state(false);
	// Aturan persetujuan form set dan tabel AP (D-139).
	let approvalMode = $state<'all' | 'any'>('all');
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let canEdit = $derived(can('A-3'));

	// Saklar mode migrasi (R-MIG-06); bawaannya mati.
	let migrationMode = $state(false);

	async function load() {
		const result = await getSystemSettings();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		approvalMode = result.data.form_approval_mode;
		migrationMode = result.data.migration_mode;
		loaded = true;
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		const result = await updateSystemSettings({
			form_approval_mode: approvalMode,
			migration_mode: migrationMode
		});
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		approvalMode = result.data.form_approval_mode;
		migrationMode = result.data.migration_mode;
		message = t('settingsSaved');
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('settingsTitle')}</h1>
<p>{t('settingsMovedHint')}</p>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if !loaded}
	<p>{t('settingsLoading')}</p>
{:else}
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<form onsubmit={save} novalidate>
		<label>
			<input type="checkbox" bind:checked={migrationMode} disabled={!canEdit} aria-describedby="migration-hint" />
			{t('settingsMigration')}
		</label>
		<p id="migration-hint"><small>{t('settingsMigrationHint')}</small></p>
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

<CompanyLogo />
