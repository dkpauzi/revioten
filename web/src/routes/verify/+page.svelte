<script lang="ts">
	// Halaman verifikasi (M6.5, D-32): QR di cetakan membuka halaman ini.
	// Menampilkan apakah revisi itu berlaku, digantikan, draft, dibatalkan,
	// atau usang, beserta hash isinya. Bisa juga dibuka dari dalam sistem.
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { verifyRevision } from '#lib/api/client.ts';
	import type { RevisionVerification } from '#lib/api/gen/index.ts';
	import { failureText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';

	let id = $derived(page.url.searchParams.get('id') ?? '');
	let result = $state<RevisionVerification | null>(null);
	let failure = $state('');

	$effect(() => {
		const current = id;
		untrack(() => load(current));
	});

	async function load(revisionId: string) {
		result = null;
		failure = '';
		if (!revisionId) {
			failure = t('verifyMissing');
			return;
		}
		const r = await verifyRevision(revisionId);
		if (!r.ok) {
			failure = failureText(r.error);
			return;
		}
		result = r.data;
	}
</script>

<h1>{t('verifyTitle')}</h1>
{#if failure}<p class="error" role="alert">{failure}</p>{/if}
{#if result}
	<p class="verdict" data-verify={result.status}>{t(`verifyStatus_${result.status}` as 'verifyStatus_effective')}</p>
	<dl class="facts">
		<dt>{t('documentNumber')}</dt>
		<dd class="mono"><a href="/documents/view?id={result.document_id}">{result.number}</a></dd>
		<dt>{t('numberColumnType')}</dt>
		<dd>{result.document_type}</dd>
		<dt>{t('documentCustomer')}</dt>
		<dd>{result.customer_code}</dd>
		<dt>{t('changeRevision')}</dt>
		<dd>{result.revision}</dd>
		{#if result.released_at}<dt>{t('verifyReleasedAt')}</dt><dd>{formatDateTime(result.released_at)}</dd>{/if}
		{#if result.sha256}<dt>SHA-256</dt><dd class="mono hash">{result.sha256}</dd>{/if}
	</dl>
{/if}

<style>
	.verdict {
		font-size: 1.25rem;
		font-weight: 600;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		background: var(--info-weak);
		color: var(--info);
	}

	.verdict[data-verify='effective'] {
		background: var(--success-weak);
		color: var(--success);
	}

	.verdict[data-verify='superseded'],
	.verdict[data-verify='void'],
	.verdict[data-verify='obsolete'] {
		background: var(--danger-weak);
		color: var(--danger);
	}

	.facts {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 4px 16px;
	}

	.facts dt {
		color: var(--text-3);
	}

	.facts dd {
		margin: 0;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	.hash {
		overflow-wrap: anywhere;
	}
</style>
