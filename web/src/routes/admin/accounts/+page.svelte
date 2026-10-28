<script lang="ts">
	// Halaman pengelolaan akun (M1.2, hak A-1). Server memeriksa hak setiap
	// operasi; halaman ini hanya menawarkan aksi yang masuk akal.
	import { onMount } from 'svelte';
	import {
		createAccount,
		disableAccount,
		enableAccount,
		listAccounts,
		listRoles,
		resetAccountPassword,
		setAccountRoles,
		type ApiFailure,
		type ApiResult
	} from '#lib/api/client.ts';
	import type { AccountInfo, NewAccount, Role, RoleGrant } from '#lib/api/gen/index.ts';
	import { accountStatus, peopleOf } from '#lib/accounts.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import RoleEditor from '#lib/RoleEditor.svelte';
	import { grantLabel } from '#lib/roles.ts';
	import { session } from '#lib/session.svelte.ts';

	// Data dari server. null berarti belum selesai dimuat.
	let accounts = $state<AccountInfo[] | null>(null);
	let roles = $state<Role[]>([]);
	let loadError = $state('');

	// Orang yang sudah punya akun, untuk pilihan "orang yang sudah ada".
	let people = $derived(accounts ? peopleOf(accounts) : []);

	// Isi formulir akun baru.
	let personMode = $state<'new' | 'existing'>('new');
	let personName = $state('');
	let employeeNumber = $state('');
	let personId = $state('');
	let kind = $state<NewAccount['kind']>('work');
	let login = $state('');
	let grants = $state<RoleGrant[]>([]);
	let creating = $state(false);
	// Pesan per field dari server, misalnya { login: 'This login name is already used.' }.
	let fieldErrors = $state<Record<string, string>>({});
	let formError = $state('');

	// Password sementara hanya dikirim server satu kali, jadi ditampilkan di
	// sini sampai ada aksi lain atau halaman ditinggalkan.
	let temporary = $state<{ login: string; password: string } | null>(null);

	// Akun yang perannya sedang diubah, beserta pilihan peran barunya.
	let editingId = $state<string | null>(null);
	let editGrants = $state<RoleGrant[]>([]);
	let busy = $state(false);
	let actionError = $state('');

	async function load() {
		// Dua permintaan dijalankan bersamaan, lalu ditunggu keduanya.
		const [rolesResult, accountsResult] = await Promise.all([listRoles(), listAccounts()]);
		if (!rolesResult.ok) {
			loadError = failureText(rolesResult.error);
			return;
		}
		if (!accountsResult.ok) {
			loadError = failureText(accountsResult.error);
			return;
		}
		roles = rolesResult.data.items;
		accounts = accountsResult.data.items;
		loadError = '';
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		creating = true;
		fieldErrors = {};
		formError = '';
		temporary = null;

		// Input dikirim apa adanya; server yang merapikan dan memeriksa (R-API-01).
		const input: NewAccount = { kind, login };
		if (personMode === 'new') {
			input.person_name = personName;
			if (employeeNumber !== '') {
				input.employee_number = employeeNumber;
			}
		} else if (personId !== '') {
			input.person_id = personId;
		}
		if (kind === 'work') {
			input.roles = grants;
		}

		const result = await createAccount(input);
		creating = false;
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			if (details.length > 0) {
				fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
			} else {
				formError = failureText(result.error);
			}
			return;
		}
		temporary = { login: result.data.account.login, password: result.data.temporary_password };
		personName = '';
		employeeNumber = '';
		personId = '';
		login = '';
		grants = [];
		await load();
	}

	// failureMessage memberi teks kegagalan sebuah aksi: masalah dari server bila
	// ada (misalnya akun admin terakhir), atau teks kegagalan umum.
	function failureMessage(error: ApiFailure): string {
		const details = error.body?.details ?? [];
		return details.length > 0 ? details.map((d) => problemText(d)).join(' ') : failureText(error);
	}

	// run menjalankan satu aksi pada akun, lalu memuat ulang daftar bila
	// berhasil. Hasilnya data dari server, atau null bila gagal.
	async function run<T>(action: () => Promise<ApiResult<T>>): Promise<T | null> {
		busy = true;
		actionError = '';
		temporary = null;
		const result = await action();
		busy = false;
		if (!result.ok) {
			actionError = failureMessage(result.error);
			return null;
		}
		await load();
		return result.data;
	}

	function startEdit(account: AccountInfo) {
		editingId = account.id;
		// Salinan: pilihan baru belum mengubah tabel sebelum disimpan.
		editGrants = account.roles.map((g) => ({ ...g }));
		actionError = '';
	}

	async function saveRoles(account: AccountInfo) {
		if (await run(() => setAccountRoles(account.id, editGrants))) {
			editingId = null;
		}
	}

	async function resetPassword(account: AccountInfo) {
		if (!confirm(t('accountResetConfirm', { login: account.login }))) {
			return;
		}
		const data = await run(() => resetAccountPassword(account.id));
		if (data) {
			temporary = { login: account.login, password: data.temporary_password };
		}
	}

	async function disable(account: AccountInfo) {
		if (!confirm(t('accountDisableConfirm', { login: account.login }))) {
			return;
		}
		await run(() => disableAccount(account.id));
	}

	async function enable(account: AccountInfo) {
		await run(() => enableAccount(account.id));
	}

	// Akun sendiri tidak bisa di-reset atau dinonaktifkan dari sini (server
	// juga menolaknya); password sendiri diganti lewat halaman ganti password.
	function isOwn(account: AccountInfo): boolean {
		return session.current?.account.id === account.id;
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('accountsTitle')}</h1>

{#if temporary}
	<div class="notice" role="status">
		<p>{t('accountTemporaryPassword', { login: temporary.login })} <code>{temporary.password}</code></p>
		<p>{t('accountTemporaryHint')}</p>
	</div>
{/if}

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else}
	<section>
		<h2>{t('accountsCreateTitle')}</h2>
		<!-- novalidate: pemeriksaan input dilakukan server, supaya aturannya hanya satu. -->
		<form onsubmit={create} novalidate>
			<fieldset>
				<legend>{t('accountPerson')}</legend>
				<label><input type="radio" bind:group={personMode} value="new" /> {t('accountPersonNew')}</label>
				<label>
					<input type="radio" bind:group={personMode} value="existing" />
					{t('accountPersonExisting')}
				</label>

				{#if personMode === 'new'}
					<div class="field">
						<label for="person-name">{t('accountPersonName')}</label>
						<input id="person-name" bind:value={personName} autocomplete="off" />
					</div>
					<div class="field">
						<label for="employee-number">{t('accountEmployeeNumber')}</label>
						<input id="employee-number" bind:value={employeeNumber} autocomplete="off" />
						{#if fieldErrors.employee_number}
							<p class="error">{fieldErrors.employee_number}</p>
						{/if}
					</div>
				{:else}
					<div class="field">
						<label for="person-id">{t('accountPersonChoose')}</label>
						<select id="person-id" bind:value={personId}>
							<option value=""></option>
							{#each people as person (person.id)}
								<option value={person.id}>
									{person.name}{person.employee_number ? ` (${person.employee_number})` : ''}
								</option>
							{/each}
						</select>
					</div>
				{/if}
				{#if fieldErrors.person_name || fieldErrors.person_id}
					<p class="error">{fieldErrors.person_name ?? fieldErrors.person_id}</p>
				{/if}
			</fieldset>

			<fieldset>
				<legend>{t('accountKind')}</legend>
				<label><input type="radio" bind:group={kind} value="work" /> {t('accountKindWork')}</label>
				<label><input type="radio" bind:group={kind} value="admin" /> {t('accountKindAdmin')}</label>
				{#if kind === 'admin'}
					<small>{t('accountKindAdminHint')}</small>
				{/if}
				{#if fieldErrors.kind}
					<p class="error">{fieldErrors.kind}</p>
				{/if}
			</fieldset>

			<div class="field">
				<label for="account-login">{t('accountLogin')}</label>
				<!-- autocomplete="off": browser tidak mengisi login admin yang sedang dipakai. -->
				<input id="account-login" bind:value={login} autocomplete="off" />
				{#if fieldErrors.login}
					<p class="error">{fieldErrors.login}</p>
				{/if}
			</div>

			{#if kind === 'work'}
				<RoleEditor {roles} bind:grants />
				{#if fieldErrors.roles}
					<p class="error">{fieldErrors.roles}</p>
				{/if}
			{/if}

			<button type="submit" disabled={creating}>
				{creating ? t('accountCreating') : t('accountCreate')}
			</button>
			{#if formError}
				<p class="error" role="alert">{formError}</p>
			{/if}
		</form>
	</section>

	<section>
		<h2>{t('accountsListTitle')}</h2>
		{#if actionError}
			<p class="error" role="alert">{actionError}</p>
		{/if}
		{#if accounts === null}
			<p>{t('accountsLoading')}</p>
		{:else}
			<table>
				<thead>
					<tr>
						<th>{t('accountLogin')}</th>
						<th>{t('accountPerson')}</th>
						<th>{t('accountKind')}</th>
						<th>{t('accountRoles')}</th>
						<th>{t('accountStatus')}</th>
						<th>{t('accountActions')}</th>
					</tr>
				</thead>
				<tbody>
					{#each accounts as account (account.id)}
						<tr>
							<td>{account.login}</td>
							<td>
								{account.person.name}
								{#if account.person.employee_number}({account.person.employee_number}){/if}
							</td>
							<td>{account.kind === 'admin' ? t('accountKindAdmin') : t('accountKindWork')}</td>
							<td>
								{#if editingId === account.id}
									<RoleEditor {roles} bind:grants={editGrants} />
								{:else if account.roles.length === 0}
									{t('accountNoRoles')}
								{:else}
									{account.roles.map(grantLabel).join(', ')}
								{/if}
							</td>
							<td>{accountStatus(account).join(', ')}</td>
							<td>
								<div class="actions">
									{#if editingId === account.id}
										<button type="button" disabled={busy} onclick={() => saveRoles(account)}>
											{t('accountSaveRoles')}
										</button>
										<button type="button" onclick={() => (editingId = null)}>
											{t('accountCancel')}
										</button>
									{:else}
										<!-- Akun admin selalu hanya memegang peran Admin (D-72). -->
										{#if account.kind === 'work'}
											<button type="button" disabled={busy} onclick={() => startEdit(account)}>
												{t('accountEditRoles')}
											</button>
										{/if}
										{#if !isOwn(account)}
											<button type="button" disabled={busy} onclick={() => resetPassword(account)}>
												{t('accountResetPassword')}
											</button>
											{#if account.disabled}
												<button type="button" disabled={busy} onclick={() => enable(account)}>
													{t('accountEnable')}
												</button>
											{:else}
												<button type="button" disabled={busy} onclick={() => disable(account)}>
													{t('accountDisable')}
												</button>
											{/if}
										{/if}
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>
{/if}

<style>
	section {
		margin-bottom: 2rem;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		max-width: 32rem;
	}

	fieldset {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	form button {
		align-self: flex-start;
	}

	.notice {
		border: 1px solid #888;
		padding: 0.5rem 1rem;
		margin-bottom: 1rem;
	}

	.error {
		color: #b00020;
		margin: 0;
	}

	table {
		border-collapse: collapse;
	}

	th,
	td {
		border: 1px solid #ccc;
		padding: 0.25rem 0.75rem;
		text-align: left;
		vertical-align: top;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
</style>
