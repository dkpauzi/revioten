<script lang="ts">
	// Halaman tanda tangan (fase 3, D-198): penanda tangan membaca ringkasan
	// revisi yang dibekukan, mengetik ulang password, lalu Sign atau Reject
	// dengan alasan. Hanya orang yang slotnya sedang ditunggu yang bisa
	// menandatangani (D-49); server memeriksanya lagi.
	import { page } from '$app/state';
	import { getSignatureRequest, rejectSignatureRequest, signSignatureRequest } from '#lib/api/client.ts';
	import type { SignatureRequest } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import PasswordInput from '#lib/PasswordInput.svelte';
	import { session } from '#lib/session.svelte.ts';

	let request = $state<SignatureRequest | null>(null);
	let loadError = $state('');
	let mode = $state<'sign' | 'reject'>('sign');
	let password = $state('');
	let reason = $state('');
	let busy = $state(false);
	let message = $state('');
	let failure = $state('');

	let me = $derived(session.current?.person.id ?? '');
	let mySlot = $derived(request?.slots.find((s) => s.order === request?.next_slot && s.person_id === me));

	$effect(() => {
		const id = page.url.searchParams.get('id') ?? '';
		load(id);
	});

	async function load(id: string) {
		const result = await getSignatureRequest(id);
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		request = result.data;
	}

	function show(error: Parameters<typeof failureText>[0]) {
		const details = error.body?.details ?? [];
		failure = details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	async function sign(event: SubmitEvent) {
		event.preventDefault();
		if (!request || !mySlot) return;
		const slot = mySlot.print_title;
		message = '';
		failure = '';
		busy = true;
		const result = await signSignatureRequest(request.id, password);
		busy = false;
		password = '';
		if (!result.ok) {
			show(result.error);
			return;
		}
		request = result.data;
		message = t('signDone', { slot });
	}

	async function reject(event: SubmitEvent) {
		event.preventDefault();
		if (!request) return;
		message = '';
		failure = '';
		busy = true;
		const result = await rejectSignatureRequest(request.id, reason, password);
		busy = false;
		password = '';
		if (!result.ok) {
			show(result.error);
			return;
		}
		request = result.data;
		mode = 'sign';
		message = t('signRejectedDone');
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

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

{#if request}
	<h1>{t('signTitle', { number: request.number, revision: String(request.revision) })}</h1>
	<p>
		<span class="badge" data-signature={request.status}>{t(statusKey(request.status))}</span>
		{request.customer_code} · {t('sigSentBy', { by: request.sent_by, at: formatDateTime(request.sent_at) })} ·
		{t('sigReleaseDate', { date: request.release_date })}
	</p>
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	<div class="layout">
		<section class="card" aria-labelledby="what-h">
			<h2 id="what-h">{t('signWhat')}</h2>
			<p class="mono">{t('signHash', { hash: request.content_sha256 })}</p>
			{#if request.changes.length > 0}<p>{t('signChanges', { changes: request.changes.join(', ') })}</p>{/if}
			{#if request.note}<p class="note">{request.note}</p>{/if}
			<p class="hint">{t('signReadHint')}</p>
			<a class="button" href={`/documents/view?id=${request.document_id}`}>{t('signaturesOpenDocument')}</a>
		</section>

		<div class="side">
			<section class="card" aria-labelledby="order-h">
				<h2 id="order-h">{t('signOrder')}</h2>
				<ol class="slots">
					{#each request.slots as slot (slot.order)}
						<li class:next={request.next_slot === slot.order}>
							<strong>{slot.print_title}</strong> · {slot.person_name}{slot.person_id === me ? ' ✓' : ''}
							<span class="hint">{slotState(request, slot.order)}</span>
						</li>
					{/each}
				</ol>
			</section>

			{#if mySlot && request.status === 'waiting'}
				{#if mode === 'sign'}
					<form class="card" onsubmit={sign} novalidate aria-labelledby="sign-h">
						<h2 id="sign-h">{t('signAs', { slot: mySlot.print_title })}</h2>
						<p class="hint">{t('signConfirm', { revision: String(request.revision) })}</p>
						<label for="sign-password">{t('sigPassword')}</label>
						<PasswordInput id="sign-password" bind:value={password} autocomplete="current-password" />
						<div class="buttons">
							<button type="submit" class="primary" disabled={busy}>{t('signButton')}</button>
							<button type="button" onclick={() => (mode = 'reject')}>{t('signReject')}</button>
						</div>
					</form>
				{:else}
					<form class="card" onsubmit={reject} novalidate aria-labelledby="reject-h">
						<h2 id="reject-h">{t('signRejectTitle', { revision: String(request.revision) })}</h2>
						<p class="hint">{t('signRejectHint')}</p>
						<label for="reject-reason">{t('signReason')}</label>
						<textarea id="reject-reason" bind:value={reason} maxlength="1000" rows="3"></textarea>
						<label for="reject-password">{t('sigPassword')}</label>
						<PasswordInput id="reject-password" bind:value={password} autocomplete="current-password" />
						<div class="buttons">
							<button type="submit" class="primary" disabled={busy}>{t('signRejectButton')}</button>
							<button type="button" onclick={() => (mode = 'sign')}>{t('signCancel')}</button>
						</div>
					</form>
				{/if}
			{:else if request.status === 'waiting' && !message}
				<p class="hint">{t('signNotYourTurn')}</p>
			{/if}
			{#if request.reject_reason}
				<p class="hint">{t('sigRejectedBy', { by: request.closed_by ?? '', reason: request.reject_reason })}</p>
			{/if}
		</div>
	</div>
{/if}

<style>
	.layout {
		display: flex;
		flex-wrap: wrap;
		gap: 1.5rem;
		align-items: flex-start;
	}

	.layout > .card {
		flex: 999 1 28rem;
		min-width: 0;
	}

	.side {
		flex: 1 1 20rem;
		max-width: 28rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.card {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
		overflow-wrap: anywhere;
	}

	.slots {
		padding-left: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.slots li.next {
		font-weight: 600;
	}

	.slots .hint {
		display: block;
	}

	.buttons {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
</style>
