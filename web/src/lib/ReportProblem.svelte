<script lang="ts">
	// Tombol "Report a problem" di bilah atas (fase 4, D-205): setiap akun
	// yang login bisa melaporkan masalah. Alamat halaman dan request ID error
	// terakhir ikut terkirim, supaya admin menemukannya di Web health.
	import { page } from '$app/state';
	import { reportProblem } from '#lib/api/client.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';

	let dialog = $state<HTMLDialogElement | null>(null);
	let text = $state('');
	let busy = $state(false);
	let message = $state('');
	let failure = $state('');

	function open() {
		text = '';
		failure = '';
		message = '';
		dialog?.showModal();
	}

	async function send(event: SubmitEvent) {
		event.preventDefault();
		busy = true;
		failure = '';
		const result = await reportProblem(page.url.pathname + page.url.search, text);
		busy = false;
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		dialog?.close();
		message = t('reportProblemSent');
	}
</script>

<button type="button" class="link" onclick={open}>{t('reportProblem')}</button>
{#if message}<p class="visually-hidden" role="status">{message}</p>{/if}

<dialog bind:this={dialog} aria-labelledby="report-h">
	<form method="dialog" onsubmit={send} novalidate>
		<h2 id="report-h">{t('reportProblemTitle')}</h2>
		<p class="hint">{t('reportProblemHint')}</p>
		<label for="report-text">{t('reportProblemText')}</label>
		<textarea id="report-text" bind:value={text} maxlength="2000" rows="5"></textarea>
		{#if failure}<p class="error" role="alert">{failure}</p>{/if}
		<div class="buttons">
			<button type="submit" class="primary" disabled={busy}>{t('reportProblemSend')}</button>
			<button type="button" onclick={() => dialog?.close()}>{t('reportProblemCancel')}</button>
		</div>
	</form>
</dialog>

<style>
	dialog {
		width: min(32rem, calc(100vw - 2rem));
		border: 1px solid var(--border-strong);
		border-radius: var(--radius);
		padding: 1.25rem;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	h2 {
		margin: 0;
	}

	.buttons {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
</style>
