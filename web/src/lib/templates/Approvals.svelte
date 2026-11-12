<script lang="ts">
	// Daftar persetujuan satu versi (D-139): siapa yang sudah setuju, dan
	// selama draft, siapa yang masih ditunggu menurut aturan di Settings.
	import type { FormApprover } from '#lib/api/gen/index.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';

	let { approvals = [], pending = [] }: { approvals?: FormApprover[]; pending?: FormApprover[] } = $props();
</script>

{#if approvals.length > 0 || pending.length > 0}
	<div class="approvals">
		{#if approvals.length > 0}
			<p>
				<strong>{t('approvalsDone')}</strong>
				{approvals.map((a) => t('approvalBy', { name: a.person_name, time: a.approved_at ? formatDateTime(a.approved_at) : '' })).join(', ')}
			</p>
		{/if}
		{#if pending.length > 0}
			<p><strong>{t('approvalsPending')}</strong> {pending.map((a) => a.person_name).join(', ')}</p>
		{/if}
	</div>
{/if}

<style>
	.approvals p {
		margin: 4px 0;
		font-size: 14px;
	}
</style>