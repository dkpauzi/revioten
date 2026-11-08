<script lang="ts">
	import { changePassword, getSession } from '#lib/api/client.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { session } from '#lib/session.svelte.ts';

	let current = $state('');
	let newPassword = $state('');
	let confirm = $state('');
	let saving = $state(false);
	// Pesan per field, misalnya { new_password: 'Use at least 15 characters.' }.
	let fieldErrors = $state<Record<string, string>>({});
	let formError = $state('');
	let changed = $state(false);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		formError = '';
		changed = false;
		// Hanya kecocokan dua isian yang diperiksa di browser; aturan password
		// diperiksa server, supaya aturannya hanya ada di satu tempat.
		if (newPassword !== confirm) {
			fieldErrors = { confirm: t('passwordMismatch') };
			return;
		}

		saving = true;
		const result = await changePassword({ current_password: current, new_password: newPassword });
		saving = false;
		if (result.ok) {
			current = '';
			newPassword = '';
			confirm = '';
			changed = true;
			// Server sudah memasang cookie sesi baru; ambil ulang data sesinya.
			const refreshed = await getSession();
			if (refreshed.ok) {
				session.current = refreshed.data;
			}
			return;
		}
		const details = result.error.body?.details ?? [];
		if (details.length > 0) {
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
		} else {
			formError = failureText(result.error);
		}
	}
</script>

<h1>{t('passwordTitle')}</h1>

{#if session.current?.must_change_password}
	<p role="alert">{t('passwordMustChange')}</p>
{/if}

<form onsubmit={submit} novalidate>
	<div class="field">
		<label for="current-password">{t('passwordCurrent')}</label>
		<input
			id="current-password"
			type="password"
			autocomplete="current-password"
			bind:value={current}
			aria-invalid={fieldErrors.current_password ? 'true' : undefined}
		/>
		{#if fieldErrors.current_password}
			<p class="error">{fieldErrors.current_password}</p>
		{/if}
	</div>

	<div class="field">
		<label for="new-password">{t('passwordNew')}</label>
		<input
			id="new-password"
			type="password"
			autocomplete="new-password"
			aria-describedby="new-password-hint"
			bind:value={newPassword}
			aria-invalid={fieldErrors.new_password ? 'true' : undefined}
		/>
		<small id="new-password-hint">{t('passwordNewHint')}</small>
		{#if fieldErrors.new_password}
			<p class="error">{fieldErrors.new_password}</p>
		{/if}
	</div>

	<div class="field">
		<label for="confirm-password">{t('passwordConfirm')}</label>
		<input
			id="confirm-password"
			type="password"
			autocomplete="new-password"
			bind:value={confirm}
			aria-invalid={fieldErrors.confirm ? 'true' : undefined}
		/>
		{#if fieldErrors.confirm}
			<p class="error">{fieldErrors.confirm}</p>
		{/if}
	</div>

	<button type="submit" disabled={saving}>{saving ? t('passwordSaving') : t('passwordSave')}</button>

	{#if formError}
		<p class="error" role="alert">{formError}</p>
	{/if}
	{#if changed}
		<p role="status">{t('passwordChanged')}</p>
	{/if}
</form>

<style>
	.field {
		margin-bottom: 1rem;
		max-width: 28rem;
	}
</style>
