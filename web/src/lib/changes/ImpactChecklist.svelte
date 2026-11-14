<script lang="ts">
	// Daftar periksa dampak (M5.7, R-CHG-05): untuk setiap dokumen terkait
	// (PFD, PFMEA, CP, WI, standar inspeksi) dicatat terdampak atau tidak,
	// dengan alasan. Alasan wajib bila tidak terdampak.
	import { untrack } from 'svelte';
	import { saveChangeImpacts } from '#lib/api/client.ts';
	import type { Change, ChangeImpactInput } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let { change, editable, onsaved }: { change: Change; editable: boolean; onsaved: (c: Change) => void } = $props();

	const kinds = ['pfd', 'pfmea', 'cp', 'wi', 'inspection_standard', 'other'] as const;
	type Line = { kind: string; document_reference: string; decision: string; reason: string };

	// Daftar yang belum pernah disimpan diisi saran: satu baris per jenis
	// dokumen, dan dokumen terdampak perubahan ini sudah ditandai terdampak.
	function initial(c: Change): Line[] {
		if (c.impacts.length > 0) {
			return c.impacts.map((x) => ({ kind: x.kind, document_reference: x.document_reference ?? '', decision: x.decision, reason: x.reason ?? '' }));
		}
		return ['pfd', 'pfmea', 'cp', 'wi', 'inspection_standard'].map((kind) => {
			const doc = c.documents.find((d) => d.document_type.key === kind);
			return { kind, document_reference: doc?.number ?? '', decision: doc ? 'affected' : '', reason: '' };
		});
	}

	// Isi awal saja; halaman membuat ulang komponen ini untuk perubahan lain.
	let lines = $state<Line[]>(untrack(() => initial(change)));
	let message = $state('');
	let failure = $state('');

	function add() {
		lines.push({ kind: 'wi', document_reference: '', decision: '', reason: '' });
	}

	// kindLabel memberi nama jenis dokumen terkait.
	function kindLabel(kind: string): string {
		return t(`impactKind_${kind}` as 'impactKind_pfd');
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		const items: ChangeImpactInput[] = lines.map((l) => ({
			kind: l.kind,
			decision: l.decision,
			document_reference: l.document_reference.trim() || undefined,
			reason: l.reason.trim() || undefined
		}));
		const result = await saveChangeImpacts(change.id, { items });
		if (!result.ok) {
			const first = (result.error.body?.details ?? [])[0];
			if (!first) {
				failure = failureText(result.error);
				return;
			}
			// Field berbentuk items.N.reason: sebut nomor barisnya.
			const match = /^items\.(\d+)\.(.+)$/.exec(first.field);
			const text = problemText(match ? { ...first, field: `impact.${match[2]}` } : first);
			failure = match ? `${t('impactLine', { line: String(Number(match[1]) + 1) })}: ${text}` : text;
			return;
		}
		message = t('impactSaved', { ref: result.data.ref });
		onsaved(result.data);
	}
</script>

<section>
	<h2>{t('impactTitle')}</h2>
	<p class="hint">{t('impactHint')}</p>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	{#if editable}
		<form onsubmit={save} novalidate>
			<table>
				<thead>
					<tr>
						<th>{t('impactKind')}</th>
						<th>{t('impactDocument')}</th>
						<th>{t('impactDecision')}</th>
						<th>{t('impactReason')}</th>
						<th>{t('masterActions')}</th>
					</tr>
				</thead>
				<tbody>
					{#each lines as line, i (i)}
						<tr>
							<td>
								<select aria-label={t('impactKindFor', { line: String(i + 1) })} bind:value={line.kind}>
									{#each kinds as k (k)}<option value={k}>{kindLabel(k)}</option>{/each}
								</select>
							</td>
							<td><input aria-label={t('impactDocumentFor', { line: String(i + 1) })} maxlength="200" bind:value={line.document_reference} /></td>
							<td>
								<select aria-label={t('impactDecisionFor', { line: String(i + 1) })} bind:value={line.decision}>
									<option value="">{t('changeChoose')}</option>
									<option value="affected">{t('impactAffected')}</option>
									<option value="not_affected">{t('impactNotAffected')}</option>
								</select>
							</td>
							<td><input aria-label={t('impactReasonFor', { line: String(i + 1) })} maxlength="500" bind:value={line.reason} /></td>
							<td><button type="button" onclick={() => lines.splice(i, 1)}>{t('changeDocumentRemove')}</button></td>
						</tr>
					{/each}
				</tbody>
			</table>
			<div class="row">
				<button type="button" onclick={add}>{t('impactAdd')}</button>
				<button type="submit" class="primary">{t('impactSave')}</button>
			</div>
		</form>
	{:else if change.impacts.length === 0}
		<p>{t('impactNone')}</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>{t('impactKind')}</th>
					<th>{t('impactDocument')}</th>
					<th>{t('impactDecision')}</th>
					<th>{t('impactReason')}</th>
				</tr>
			</thead>
			<tbody>
				{#each change.impacts as x (x.id)}
					<tr>
						<td>{kindLabel(x.kind)}</td>
						<td class="mono">{x.document_reference ?? ''}</td>
						<td>{t(x.decision === 'affected' ? 'impactAffected' : 'impactNotAffected')}</td>
						<td>{x.reason ?? ''}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</section>

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		margin-top: 0.75rem;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
