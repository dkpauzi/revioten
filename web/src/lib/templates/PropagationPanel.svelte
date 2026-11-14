<script lang="ts">
	// Propagasi template (M5.8, R-TPL-06 sampai R-TPL-09). Untuk versi yang
	// sudah terbit: pratinjau dampak ke dokumen yang tertaut, lalu jalankan.
	// Hasilnya perubahan bermekanisme template per customer, yang menunggu
	// verifikasi; isi baru masuk ke draft saat perubahan diterima.
	import { listChangeTypes, previewPropagation, propagateTemplate } from '#lib/api/client.ts';
	import type { ChangeType, PropagationItem, ProcessTemplateVersion } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let { version }: { version: ProcessTemplateVersion } = $props();

	let scope = $state('following');
	let items = $state<PropagationItem[] | null>(null);
	let types = $state<ChangeType[]>([]);
	let changeTypeId = $state('');
	let conflictPolicy = $state('keep');
	let reason = $state('');
	let sourceReference = $state('');
	let effectiveDate = $state('');
	let message = $state('');
	let failure = $state('');
	let included = $derived(items?.filter((i) => i.included).length ?? 0);

	async function preview() {
		message = '';
		failure = '';
		const result = await previewPropagation(version.id, scope);
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		items = result.data.items;
		if (types.length === 0) {
			const list = await listChangeTypes();
			types = list.ok ? list.data.items : [];
		}
	}

	async function run(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		const result = await propagateTemplate(version.id, {
			scope,
			conflict_policy: conflictPolicy,
			change_type_id: changeTypeId,
			reason: reason.trim(),
			source_reference: sourceReference.trim() || undefined,
			effective_date: effectiveDate || undefined
		});
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? details.map((d) => problemText(d)).join(' ') : failureText(result.error);
			return;
		}
		message = t('propagationDone', { refs: result.data.items.map((c) => c.ref).join(', ') });
		items = null;
	}

	function skippedText(item: PropagationItem): string {
		return item.skipped ? t(`propagationSkipped_${item.skipped}` as 'propagationSkipped_pinned') : '';
	}
</script>

<section>
	<h2>{t('propagationTitle', { version: String(version.version) })}</h2>
	<p class="hint">{t('propagationHint')}</p>
	<div class="row">
		<div class="field">
			<label for="propagation-scope">{t('propagationScope')}</label>
			<select id="propagation-scope" bind:value={scope} onchange={() => (items = null)}>
				<option value="following">{t('propagationScopeFollowing')}</option>
				<option value="all">{t('propagationScopeAll')}</option>
			</select>
		</div>
		<button type="button" onclick={preview}>{t('propagationPreview')}</button>
	</div>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if items}
		{#if items.length === 0}
			<p>{t('propagationNoDocuments')}</p>
		{:else}
			<table class="preview">
				<thead>
					<tr>
						<th>{t('documentNumber')}</th>
						<th>{t('propagationMode')}</th>
						<th>{t('propagationUpdates')}</th>
						<th>{t('propagationConflicts')}</th>
						<th>{t('propagationRemoved')}</th>
						<th>{t('propagationTakesPart')}</th>
					</tr>
				</thead>
				<tbody>
					{#each items as item (item.document_id)}
						<tr data-included={item.included}>
							<td class="mono">{item.number}</td>
							<td>{t(`templateMode_${item.mode}` as 'templateMode_follow')}</td>
							<td>{item.updates}</td>
							<td>{item.conflicts}</td>
							<td>{item.removed}</td>
							<td>{item.included ? t('propagationYes') : skippedText(item)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
		{#if included > 0}
			<form onsubmit={run} novalidate>
				<div class="row">
					<div class="field">
						<label for="propagation-type">{t('changeType')}</label>
						<select id="propagation-type" bind:value={changeTypeId}>
							<option value="">{t('changeChoose')}</option>
							{#each types as ct (ct.id)}<option value={ct.id}>{ct.name}</option>{/each}
						</select>
					</div>
					<div class="field">
						<label for="propagation-conflict">{t('propagationConflictPolicy')}</label>
						<select id="propagation-conflict" bind:value={conflictPolicy}>
							<option value="keep">{t('propagationKeep')}</option>
							<option value="overwrite">{t('propagationOverwrite')}</option>
						</select>
					</div>
				</div>
				<div class="field">
					<label for="propagation-reason">{t('changeReason')}</label>
					<textarea id="propagation-reason" rows="2" maxlength="2000" bind:value={reason}></textarea>
				</div>
				{#if scope === 'all'}
					<div class="row">
						<div class="field">
							<label for="propagation-sqam">{t('propagationSqam')}</label>
							<input id="propagation-sqam" maxlength="200" bind:value={sourceReference} />
						</div>
						<div class="field">
							<label for="propagation-date">{t('propagationEffectiveDate')}</label>
							<input id="propagation-date" type="date" bind:value={effectiveDate} />
						</div>
					</div>
				{/if}
				<button type="submit" class="primary">{t('propagationRun', { count: String(included) })}</button>
			</form>
		{/if}
	{/if}
</section>

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	.row > button {
		margin-top: var(--label-space);
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	tr[data-included='false'] {
		color: var(--text-3);
	}
</style>
