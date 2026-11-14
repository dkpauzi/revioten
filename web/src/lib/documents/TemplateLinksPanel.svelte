<script lang="ts">
	// Mode tautan dokumen ke process template (M5.8, R-TPL-08): follow ikut
	// setiap versi baru, pinned tetap di versinya kecuali propagasi bercakupan
	// all, detached tidak pernah ikut. Pinned dan detached wajib alasan.
	import { listDocumentTemplates, setDocumentTemplateMode } from '#lib/api/client.ts';
	import type { Document, DocumentTemplate } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let { doc }: { doc: Document } = $props();

	let items = $state<DocumentTemplate[]>([]);
	let editing = $state<string | null>(null);
	let mode = $state('follow');
	let reason = $state('');
	let message = $state('');
	let failure = $state('');
	let canEdit = $derived(can('E-2'));

	$effect(() => {
		void doc.updated_at;
		load(doc.id);
	});

	async function load(id: string) {
		const result = await listDocumentTemplates(id);
		items = result.ok ? result.data.items : [];
	}

	function edit(item: DocumentTemplate) {
		editing = item.process_template_id;
		mode = item.mode;
		reason = item.reason ?? '';
		message = '';
		failure = '';
	}

	async function save(event: SubmitEvent, item: DocumentTemplate) {
		event.preventDefault();
		failure = '';
		const result = await setDocumentTemplateMode(doc.id, item.process_template_id, { mode, reason: reason.trim() || undefined });
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		items = result.data.items;
		editing = null;
		message = t('templateModeSaved', { process: item.process_name, mode: t(`templateMode_${mode}` as 'templateMode_follow') });
	}
</script>

{#if items.length > 0}
	<section>
		<h2>{t('templateLinksTitle')}</h2>
		<p class="hint">{t('templateLinksHint')}</p>
		{#if message}<p role="status">{message}</p>{/if}
		{#if failure}<p class="error" role="alert">{failure}</p>{/if}
		<table>
			<thead>
				<tr>
					<th>{t('templateLinkTemplate')}</th>
					<th>{t('templateLinkVersion')}</th>
					<th>{t('propagationMode')}</th>
					{#if canEdit}<th>{t('masterActions')}</th>{/if}
				</tr>
			</thead>
			<tbody>
				{#each items as item (item.process_template_id)}
					<tr>
						<td>{item.process_name}</td>
						<td>{item.version}</td>
						<td>{t(`templateMode_${item.mode}` as 'templateMode_follow')}{item.reason ? ` · ${item.reason}` : ''}</td>
						{#if canEdit}
							<td>
								{#if editing === item.process_template_id}
									<form class="row" onsubmit={(e) => save(e, item)} novalidate>
										<select aria-label={t('templateModeFor', { process: item.process_name })} bind:value={mode}>
											<option value="follow">{t('templateMode_follow')}</option>
											<option value="pinned">{t('templateMode_pinned')}</option>
											<option value="detached">{t('templateMode_detached')}</option>
										</select>
										<input aria-label={t('templateModeReasonFor', { process: item.process_name })} maxlength="500" placeholder={t('changeReason')} bind:value={reason} />
										<button type="submit">{t('templateModeSave')}</button>
										<button type="button" onclick={() => (editing = null)}>{t('masterCancel')}</button>
									</form>
								{:else}
									<button type="button" aria-label={t('templateModeChangeFor', { process: item.process_name })} onclick={() => edit(item)}>{t('templateModeChange')}</button>
								{/if}
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</section>
{/if}

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		align-items: center;
	}
</style>
