<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { LayoutProps } from './$types';
	import { getSession, logout, setUnauthenticatedHandler } from '#lib/api/client.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can, session } from '#lib/session.svelte.ts';

	// children adalah isi halaman aktif (+page.svelte) yang ditaruh di dalam layout ini.
	let { children }: LayoutProps = $props();

	// Halaman login terbuka tanpa sesi. Halaman lain baru tampil setelah sesi
	// terbukti ada; datanya tetap dijaga API di server (M1.1).
	let onLoginPage = $derived(page.url.pathname === '/login');
	let sessionError = $state('');

	// toLogin membawa pengguna ke halaman login, dengan alamat sekarang sebagai next.
	function toLogin() {
		session.current = null;
		if (page.url.pathname !== '/login') {
			goto('/login?next=' + encodeURIComponent(page.url.pathname + page.url.search));
		}
	}

	async function checkSession() {
		const result = await getSession();
		if (result.ok) {
			session.current = result.data;
			if (result.data.must_change_password && page.url.pathname !== '/account/password') {
				goto('/account/password');
			}
		} else if (result.error.status !== 401) {
			// 401 sudah ditangani toLogin; selain itu tampilkan masalahnya.
			sessionError = failureText(result.error);
		}
	}

	async function onLogout() {
		await logout();
		session.current = null;
		goto('/login');
	}

	onMount(() => {
		// Setiap jawaban 401 dari API (sesi berakhir) membawa ke halaman login.
		setUnauthenticatedHandler(toLogin);
		if (!onLoginPage) {
			checkSession();
		}
		return () => setUnauthenticatedHandler(null);
	});
</script>

<svelte:head>
	<title>{t('appTitle')}</title>
</svelte:head>

<header>
	<strong>{t('appTitle')}</strong>
	{#if session.current}
		{#if !session.current.must_change_password}
			<nav>
				<a href="/">{t('navHome')}</a>
				<a href="/customers">{t('navCustomers')}</a>
				<a href="/models">{t('navModels')}</a>
				<!-- Menu hanya untuk pemegang hak A-1; server tetap memeriksa setiap operasi. -->
				{#if can('A-1')}
					<a href="/admin/accounts">{t('navAccounts')}</a>
				{/if}
				<!-- Audit log hanya lewat akun admin (A-4, D-75). -->
				{#if can('A-4')}
					<a href="/admin/audit">{t('navAuditLog')}</a>
				{/if}
				<a href="/about">{t('navAbout')}</a>
			</nav>
		{/if}
		<!-- Logout selalu terlihat di setiap halaman yang butuh login (ASVS 7.4.4). -->
		<span class="account">
			{t('loggedInAs', { name: session.current.person.name, login: session.current.account.login })}
			{#if session.current.account.kind === 'admin'}
				<em>{t('accountKindAdmin')}</em>
			{/if}
			<a href="/account/password">{t('navAccount')}</a>
			<button type="button" onclick={onLogout}>{t('logout')}</button>
		</span>
	{/if}
</header>

<main>
	{#if onLoginPage || session.current}
		{@render children()}
	{:else if sessionError}
		<p class="error" role="alert">{sessionError}</p>
	{:else}
		<p>{t('sessionChecking')}</p>
	{/if}
</main>

<style>
	header {
		display: flex;
		flex-wrap: wrap;
		gap: 1.5rem;
		align-items: baseline;
		border-bottom: 1px solid #ccc;
		padding: 0.5rem 1rem;
	}

	nav,
	.account {
		display: flex;
		gap: 1rem;
		align-items: baseline;
	}

	.account {
		margin-left: auto;
	}

	main {
		padding: 1rem;
	}

	.error {
		color: #b00020;
	}
</style>
