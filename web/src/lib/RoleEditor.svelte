<script lang="ts">
	// Pilihan peran untuk akun kerja (M1.2): satu kotak centang per peran, dan
	// pilihan cakupan untuk peran yang boleh dibatasi ke Project Group sendiri.
	import type { Role, RoleGrant } from './api/gen/index.ts';
	import { t } from './i18n/index.ts';
	import { assignableRoles, groupScopedRoles, roleLabel } from './roles.ts';

	// roles adalah katalog peran dari server. grants adalah peran yang dipilih.
	// $bindable membuat induk bisa menulis bind:grants={...}: perubahan di sini
	// langsung terlihat di variabel milik induk.
	let { roles, grants = $bindable() }: { roles: Role[]; grants: RoleGrant[] } = $props();

	function grantOf(roleId: string): RoleGrant | undefined {
		return grants.find((g) => g.role === roleId);
	}

	// toggle menambah atau membuang satu peran. Peran baru bercakupan semua customer.
	function toggle(roleId: string, checked: boolean) {
		grants = checked
			? [...grants, { role: roleId, scope: 'all' }]
			: grants.filter((g) => g.role !== roleId);
	}

	function setScope(roleId: string, scope: RoleGrant['scope']) {
		grants = grants.map((g) => (g.role === roleId ? { ...g, scope } : g));
	}
</script>

<fieldset>
	<legend>{t('accountRoles')}</legend>
	<small>{t('accountRolesHint')}</small>
	{#each assignableRoles(roles) as role (role.id)}
		{@const grant = grantOf(role.id)}
		<div class="role">
			<label>
				<input
					type="checkbox"
					checked={grant !== undefined}
					onchange={(event) => toggle(role.id, event.currentTarget.checked)}
				/>
				{roleLabel(role.id)}
			</label>
			{#if grant && groupScopedRoles.includes(role.id)}
				<select
					aria-label={t('accountRoleScope', { role: roleLabel(role.id) })}
					value={grant.scope}
					onchange={(event) =>
						setScope(role.id, event.currentTarget.value as RoleGrant['scope'])}
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
	}
</style>
