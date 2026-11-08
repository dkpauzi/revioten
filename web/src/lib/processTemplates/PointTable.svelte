<script lang="ts">
	// Tabel poin satu sub-proses untuk satu jenis dokumen (M3.2, R-TPL-04):
	// ganti teks, sembunyikan, dan ubah urutan. Poin tidak pernah dihapus,
	// hanya disembunyikan; ID-nya tetap, karena baris dokumen nanti menunjuk
	// ID itu. Server tetap memeriksa setiap perubahan.
	import { updateTemplatePoint } from '#lib/api/client.ts';
	import type { ProcessTemplateVersion, TemplatePoint, TemplatePointUpdate } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		version,
		points,
		canEdit,
		onsaved
	}: {
		version: ProcessTemplateVersion;
		points: TemplatePoint[];
		canEdit: boolean;
		onsaved: (v: ProcessTemplateVersion, message: string) => void;
	} = $props();

	// Poin yang sedang diubah, beserta isiannya.
	let editingId = $state<string | null>(null);
	let text = $state('');
	let hidden = $state(false);
	let fieldErrors = $state<Record<string, string>>({});
	let failure = $state('');

	function startEdit(p: TemplatePoint) {
		editingId = p.id;
		text = p.text;
		hidden = p.hidden;
		fieldErrors = {};
		failure = '';
	}

	// send mengirim perubahan satu poin, lalu menyerahkan versi barunya.
	async function send(p: TemplatePoint, input: TemplatePointUpdate) {
		fieldErrors = {};
		failure = '';
		const result = await updateTemplatePoint(version.id, p.id, input);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'processTemplate')]));
			if (details.length === 0 || editingId === null) {
				failure = details.length > 0 ? problemText(details[0], 'processTemplate') : failureText(result.error);
			}
			return;
		}
		editingId = null;
		onsaved(result.data, t('pointSaved'));
	}

	function save(event: SubmitEvent, p: TemplatePoint) {
		event.preventDefault();
		send(p, { text, hidden });
	}

	// move memindahkan poin satu tempat; isian lain tidak dikirim, jadi tetap (D-111).
	function move(p: TemplatePoint, delta: number) {
		send(p, { text: p.text, position: p.position + delta });
	}
</script>

{#if failure}<p class="error" role="alert">{failure}</p>{/if}
{#if points.length === 0}
	<p>{t('pointNone')}</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>{t('pointColumnNo')}</th>
				<th>{t('pointColumnText')}</th>
				<th>{t('pointColumnHidden')}</th>
				{#if canEdit}<th>{t('masterActions')}</th>{/if}
			</tr>
		</thead>
		<tbody>
			{#each points as p, i (p.id)}
				<tr class:hidden={p.hidden}>
					<td>{p.position}</td>
					{#if editingId === p.id}
						<td colspan={canEdit ? 3 : 2}>
							<form class="edit" onsubmit={(event) => save(event, p)} novalidate>
								<div class="field">
									<label for="point-text-{p.id}">{t('pointText')}</label>
									<input id="point-text-{p.id}" bind:value={text} autocomplete="off" maxlength="500" size="50" />
									{#if fieldErrors.text}<p class="error">{fieldErrors.text}</p>{/if}
								</div>
								<label class="check"><input type="checkbox" bind:checked={hidden} /> {t('pointHidden')}</label>
								<button type="submit">{t('masterSave')}</button>
								<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
							</form>
						</td>
					{:else}
						<td>{p.text}</td>
						<td>{p.hidden ? t('processTemplateYes') : ''}</td>
						{#if canEdit}
							<td class="actions">
								<button type="button" onclick={() => startEdit(p)}>{t('processTemplateEdit')}</button>
								<button
									type="button"
									disabled={i === 0}
									aria-label={t('pointMoveUp', { text: p.text })}
									onclick={() => move(p, -1)}>↑</button
								>
								<button
									type="button"
									disabled={i === points.length - 1}
									aria-label={t('pointMoveDown', { text: p.text })}
									onclick={() => move(p, 1)}>↓</button
								>
							</td>
						{/if}
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
