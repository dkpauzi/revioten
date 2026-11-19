<script lang="ts">
	// Halaman Roles (M8.1, D-66, D-199): matriks hak per peran, gaya Spatie
	// Laravel Permission. Admin mencentang hak mana yang dipegang setiap peran
	// kerja, menambah peran, mengganti namanya, dan menonaktifkan peran yang
	// tidak dipegang akun mana pun. Katalog hak tetap. Aturan yang tidak bisa
	// dimatikan (R-SEC-11) juga tampak di layar: peran Admin dan Viewer tidak
	// bisa diubah, hak admin (A-) tidak bisa diberikan ke peran kerja, dan hak
	// dasar L-1 sampai L-3 tidak bisa dicabut. Server dan database tetap
	// memeriksa semuanya.
	import { onMount } from 'svelte';
	import { createRole, listPermissions, listRoles, setRoleActive, setRolePermissions, updateRole, type ApiResult } from '#lib/api/client.ts';
	import type { PermissionInfo, Role } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { roleLabel } from '#lib/roles.ts';

	let roles = $state<Role[]>([]);
	let permissions = $state<PermissionInfo[]>([]);
	// Pilihan di layar, per ID peran.
	let chosen = $state<Record<string, string[]>>({});
	let showInactive = $state(false);
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');
	let busy = $state(false);

	// Isian peran baru.
	let draft = $state({ nameEn: '', nameId: '', groupScope: false, copyFrom: '' });
	// Peran yang sedang diganti namanya.
	let renaming = $state<{ id: string; nameEn: string; nameId: string; groupScope: boolean } | null>(null);

	const basic = ['L-1', 'L-2', 'L-3'];

	let shown = $derived(roles.filter((r) => r.active || showInactive));
	let inactiveCount = $derived(roles.filter((r) => !r.active).length);

	// Nama hak di katalog teks: L-1 menjadi permL1, E-10 menjadi permE10.
	function permissionLabel(code: string): string {
		return t(`perm${code.replace('-', '')}` as MessageKey);
	}

	function fixedRole(role: Role): boolean {
		return role.id === 'admin' || role.id === 'viewer';
	}

	// Kotak yang tidak bisa diubah: peran tetap, hak admin, dan hak dasar.
	function locked(role: Role, code: string): boolean {
		return fixedRole(role) || code.startsWith('A-') || basic.includes(code);
	}

	function changed(role: Role): boolean {
		const now = chosen[role.id] ?? [];
		return now.length !== role.permissions.length || now.some((c) => !role.permissions.includes(c));
	}

	async function load() {
		const [roleResult, permissionResult] = await Promise.all([listRoles(), listPermissions()]);
		if (!roleResult.ok) {
			loadError = failureText(roleResult.error);
			return;
		}
		if (!permissionResult.ok) {
			loadError = failureText(permissionResult.error);
			return;
		}
		roles = roleResult.data.items;
		permissions = permissionResult.data.items;
		chosen = Object.fromEntries(roles.map((r) => [r.id, [...r.permissions]]));
	}

	function toggle(role: Role, code: string, checked: boolean) {
		const now = chosen[role.id] ?? [];
		chosen[role.id] = checked ? [...now, code] : now.filter((c) => c !== code);
	}

	// replace menaruh peran hasil server ke daftar, menggantikan versi lamanya.
	function replace(role: Role) {
		roles = roles.some((r) => r.id === role.id) ? roles.map((r) => (r.id === role.id ? role : r)) : [...roles, role];
		chosen[role.id] = [...role.permissions];
	}

	// run menjalankan satu aksi dan menulis hasilnya sebagai pesan.
	async function run<T>(action: () => Promise<ApiResult<T>>, done: (data: T) => string): Promise<boolean> {
		message = '';
		failure = '';
		busy = true;
		const result = await action();
		busy = false;
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return false;
		}
		message = done(result.data);
		return true;
	}

	function save(role: Role) {
		return run(
			() => setRolePermissions(role.id, chosen[role.id] ?? []),
			(r) => {
				replace(r);
				return t('rolesSaved', { role: roleLabel(r) });
			}
		);
	}

	async function add(event: SubmitEvent) {
		event.preventDefault();
		const ok = await run(
			() =>
				createRole({
					name_en: draft.nameEn,
					name_id: draft.nameId,
					group_scope: draft.groupScope,
					copy_from: draft.copyFrom || undefined
				}),
			(r) => {
				replace(r);
				return t('rolesAdded', { role: roleLabel(r) });
			}
		);
		if (ok) {
			draft = { nameEn: '', nameId: '', groupScope: false, copyFrom: '' };
		}
	}

	function startRename(role: Role) {
		renaming = { id: role.id, nameEn: role.name_en, nameId: role.name_id, groupScope: role.group_scope };
	}

	async function saveRename(event: SubmitEvent) {
		event.preventDefault();
		if (!renaming) return;
		const edit = renaming;
		const ok = await run(
			() => updateRole(edit.id, { name_en: edit.nameEn, name_id: edit.nameId, group_scope: edit.groupScope }),
			(r) => {
				replace(r);
				return t('rolesRenamed', { role: roleLabel(r) });
			}
		);
		if (ok) renaming = null;
	}

	function setActive(role: Role, active: boolean) {
		if (!active && !confirm(t('rolesDeactivateConfirm', { role: roleLabel(role) }))) return;
		return run(
			() => setRoleActive(role.id, active),
			(r) => {
				replace(r);
				return t(active ? 'rolesActivated' : 'rolesDeactivated', { role: roleLabel(r) });
			}
		);
	}

	onMount(load);
</script>

<h1>{t('rolesTitle')}</h1>
<p class="hint">{t('rolesHint')}</p>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}
{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

{#if roles.length > 0}
	{#if inactiveCount > 0}
		<label class="inline">
			<input type="checkbox" bind:checked={showInactive} />
			{t('rolesShowInactive')} ({inactiveCount})
		</label>
	{/if}
	<div class="scroll">
		<table>
			<thead>
				<tr>
					<th>{t('rolesPermission')}</th>
					{#each shown as role (role.id)}
						<th class="cell">
							{roleLabel(role)}
							<small class="holders">{t('rolesHolders', { count: String(role.holders) })}</small>
							{#if !role.active}<span class="badge">{t('rolesInactive')}</span>{/if}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each permissions as p (p.code)}
					<tr>
						<td><span class="mono">{p.code}</span> {permissionLabel(p.code)}</td>
						{#each shown as role (role.id)}
							<td class="cell">
								<input
									type="checkbox"
									aria-label="{roleLabel(role)}: {p.code}"
									checked={(chosen[role.id] ?? []).includes(p.code)}
									disabled={locked(role, p.code)}
									onchange={(event) => toggle(role, p.code, event.currentTarget.checked)}
								/>
							</td>
						{/each}
					</tr>
				{/each}
				<tr>
					<td></td>
					{#each shown as role (role.id)}
						<td class="cell">
							{#if !fixedRole(role)}
								<button type="button" disabled={busy || !changed(role)} onclick={() => save(role)} aria-label={t('rolesSaveFor', { role: roleLabel(role) })}>
									{t('rolesSave')}
								</button>
							{/if}
						</td>
					{/each}
				</tr>
				<tr>
					<td>{t('rolesActions')}</td>
					{#each shown as role (role.id)}
						<td class="cell actions">
							{#if !fixedRole(role)}
								<button type="button" class="link" disabled={busy} onclick={() => startRename(role)} aria-label={t('rolesRenameFor', { role: roleLabel(role) })}>
									{t('rolesRename')}
								</button>
								{#if role.active}
									<button
										type="button"
										class="link"
										disabled={busy || role.holders > 0}
										title={role.holders > 0 ? t('rolesHeldCannotDeactivate') : undefined}
										onclick={() => setActive(role, false)}
										aria-label={t('rolesDeactivateFor', { role: roleLabel(role) })}
									>
										{t('rolesDeactivate')}
									</button>
								{:else}
									<button type="button" class="link" disabled={busy} onclick={() => setActive(role, true)} aria-label={t('rolesActivateFor', { role: roleLabel(role) })}>
										{t('rolesActivate')}
									</button>
								{/if}
							{/if}
						</td>
					{/each}
				</tr>
			</tbody>
		</table>
	</div>

	{#if renaming}
		<form class="panel" onsubmit={saveRename}>
			<h2>{t('rolesRenameFor', { role: roleLabel(renaming.id, roles) })}</h2>
			<label for="rename-en">{t('rolesNameEn')}</label>
			<input id="rename-en" bind:value={renaming.nameEn} maxlength="60" required autocomplete="off" />
			<label for="rename-id">{t('rolesNameId')}</label>
			<input id="rename-id" bind:value={renaming.nameId} maxlength="60" required autocomplete="off" />
			<label class="inline"><input type="checkbox" bind:checked={renaming.groupScope} /> {t('rolesGroupScope')}</label>
			<div class="buttons">
				<button type="submit" class="primary" disabled={busy}>{t('rolesRenameSave')}</button>
				<button type="button" onclick={() => (renaming = null)}>{t('rolesCancel')}</button>
			</div>
		</form>
	{/if}

	<form class="panel" onsubmit={add}>
		<h2>{t('rolesAddTitle')}</h2>
		<p class="hint">{t('rolesAddHint')}</p>
		<label for="role-en">{t('rolesNameEn')}</label>
		<input id="role-en" bind:value={draft.nameEn} maxlength="60" required autocomplete="off" />
		<label for="role-id">{t('rolesNameId')}</label>
		<input id="role-id" bind:value={draft.nameId} maxlength="60" required autocomplete="off" />
		<label for="role-copy">{t('rolesCopyFrom')}</label>
		<select id="role-copy" bind:value={draft.copyFrom}>
			<option value="">{t('rolesCopyNone')}</option>
			{#each roles.filter((r) => r.active && r.id !== 'admin') as role (role.id)}
				<option value={role.id}>{roleLabel(role)}</option>
			{/each}
		</select>
		<label class="inline"><input type="checkbox" bind:checked={draft.groupScope} /> {t('rolesGroupScope')}</label>
		<div class="buttons">
			<button type="submit" class="primary" disabled={busy}>{t('rolesAdd')}</button>
		</div>
	</form>
{/if}

<style>
	.scroll {
		overflow-x: auto;
	}

	.cell {
		text-align: center;
	}

	.holders {
		display: block;
		font-weight: normal;
	}

	.actions {
		white-space: nowrap;
	}

	.actions button {
		display: block;
		margin: 0 auto;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}

	.panel {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		max-width: 32rem;
		margin-top: 1.5rem;
	}

	.inline {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.buttons {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.5rem;
	}
</style>
