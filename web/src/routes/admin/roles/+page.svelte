<script lang="ts">
	// Halaman Roles (M8.1, D-66, R-SEC-10): admin mengatur hak mana yang
	// dipegang setiap peran kerja. Katalog hak tetap. Aturan yang tidak bisa
	// dimatikan (R-SEC-11) juga tampak di layar: peran Admin dan Viewer tidak
	// bisa diubah, hak admin (A-) tidak bisa diberikan ke peran kerja, dan hak
	// dasar L-1 sampai L-3 tidak bisa dicabut. Server dan database tetap
	// memeriksa semuanya.
	import { onMount } from 'svelte';
	import { listPermissions, listRoles, setRolePermissions } from '#lib/api/client.ts';
	import type { PermissionInfo, Role } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { roleLabel } from '#lib/roles.ts';

	let roles = $state<Role[]>([]);
	let permissions = $state<PermissionInfo[]>([]);
	// Pilihan di layar, per ID peran.
	let chosen = $state<Record<string, string[]>>({});
	let loadError = $state('');
	let message = $state('');
	let failure = $state('');

	const basic = ['L-1', 'L-2', 'L-3'];

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

	async function save(role: Role) {
		message = '';
		failure = '';
		const result = await setRolePermissions(role.id, chosen[role.id] ?? []);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			failure = details.length > 0 ? problemText(details[0]) : failureText(result.error);
			return;
		}
		roles = roles.map((r) => (r.id === role.id ? result.data : r));
		chosen[role.id] = [...result.data.permissions];
		message = t('rolesSaved', { role: roleLabel(role.id) });
	}

	onMount(load);
</script>

<h1>{t('rolesTitle')}</h1>
<p class="hint">{t('rolesHint')}</p>

{#if loadError}<p class="error" role="alert">{loadError}</p>{/if}
{#if message}<p role="status">{message}</p>{/if}
{#if failure}<p class="error" role="alert">{failure}</p>{/if}

{#if roles.length > 0}
	<div class="scroll">
		<table>
			<thead>
				<tr>
					<th>{t('rolesPermission')}</th>
					{#each roles as role (role.id)}
						<th>{roleLabel(role.id)}</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each permissions as p (p.code)}
					<tr>
						<td><span class="mono">{p.code}</span> {permissionLabel(p.code)}</td>
						{#each roles as role (role.id)}
							<td class="cell">
								<input
									type="checkbox"
									aria-label="{roleLabel(role.id)}: {p.code}"
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
					{#each roles as role (role.id)}
						<td class="cell">
							{#if !fixedRole(role)}
								<button type="button" disabled={!changed(role)} onclick={() => save(role)} aria-label={t('rolesSaveFor', { role: roleLabel(role.id) })}>
									{t('rolesSave')}
								</button>
							{/if}
						</td>
					{/each}
				</tr>
			</tbody>
		</table>
	</div>
{/if}

<style>
	.scroll {
		overflow-x: auto;
	}

	.cell {
		text-align: center;
	}

	.mono {
		font-family: 'Geist Mono Variable', monospace;
	}
</style>
