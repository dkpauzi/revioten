<script lang="ts">
	// Documents → Signatures (fase 3, D-198): permintaan yang menunggu tanda
	// tangan saya, yang saya kirim, dan yang sudah lengkap menunggu rilis
	// (hanya untuk pemegang E-7).
	import { onMount } from 'svelte';
	import { listSignatureRequests } from '#lib/api/client.ts';
	import type { SignatureOverview, SignatureRequest } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { session } from '#lib/session.svelte.ts';

	type Tab = 'waiting' | 'sent' | 'release';

	let overview = $state<SignatureOverview | null>(null);
	let tab = $state<Tab>('waiting');
	let loadError = $state('');

	let me = $derived(session.current?.person.id ?? '');
	let list = $derived(
		!overview ? [] : tab === 'waiting' ? overview.waiting : tab === 'sent' ? overview.sent : overview.to_release
	);

	onMount(async () => {
		const result = await listSignatureRequests();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		overview = result.data;
		if (overview.waiting.length === 0 && overview.sent.length > 0) tab = 'sent';
	});

	function mySlot(r: SignatureRequest): string {
		return r.slots.find((s) => s.person_id === me && s.order === r.next_slot)?.print_title ?? '';
	}

	function next(r: SignatureRequest): string {
		return r.slots.find((s) => s.order === r.next_slot)?.person_name ?? '';
	}

	function statusKey(status: string): MessageKey {
		return `sigStatus_${status}` as MessageKey;
	}
</script>

<h1>{t('signaturesTitle')}</h1>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}

{#if overview}
	<div class="segmented tabs" role="group" aria-label={t('signaturesTitle')}>
		<button type="button" aria-pressed={tab === 'waiting'} onclick={() => (tab = 'waiting')}>
			{t('signaturesTabWaiting', { count: String(overview.waiting.length) })}
		</button>
		<button type="button" aria-pressed={tab === 'sent'} onclick={() => (tab = 'sent')}>
			{t('signaturesTabSent', { count: String(overview.sent.length) })}
		</button>
		{#if overview.to_release.length > 0}
			<button type="button" aria-pressed={tab === 'release'} onclick={() => (tab = 'release')}>
				{t('signaturesTabRelease', { count: String(overview.to_release.length) })}
			</button>
		{/if}
	</div>

	{#if list.length === 0}
		<p class="hint">{t('signaturesEmpty')}</p>
	{:else}
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t('signaturesDocument')}</th>
						<th>{t('signaturesCustomer')}</th>
						<th>{tab === 'waiting' ? t('signaturesSlot') : t('signaturesSigners')}</th>
						<th>{t('signaturesStatus')}</th>
						<th>{t('signaturesSent')}</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each list as r (r.id)}
						<tr>
							<td><span class="mono">{r.number}</span> · {t('changeRevision')} {r.revision}</td>
							<td>{r.customer_code}</td>
							<td>
								{#if tab === 'waiting'}
									{mySlot(r)}
								{:else}
									<span class="chain" aria-label={r.slots.map((s) => `${s.print_title}: ${s.person_name}`).join(', ')}>
										{#each r.slots as s (s.order)}
											<span class="dot" class:done={s.signed_at} class:next={r.next_slot === s.order} title={`${s.print_title} · ${s.person_name}`}>{s.order}</span>
										{/each}
									</span>
									{#if r.next_slot}<span class="hint">{t('signaturesNext', { name: next(r) })}</span>{/if}
								{/if}
							</td>
							<td><span class="badge" data-signature={r.status}>{t(statusKey(r.status))}</span></td>
							<td>{r.sent_by} · {formatDateTime(r.sent_at)}</td>
							<td>
								{#if tab === 'waiting'}
									<a class="button primary" href={`/signatures/sign?id=${r.id}`}>{t('sigOpenSign')}</a>
								{:else}
									<a class="button" href={`/documents/view?id=${r.document_id}`}>{t('signaturesOpenDocument')}</a>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{/if}

<style>
	/* Di HP tab boleh turun ke baris berikutnya, supaya halaman tidak melebar. */
	.tabs {
		display: flex;
		flex-wrap: wrap;
		max-width: 100%;
	}

	.scroll {
		overflow-x: auto;
		margin-top: 1rem;
	}

	.scroll table {
		width: 100%;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	.chain {
		display: inline-flex;
		gap: 0.25rem;
		margin-right: 0.5rem;
	}

	.dot {
		width: 1.4rem;
		height: 1.4rem;
		border-radius: 50%;
		display: inline-grid;
		place-items: center;
		font-size: 0.7rem;
		font-weight: 600;
		background: var(--surface-muted, #f2f4f7);
		color: var(--text-muted, #475467);
	}

	.dot.done {
		background: var(--success-weak);
		color: var(--success);
	}

	.dot.next {
		background: var(--accent);
		color: #ffffff;
	}
</style>
