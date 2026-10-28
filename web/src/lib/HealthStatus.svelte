<script lang="ts">
	import { onMount } from 'svelte';
	import { checkHealth, type HealthResult } from '#lib/health.ts';
	import { t } from '#lib/i18n/index.ts';

	// Status yang tampil di layar. Rune $state membuat layar ikut berubah
	// setiap kali nilai ini diganti.
	let status = $state<'checking' | HealthResult>('checking');

	async function refresh() {
		status = 'checking';
		status = await checkHealth();
	}

	// Periksa sekali saat komponen pertama kali tampil.
	onMount(() => {
		refresh();
	});
</script>

<p>
	{#if status === 'checking'}
		{t('healthChecking')}
	{:else if status === 'ok'}
		{t('healthOk')}
	{:else if status === 'database_unreachable'}
		{t('healthDatabaseDown')}
	{:else}
		{t('healthError')}
	{/if}
</p>

<button onclick={refresh} disabled={status === 'checking'}>{t('healthRetry')}</button>
