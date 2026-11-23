<script lang="ts">
	// Perubahan logo (D-223, E-145 opsi a). Bila admin mengganti logo dan
	// menjawab "ya, revisi dokumen", setiap customer yang punya dokumen aktif
	// menunggu satu perubahan logo. Formulir ini hanya tampil selama ada yang
	// menunggu; perubahan itu mencakup semua dokumen aktif customer, lalu
	// diverifikasi, ditandatangani, dan dirilis seperti perubahan lain.
	import { onMount } from 'svelte';
	import { getPendingLogoChange, submitLogoChange } from '#lib/api/client.ts';
	import type { Change, ChangeType, FourMCategory, PendingLogoChange } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		customerId,
		types,
		categories,
		onsubmitted
	}: { customerId: string; types: ChangeType[]; categories: FourMCategory[]; onsubmitted: (c: Change) => void } = $props();

	let pending = $state<PendingLogoChange | null>(null);
	let changeTypeId = $state('');
	let fourMId = $state('');
	let reason = $state(t('logoChangeDefaultReason'));
	let failure = $state('');

	onMount(async () => {
		const result = await getPendingLogoChange(customerId);
		pending = result.ok ? result.data : null;
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		failure = '';
		if (!pending?.logo_change_id) {
			return;
		}
		const result = await submitLogoChange({
			customer_id: customerId,
			logo_change_id: pending.logo_change_id,
			change_type_id: changeTypeId,
			four_m_category_id: fourMId || undefined,
			reason: reason.trim()
		});
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? details.map((d) => problemText(d)).join(' ') : failureText(result.error);
			return;
		}
		pending = { pending: false };
		onsubmitted(result.data);
	}
</script>

{#if pending?.pending}
	<section aria-labelledby="logo-change-title">
		<h2 id="logo-change-title">{t('logoChangeTitle')}</h2>
		<p class="hint">{t('logoChangeHint', { date: formatDateTime(pending.changed_at ?? '') })}</p>
		{#if failure}<p class="error" role="alert">{failure}</p>{/if}
		<form class="grid" onsubmit={submit} novalidate>
			<div class="field">
				<label for="logo-change-type">{t('changeType')}</label>
				<select id="logo-change-type" bind:value={changeTypeId}>
					<option value="">{t('changeChoose')}</option>
					{#each types as ct (ct.id)}<option value={ct.id}>{ct.name}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="logo-change-4m">{t('changeFourM')}</label>
				<select id="logo-change-4m" bind:value={fourMId}>
					<option value="">{t('changeChoose')}</option>
					{#each categories as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
				</select>
			</div>
			<div class="field wide">
				<label for="logo-change-reason">{t('changeReason')}</label>
				<textarea id="logo-change-reason" rows="2" maxlength="2000" bind:value={reason}></textarea>
			</div>
			<p class="wide"><button type="submit" class="primary">{t('logoChangeSubmit')}</button></p>
		</form>
	</section>
{/if}

<style>
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
		gap: 0 1rem;
	}

	.wide {
		grid-column: 1 / -1;
	}
</style>
