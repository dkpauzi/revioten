<script lang="ts">
	// Tabel poin satu sub-proses untuk satu jenis dokumen (M3.2, R-TPL-04):
	// ganti teks, sembunyikan, dan ubah urutan. Poin tidak pernah dihapus,
	// hanya disembunyikan; ID-nya tetap, karena baris dokumen nanti menunjuk
	// ID itu. Server tetap memeriksa setiap perubahan.
	//
	// Sejak M3.3 (D-129) setiap poin menampilkan cakupannya, dan variannya
	// tampil tepat di bawahnya. Varian mengganti teks poin untuk cakupan yang
	// lebih sempit, misalnya laser type untuk satu model.
	import { addPointVariant, updatePointVariant, updateTemplatePoint, type ApiFailure } from '#lib/api/client.ts';
	import type {
		Customer,
		PointVariant,
		ProcessTemplateVersion,
		TemplatePoint,
		TemplatePointUpdate
	} from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import ScopePicker from '#lib/processTemplates/ScopePicker.svelte';
	import { scopeText, variantLevels, type ScopeLevel } from '#lib/processTemplates.ts';

	let {
		version,
		points,
		customers,
		canEdit,
		onsaved
	}: {
		version: ProcessTemplateVersion;
		points: TemplatePoint[];
		customers: Customer[];
		canEdit: boolean;
		onsaved: (v: ProcessTemplateVersion, message: string) => void;
	} = $props();

	// Poin atau varian yang sedang diubah, beserta isiannya.
	let editingId = $state<string | null>(null);
	let text = $state('');
	let hidden = $state(false);
	let fieldErrors = $state<Record<string, string>>({});
	let failure = $state('');

	// Poin yang sedang diberi varian baru, beserta isiannya.
	let variantFor = $state<string | null>(null);
	let variantLevel = $state<ScopeLevel>('customer');
	let variantTarget = $state('');
	let variantText = $state('');

	let columns = $derived(canEdit ? 5 : 4);

	function clearErrors() {
		fieldErrors = {};
		failure = '';
	}

	function startEdit(id: string, currentText: string, currentHidden: boolean) {
		editingId = id;
		variantFor = null;
		text = currentText;
		hidden = currentHidden;
		clearErrors();
	}

	function startVariant(p: TemplatePoint) {
		variantFor = p.id;
		editingId = null;
		variantLevel = variantLevels(p.scope.level)[0];
		variantTarget = '';
		variantText = p.text;
		clearErrors();
	}

	// showErrors menampilkan masalah per field, atau pesan umum bila tidak ada.
	function showErrors(error: ApiFailure, formOpen: boolean) {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'processTemplate')]));
		if (details.length === 0 || !formOpen) {
			failure = details.length > 0 ? problemText(details[0], 'processTemplate') : failureText(error);
		}
	}

	// send mengirim perubahan satu poin, lalu menyerahkan versi barunya.
	async function send(p: TemplatePoint, input: TemplatePointUpdate) {
		clearErrors();
		const result = await updateTemplatePoint(version.id, p.id, input);
		if (!result.ok) {
			showErrors(result.error, editingId !== null);
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

	async function saveVariant(event: SubmitEvent, v: PointVariant) {
		event.preventDefault();
		clearErrors();
		const result = await updatePointVariant(version.id, v.id, { text, hidden });
		if (!result.ok) {
			showErrors(result.error, true);
			return;
		}
		editingId = null;
		onsaved(result.data, t('variantSaved'));
	}

	async function addVariant(event: SubmitEvent, p: TemplatePoint) {
		event.preventDefault();
		clearErrors();
		const result = await addPointVariant(version.id, p.id, {
			scope_level: variantLevel,
			scope_target_id: variantTarget,
			text: variantText
		});
		if (!result.ok) {
			showErrors(result.error, true);
			return;
		}
		variantFor = null;
		onsaved(result.data, t('variantAdded'));
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
				<th>{t('pointColumnScope')}</th>
				<th>{t('pointColumnHidden')}</th>
				{#if canEdit}<th>{t('masterActions')}</th>{/if}
			</tr>
		</thead>
		<tbody>
			{#each points as p, i (p.id)}
				<tr class:hidden={p.hidden}>
					<td>{p.position}</td>
					{#if editingId === p.id}
						<td colspan={columns - 1}>
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
						<td>{scopeText(p.scope)}</td>
						<td>{p.hidden ? t('processTemplateYes') : ''}</td>
						{#if canEdit}
							<td class="actions">
								<button type="button" onclick={() => startEdit(p.id, p.text, p.hidden)}>{t('processTemplateEdit')}</button>
								{#if variantLevels(p.scope.level).length > 0}
									<button type="button" onclick={() => startVariant(p)}>{t('variantAdd')}</button>
								{/if}
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

				<!-- Varian poin ini, dari cakupan terluas ke tersempit (M3.3). -->
				{#each p.variants as v (v.id)}
					<tr class="variant" class:hidden={v.hidden}>
						<td></td>
						{#if editingId === v.id}
							<td colspan={columns - 1}>
								<form class="edit" onsubmit={(event) => saveVariant(event, v)} novalidate>
									<div class="field">
										<label for="variant-text-{v.id}">{t('variantText')}</label>
										<input id="variant-text-{v.id}" bind:value={text} autocomplete="off" maxlength="500" size="50" />
										{#if fieldErrors.text}<p class="error">{fieldErrors.text}</p>{/if}
									</div>
									<label class="check"><input type="checkbox" bind:checked={hidden} /> {t('pointHidden')}</label>
									<button type="submit">{t('masterSave')}</button>
									<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
								</form>
							</td>
						{:else}
							<td><span class="arrow" aria-hidden="true">↳</span> {v.text}</td>
							<td>{t('variantFor', { scope: scopeText(v.scope) })}</td>
							<td>{v.hidden ? t('processTemplateYes') : ''}</td>
							{#if canEdit}
								<td class="actions">
									<button
										type="button"
										aria-label={t('variantEdit', { scope: scopeText(v.scope) })}
										onclick={() => startEdit(v.id, v.text, v.hidden)}>{t('processTemplateEdit')}</button
									>
								</td>
							{/if}
						{/if}
					</tr>
				{/each}

				{#if variantFor === p.id}
					<tr class="variant">
						<td></td>
						<td colspan={columns - 1}>
							<form class="edit" onsubmit={(event) => addVariant(event, p)} novalidate>
								<p class="hint">{t('variantHint')}</p>
								<ScopePicker
									id="variant-{p.id}"
									{customers}
									levels={variantLevels(p.scope.level)}
									bind:level={variantLevel}
									bind:targetId={variantTarget}
									errors={fieldErrors}
								/>
								<div class="field">
									<label for="variant-new-text-{p.id}">{t('variantText')}</label>
									<input id="variant-new-text-{p.id}" bind:value={variantText} autocomplete="off" maxlength="500" size="40" />
									{#if fieldErrors.text}<p class="error">{fieldErrors.text}</p>{/if}
								</div>
								<button type="submit">{t('variantAdd')}</button>
								<button type="button" onclick={() => (variantFor = null)}>{t('masterCancel')}</button>
							</form>
						</td>
					</tr>
				{/if}
			{/each}
		</tbody>
	</table>
{/if}

<style>
	.edit {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	/* Field dirapikan ke atas; tombol diturunkan setinggi label supaya sejajar
	   dengan kotak isian, juga bila ada teks bantuan atau pesan masalah. */
	.edit > button {
		margin-top: var(--label-space);
	}

	.edit .hint {
		flex-basis: 100%;
		margin: 0;
	}

	.actions {
		white-space: nowrap;
	}

	.variant td {
		background: var(--surface-2);
	}

	.arrow {
		color: var(--muted);
	}
</style>
