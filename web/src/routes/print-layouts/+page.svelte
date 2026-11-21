<script lang="ts">
	// Layout cetak (M6.1, R-PRT-02): layout adalah data. PPAP Engineer (E-11) mengubah
	// kertas, arah, huruf, margin, cover, tanda revisi, dan lebar kolom per key
	// field, menambah layout, dan memilih layout lain untuk satu customer.
	import { onMount } from 'svelte';
	import {
		createPrintLayout,
		listCustomerPrintLayouts,
		listCustomers,
		listDocumentTypes,
		listPrintLayouts,
		setCustomerPrintLayout,
		updatePrintLayout,
		type ApiFailure
	} from '#lib/api/client.ts';
	import type { Customer, DocumentType, PrintLayout } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let layouts = $state<PrintLayout[]>([]);
	let types = $state<DocumentType[]>([]);
	let customers = $state<Customer[]>([]);
	let customerId = $state('');
	let choices = $state<Record<string, string>>({});
	let editing = $state<string | null>(null);
	let name = $state('');
	let content = $state('');
	let newType = $state('');
	let message = $state('');
	let failure = $state('');
	let canEdit = $derived(can('E-11'));

	onMount(async () => {
		const [l, ty, c] = await Promise.all([listPrintLayouts(), listDocumentTypes(), listCustomers()]);
		if (!l.ok) {
			failure = failureText(l.error);
			return;
		}
		layouts = l.data.items;
		types = ty.ok ? ty.data.items.filter((d) => d.key === 'pfd' || d.key === 'pfmea' || d.key === 'cp') : [];
		customers = c.ok ? c.data.items.filter((x) => x.active) : [];
	});

	function showError(error: ApiFailure) {
		const details = error.body?.details ?? [];
		failure = details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	function edit(l: PrintLayout | null) {
		message = '';
		failure = '';
		editing = l?.id ?? 'new';
		name = l?.name ?? '';
		newType = types[0]?.id ?? '';
		content = JSON.stringify(
			l?.layout ?? { paper: 'A3', orientation: 'landscape', font_size: 8, margins_mm: { top: 10, right: 8, bottom: 12, left: 8 }, cover: true, revision_marks: true, widths: {} },
			null,
			2
		);
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		failure = '';
		let layout: Record<string, unknown>;
		try {
			layout = JSON.parse(content);
		} catch {
			failure = t('layoutInvalidJson');
			return;
		}
		const result =
			editing === 'new'
				? await createPrintLayout({ name: name.trim(), document_type_id: newType, layout })
				: await updatePrintLayout(editing ?? '', { name: name.trim(), layout });
		if (!result.ok) {
			showError(result.error);
			return;
		}
		const list = await listPrintLayouts();
		layouts = list.ok ? list.data.items : layouts;
		editing = null;
		message = t('layoutSaved', { name: result.data.name });
	}

	async function chooseCustomer() {
		choices = {};
		if (!customerId) return;
		const result = await listCustomerPrintLayouts(customerId);
		if (result.ok) {
			choices = Object.fromEntries(result.data.items.map((x) => [x.document_type_id, x.layout_id]));
		}
	}

	async function assign(typeId: string, layoutId: string) {
		message = '';
		failure = '';
		const result = await setCustomerPrintLayout(customerId, typeId, layoutId || undefined);
		if (!result.ok) {
			showError(result.error);
			return;
		}
		choices = Object.fromEntries(result.data.items.map((x) => [x.document_type_id, x.layout_id]));
		message = t('layoutAssigned');
	}

	function typeName(id: string): string {
		return types.find((x) => x.id === id)?.name ?? '';
	}
</script>

<h1>{t('layoutsTitle')}</h1>
<p class="hint">{t('layoutsHint')}</p>
{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

<table>
	<thead>
		<tr>
			<th>{t('layoutName')}</th>
			<th>{t('numberColumnType')}</th>
			<th>{t('layoutPaper')}</th>
			<th>{t('layoutFont')}</th>
			{#if canEdit}<th>{t('masterActions')}</th>{/if}
		</tr>
	</thead>
	<tbody>
		{#each layouts as l (l.id)}
			<tr>
				<td>{l.name}{l.is_default ? ` · ${t('layoutDefault')}` : ''}</td>
				<td>{typeName(l.document_type_id)}</td>
				<td>{l.layout.paper} {l.layout.orientation}</td>
				<td>{l.layout.font_size} pt</td>
				{#if canEdit}<td><button type="button" aria-label={t('layoutEditFor', { name: l.name })} onclick={() => edit(l)}>{t('layoutEdit')}</button></td>{/if}
			</tr>
		{/each}
	</tbody>
</table>
{#if canEdit && editing === null}
	<p><button type="button" onclick={() => edit(null)}>{t('layoutAdd')}</button></p>
{/if}

{#if editing !== null}
	<section>
		<h2>{editing === 'new' ? t('layoutAdd') : name}</h2>
		<form onsubmit={save} novalidate>
			<div class="field">
				<label for="layout-name">{t('layoutName')}</label>
				<input id="layout-name" maxlength="100" bind:value={name} />
			</div>
			{#if editing === 'new'}
				<div class="field">
					<label for="layout-type">{t('numberColumnType')}</label>
					<select id="layout-type" bind:value={newType}>
						{#each types as ty (ty.id)}<option value={ty.id}>{ty.name}</option>{/each}
					</select>
				</div>
			{/if}
			<div class="field">
				<label for="layout-content">{t('layoutContent')}</label>
				<textarea id="layout-content" rows="14" class="mono" bind:value={content}></textarea>
				<p class="hint">{t('layoutContentHint')}</p>
			</div>
			<div class="row">
				<button type="submit" class="primary">{t('layoutSave')}</button>
				<button type="button" onclick={() => (editing = null)}>{t('masterCancel')}</button>
			</div>
		</form>
	</section>
{/if}

<section>
	<h2>{t('layoutCustomerTitle')}</h2>
	<div class="field">
		<label for="layout-customer">{t('documentCustomer')}</label>
		<select id="layout-customer" bind:value={customerId} onchange={chooseCustomer}>
			<option value="">{t('changeChoose')}</option>
			{#each customers as c (c.id)}<option value={c.id}>{c.code} · {c.name}</option>{/each}
		</select>
	</div>
	{#if customerId}
		<table>
			<thead>
				<tr><th>{t('numberColumnType')}</th><th>{t('layoutName')}</th></tr>
			</thead>
			<tbody>
				{#each types as ty (ty.id)}
					<tr>
						<td>{ty.name}</td>
						<td>
							<select aria-label={t('layoutForType', { type: ty.name })} value={choices[ty.id] ?? ''} disabled={!canEdit} onchange={(e) => assign(ty.id, e.currentTarget.value)}>
								<option value="">{t('layoutUseDefault')}</option>
								{#each layouts.filter((l) => l.document_type_id === ty.id && !l.is_default) as l (l.id)}<option value={l.id}>{l.name}</option>{/each}
							</select>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</section>

<style>
	.row {
		display: flex;
		gap: 0.5rem;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	textarea {
		width: 100%;
	}
</style>
