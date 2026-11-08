<script lang="ts">
	// Tabel sub-proses satu versi process template (M3.2): ganti nama,
	// sembunyikan, ubah urutan, dan pilih sub-proses untuk melihat poinnya.
	// Sub-proses tidak pernah dihapus, hanya disembunyikan; ID-nya tetap, jadi
	// poin dan rujukan lain tetap utuh. Server tetap memeriksa setiap perubahan.
	import { updateSubprocess } from '#lib/api/client.ts';
	import type { DocumentType, ProcessTemplateVersion, Subprocess, SubprocessUpdate } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { pointSummary } from '#lib/processTemplates.ts';

	// onselect menerima ID sub-proses yang poinnya ingin dilihat; onsaved
	// menerima versi sesudah perubahan dan pesan untuk layar.
	let {
		version,
		types,
		canEdit,
		selectedId,
		onselect,
		onsaved
	}: {
		version: ProcessTemplateVersion;
		types: DocumentType[];
		canEdit: boolean;
		selectedId: string;
		onselect: (id: string) => void;
		onsaved: (v: ProcessTemplateVersion, message: string) => void;
	} = $props();

	// Sub-proses yang sedang diubah, beserta isiannya.
	let editingId = $state<string | null>(null);
	let name = $state('');
	let hidden = $state(false);
	let fieldErrors = $state<Record<string, string>>({});
	let failure = $state('');

	function startEdit(s: Subprocess) {
		editingId = s.id;
		name = s.name;
		hidden = s.hidden;
		fieldErrors = {};
		failure = '';
	}

	// send mengirim perubahan satu sub-proses, lalu menyerahkan versi barunya.
	async function send(s: Subprocess, input: SubprocessUpdate) {
		fieldErrors = {};
		failure = '';
		const result = await updateSubprocess(version.id, s.id, input);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'processTemplate')]));
			if (details.length === 0 || editingId === null) {
				failure = details.length > 0 ? problemText(details[0], 'processTemplate') : failureText(result.error);
			}
			return;
		}
		editingId = null;
		onsaved(result.data, t('subprocessSaved', { name: input.name }));
	}

	function save(event: SubmitEvent, s: Subprocess) {
		event.preventDefault();
		send(s, { name, hidden });
	}

	// move memindahkan sub-proses satu tempat; isian lain tidak dikirim, jadi tetap (D-111).
	function move(s: Subprocess, delta: number) {
		send(s, { name: s.name, position: s.position + delta });
	}
</script>

{#if failure}<p class="error" role="alert">{failure}</p>{/if}
{#if version.subprocesses.length === 0}
	<p>{t('subprocessNone')}</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>{t('subprocessColumnNo')}</th>
				<th>{t('subprocessColumnName')}</th>
				<th>{t('subprocessColumnPoints')}</th>
				<th>{t('subprocessColumnHidden')}</th>
				<th>{t('masterActions')}</th>
			</tr>
		</thead>
		<tbody>
			{#each version.subprocesses as s, i (s.id)}
				<tr class:hidden={s.hidden} class:selected={s.id === selectedId}>
					<td>{s.position}</td>
					{#if editingId === s.id}
						<td colspan="4">
							<form class="edit" onsubmit={(event) => save(event, s)} novalidate>
								<div class="field">
									<label for="subprocess-name-{s.id}">{t('subprocessName')}</label>
									<input id="subprocess-name-{s.id}" bind:value={name} autocomplete="off" maxlength="200" size="30" />
									{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
								</div>
								<label class="check"><input type="checkbox" bind:checked={hidden} /> {t('subprocessHidden')}</label>
								<button type="submit">{t('masterSave')}</button>
								<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
							</form>
						</td>
					{:else}
						<td>{s.name}</td>
						<td>{pointSummary(s, types)}</td>
						<td>{s.hidden ? t('processTemplateYes') : ''}</td>
						<td class="actions">
							<button
								type="button"
								aria-label={t('subprocessShowPointsOf', { name: s.name })}
								aria-pressed={s.id === selectedId}
								onclick={() => onselect(s.id)}>{t('subprocessShowPoints')}</button
							>
							{#if canEdit}
								<button type="button" onclick={() => startEdit(s)}>{t('processTemplateEdit')}</button>
								<button
									type="button"
									disabled={i === 0}
									aria-label={t('subprocessMoveUp', { name: s.name })}
									onclick={() => move(s, -1)}>↑</button
								>
								<button
									type="button"
									disabled={i === version.subprocesses.length - 1}
									aria-label={t('subprocessMoveDown', { name: s.name })}
									onclick={() => move(s, 1)}>↓</button
								>
							{/if}
						</td>
					{/if}
				</tr>
			{/each}
		</tbody>
	</table>
{/if}

<style>
	.edit {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-end;
	}

	.actions {
		white-space: nowrap;
	}
</style>
