<script lang="ts">
	import type { LayoutProps } from './$types';
	import { t } from '#lib/i18n/index.ts';

	// children adalah isi halaman aktif (+page.svelte) yang ditaruh di dalam layout ini.
	let { children }: LayoutProps = $props();
</script>

<svelte:head>
	<title>{t('appTitle')}</title>
</svelte:head>

<header>
	<strong>{t('appTitle')}</strong>
	<nav>
		<a href="/">{t('navHome')}</a>
		<a href="/about">{t('navAbout')}</a>
	</nav>
</header>

<main>
	{@render children()}
</main>

<style>
	header {
		display: flex;
		gap: 1.5rem;
		align-items: baseline;
		border-bottom: 1px solid #ccc;
		padding: 0.5rem 1rem;
	}

	nav {
		display: flex;
		gap: 1rem;
	}

	main {
		padding: 1rem;
	}
</style>
