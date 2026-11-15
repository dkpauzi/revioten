<script lang="ts">
	// Service account API integrasi (M9.1, R-API-04): aplikasi lain yang boleh
	// membaca data rilis dan mengirim usulan perubahan lewat /integration/v1.
	// Hanya admin (A-1). Nilai token tampil SATU KALI saat diterbitkan dan
	// langsung disalin ke clipboard; yang disimpan server hanya hash-nya.
	import { onMount } from 'svelte';
	import {
		createServiceAccount,
		disableServiceAccount,
		issueServiceToken,
		listCustomers,
		listServiceAccounts,
		revokeServiceToken
	} from '#lib/api/client.ts';
	import type { Customer, ServiceAccount } from '#lib/api/gen/index.ts';
	import { copyText } from '#lib/clipboard.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { formatDateTime } from '#lib/format.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';

	let accounts = $state<ServiceAccount[]>([]);
	let scopes = $state<string[]>([]);
	let customers = $state<Customer[]>([]);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	// Token yang baru terbit: tampil sekali, lalu hilang saat halaman ditinggalkan.
	let issued = $state<{ name: string; token: string; copied: boolean } | null>(null);

	const empty = { name: '', ownerTeam: '', contact: '', purpose: '', scopes: [] as string[], customerIds: [] as string[], networks: '' };
	let draft = $state({ ...empty });
	let days = $state<Record<string, number | null>>({});

	function scopeLabel(scope: string): string {
		return t(`scope_${scope.replace(':', '_')}` as MessageKey);
	}

	function tokenState(token: ServiceAccount['tokens'][number]): string {
		if (token.revoked_at) return t('serviceTokenRevoked');
		if (new Date(token.expires_at).getTime() <= Date.now()) return t('serviceTokenExpired');
		return t('serviceTokenActive');
	}

	async function load() {
		const [list, customerList] = await Promise.all([listServiceAccounts(), listCustomers()]);
		if (!list.ok) {
			loadError = failureText(list.error);
			return;
		}
		accounts = list.data.items;
		scopes = list.data.scopes;
		customers = customerList.ok ? customerList.data.items.filter((c) => c.active) : [];
	}

	function fail(error: Parameters<typeof failureText>[0]) {
		const details = error.body?.details ?? [];
		failure = details.length > 0 ? problemText(details[0]) : failureText(error);
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		message = '';
		failure = '';
		fieldErrors = {};
		const result = await createServiceAccount({
			name: draft.name,
			owner_team: draft.ownerTeam,
			contact: draft.contact,
			purpose: draft.purpose,
			scopes: draft.scopes,
			customer_ids: draft.customerIds,
			networks: draft.networks
				.split(/[\s,]+/)
				.map((n) => n.trim())
				.filter((n) => n !== '')
		});
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
			if (details.length === 0) failure = failureText(result.error);
			return;
		}
		message = t('serviceAccountCreated', { name: result.data.name });
		draft = { ...empty };
		await load();
	}

	async function issue(account: ServiceAccount) {
		message = '';
		failure = '';
		const result = await issueServiceToken(account.id, days[account.id] ?? undefined);
		if (!result.ok) {
			fail(result.error);
			return;
		}
		const copied = await copyText(result.data.token);
		issued = { name: account.name, token: result.data.token, copied };
		await load();
	}

	async function revoke(account: ServiceAccount, tokenId: string, prefix: string) {
		message = '';
		failure = '';
		if (!confirm(t('serviceTokenRevokeConfirm', { prefix, name: account.name }))) return;
		const result = await revokeServiceToken(account.id, tokenId);
		if (!result.ok) {
			fail(result.error);
			return;
		}
		message = t('serviceTokenRevokedDone', { prefix });
		await load();
	}

	async function disable(account: ServiceAccount) {
		message = '';
		failure = '';
		if (!confirm(t('serviceAccountDisableConfirm', { name: account.name }))) return;
		const result = await disableServiceAccount(account.id);
		if (!result.ok) {
			fail(result.error);
			return;
		}
		message = t('serviceAccountDisabledDone', { name: account.name });
		await load();
	}

	onMount(load);
</script>

<h1>{t('serviceAccountsTitle')}</h1>
<p class="hint">{t('serviceAccountsHint')}</p>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}
{#if issued}
	<div class="issued" role="status">
		<p>{t('serviceTokenIssued', { name: issued.name })}</p>
		<p><code>{issued.token}</code></p>
		<p class="hint">{issued.copied ? t('serviceTokenCopied') : t('serviceTokenCopyYourself')}</p>
	</div>
{/if}
{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

{#if accounts.length === 0}
	<p>{t('serviceAccountsNone')}</p>
{/if}
{#each accounts as account (account.id)}
	<section class="account">
		<h2>
			{account.name}
			{#if account.disabled}<span class="badge" data-status="void">{t('serviceAccountDisabled')}</span>{/if}
		</h2>
		<dl class="facts">
			<dt>{t('serviceAccountOwner')}</dt>
			<dd>{account.owner_team} · {account.contact}</dd>
			<dt>{t('serviceAccountPurpose')}</dt>
			<dd>{account.purpose}</dd>
			<dt>{t('serviceAccountScopes')}</dt>
			<dd>{account.scopes.map(scopeLabel).join(', ')}</dd>
			<dt>{t('serviceAccountCustomers')}</dt>
			<dd>{account.customer_codes.join(', ')}</dd>
			<dt>{t('serviceAccountNetworks')}</dt>
			<dd>{account.networks.length > 0 ? account.networks.join(', ') : t('serviceAccountAnyNetwork')}</dd>
		</dl>
		<div class="scroll">
			<table>
				<thead>
					<tr>
						<th>{t('serviceTokenPrefix')}</th>
						<th>{t('numberColumnStatus')}</th>
						<th>{t('serviceTokenExpires')}</th>
						<th>{t('serviceTokenLastUsed')}</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each account.tokens as token (token.id)}
						<tr>
							<td class="mono">{token.prefix}</td>
							<td>{tokenState(token)}</td>
							<td>{formatDateTime(token.expires_at)}</td>
							<td>{token.last_used_at ? formatDateTime(token.last_used_at) : '-'}</td>
							<td>
								{#if !token.revoked_at}
									<button type="button" onclick={() => revoke(account, token.id, token.prefix)}>{t('serviceTokenRevoke')}</button>
								{/if}
							</td>
						</tr>
					{:else}
						<tr><td colspan="5">{t('serviceTokensNone')}</td></tr>
					{/each}
				</tbody>
			</table>
		</div>
		{#if !account.disabled}
			<div class="row">
				<div class="field">
					<label for="days-{account.id}">{t('serviceTokenDays')}</label>
					<input id="days-{account.id}" type="number" min="1" max="365" placeholder="90" bind:value={days[account.id]} />
				</div>
				<button type="button" class="primary" onclick={() => issue(account)}>{t('serviceTokenIssue')}</button>
				<button type="button" onclick={() => disable(account)}>{t('serviceAccountDisable')}</button>
			</div>
		{/if}
	</section>
{/each}

<section>
	<h2>{t('serviceAccountAdd')}</h2>
	<form onsubmit={create} novalidate>
		<div class="grid">
			<div class="field">
				<label for="sa-name">{t('serviceAccountName')}</label>
				<input id="sa-name" bind:value={draft.name} />
				{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
			</div>
			<div class="field">
				<label for="sa-team">{t('serviceAccountTeam')}</label>
				<input id="sa-team" bind:value={draft.ownerTeam} />
				{#if fieldErrors.owner_team}<p class="error">{fieldErrors.owner_team}</p>{/if}
			</div>
			<div class="field">
				<label for="sa-contact">{t('serviceAccountContact')}</label>
				<input id="sa-contact" bind:value={draft.contact} />
				{#if fieldErrors.contact}<p class="error">{fieldErrors.contact}</p>{/if}
			</div>
			<div class="field">
				<label for="sa-purpose">{t('serviceAccountPurpose')}</label>
				<input id="sa-purpose" bind:value={draft.purpose} />
				{#if fieldErrors.purpose}<p class="error">{fieldErrors.purpose}</p>{/if}
			</div>
		</div>
		<fieldset>
			<legend>{t('serviceAccountScopes')}</legend>
			{#each scopes as scope (scope)}
				<label><input type="checkbox" bind:group={draft.scopes} value={scope} /> {scopeLabel(scope)} <span class="mono hint">{scope}</span></label>
			{/each}
			{#if fieldErrors.scopes}<p class="error">{fieldErrors.scopes}</p>{/if}
		</fieldset>
		<fieldset>
			<legend>{t('serviceAccountCustomers')}</legend>
			{#each customers as c (c.id)}
				<label><input type="checkbox" bind:group={draft.customerIds} value={c.id} /> {c.code} · {c.name}</label>
			{/each}
			{#if fieldErrors.customer_ids}<p class="error">{fieldErrors.customer_ids}</p>{/if}
		</fieldset>
		<div class="field">
			<label for="sa-networks">{t('serviceAccountNetworks')}</label>
			<input id="sa-networks" placeholder="192.168.10.0/24" bind:value={draft.networks} aria-describedby="sa-networks-hint" />
			<p id="sa-networks-hint" class="hint">{t('serviceAccountNetworksHint')}</p>
			{#if fieldErrors.networks}<p class="error">{fieldErrors.networks}</p>{/if}
		</div>
		<p><button type="submit">{t('serviceAccountCreate')}</button></p>
	</form>
</section>

<style>
	.account {
		padding: 1rem 0;
		border-top: 1px solid var(--border);
	}

	.issued {
		padding: 1rem;
		border: 1.5px solid var(--accent);
		border-radius: var(--radius-lg);
		margin-bottom: 1rem;
		overflow-wrap: anywhere;
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 0 1rem;
	}

	fieldset {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1.25rem;
		margin: 1rem 0;
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1rem;
		align-items: flex-end;
	}

	.row .field {
		max-width: 12rem;
		margin-bottom: 0;
	}

	.scroll {
		overflow-x: auto;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
