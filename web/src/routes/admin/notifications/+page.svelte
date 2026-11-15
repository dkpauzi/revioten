<script lang="ts">
	// Email notifikasi (D-194): keadaan pengaturan email (diisi IT di server),
	// saklar per jenis peristiwa, email uji, dan 100 email terbaru di kotak
	// keluar. Butuh A-3; server tetap memeriksa setiap operasi.
	import { onMount } from 'svelte';
	import { getNotifications, sendTestEmail, setNotificationEvent } from '#lib/api/client.ts';
	import type { NotificationOverview } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';

	let overview = $state<NotificationOverview | null>(null);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');

	function eventLabel(key: string): string {
		return t(`notifyEvent_${key}` as MessageKey);
	}

	function statusLabel(status: string): string {
		return t(`notifyStatus_${status}` as MessageKey);
	}

	function reasonLabel(reason: string | undefined): string {
		if (!reason) return '';
		if (reason === 'no_email' || reason === 'mail_not_configured') return t(`notifyReason_${reason}` as MessageKey);
		return reason;
	}

	async function load() {
		const result = await getNotifications();
		if (!result.ok) {
			loadError = failureText(result.error);
			return;
		}
		overview = result.data;
	}

	async function toggle(key: string, enabled: boolean) {
		message = '';
		failure = '';
		const result = await setNotificationEvent(key, enabled);
		if (!result.ok) {
			failure = failureText(result.error);
			await load();
			return;
		}
		message = t(enabled ? 'notifyEventOn' : 'notifyEventOff', { event: eventLabel(key) });
		await load();
	}

	async function test() {
		message = '';
		failure = '';
		const result = await sendTestEmail();
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		message = t('notifyTestQueued');
		await load();
	}

	onMount(load);
</script>

<h1>{t('notifyTitle')}</h1>
<p class="hint">{t('notifyHint')}</p>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}
{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

{#if overview}
	<section>
		<h2>{t('notifyServer')}</h2>
		<dl class="facts">
			<dt>{t('notifyProvider')}</dt>
			<dd>{overview.provider ? t(`notifyProvider_${overview.provider}` as MessageKey) : t('notifyProviderNone')}</dd>
			{#if overview.provider}
				<dt>{t('notifyFrom')}</dt>
				<dd>{overview.from}</dd>
				<dt>{t('notifyServerAddress')}</dt>
				<dd>{overview.server}</dd>
			{/if}
		</dl>
		<p class="hint">{t('notifyServerHint')}</p>
		<p><button type="button" onclick={test}>{t('notifyTest')}</button></p>
	</section>

	<section>
		<h2>{t('notifyEvents')}</h2>
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t('notifyEvent')}</th>
						<th>{t('notifyRecipients')}</th>
						<th>{t('notifyOn')}</th>
					</tr>
				</thead>
				<tbody>
					{#each overview.events as e (e.key)}
						<tr>
							<td>{eventLabel(e.key)}</td>
							<td>{t(`notifyTo_${e.key}` as MessageKey)}</td>
							<td>
								<input
									type="checkbox"
									aria-label={eventLabel(e.key)}
									checked={e.enabled}
									onchange={(event) => toggle(e.key, event.currentTarget.checked)}
								/>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</section>

	<section>
		<h2>{t('notifyOutbox')}</h2>
		{#if overview.outbox.length === 0}
			<p>{t('notifyOutboxEmpty')}</p>
		{:else}
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th>{t('notifyCreated')}</th>
							<th>{t('notifyEvent')}</th>
							<th>{t('notifyRecipient')}</th>
							<th>{t('notifySubject')}</th>
							<th>{t('numberColumnStatus')}</th>
						</tr>
					</thead>
					<tbody>
						{#each overview.outbox as o (o.id)}
							<tr>
								<td>{formatDateTime(o.created_at)}</td>
								<td>{eventLabel(o.event)}</td>
								<td>{o.recipient}{o.email ? ` · ${o.email}` : ''}</td>
								<td>{o.subject}</td>
								<td data-outbox-status={o.status}>
									{statusLabel(o.status)}{o.reason ? ` · ${reasonLabel(o.reason)}` : ''}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
{/if}

<style>
	.scroll {
		overflow-x: auto;
	}
</style>
