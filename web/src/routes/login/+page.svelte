<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { getSession, login } from '#lib/api/client.ts';
	import type { Session } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { safeNext } from '#lib/navigation.ts';
	import { session } from '#lib/session.svelte.ts';

	let loginName = $state('');
	let password = $state('');
	let submitting = $state(false);
	let errorText = $state('');
	// Isi password bisa ditampilkan sebentar untuk memeriksa ketikan (D-187).
	let showPassword = $state(false);

	// enter menyimpan sesi lalu membuka halaman tujuan. Akun yang passwordnya
	// direset admin wajib ganti password dulu (E-08).
	function enter(data: Session) {
		session.current = data;
		if (data.must_change_password) {
			goto('/account/password');
			return;
		}
		// next diperiksa safeNext supaya tidak bisa membawa ke situs lain.
		goto(safeNext(page.url.searchParams.get('next'))).catch(() => goto('/'));
	}

	onMount(async () => {
		// Bila sudah login (misalnya halaman ini dibuka lewat tautan), langsung lanjut.
		const result = await getSession();
		if (result.ok) {
			enter(result.data);
		}
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		submitting = true;
		errorText = '';
		const result = await login({ login: loginName, password });
		submitting = false;
		password = '';
		showPassword = false;
		if (result.ok) {
			enter(result.data);
			return;
		}
		// Pesan login gagal sengaja umum: tidak menyebut apakah nama login ada.
		errorText = result.error.status === 401 ? t('loginFailed') : failureText(result.error);
	}
</script>

<h1>{t('loginTitle')}</h1>

<form onsubmit={submit}>
	<div class="field">
		<label for="login-name">{t('loginName')}</label>
		<input id="login-name" name="username" autocomplete="username" bind:value={loginName} />
	</div>

	<div class="field">
		<label for="login-password">{t('loginPassword')}</label>
		<!-- type=password menyamarkan isi, dan tempel (paste) tetap boleh (ASVS 6.2.6, 6.2.7). -->
		<div class="password">
			<input
				id="login-password"
				name="password"
				type={showPassword ? 'text' : 'password'}
				autocomplete="current-password"
				bind:value={password}
			/>
			<!-- type=button supaya tombol ini tidak mengirim formulir. -->
			<button type="button" aria-pressed={showPassword} aria-controls="login-password" onclick={() => (showPassword = !showPassword)}>
				{showPassword ? t('loginHide') : t('loginShow')}
			</button>
		</div>
	</div>

	<button type="submit" disabled={submitting}>
		{submitting ? t('loginSubmitting') : t('loginSubmit')}
	</button>

	{#if errorText}
		<p class="error" role="alert">{errorText}</p>
	{/if}
	<p><small>{t('loginHelp')}</small></p>
</form>

<style>
	.password {
		display: flex;
		gap: 0.5rem;
	}

	.password input {
		flex: 1;
		min-width: 0;
	}

	.field {
		margin-bottom: 1rem;
		max-width: 20rem;
	}
</style>
