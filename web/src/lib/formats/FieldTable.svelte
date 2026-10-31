<script lang="ts">
	// Tabel field satu area format template (M3.1): field header atau kolom
	// baris, dengan ubah label, wajib, tersembunyi, pilihan, dan urutan.
	// Field tidak pernah dihapus, hanya disembunyikan (R-TPL-02). Server tetap
	// memeriksa setiap perubahan.
	import { updateFormatField } from '#lib/api/client.ts';
	import type { FormatField, FormatFieldUpdate, FormatVersion } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { fieldsOf, kindText, mergeOptions, optionLines } from '#lib/formats.ts';
	import { t } from '#lib/i18n/index.ts';

	// onsaved menerima versi sesudah perubahan dan pesan untuk layar.
	let {
		version,
		area,
		canEdit,
		onsaved
	}: {
		version: FormatVersion;
		area: FormatField['area'];
		canEdit: boolean;
		onsaved: (v: FormatVersion, message: string) => void;
	} = $props();

	let fields = $derived(fieldsOf(version, area));

	// Field yang sedang diubah, beserta isiannya.
	let editingId = $state<string | null>(null);
	let label = $state('');
	let required = $state(false);
	let hidden = $state(false);
	let optionsText = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let failure = $state('');

	function startEdit(f: FormatField) {
		editingId = f.id;
		label = f.label;
		required = f.required;
		hidden = f.hidden;
		optionsText = optionLines(f.options);
		fieldErrors = {};
		failure = '';
	}

	// send mengirim perubahan satu field, lalu menyerahkan versi barunya.
	async function send(f: FormatField, input: FormatFieldUpdate) {
		fieldErrors = {};
		failure = '';
		const result = await updateFormatField(version.id, f.id, input);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
			if (details.length === 0 || editingId === null) {
				failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			}
			return;
		}
		editingId = null;
		onsaved(result.data, t('formatFieldSaved', { label: input.label }));
	}

	function save(event: SubmitEvent, f: FormatField) {
		event.preventDefault();
		const input: FormatFieldUpdate = { label, required, hidden };
		if (f.kind === 'select') {
			input.options = mergeOptions(f.options ?? [], optionsText);
		}
		send(f, input);
	}

	// move memindahkan field satu tempat; isian lain tidak dikirim, jadi tetap (D-111).
	function move(f: FormatField, delta: number) {
		send(f, { label: f.label, position: f.position + delta });
	}
</script>

{#if failure}<p class="error" role="alert">{failure}</p>{/if}
{#if fields.length === 0}
	<p>{t('formatNoFields')}</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>{t('formatColumnNo')}</th>
				<th>{t('formatColumnLabel')}</th>
				<th>{t('formatColumnKind')}</th>
				<th>{t('formatColumnRequired')}</th>
				<th>{t('formatColumnHidden')}</th>
				<th>{t('formatColumnOptions')}</th>
				{#if canEdit}<th>{t('masterActions')}</th>{/if}
			</tr>
		</thead>
		<tbody>
			{#each fields as f, i (f.id)}
				<tr class:hidden={f.hidden}>
					<td>{f.position}</td>
					{#if editingId === f.id}
						<td colspan={canEdit ? 6 : 5}>
							<form class="edit" onsubmit={(event) => save(event, f)} novalidate>
								<div class="field">
									<label for="field-label-{f.id}">{t('formatColumnLabel')}</label>
									<input id="field-label-{f.id}" bind:value={label} autocomplete="off" maxlength="100" />
									{#if fieldErrors.label}<p class="error">{fieldErrors.label}</p>{/if}
								</div>
								<label class="check"><input type="checkbox" bind:checked={required} /> {t('formatRequired')}</label>
								<label class="check"><input type="checkbox" bind:checked={hidden} /> {t('formatHidden')}</label>
								{#if fieldErrors.required}<p class="error">{fieldErrors.required}</p>{/if}
								{#if f.kind === 'select'}
									<div class="field">
										<label for="field-options-{f.id}">{t('formatOptions')}</label>
										<textarea id="field-options-{f.id}" bind:value={optionsText} rows="5"></textarea>
										<p class="hint">{t('formatOptionsHint')}</p>
										{#if fieldErrors.options}<p class="error">{fieldErrors.options}</p>{/if}
									</div>
								{/if}
								<button type="submit">{t('masterSave')}</button>
								<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
							</form>
						</td>
					{:else}
						<td>{f.label}</td>
						<td>{kindText(f)}</td>
						<td>{f.required ? t('formatYes') : ''}</td>
						<td>{f.hidden ? t('formatYes') : ''}</td>
						<td>{(f.options ?? []).map((o) => o.label).join(', ')}</td>
						{#if canEdit}
							<td class="actions">
								<button type="button" onclick={() => startEdit(f)}>{t('formatEdit')}</button>
								<button
									type="button"
									disabled={i === 0}
									aria-label={t('formatMoveUp', { label: f.label })}
									onclick={() => move(f, -1)}>↑</button
								>
								<button
									type="button"
									disabled={i === fields.length - 1}
									aria-label={t('formatMoveDown', { label: f.label })}
									onclick={() => move(f, 1)}>↓</button
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
	table {
		border-collapse: collapse;
		margin-bottom: 1rem;
	}

	th,
	td {
		border-bottom: 1px solid #ddd;
		padding: 0.25rem 0.5rem;
		text-align: left;
		vertical-align: top;
	}

	tr.hidden td {
		color: #777;
	}

	.edit {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-end;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.actions {
		white-space: nowrap;
	}

	.error {
		color: #b00020;
		margin: 0;
	}

	.hint {
		color: #555;
		margin: 0;
	}
</style>
