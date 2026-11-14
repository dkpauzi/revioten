<script lang="ts">
	// Bilah bagian per PCB (D-155 pola 2, referensi D-159). Satu tab per
	// bagian, misalnya PCB A, Heatsink Assy (R/L), Base Assy (R/L), seperti
	// satu lembar per bagian di CP kertas. Bagian berlaku untuk seluruh set
	// (PFD, PFMEA, CP). Server tetap memeriksa setiap perubahan.
	import { addDocumentSection, removeDocumentSection, updateDocumentSection, type ApiFailure } from '#lib/api/client.ts';
	import type { Document, DocumentSection } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		doc,
		current,
		canEdit,
		onselect,
		onchanged
	}: {
		doc: Document;
		current: DocumentSection | null;
		canEdit: boolean;
		// onselect menerima kunci bagian yang dipilih.
		onselect: (key: string) => void;
		// onchanged menerima dokumen sesudah perubahan, pesan, dan kunci bagian yang dipilih sesudahnya.
		onchanged: (doc: Document, message: string, key: string) => void;
	} = $props();

	// Formulir yang sedang terbuka: tambah bagian, atau ubah bagian ini.
	let mode = $state<'add' | 'edit' | null>(null);
	let name = $state('');
	let parts = $state<string[]>([]);
	let fieldErrors = $state<Record<string, string>>({});
	let failure = $state('');

	// Part number yang sudah milik bagian lain tidak ditawarkan lagi.
	let taken = $derived(
		new Set(doc.sections.filter((s) => mode !== 'edit' || s.id !== current?.id).flatMap((s) => s.part_numbers.map((p) => p.id)))
	);

	function open(next: 'add' | 'edit') {
		mode = next;
		fieldErrors = {};
		failure = '';
		name = next === 'edit' && current ? current.name : '';
		parts = next === 'edit' && current ? current.part_numbers.map((p) => p.id) : [];
	}

	function showErrors(error: ApiFailure) {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
		failure = details.length === 0 ? failureText(error) : '';
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		const result =
			mode === 'edit' && current
				? await updateDocumentSection(doc.id, current.id, { name, part_number_ids: parts })
				: await addDocumentSection(doc.id, { name, part_number_ids: parts });
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		const saved = result.data.sections.find((s) => s.name === name.trim()) ?? result.data.sections.at(-1);
		mode = null;
		onchanged(result.data, t('sectionSaved', { name: name.trim() }), saved?.key ?? '');
	}

	async function move(delta: number) {
		if (!current) {
			return;
		}
		const result = await updateDocumentSection(doc.id, current.id, { name: current.name, position: current.position + delta });
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		onchanged(result.data, t('sectionSaved', { name: current.name }), current.key);
	}

	async function remove() {
		if (!current || !confirm(t('sectionRemoveConfirm', { name: current.name }))) {
			return;
		}
		const result = await removeDocumentSection(doc.id, current.id);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		onchanged(result.data, t('sectionRemoved', { name: current.name }), result.data.sections[0]?.key ?? '');
	}
</script>

{#if doc.sections.length > 0}
	<nav class="sections" aria-label={t('sectionTabs')}>
		{#each doc.sections as s (s.id)}
			<button type="button" class="tab" aria-pressed={s.id === current?.id} onclick={() => onselect(s.key)}>
				<span class="name">{s.name}</span>
				{#if s.part_numbers.length > 0}<span class="parts">{s.part_numbers.map((p) => p.number).join(', ')}</span>{/if}
			</button>
		{/each}
	</nav>
{/if}

{#if canEdit}
	<p class="actions">
		<button type="button" onclick={() => open('add')}>{doc.sections.length === 0 ? t('sectionSplit') : t('sectionAdd')}</button>
		{#if current}
			<button type="button" onclick={() => open('edit')}>{t('sectionEdit')}</button>
			<button type="button" disabled={current.position === 1} aria-label={t('sectionMoveLeft', { name: current.name })} onclick={() => move(-1)}>←</button>
			<button type="button" disabled={current.position === doc.sections.length} aria-label={t('sectionMoveRight', { name: current.name })} onclick={() => move(1)}>→</button>
			<button type="button" onclick={remove}>{t('sectionRemove')}</button>
		{/if}
	</p>
	{#if doc.sections.length === 0 && mode === null}<p class="hint">{t('sectionHint')}</p>{/if}
{/if}
{#if failure || fieldErrors.section}<p class="error" role="alert">{failure || fieldErrors.section}</p>{/if}

{#if mode}
	<form class="section-form" onsubmit={save} novalidate>
		<div class="field">
			<label for="section-name">{t('sectionName')}</label>
			<input id="section-name" bind:value={name} maxlength="200" size="30" autocomplete="off" />
			{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
		</div>
		<fieldset>
			<legend>{t('sectionParts')}</legend>
			{#if doc.part_numbers.length === 0}
				<p class="hint">{t('sectionNoParts')}</p>
			{/if}
			{#each doc.part_numbers as p (p.id)}
				<label class="check">
					<input type="checkbox" value={p.id} bind:group={parts} disabled={taken.has(p.id)} />
					{p.number}{p.name ? ` · ${p.name}` : ''}
				</label>
			{/each}
			{#if fieldErrors.part_number_ids}<p class="error">{fieldErrors.part_number_ids}</p>{/if}
		</fieldset>
		<p class="actions">
			<button type="submit" class="primary">{t('masterSave')}</button>
			<button type="button" onclick={() => (mode = null)}>{t('masterCancel')}</button>
		</p>
	</form>
{/if}

<style>
	.sections {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-bottom: 8px;
	}

	.tab {
		display: inline-flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 2px;
		min-height: 44px;
		padding: 6px 12px;
		text-align: left;
	}

	.tab[aria-pressed='true'] {
		border-color: var(--accent);
		background: var(--accent-weak, var(--surface-2));
		color: var(--accent-ink);
		font-weight: 600;
	}

	.parts {
		color: var(--text-3);
		font-size: 12px;
		font-family: 'Geist Mono Variable', monospace;
		font-weight: 400;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.section-form {
		max-width: 40rem;
		margin-bottom: 1rem;
	}

	fieldset {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 8px 12px;
	}

	.check {
		display: flex;
		gap: 8px;
		align-items: center;
		padding: 2px 0;
	}
</style>
