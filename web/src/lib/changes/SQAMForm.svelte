<script lang="ts">
	// Perubahan surut SQAM (M5.10, R-CHG-01, R-TPL-07): satu perubahan untuk
	// semua dokumen aktif customer, dengan rujukan revisi SQAM dan tanggal
	// berlaku. Saat diterima, setiap dokumen mendapat draft baru.
	import { submitSQAMChange } from '#lib/api/client.ts';
	import type { Change, ChangeType, FourMCategory } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		customerId,
		types,
		categories,
		onsubmitted
	}: { customerId: string; types: ChangeType[]; categories: FourMCategory[]; onsubmitted: (c: Change) => void } = $props();

	let open = $state(false);
	let changeTypeId = $state('');
	let fourMId = $state('');
	let reason = $state('');
	let reference = $state('');
	let effectiveDate = $state('');
	let failure = $state('');

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		failure = '';
		const result = await submitSQAMChange({
			customer_id: customerId,
			change_type_id: changeTypeId,
			four_m_category_id: fourMId || undefined,
			reason: reason.trim(),
			sqam_reference: reference.trim(),
			effective_date: effectiveDate || undefined
		});
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? details.map((d) => problemText(d)).join(' ') : failureText(result.error);
			return;
		}
		open = false;
		reason = reference = effectiveDate = '';
		onsubmitted(result.data);
	}
</script>

<section>
	<h2>{t('sqamTitle')}</h2>
	<p class="hint">{t('sqamHint')}</p>
	{#if !open}
		<button type="button" onclick={() => (open = true)}>{t('sqamOpen')}</button>
	{:else}
		{#if failure}<p class="error" role="alert">{failure}</p>{/if}
		<form class="grid" onsubmit={submit} novalidate>
			<div class="field">
				<label for="sqam-reference">{t('propagationSqam')}</label>
				<input id="sqam-reference" maxlength="200" bind:value={reference} />
			</div>
			<div class="field">
				<label for="sqam-date">{t('propagationEffectiveDate')}</label>
				<input id="sqam-date" type="date" bind:value={effectiveDate} />
			</div>
			<div class="field">
				<label for="sqam-type">{t('changeType')}</label>
				<select id="sqam-type" bind:value={changeTypeId}>
					<option value="">{t('changeChoose')}</option>
					{#each types as ct (ct.id)}<option value={ct.id}>{ct.name}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="sqam-fourm">{t('changeFourM')}</label>
				<select id="sqam-fourm" bind:value={fourMId}>
					<option value="">{t('changeChoose')}</option>
					{#each categories as f (f.id)}<option value={f.id}>{f.name}</option>{/each}
				</select>
			</div>
			<div class="field wide">
				<label for="sqam-reason">{t('changeReason')}</label>
				<textarea id="sqam-reason" rows="2" maxlength="2000" bind:value={reason}></textarea>
			</div>
			<p class="wide"><button type="submit" class="primary">{t('sqamSubmit')}</button></p>
		</form>
	{/if}
</section>

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
