<script lang="ts">
	// Pilihan peran untuk akun kerja (M1.2, D-199): satu kotak centang per
	// peran aktif, dan pilihan cakupan untuk peran yang boleh dibatasi ke
	// Project Group sendiri (group_scope, misalnya Project Leader).
	import type { Role, RoleGrant } from './api/gen/index.ts';
	import { t } from './i18n/index.ts';
	import { assignableRoles, roleLabel } from './roles.ts';

	// roles adalah katalog peran dari server. grants adalah peran yang dipilih.
	// $bindable membuat induk bisa menulis bind:grants={...}: perubahan di sini
	// langsung terlihat di variabel milik induk.
	let { roles, grants = $bindable() }: { roles: Role[]; grants: RoleGrant[] } = $props();

	// Peran yang dipegang saat editor dibuka, supaya peran nonaktif yang masih
	// dipegang tetap tampil dan bisa dilepas.
	const held = [...grants];

	function grantOf(roleId: string): RoleGrant | undefined {
		return grants.find((g) => g.role === roleId);
	}

	// toggle menambah atau membuang satu peran. Peran bercakupan Project Group
	// (Project Leader) bawaannya untuk Project Group sendiri (D-71).
	function toggle(role: Role, checked: boolean) {
		const scope: RoleGrant['scope'] = role.group_scope ? 'own_project_group' : 'all';
		grants = checked ? [...grants, { role: role.id, scope }] : grants.filter((g) => g.role !== role.id);
	}

	function setScope(roleId: string, scope: RoleGrant['scope']) {
		grants = grants.map((g) => (g.role === roleId ? { ...g, scope } : g));
	}
</script>

<fieldset>
	<legend>{t('accountRoles')}</legend>
	<small>{t('accountRolesHint')}</small>
	{#each assignableRoles(roles, held) as role (role.id)}
		{@const grant = grantOf(role.id)}
		<div class="role">
			<label>
				<input type="checkbox" checked={grant !== undefined} onchange={(event) => toggle(role, event.currentTarget.checked)} />
				{roleLabel(role)}
				{#if !role.active}<span class="badge">{t('rolesInactive')}</span>{/if}
			</label>
			{#if grant && role.group_scope}
				<select
					aria-label={t('accountRoleScope', { role: roleLabel(role) })}
					value={grant.scope}
					onchange={(event) => setScope(role.id, event.currentTarget.value as RoleGrant['scope'])}
				>
					<option value="all">{t('scopeAll')}</option>
					<option value="own_project_group">{t('scopeOwnProjectGroup')}</option>
				</select>
			{/if}
		</div>
	{/each}
</fieldset>

<style>
	fieldset {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.role {
		display: flex;
		gap: 0.75rem;
		align-items: baseline;
		flex-wrap: wrap;
	}
</style>
