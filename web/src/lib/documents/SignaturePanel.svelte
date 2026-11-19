<script lang="ts">
	// Tanda tangan digital di halaman dokumen (fase 3; D-198, D-203, D-204).
	// Pemegang E-5 mengirim draft untuk tanda tangan: isi dibekukan, penanda
	// tangan setiap slot dipilih, dan password diketik ulang. Di bawahnya
	// tampil permintaan yang berjalan beserta slotnya. Server memeriksa
	// semua aturan; layar ini hanya menawarkan pilihan yang masuk akal.
	import {
		getSignaturePlan,
		listDocumentSignatureRequests,
		sendForSignature,
		withdrawSignatureRequest
	} from '#lib/api/client.ts';
	import type { Document, SignaturePlan, SignatureRequest } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import PasswordInput from '#lib/PasswordInput.svelte';
	import { can, session } from '#lib/session.svelte.ts';

	let { doc, onchanged }: { doc: Document; onchanged: () => void } = $props();

	let plan = $state<SignaturePlan | null>(null);
	let requests = $state<SignatureRequest[]>([]);
	// Pilihan penanda tangan per urutan slot.
	let chosen = $state<Record<number, string>>({});
	let note = $state('');
	let password = $state('');
	let busy = $state(false);
	let message = $state('');
	let failure = $state('');

	let me = $derived(session.current?.person.id ?? '');
	let current = $derived(requests.find((r) => r.status === 'waiting' || r.status === 'completed'));
	let earlier = $derived(requests.filter((r) => r !== current));
	let canSend = $derived(can('E-5') && doc.status === 'draft' && !doc.signing && doc.baseline === undefined);

	$effect(() => {
		void doc.revision;
		void doc.signing;
		void doc.status;
		load(doc.id);
	});

	async function load(id: string) {
		const list = await listDocumentSignatureRequests(id);
		requests = list.ok ? list.data.items : [];
		plan = null;
		if (can('E-5') && doc.status === 'draft' && !doc.signing && doc.baseline === undefined) {
			const p = await getSignaturePlan(id);
			if (p.ok) {
				plan = p.data;
				chosen = {};
				for (const slot of p.data.slots) {
					if (slot.candidates.length === 1) chosen[slot.order] = slot.candidates[0].person_id;
				}
			}
		}
	}

	// candidates memberi calon satu slot sesuai pilihan slot sebelumnya
	// (D-203): orang yang sudah dipilih tidak muncul lagi, dan bila Project
	// Leader (Leader) menjadi Prepared, Checked hanya Project Leader (Supervisor).
	function candidates(order: number) {
		if (!plan) return [];
		const slot = plan.slots.find((s) => s.order === order);
		if (!slot) return [];
		const taken = Object.entries(chosen)
			.filter(([o]) => Number(o) < order)
			.map(([, p]) => p);
		const prepared = plan.slots.find((s) => s.stage === 'prepared');
		const preparedKey = prepared?.candidates.find((c) => c.person_id === chosen[prepared.order])?.position_key;
		return slot.candidates.filter(
			(c) =>
				!taken.includes(c.person_id) &&
				!(slot.stage === 'checked' && preparedKey === 'project_engineer' && c.position_key !== 'project_leader')
		);
	}

	function pick(order: number, person: string) {
		chosen[order] = person;
		// Pilihan slot sesudahnya yang tidak lagi sah dibuang.
		for (const slot of plan?.slots ?? []) {
			if (slot.order > order && chosen[slot.order] && !candidates(slot.order).some((c) => c.person_id === chosen[slot.order])) {
				const left = candidates(slot.order);
				chosen[slot.order] = left.length === 1 ? left[0].person_id : '';
			}
		}
	}

	function hintFor(stage: string): string {
		if (stage === 'prepared') return t('sigPreparedHint');
		if (stage === 'checked') return t('sigCheckedHint');
		return '';
	}

	function show(error: Parameters<typeof failureText>[0]) {
		const details = error.body?.details ?? [];
		failure = details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	async function send(event: SubmitEvent) {
		event.preventDefault();
		if (!plan) return;
		message = '';
		failure = '';
		busy = true;
		const signers = plan.slots.filter((s) => chosen[s.order]).map((s) => ({ slot: s.order, person_id: chosen[s.order] }));
		const result = await sendForSignature(doc.id, { signers, note: note || undefined, password });
		busy = false;
		password = '';
		if (!result.ok) {
			show(result.error);
			return;
		}
		note = '';
		message = t('sigSent', { revision: String(result.data.revision) });
		onchanged();
	}

	async function withdraw(r: SignatureRequest) {
		if (!confirm(t('sigWithdrawConfirm'))) return;
		message = '';
		failure = '';
		const result = await withdrawSignatureRequest(r.id);
		if (!result.ok) {
			show(result.error);
			return;
		}
		message = t('sigWithdrawn');
		onchanged();
	}

	function slotState(r: SignatureRequest, order: number): string {
		const slot = r.slots.find((s) => s.order === order);
		if (slot?.signed_at) return t('sigSlotSigned', { at: formatDateTime(slot.signed_at) });
		return r.next_slot === order ? t('sigSlotWaiting') : t('sigSlotLater');
	}

	function statusKey(status: string): MessageKey {
		return `sigStatus_${status}` as MessageKey;
	}
</script>

{#if requests.length > 0 || canSend}
	<section aria-labelledby="sig-h">
		<h2 id="sig-h">{t('sigTitle')}</h2>
		{#if message}<p role="status">{message}</p>{/if}
		{#if failure}<p class="error" role="alert">{failure}</p>{/if}

		{#if doc.signing}
			<p class="notice">{t(doc.signing === 'waiting' ? 'sigLockedWaiting' : 'sigLockedCompleted')}</p>
		{/if}

		{#if current}
			<div class="request">
				<p>
					<span class="badge" data-signature={current.status}>{t(statusKey(current.status))}</span>
					{t('sigSentBy', { by: current.sent_by, at: formatDateTime(current.sent_at) })} ·
					{t('sigReleaseDate', { date: current.release_date })} ·
					<span class="mono" title={current.content_sha256}>SHA-256 {current.content_sha256.slice(0, 12)}…</span>
				</p>
				{#if current.note}<p class="hint">{current.note}</p>{/if}
				<ol class="slots">
					{#each current.slots as slot (slot.order)}
						<li class:next={current.next_slot === slot.order}>
							<strong>{slot.print_title}</strong>
							<span>{slot.person_name}</span>
							<span class="hint">{slotState(current, slot.order)}</span>
						</li>
					{/each}
				</ol>
				<div class="buttons">
					{#if current.status === 'waiting' && current.slots.find((s) => s.order === current.next_slot)?.person_id === me}
						<a class="button primary" href={`/signatures/sign?id=${current.id}`}>{t('sigOpenSign')}</a>
					{/if}
					{#if current.status === 'waiting' && current.sent_by_id === me}
						<button type="button" onclick={() => withdraw(current)}>{t('sigWithdraw')}</button>
					{/if}
				</div>
			</div>
		{/if}

		{#if canSend && plan}
			<h3>{t('sigSendTitle')}</h3>
			{#if plan.mode === 'wet_only'}
				<p class="hint">{t('sigWetOnly', { code: doc.customer_code })}</p>
			{:else if !plan.route}
				<p class="hint">{t('sigNoRoute')}</p>
			{:else}
				<form class="send" onsubmit={send} novalidate>
					<p class="hint">{t('sigSendHint', { revision: String(doc.revision) })}</p>
					<p>{t('sigRoute', { route: plan.route.name })}</p>
					{#each plan.slots as slot (slot.order)}
						{@const options = candidates(slot.order)}
						<div class="field">
							<label for="signer-{slot.order}">{t('sigSlotChoose', { slot: slot.print_title })}</label>
							{#if slot.candidates.length === 0}
								<p class="error" id="signer-{slot.order}">{t('sigNobody')}</p>
							{:else}
								<select
									id="signer-{slot.order}"
									value={chosen[slot.order] ?? ''}
									onchange={(event) => pick(slot.order, event.currentTarget.value)}
								>
									<option value="">{t('sigChoose')}</option>
									{#each options as c (c.person_id)}
										<option value={c.person_id}>{c.name} · {c.position}</option>
									{/each}
								</select>
								{#if hintFor(slot.stage)}<small>{hintFor(slot.stage)}</small>{/if}
							{/if}
						</div>
					{/each}
					<div class="field">
						<label for="sig-note">{t('sigNote')}</label>
						<textarea id="sig-note" bind:value={note} maxlength="1000" rows="2"></textarea>
					</div>
					<div class="field">
						<label for="sig-password">{t('sigPassword')}</label>
						<PasswordInput id="sig-password" bind:value={password} autocomplete="current-password" />
					</div>
					<div class="buttons">
						<button type="submit" class="primary" disabled={busy}>{t('sigSend')}</button>
					</div>
				</form>
			{/if}
		{/if}

		{#if earlier.length > 0}
			<details>
				<summary>{t('sigHistory')} ({earlier.length})</summary>
				<ul class="history">
					{#each earlier as r (r.id)}
						<li>
							<span class="badge" data-signature={r.status}>{t(statusKey(r.status))}</span>
							{t('changeRevision')} {r.revision} · {t('sigSentBy', { by: r.sent_by, at: formatDateTime(r.sent_at) })}
							{#if r.reject_reason}· {t('sigRejectedBy', { by: r.closed_by ?? '', reason: r.reject_reason })}{/if}
						</li>
					{/each}
				</ul>
			</details>
		{/if}
	</section>
{/if}

<style>
	.notice {
		padding: 0.5rem 0.75rem;
		border-radius: 8px;
		background: var(--warning-weak);
		color: var(--warning);
	}

	.slots {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		padding-left: 1.25rem;
	}

	.slots li {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
		align-items: baseline;
	}

	.slots li.next {
		font-weight: 600;
	}

	.send {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		max-width: 36rem;
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
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	.history {
		padding-left: 1.25rem;
	}
</style>
