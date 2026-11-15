<script lang="ts">
	// Spanduk review berkala (D-196) di halaman dokumen: tampil bila set dokumen
	// ini akan atau sudah lewat jatuh tempo review 3 tahun. Tidak tampil bila
	// belum jatuh tempo atau dokumen belum pernah terbit.
	import { listReviewSets } from '#lib/api/client.ts';
	import type { ReviewSet } from '#lib/api/gen/index.ts';
	import { t } from '#lib/i18n/index.ts';

	let { documentId }: { documentId: string } = $props();
	let set = $state<ReviewSet | null>(null);

	$effect(() => {
		const id = documentId;
		set = null;
		listReviewSets({ documentId: id }).then((result) => {
			if (result.ok && id === documentId) {
				set = result.data.items[0] ?? null;
			}
		});
	});
</script>

{#if set}
	<div class="review" data-review={set.status} data-testid="review-banner">
		<span class="badge" data-review={set.status}>
			{set.status === 'overdue' ? t('reviewOverdue', { days: String(-set.days_left) }) : t('reviewDueIn', { days: String(set.days_left) })}
		</span>
		<span>{t('reviewBanner', { date: set.due_on })}</span>
		<a href="/periodic-reviews">{t('reviewBannerLink')}</a>
	</div>
{/if}

<style>
	.review {
		display: flex;
		flex-wrap: wrap;
		gap: 10px;
		align-items: center;
		margin: 0 0 1rem;
		padding: 12px 16px;
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
	}
</style>
