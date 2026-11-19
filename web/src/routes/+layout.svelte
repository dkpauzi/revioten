<script lang="ts">
	import ReportProblem from '#lib/ReportProblem.svelte';
	// Gaya global dan huruf yang dibundel dengan aplikasi (D-127).
	import '@fontsource-variable/geist';
	import '@fontsource-variable/geist-mono';
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { LayoutProps } from './$types';
	import LanguageSwitch from '#lib/LanguageSwitch.svelte';
	import Toaster from '#lib/Toaster.svelte';
	import { getSession, logout, setUnauthenticatedHandler } from '#lib/api/client.ts';
	import { failureText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { language } from '#lib/i18n/language.svelte.ts';
	import { activeGroup, buildMenu, type MenuGroup } from '#lib/menu.ts';
	import { can, session } from '#lib/session.svelte.ts';

	// children adalah isi halaman aktif (+page.svelte) yang ditaruh di dalam layout ini.
	let { children }: LayoutProps = $props();

	// Halaman login terbuka tanpa sesi. Halaman lain baru tampil setelah sesi
	// terbukti ada; datanya tetap dijaga API di server (M1.1).
	let onLoginPage = $derived(page.url.pathname === '/login');
	let sessionError = $state('');

	// Menu atas berkelompok (D-127). Disusun ulang setiap kali sesi berubah,
	// karena isinya mengikuti hak akun yang login. Akun yang wajib ganti
	// password dulu belum mendapat menu.
	let showMenu = $derived(session.current !== null && !session.current.must_change_password);
	let menu = $derived(showMenu ? buildMenu(can) : []);
	let current = $derived(activeGroup(menu, page.url.pathname));

	// Kelompok menu yang sedang terbuka, atau null bila semua tertutup.
	let open = $state<MenuGroup['id'] | null>(null);
	// Panel menu di HP dan tablet (D-128): di layar sempit menu dilipat ke
	// tombol Menu. Di laptop panel ini selalu tampil dan nilainya diabaikan.
	let panelOpen = $state(false);
	let nav = $state<HTMLElement>();
	let menuToggle = $state<HTMLButtonElement>();

	// Pindah halaman selalu menutup menu dan panelnya.
	$effect(() => {
		void page.url.pathname;
		open = null;
		panelOpen = false;
	});

	// Atribut lang di <html> mengikuti bahasa layar (D-128), juga saat pertama dibuka.
	$effect(() => {
		document.documentElement.lang = language.current;
	});

	function toggle(id: MenuGroup['id']) {
		open = open === id ? null : id;
	}

	// Escape menutup kelompok yang terbuka lebih dulu, lalu panel menu, dan
	// mengembalikan fokus ke tombolnya supaya pengguna keyboard tidak kehilangan posisi.
	function onMenuKeydown(event: KeyboardEvent) {
		if (event.key !== 'Escape') {
			return;
		}
		if (open) {
			const button = nav?.querySelector<HTMLButtonElement>(`#menu-button-${open}`);
			open = null;
			button?.focus();
		} else if (panelOpen) {
			panelOpen = false;
			menuToggle?.focus();
		}
	}

	// Klik di luar menu menutup kelompok yang terbuka.
	function onWindowClick(event: MouseEvent) {
		if (open && nav && !nav.contains(event.target as Node)) {
			open = null;
		}
	}

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

<svelte:window onclick={onWindowClick} onkeydown={onMenuKeydown} />

<header class="top">
	<a class="brand" href="/" aria-label={t('appTitle')}>
		<span class="mark" aria-hidden="true">P</span>
		<span class="title" aria-hidden="true">{t('appTitle')}</span>
	</a>

	{#if session.current}
		<!-- Di laptop isi panel ini tampil langsung di bar atas; di HP dan tablet
		     baru tampil setelah tombol Menu ditekan. -->
		<div class="panel" id="menu-panel" class:open={panelOpen} class:always={!showMenu}>
			{#if showMenu}
				<!-- Menu atas berkelompok (D-127). Menu hanya menyembunyikan halaman yang
				     pasti ditolak; server tetap memeriksa setiap operasi (M1.2). -->
				<nav bind:this={nav} aria-label={t('navMain')}>
					<ul class="groups">
						{#each menu as group (group.id)}
							<li class="group">
								{#if group.id === 'home'}
									<a class="entry" href="/" data-active={current === 'home'} aria-current={page.url.pathname === '/' ? 'page' : undefined}>{t(group.label)}</a>
								{:else}
									<button
										type="button"
										id="menu-button-{group.id}"
										class="entry"
										aria-expanded={open === group.id}
										aria-controls="menu-{group.id}"
										data-active={current === group.id}
										onclick={() => toggle(group.id)}
									>
										{t(group.label)}
										<svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
											<path d="M3 4.5 6 7.5 9 4.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
										</svg>
									</button>
									{#if open === group.id}
										<ul class="popup" id="menu-{group.id}">
											{#each group.items as item (item.href)}
												<li>
													<a href={item.href} aria-current={page.url.pathname === item.href ? 'page' : undefined}>{t(item.label)}</a>
												</li>
											{/each}
										</ul>
									{/if}
								{/if}
							</li>
						{/each}
					</ul>
				</nav>
			{/if}
			<div class="account">
				<span class="who">
					{t('loggedInAs', { name: session.current.person.name, login: session.current.account.login })}
					{#if session.current.account.kind === 'admin'}
						<em class="badge">{t('accountKindAdmin')}</em>
					{/if}
				</span>
				<a href="/account/password">{t('navAccount')}</a>
				<a href="/help">{t('navHelp')}</a>
				<ReportProblem />
				<a href="/about">{t('navAbout')}</a>
			</div>
		</div>
	{/if}

	<div class="tools">
		<LanguageSwitch />
		{#if showMenu}
			<button
				type="button"
				class="menu-toggle"
				bind:this={menuToggle}
				aria-expanded={panelOpen}
				aria-controls="menu-panel"
				onclick={() => (panelOpen = !panelOpen)}
			>
				<svg width="18" height="18" viewBox="0 0 18 18" aria-hidden="true">
					<path d="M3 5h12M3 9h12M3 13h12" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
				</svg>
				{t('navMenu')}
			</button>
		{/if}
		{#if session.current}
			<!-- Logout selalu terlihat di setiap halaman yang butuh login, juga di HP (ASVS 7.4.4). -->
			<button type="button" onclick={onLogout}>{t('logout')}</button>
		{/if}
	</div>
</header>

<main class="page" class:login={onLoginPage}>
	{#if onLoginPage || session.current}
		{@render children()}
	{:else if sessionError}
		<p class="error" role="alert">{sessionError}</p>
	{:else}
		<p>{t('sessionChecking')}</p>
	{/if}
</main>

<!-- Pop-up untuk setiap pesan berhasil atau gagal di halaman. -->
<Toaster />

<style>
	.top {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 16px;
		padding: 10px 28px;
		background: var(--surface);
		border-bottom: 1px solid var(--border);
	}

	.brand {
		display: flex;
		align-items: center;
		gap: 10px;
		color: var(--ink);
		font-weight: 600;
	}

	.brand:hover {
		text-decoration: none;
	}

	.mark {
		display: grid;
		place-items: center;
		flex: none;
		width: 34px;
		height: 34px;
		border-radius: 9px;
		background: var(--ink);
		color: #ffffff;
		font-weight: 700;
	}

	/* Isi panel (menu dan akun) di laptop: satu baris di antara logo dan alat. */
	.panel {
		display: flex;
		flex: 1 1 auto;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 16px;
		min-width: 0;
	}

	.groups {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.group {
		position: relative;
	}

	/* Tombol kelompok dan tautan Home tampil sama: teks berlabel, tanpa bingkai. */
	.entry {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		min-height: 40px;
		padding: 0 14px;
		border: 0;
		border-radius: var(--radius);
		background: transparent;
		color: var(--text-2);
		font-weight: 500;
	}

	.entry:hover {
		background: #f2f4f7;
		text-decoration: none;
	}

	.entry[data-active='true'],
	.entry[aria-expanded='true'] {
		background: var(--accent-weak);
		color: var(--accent-ink);
	}

	.popup {
		position: absolute;
		top: calc(100% + 6px);
		left: 0;
		z-index: 10;
		min-width: 220px;
		margin: 0;
		padding: 6px;
		list-style: none;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 12px;
		box-shadow: 0 12px 32px rgba(16, 24, 40, 0.12);
	}

	.popup a {
		display: block;
		padding: 9px 12px;
		border-radius: var(--radius);
		color: var(--text-2);
		white-space: nowrap;
	}

	.popup a:hover {
		background: #f2f4f7;
		text-decoration: none;
	}

	.popup a[aria-current='page'] {
		color: var(--accent-ink);
		font-weight: 600;
	}

	.account {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 16px;
		margin-left: auto;
		font-size: 14px;
	}

	.who {
		color: var(--text-3);
	}

	.who em {
		margin-left: 6px;
		font-style: normal;
	}

	.tools {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.menu-toggle {
		display: none;
	}

	/* Halaman login: satu kartu di tengah. */
	.login {
		max-width: 440px;
		margin-top: 56px;
		padding: 32px;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}

	/* Laptop (1024px ke atas): dua baris yang tetap. Baris atas berisi logo,
	   akun, dan alat; baris bawah berisi menu. Tanpa aturan ini isi bilah atas
	   terlipat tidak teratur begitu teks akun lebih panjang, misalnya dalam
	   bahasa Indonesia. display: contents membuat menu dan akun ikut diatur
	   langsung oleh .top. */
	@media (min-width: 1024px) {
		.panel {
			display: contents;
		}

		.account {
			order: 2;
			flex-wrap: nowrap;
			min-width: 0;
			white-space: nowrap;
		}

		.who {
			overflow: hidden;
			text-overflow: ellipsis;
		}

		.tools {
			order: 3;
		}

		nav {
			order: 4;
			flex-basis: 100%;
		}
	}

	/* HP dan tablet (di bawah 1024px): menu dilipat ke tombol Menu, dan panelnya
	   turun sebagai baris penuh di bawah bar atas. */
	@media (max-width: 1023px) {
		.top {
			padding: 10px 20px;
		}

		.tools {
			margin-left: auto;
		}

		.menu-toggle {
			display: inline-flex;
		}

		.panel:not(.always) {
			display: none;
		}

		.panel.open {
			display: flex;
			flex-direction: column;
			align-items: stretch;
			order: 3;
			flex-basis: 100%;
			padding: 8px 0 4px;
			border-top: 1px solid var(--border);
		}

		.panel.always {
			order: 3;
			flex-basis: 100%;
		}

		.panel.open .groups {
			flex-direction: column;
		}

		.panel.open .entry {
			width: 100%;
			min-height: 48px;
			justify-content: space-between;
		}

		/* Isi kelompok tampil di dalam panel, bukan melayang. */
		.panel.open .popup {
			position: static;
			min-width: 0;
			margin: 2px 0 6px 12px;
			border: 0;
			border-left: 2px solid var(--border);
			border-radius: 0;
			box-shadow: none;
		}

		.panel.open .popup a {
			padding: 12px;
		}

		.account {
			margin-left: 0;
			padding-top: 8px;
			border-top: 1px solid var(--border-soft);
		}
	}

	/* HP (di bawah 640px): nama aplikasi disembunyikan dari tampilan supaya bar
	   atas muat satu baris; namanya tetap dibacakan lewat aria-label tautan. */
	@media (max-width: 639px) {
		.top {
			padding: 8px 16px;
		}

		.title {
			display: none;
		}

		.login {
			margin: 24px 0 0;
			padding: 24px 20px;
		}
	}
</style>
