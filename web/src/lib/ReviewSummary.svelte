<script lang="ts">
	// Kartu review berkala di Home dan dasbor (D-196): jumlah set dokumen yang
	// akan jatuh tempo dan yang sudah lewat, untuk customer yang boleh dilihat
	// akun ini. Tautannya membuka daftar review.
	import { onMount } from 'svelte';
	import { listReviewSets } from '#lib/api/client.ts';
	import type { ReviewSetList } from '#lib/api/gen/index.ts';
	import { t } from '#lib/i18n/index.ts';

	let { customerId = '' }: { customerId?: string } = $props();
	let overview = $state<ReviewSetList | null>(null);

	async function load(id: string) {
		const result = await listReviewSets({ customerId: id });
		overview = result.ok ? result.data : null;
	}

	onMount(() => load(customerId));
	$effect(() => {
		load(customerId);
	});
</script>

{#if overview}
	<section class="summary" aria-labelledby="review-summary" data-testid="review-summary">
		<h2 id="review-summary">{t('reviewTitle')}</h2>
		{#if overview.overdue + overview.due_soon === 0}
			<p>{t('reviewNone')}</p>
		{:else}
			<p class="counts">
				<span class="badge" data-review="overdue">{t('reviewCountOverdue', { count: String(overview.overdue) })}</span>
				<span class="badge" data-review="due_soon">{t('reviewCountDueSoon', { count: String(overview.due_soon) })}</span>
			</p>
		{/if}
		<p><a href="/periodic-reviews">{t('reviewBannerLink')}</a></p>
	</section>
{/if}

<style>
	.summary {
		margin-bottom: 24px;
		padding: 22px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
	}

	.summary h2 {
		font-size: 18px;
	}

	.counts {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
</style>
