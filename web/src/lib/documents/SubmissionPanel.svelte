<script lang="ts">
	// Form Control Submission (D-207): satu daftar per dokumen, satu baris per
	// revisi. Tombol PFD · PFMEA · Control Plan berpindah ke daftar dokumen
	// lain satu set. Kolom tanggal dihitung server; pemegang E-7 mengisi
	// pengiriman ke customer, hasilnya, catatan, dan bukti (opsional).
	import { getSubmissionControl, saveSubmissionEntry, submissionExportUrl, submissionScanUrl, uploadSubmissionScan } from '#lib/api/client.ts';
	import type { Document, SubmissionControl, SubmissionEntry, SubmissionRow } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { can } from '#lib/session.svelte.ts';

	let { doc }: { doc: Document } = $props();

	let currentId = $state('');
	let control = $state<SubmissionControl | null>(null);
	let showAll = $state(true);
	let editing = $state<number | null>(null);
	let draft = $state<{ required: boolean; reason: string; submitted: string; pkg: string; result: '' | 'waiting' | 'approved' | 'rejected'; resultOn: string; remark: string }>({
		required: true,
		reason: '',
		submitted: '',
		pkg: '',
		result: '',
		resultOn: '',
		remark: ''
	});
	let scanInput = $state<HTMLInputElement | null>(null);
	let message = $state('');
	let failure = $state('');
	let busy = $state(false);

	let canEdit = $derived(can('E-7'));
	let rows = $derived((control?.rows ?? []).filter((r) => showAll || r.required));
	let latest = $derived(control && control.rows.length > 0 ? control.rows[control.rows.length - 1] : null);

	$effect(() => {
		void doc.revision;
		void doc.status;
		void doc.signing;
		load(currentId || doc.id);
	});

	async function load(id: string) {
		const result = await getSubmissionControl(id);
		if (!result.ok) {
			failure = failureText(result.error);
			return;
		}
		currentId = id;
		control = result.data;
	}

	function pick(id: string) {
		editing = null;
		message = '';
		failure = '';
		load(id);
	}

	function edit(r: SubmissionRow) {
		editing = r.revision;
		draft = {
			required: r.required,
			reason: r.override_reason ?? '',
			submitted: r.submitted_on ?? '',
			pkg: r.package ?? '',
			result: r.result ?? '',
			resultOn: r.result_on ?? '',
			remark: r.remark ?? ''
		};
	}

	function show(error: Parameters<typeof failureText>[0]) {
		const details = error.body?.details ?? [];
		failure = details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (editing === null || !control) return;
		message = '';
		failure = '';
		busy = true;
		const entry: SubmissionEntry = { remark: draft.remark, package: draft.pkg };
		if (draft.required !== control.submission_required) {
			entry.required = draft.required;
			entry.override_reason = draft.reason;
		}
		if (draft.submitted) entry.submitted_on = draft.submitted;
		if (draft.result) entry.result = draft.result;
		if (draft.resultOn) entry.result_on = draft.resultOn;
		const revision = editing;
		const result = await saveSubmissionEntry(control.document_id, revision, entry);
		busy = false;
		if (!result.ok) {
			show(result.error);
			return;
		}
		control = result.data;
		editing = null;
		message = t('subSaved', { revision: String(revision) });
	}

	async function upload() {
		const file = scanInput?.files?.[0];
		if (!file || editing === null || !control) return;
		message = '';
		failure = '';
		const revision = editing;
		const result = await uploadSubmissionScan(control.document_id, revision, file);
		if (!result.ok) {
			show(result.error);
			return;
		}
		control = result.data;
		if (scanInput) scanInput.value = '';
		message = t('subScanUploaded', { revision: String(revision) });
	}

	function day(iso: string | undefined): string {
		if (!iso) return '';
		const [y, m, d] = iso.split('-').map(Number);
		return new Date(Date.UTC(y, m - 1, d)).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' });
	}

	type Step = { label: MessageKey; state: 'done' | 'now' | 'next' };

	// steps menyusun panel "Revision N needs" untuk revisi terbaru.
	function steps(r: SubmissionRow): Step[] {
		const released = r.status === 'released';
		const signing = r.signing !== undefined;
		const out: Step[] = [{ label: 'subStepChange', state: 'done' }];
		out.push({ label: 'subStepEdit', state: released || signing ? 'done' : 'now' });
		out.push({ label: 'subStepSign', state: released && r.top_sign_on ? 'done' : signing ? 'now' : 'next' });
		if (r.required) {
			out.push({ label: 'subStepSubmit', state: r.submitted_on ? 'done' : released ? 'now' : 'next' });
		}
		return out;
	}

	const stepKey: Record<Step['state'], MessageKey> = { done: 'subStepDone', now: 'subStepNow', next: 'subStepNext' };
</script>

<section aria-labelledby="sub-h">
	<h2 id="sub-h">{t('subTitle')}</h2>
	<p class="hint">{t('subLead')}</p>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#if doc.set.length > 1}
		<div class="segmented tabs" role="group" aria-label={t('subTitle')}>
			{#each doc.set as member (member.id)}
				<button type="button" aria-pressed={member.id === (currentId || doc.id)} onclick={() => pick(member.id)}>
					{member.document_type.name} · <span class="mono">{member.number}</span>
				</button>
			{/each}
		</div>
	{/if}

	{#if control}
		<div class="layout">
			<div class="main">
				<div class="head">
					<h3>{t('subRevisions', { number: control.number, count: String(control.rows.length) })}</h3>
					<label class="inline"><input type="checkbox" bind:checked={showAll} /> {t('subShowAll')}</label>
				</div>
				<div class="scroll">
					<table>
						<thead>
							<tr>
								<th>{t('subNo')}</th>
								<th>{t('subRevision')}</th>
								<th>{t('subReason')}</th>
								<th>{t('subStart')}</th>
								<th>{t('subFinish')}</th>
								<th>{t('subTopSign')}</th>
								<th>{t('subSubmit')}</th>
								<th>{t('subResult')}</th>
								<th>{t('subRemark')}</th>
								{#if canEdit}<th></th>{/if}
							</tr>
						</thead>
						<tbody>
							{#each rows as r, i (r.revision)}
								<tr class:current={r.status !== 'released'}>
									<td>{i + 1}</td>
									<td>
										{r.revision}
										<span class="badge" data-status={r.status}>{t(r.status === 'released' ? 'subReleased' : 'subInProgress')}</span>
									</td>
									<td>{r.reason}</td>
									<td>{day(r.started_on)}</td>
									<td>
										{#if r.finished_on}{day(r.finished_on)}<small>{t('subDaysTaken', { days: String(r.days) })}</small>{:else}<small>{t('subDays', { days: String(r.days) })}</small>{/if}
									</td>
									<td>
										{#if r.top_sign_on}{day(r.top_sign_on)}<small>{t(r.top_sign_by === 'digital' ? 'subSignedDigitally' : 'subSignedScan')}</small>{/if}
									</td>
									<td>
										{#if !r.required}
											<span class="badge">{t('subNotRequired')}</span>
											{#if r.override_reason}<small>{r.override_reason}</small>{/if}
										{:else if r.submitted_on}
											{day(r.submitted_on)}{#if r.package}<small>{r.package}</small>{/if}
										{:else}
											<small>{t('subRequired')}</small>
										{/if}
									</td>
									<td>
										{#if r.result}
											<span class="badge" data-result={r.result}>{t(`subResult_${r.result}` as MessageKey)}</span>
											{#if r.result_on}<small>{day(r.result_on)}</small>{/if}
										{/if}
										{#if r.scan_name}<a href={submissionScanUrl(control.document_id, r.revision)} download={r.scan_name}>{r.scan_name}</a>{/if}
									</td>
									<td>{r.remark ?? ''}</td>
									{#if canEdit}
										<td><button type="button" class="link" onclick={() => edit(r)} aria-label={t('subEditFor', { revision: String(r.revision) })}>{t('subEdit')}</button></td>
									{/if}
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				{#if editing !== null}
					<form class="edit" onsubmit={save} novalidate aria-labelledby="sub-edit-h">
						<h3 id="sub-edit-h">{t('subEditTitle', { revision: String(editing) })}</h3>
						<label class="inline"><input type="checkbox" bind:checked={draft.required} /> {t('subNeedsSubmission')}</label>
						{#if draft.required !== control.submission_required}
							<label for="sub-reason">{t('subOverrideReason')}</label>
							<input id="sub-reason" bind:value={draft.reason} maxlength="500" autocomplete="off" />
						{/if}
						{#if draft.required}
							<div class="grid">
								<div class="field">
									<label for="sub-submitted">{t('subSubmittedOn')}</label>
									<input id="sub-submitted" type="date" bind:value={draft.submitted} />
								</div>
								<div class="field">
									<label for="sub-package">{t('subPackage')}</label>
									<input id="sub-package" bind:value={draft.pkg} maxlength="500" autocomplete="off" />
								</div>
								<div class="field">
									<label for="sub-result">{t('subResultLabel')}</label>
									<select id="sub-result" bind:value={draft.result}>
										<option value="">{t('subResultNone')}</option>
										<option value="waiting">{t('subResult_waiting')}</option>
										<option value="approved">{t('subResult_approved')}</option>
										<option value="rejected">{t('subResult_rejected')}</option>
									</select>
								</div>
								<div class="field">
									<label for="sub-result-on">{t('subResultOn')}</label>
									<input id="sub-result-on" type="date" bind:value={draft.resultOn} />
								</div>
							</div>
						{/if}
						<label for="sub-remark">{t('subRemarkLabel')}</label>
						<textarea id="sub-remark" bind:value={draft.remark} maxlength="1000" rows="2"></textarea>
						<div class="buttons">
							<button type="submit" class="primary" disabled={busy}>{t('subSave')}</button>
							<button type="button" onclick={() => (editing = null)}>{t('subCancel')}</button>
						</div>
						{#if draft.required}
							<div class="field">
								<label for="sub-scan">{t('subScan')}</label>
								<div class="buttons">
									<input id="sub-scan" type="file" accept=".pdf,.png,.jpg,.jpeg" bind:this={scanInput} />
									<button type="button" onclick={upload}>{t('subScanUpload')}</button>
								</div>
							</div>
						{/if}
					</form>
				{/if}
			</div>

			<aside class="side">
				<div class="card">
					<h3>{t('subPrintTitle')}</h3>
					<p class="hint">{control.form_number ? `${control.form_number} · ` : ''}{t('subPrintHint')}</p>
					<a class="button primary" href={submissionExportUrl(control.document_id, 'pdf')} download>{t('subDownloadPdf')}</a>
					<a class="button" href={submissionExportUrl(control.document_id, 'xlsx')} download>{t('subDownloadExcel')}</a>
				</div>
				{#if latest}
					<div class="card">
						<h3>{t('subNeeds', { revision: String(latest.revision) })}</h3>
						<ul class="needs">
							{#each steps(latest) as s (s.label)}
								<li><span class="badge" data-step={s.state}>{t(stepKey[s.state])}</span> {t(s.label)}</li>
							{/each}
						</ul>
					</div>
				{/if}
				<div class="card note">
					<h3>{t('subWhenTitle')}</h3>
					<p class="hint">{t('subWhen')}</p>
				</div>
			</aside>
		</div>
	{/if}
</section>

<style>
	.tabs {
		display: flex;
		flex-wrap: wrap;
		max-width: 100%;
		margin-bottom: 0.75rem;
	}

	.layout {
		display: flex;
		flex-wrap: wrap;
		gap: 1.5rem;
		align-items: flex-start;
	}

	.main {
		flex: 999 1 36rem;
		min-width: 0;
	}

	.side {
		flex: 1 1 16rem;
		max-width: 22rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.card {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 0.75rem 1rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
	}

	.card h3 {
		margin: 0;
	}

	.note {
		background: var(--warning-weak);
	}

	.head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: baseline;
		gap: 0.75rem;
	}

	.inline {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.scroll {
		overflow-x: auto;
	}

	.scroll table {
		width: 100%;
	}

	td small {
		display: block;
		color: var(--text-3);
	}

	tr.current td {
		background: var(--warning-weak);
	}

	.edit {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		margin-top: 1rem;
		max-width: 44rem;
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		gap: 0.5rem 1rem;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.buttons {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		align-items: center;
	}

	.needs {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
