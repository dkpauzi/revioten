<script lang="ts">
	import HealthStatus from '#lib/HealthStatus.svelte';
	import { t } from '#lib/i18n/index.ts';
	import { buildMenu } from '#lib/menu.ts';
	import { can } from '#lib/session.svelte.ts';

	// Ubin di beranda mengikuti kelompok menu atas (D-127), jadi hanya menuju
	// halaman yang sudah dibangun dan yang boleh dibuka akun ini. Ubin untuk
	// fitur tahap berikutnya (misalnya membuat set dokumen) belum dipasang.
	let tiles = $derived(buildMenu(can).filter((group) => group.id !== 'home'));
</script>

<h1>{t('homeTitle')}</h1>

<div class="tiles">
	{#each tiles as group (group.id)}
		<section class="tile" aria-labelledby="tile-{group.id}">
			<h2 id="tile-{group.id}">{t(group.label)}</h2>
			<ul>
				{#each group.items as item (item.href)}
					<li><a href={item.href}>{t(item.label)}</a></li>
				{/each}
			</ul>
		</section>
	{/each}
</div>

<div class="health">
	<HealthStatus />
</div>

<style>
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 16px;
		margin-bottom: 28px;
	}

	.tile {
		padding: 22px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
	}

	.tile:hover {
		border-color: var(--accent);
		box-shadow: 0 6px 20px rgba(67, 56, 202, 0.1);
	}

	.tile h2 {
		font-size: 18px;
	}

	.tile ul {
		display: flex;
		flex-direction: column;
		gap: 6px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.health {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px;
		color: var(--muted);
		font-size: 14px;
	}

	.health :global(p) {
		margin: 0;
	}
</style>
