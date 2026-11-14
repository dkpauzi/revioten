<script lang="ts">
	// Master dokumen terkait (D-130, D-156, R-DOC-11): WI, SOP, QEP, dan
	// dokumen sejenis dicatat sekali, lalu dijadikan dokumen bawaan proses
	// dengan cakupan umum, customer, group model, atau model. Saat dokumen
	// dirakit, kolom Relevant Document di CP terisi dari daftar ini. Semua akun
	// bisa melihat (L-2); mengubah hanya untuk pemegang E-6 (ASUMSI E-96).
	import { onMount } from 'svelte';
	import {
		addReferencedDocumentLink,
		createReferencedDocument,
		listCustomers,
		listProcesses,
		listReferencedDocuments,
		removeReferencedDocumentLink,
		updateReferencedDocument,
		type ApiFailure
	} from '#lib/api/client.ts';
	import type { Customer, Process, ReferencedDocument } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import ScopePicker from '#lib/processTemplates/ScopePicker.svelte';
	import { scopeLevels, scopeText, type ScopeLevel } from '#lib/processTemplates.ts';
	import { can } from '#lib/session.svelte.ts';

	let docs = $state<ReferencedDocument[] | null>(null);
	let processes = $state<Process[]>([]);
	let customers = $state<Customer[]>([]);
	let loadError = $state('');
	let canEdit = $derived(can('E-6'));

	let message = $state('');
	let failure = $state('');
	let fieldErrors = $state<Record<string, string>>({});

	// Formulir tambah.
	let draft = $state({ number: '', title: '', level: '', revision: '', revisionDate: '' });
	// Dokumen yang sedang diubah, beserta isiannya.
	let editingId = $state<string | null>(null);
	let edit = $state({ title: '', level: '', revision: '', revisionDate: '', active: true });
	// Dokumen yang tautan prosesnya sedang dibuka, beserta isian tautan baru.
	let linksFor = $state<string | null>(null);
	let link = $state<{ processId: string; level: ScopeLevel; targetId: string }>({ processId: '', level: 'general', targetId: '' });

	onMount(async () => {
		const [processResult, customerResult] = await Promise.all([listProcesses(), listCustomers()]);
		processes = processResult.ok ? processResult.data.items : [];
		customers = customerResult.ok ? customerResult.data.items : [];
		await load();
	});

	async function load() {
		const result = await listReferencedDocuments();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		docs = result.data.items;
		loadError = '';
	}

	function reset() {
		message = '';
		failure = '';
		fieldErrors = {};
	}

	function showErrors(error: ApiFailure) {
		const details = error.body?.details ?? [];
		fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'refDoc')]));
		if (details.length === 0) {
			failure = failureText(error);
		}
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		reset();
		const result = await createReferencedDocument({
			number: draft.number,
			title: draft.title,
			level: draft.level || undefined,
			revision: draft.revision || undefined,
			revision_date: draft.revisionDate || undefined
		});
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		message = t('refDocSaved', { number: result.data.number });
		draft = { number: '', title: '', level: '', revision: '', revisionDate: '' };
		await load();
	}

	function startEdit(d: ReferencedDocument) {
		reset();
		linksFor = null;
		editingId = d.id;
		edit = { title: d.title, level: d.level ?? '', revision: d.revision ?? '', revisionDate: d.revision_date ?? '', active: d.active };
	}

	async function save(event: SubmitEvent, d: ReferencedDocument) {
		event.preventDefault();
		reset();
		const result = await updateReferencedDocument(d.id, {
			title: edit.title,
			level: edit.level,
			revision: edit.revision,
			revision_date: edit.revisionDate,
			active: edit.active
		});
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		editingId = null;
		message = t('refDocSaved', { number: d.number });
		await load();
	}

	function openLinks(d: ReferencedDocument) {
		reset();
		editingId = null;
		linksFor = linksFor === d.id ? null : d.id;
		link = { processId: '', level: 'general', targetId: '' };
	}

	async function addLink(event: SubmitEvent, d: ReferencedDocument) {
		event.preventDefault();
		reset();
		const result = await addReferencedDocumentLink(d.id, {
			process_id: link.processId,
			scope_level: link.level,
			scope_target_id: link.level === 'general' ? undefined : link.targetId
		});
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		message = t('refDocLinked', { number: d.number });
		link = { ...link, processId: '' };
		await load();
	}

	async function removeLink(d: ReferencedDocument, linkId: string, processName: string) {
		reset();
		const result = await removeReferencedDocumentLink(d.id, linkId);
		if (!result.ok) {
			showErrors(result.error);
			return;
		}
		message = t('refDocUnlinked', { number: d.number, process: processName });
		await load();
	}
</script>

<h1>{t('refDocTitle')}</h1>
<p>{t('refDocIntro')}</p>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if docs === null}
	<p>{t('formatLoading')}</p>
{:else}
	{#if !canEdit}<p class="hint">{t('refDocReadOnly')}</p>{/if}
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if canEdit}
		<section>
			<h2>{t('refDocAddTitle')}</h2>
			<form class="row" onsubmit={create} novalidate>
				<div class="field">
					<label for="ref-number">{t('refDocNumber')}</label>
					<input id="ref-number" bind:value={draft.number} maxlength="40" size="16" autocomplete="off" />
					{#if fieldErrors.number}<p class="error">{fieldErrors.number}</p>{/if}
				</div>
				<div class="field">
					<label for="ref-title">{t('refDocDocTitle')}</label>
					<input id="ref-title" bind:value={draft.title} maxlength="200" size="32" autocomplete="off" />
					{#if fieldErrors.title}<p class="error">{fieldErrors.title}</p>{/if}
				</div>
				<div class="field">
					<label for="ref-level">{t('refDocLevel')}</label>
					<input id="ref-level" bind:value={draft.level} maxlength="60" size="16" autocomplete="off" />
				</div>
				<div class="field">
					<label for="ref-revision">{t('refDocRevision')}</label>
					<input id="ref-revision" bind:value={draft.revision} maxlength="10" size="5" autocomplete="off" />
					{#if fieldErrors.revision}<p class="error">{fieldErrors.revision}</p>{/if}
				</div>
				<div class="field">
					<label for="ref-date">{t('refDocRevisionDate')}</label>
					<input id="ref-date" type="date" bind:value={draft.revisionDate} />
					{#if fieldErrors.revision_date}<p class="error">{fieldErrors.revision_date}</p>{/if}
				</div>
				<button type="submit">{t('refDocAdd')}</button>
			</form>
		</section>
	{/if}

	<section>
		<h2>{t('refDocListTitle')}</h2>
		{#if docs.length === 0}
			<p>{t('refDocNone')}</p>
		{:else}
			<table>
				<thead>
					<tr>
						<th>{t('refDocNumber')}</th>
						<th>{t('refDocDocTitle')}</th>
						<th>{t('refDocLevel')}</th>
						<th>{t('refDocRevision')}</th>
						<th>{t('refDocDefaultFor')}</th>
						<th>{t('masterActions')}</th>
					</tr>
				</thead>
				<tbody>
					{#each docs as d (d.id)}
						<tr class:inactive={!d.active}>
							<td class="mono">{d.number}{#if !d.active}<small>{t('refDocInactive')}</small>{/if}</td>
							{#if editingId === d.id}
								<td colspan="4">
									<form class="row" onsubmit={(event) => save(event, d)} novalidate>
										<div class="field">
											<label for="edit-title-{d.id}">{t('refDocDocTitle')}</label>
											<input id="edit-title-{d.id}" bind:value={edit.title} maxlength="200" size="28" />
											{#if fieldErrors.title}<p class="error">{fieldErrors.title}</p>{/if}
										</div>
										<div class="field">
											<label for="edit-level-{d.id}">{t('refDocLevel')}</label>
											<input id="edit-level-{d.id}" bind:value={edit.level} maxlength="60" size="14" />
										</div>
										<div class="field">
											<label for="edit-revision-{d.id}">{t('refDocRevision')}</label>
											<input id="edit-revision-{d.id}" bind:value={edit.revision} maxlength="10" size="5" />
										</div>
										<div class="field">
											<label for="edit-date-{d.id}">{t('refDocRevisionDate')}</label>
											<input id="edit-date-{d.id}" type="date" bind:value={edit.revisionDate} />
										</div>
										<label class="check"><input type="checkbox" bind:checked={edit.active} /> {t('refDocActive')}</label>
										<button type="submit">{t('masterSave')}</button>
										<button type="button" onclick={() => (editingId = null)}>{t('masterCancel')}</button>
									</form>
								</td>
							{:else}
								<td>{d.title}</td>
								<td>{d.level ?? ''}</td>
								<td>{d.revision ?? ''}{d.revision_date ? ` · ${d.revision_date}` : ''}</td>
								<td>
									{#each d.links as l (l.id)}
										<span class="link">{l.process.name} · {scopeText(l.scope)}</span>
									{/each}
								</td>
							{/if}
							<td class="actions">
								<button type="button" aria-label={t('refDocLinksFor', { number: d.number })} onclick={() => openLinks(d)}>{t('refDocLinks')}</button>
								{#if canEdit && editingId !== d.id}
									<button type="button" aria-label={t('refDocEditFor', { number: d.number })} onclick={() => startEdit(d)}>{t('processTemplateEdit')}</button>
								{/if}
							</td>
						</tr>
						{#if linksFor === d.id}
							<tr class="links">
								<td></td>
								<td colspan="5">
									<h3>{t('refDocLinksTitle', { number: d.number })}</h3>
									{#if d.links.length === 0}<p class="hint">{t('refDocNoLinks')}</p>{/if}
									<ul>
										{#each d.links as l (l.id)}
											<li>
												{l.process.name} · {scopeText(l.scope)}
												{#if canEdit}
													<button
														type="button"
														aria-label={t('refDocUnlinkFor', { process: l.process.name, scope: scopeText(l.scope) })}
														onclick={() => removeLink(d, l.id, l.process.name)}>{t('refDocUnlink')}</button
													>
												{/if}
											</li>
										{/each}
									</ul>
									{#if canEdit}
										<form class="row" onsubmit={(event) => addLink(event, d)} novalidate>
											<div class="field">
												<label for="link-process-{d.id}">{t('refDocProcess')}</label>
												<select id="link-process-{d.id}" bind:value={link.processId}>
													<option value="">{t('refDocProcessChoose')}</option>
													{#each processes as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
												</select>
												{#if fieldErrors.process_id}<p class="error">{fieldErrors.process_id}</p>{/if}
											</div>
											<ScopePicker id="link-scope-{d.id}" {customers} levels={scopeLevels} bind:level={link.level} bind:targetId={link.targetId} errors={fieldErrors} />
											<button type="submit">{t('refDocLink')}</button>
										</form>
									{/if}
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		{/if}
	</section>
{/if}

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	.row > button,
	.row > .check {
		margin-top: var(--label-space);
	}

	.check {
		display: inline-flex;
		gap: 8px;
		align-items: center;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
		white-space: nowrap;
	}

	.mono small {
		display: block;
		color: var(--text-3);
		font-family: inherit;
	}

	tr.inactive td {
		color: var(--text-3);
	}

	.link {
		display: block;
	}

	.actions {
		white-space: nowrap;
	}

	tr.links td {
		background: var(--surface-2);
	}

	tr.links ul {
		list-style: none;
		margin: 0 0 12px;
		padding: 0;
	}

	tr.links li {
		display: flex;
		gap: 12px;
		align-items: center;
		padding: 2px 0;
	}
</style>
