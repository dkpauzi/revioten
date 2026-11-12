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
	import { contentInput, scopeText, variantLevels, type PointColumn, type ScopeLevel } from '#lib/processTemplates.ts';

	let {
		version,
		points,
		columns,
		customers,
		canEdit,
		onsaved
	}: {
		version: ProcessTemplateVersion;
		points: TemplatePoint[];
		// columns adalah kolom form yang bisa diisi poin jenis dokumen ini (D-146).
		columns: PointColumn[];
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

	// Poin atau varian yang isi per kolomnya sedang dibuka (D-146).
	let contentFor = $state<string | null>(null);
	let contentValues = $state<Record<string, string>>({});

	// Kolom Actions selalu ada sejak D-146: tombol Columns juga untuk yang hanya melihat.
	const span = 5;

	function clearErrors() {
		fieldErrors = {};
		failure = '';
	}

	// startContent membuka isi per kolom satu poin atau varian.
	function startContent(id: string, content: Record<string, unknown>) {
		contentFor = id;
		editingId = null;
		variantFor = null;
		contentValues = Object.fromEntries(columns.map((c) => [c.key, content[c.key] === undefined ? '' : String(content[c.key])]));
		clearErrors();
	}

	async function saveContent(event: SubmitEvent, p: TemplatePoint) {
		event.preventDefault();
		clearErrors();
		const result = await updateTemplatePoint(version.id, p.id, { text: p.text, content: contentInput(columns, contentValues, p.content) });
		if (!result.ok) {
			showErrors(result.error, true);
			return;
		}
		contentFor = null;
		onsaved(result.data, t('pointContentSaved'));
	}

	async function saveVariantContent(event: SubmitEvent, v: PointVariant) {
		event.preventDefault();
		clearErrors();
		const result = await updatePointVariant(version.id, v.id, { text: v.text, content: contentInput(columns, contentValues, v.content) });
		if (!result.ok) {
			showErrors(result.error, true);
			return;
		}
		contentFor = null;
		onsaved(result.data, t('pointContentSaved'));
	}

	// filled menulis banyaknya kolom yang terisi, atau '' bila belum ada.
	function filled(content: Record<string, unknown>): string {
		const n = Object.keys(content).length;
		return n === 0 ? '' : t('pointContentFilled', { n: String(n) });
	}

	function startEdit(id: string, currentText: string, currentHidden: boolean) {
		editingId = id;
		variantFor = null;
		contentFor = null;
		text = currentText;
		hidden = currentHidden;
		clearErrors();
	}

	function startVariant(p: TemplatePoint) {
		variantFor = p.id;
		editingId = null;
		contentFor = null;
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

{#snippet contentForm(id: string, title: string, onsubmit: (event: SubmitEvent) => void)}
	<tr class="variant">
		<td></td>
		<td colspan={span - 1}>
			<form class="content" {onsubmit} novalidate>
				<p class="hint">{t('pointContentHint')}</p>
				{#if columns.length === 0}
					<p>{t('pointContentNone')}</p>
				{/if}
				<div class="content-grid">
					{#each columns as c (c.key)}
						<div class="field">
							<label for="content-{id}-{c.key}">{c.label}</label>
							{#if c.kind === 'rating'}
								<input id="content-{id}-{c.key}" type="number" min="1" max="10" step="1" bind:value={contentValues[c.key]} disabled={!canEdit} />
							{:else}
								<textarea id="content-{id}-{c.key}" rows="2" maxlength="2000" bind:value={contentValues[c.key]} disabled={!canEdit}></textarea>
							{/if}
							<p class="hint">{t('pointContentForms', { forms: c.forms.join(', ') })}</p>
						</div>
					{/each}
				</div>
				{#if fieldErrors.content}<p class="error">{fieldErrors.content}</p>{/if}
				<p class="buttons">
					{#if canEdit}<button type="submit" aria-label={t('pointContentSave', { text: title })}>{t('masterSave')}</button>{/if}
					<button type="button" onclick={() => (contentFor = null)}>{t('masterCancel')}</button>
				</p>
			</form>
		</td>
	</tr>
{/snippet}

{#if failure}<p class="error" role="alert">{failure}</p>{/if}{#if points.length === 0}
	<p>{t('pointNone')}</p>
{:else}
	<table>
		<thead>
			<tr>
				<th>{t('pointColumnNo')}</th>
				<th>{t('pointColumnText')}</th>
				<th>{t('pointColumnScope')}</th>
				<th>{t('pointColumnHidden')}</th>
				<th>{t('masterActions')}</th>
			</tr>
		</thead>
		<tbody>
			{#each points as p, i (p.id)}
				<tr class:hidden={p.hidden}>
					<td>{p.position}</td>
					{#if editingId === p.id}
						<td colspan={span - 1}>
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
						<td>{p.text}{#if filled(p.content)}<small class="filled">{filled(p.content)}</small>{/if}</td>
						<td>{scopeText(p.scope)}</td>
						<td>{p.hidden ? t('processTemplateYes') : ''}</td>
						<td class="actions">
							<button type="button" aria-label={t('pointContentFor', { text: p.text })} onclick={() => startContent(p.id, p.content)}>{t('pointContent')}</button>
							{#if canEdit}
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
							{/if}
						</td>
					{/if}
				</tr>

				{#if contentFor === p.id}
					{@render contentForm(p.id, p.text, (event) => saveContent(event, p))}
				{/if}

				<!-- Varian poin ini, dari cakupan terluas ke tersempit (M3.3). -->				{#each p.variants as v (v.id)}
					<tr class="variant" class:hidden={v.hidden}>
						<td></td>
						{#if editingId === v.id}
							<td colspan={span - 1}>
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
							<td><span class="arrow" aria-hidden="true">↳</span> {v.text}{#if filled(v.content)}<small class="filled">{filled(v.content)}</small>{/if}</td>
							<td>{t('variantFor', { scope: scopeText(v.scope) })}</td>
							<td>{v.hidden ? t('processTemplateYes') : ''}</td>
							<td class="actions">
								<button type="button" aria-label={t('pointContentFor', { text: v.text })} onclick={() => startContent(v.id, v.content)}>{t('pointContent')}</button>
								{#if canEdit}
									<button
										type="button"
										aria-label={t('variantEdit', { scope: scopeText(v.scope) })}
										onclick={() => startEdit(v.id, v.text, v.hidden)}>{t('processTemplateEdit')}</button
									>
								{/if}
							</td>
						{/if}
					</tr>
					{#if contentFor === v.id}
						{@render contentForm(v.id, v.text, (event) => saveVariantContent(event, v))}
					{/if}
				{/each}

				{#if variantFor === p.id}
					<tr class="variant">
						<td></td>
						<td colspan={span - 1}>
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

	.filled {
		display: block;
		color: var(--text-3);
	}

	.content-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr));
		gap: 0.5rem 1rem;
	}

	.content-grid textarea,
	.content-grid input {
		width: 100%;
	}

	.content .hint {
		margin: 0.25rem 0 0;
	}

	.buttons {
		display: flex;
		gap: 8px;
	}

	.arrow {
		color: var(--muted);
	}
</style>
