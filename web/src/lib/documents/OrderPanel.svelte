<script lang="ts">
	// Urutan dan nomor operasi (M4.6, R-CMP-01). Setiap butir adalah satu
	// langkah: baris berurutan dengan proses yang sama. Langkah dipindah dengan
	// menyeretnya, atau dengan tombol panah (untuk keyboard dan layar sentuh).
	// Belum ada yang disimpan sebelum tombol Save order ditekan.
	import { untrack } from 'svelte';
	import type { Document, DocumentRow } from '#lib/api/gen/index.ts';
	import { t } from '#lib/i18n/index.ts';
	import { moveStep, stepsOf } from '#lib/order.ts';

	let {
		doc,
		onsave,
		oncancel
	}: {
		doc: Document;
		// onsave menyimpan urutan; true bila berhasil.
		onsave: (rowIds: string[], renumber: boolean) => Promise<boolean>;
		oncancel: () => void;
	} = $props();

	// Urutan yang sedang disusun; mulai dari urutan dokumen saat panel dibuka.
	let ordered = $state<DocumentRow[]>(untrack(() => [...doc.rows]));
	let renumber = $state(true);
	let busy = $state(false);
	let dragging = $state<number | null>(null);

	let steps = $derived(stepsOf(ordered));

	function move(from: number, to: number) {
		ordered = moveStep(ordered, from, to);
	}

	function drop(event: DragEvent, to: number) {
		event.preventDefault();
		if (dragging !== null) {
			move(dragging, to);
		}
		dragging = null;
	}

	function label(index: number, name: string): string {
		return `${index + 1}. ${name || t('orderNoProcess')}`;
	}

	async function save() {
		busy = true;
		await onsave(
			ordered.map((r) => r.id),
			renumber
		);
		busy = false;
	}
</script>

<h2>{t('orderTitle')}</h2>
<p class="hint">{t('orderHint')}</p>
{#if steps.length === 0}
	<p>{t('orderEmpty')}</p>
{:else}
	<ol class="steps">
		{#each steps as step, i (step.rowIds[0])}
			<li
				draggable="true"
				class:dragging={dragging === i}
				ondragstart={() => (dragging = i)}
				ondragend={() => (dragging = null)}
				ondragover={(event) => event.preventDefault()}
				ondrop={(event) => drop(event, i)}
			>
				<span class="grip" aria-hidden="true">⋮⋮</span>
				<span class="name">{step.processName || t('orderNoProcess')}</span>
				<span class="count">{t('orderStepRows', { n: String(step.rowIds.length) })}</span>
				<span class="number" data-testid="order-number">
					{#if renumber}
						{t('orderNumber', { number: step.next })}
					{:else}
						{step.current ? t('orderNumber', { number: step.current }) : t('orderNumberNone')}
					{/if}
				</span>
				<span class="arrows">
					<button type="button" disabled={i === 0} aria-label={t('orderMoveUp', { name: label(i, step.processName) })} onclick={() => move(i, i - 1)}>↑</button>
					<button
						type="button"
						disabled={i === steps.length - 1}
						aria-label={t('orderMoveDown', { name: label(i, step.processName) })}
						onclick={() => move(i, i + 1)}>↓</button
					>
				</span>
			</li>
		{/each}
	</ol>
	<p>
		<label class="check"><input type="checkbox" bind:checked={renumber} /> {t('orderRenumber')}</label>
	</p>
{/if}
<p class="actions">
	{#if steps.length > 0}<button type="button" class="primary" onclick={save} disabled={busy}>{t('orderSave')}</button>{/if}
	<button type="button" onclick={oncancel}>{t('composeClose')}</button>
</p>

<style>
	.steps {
		list-style: none;
		margin: 0 0 1rem;
		padding: 0;
		max-width: 44rem;
	}

	.steps li {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 12px;
		padding: 8px 12px;
		margin-bottom: 6px;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		cursor: grab;
	}

	.steps li.dragging {
		opacity: 0.5;
	}

	.grip {
		color: var(--text-3);
		letter-spacing: -2px;
	}

	.name {
		font-weight: 600;
		flex: 1 1 10rem;
	}

	.count,
	.number {
		color: var(--text-2);
		font-size: 13px;
	}

	.number {
		min-width: 5rem;
		font-family: 'Geist Mono Variable', monospace;
	}

	.arrows {
		display: inline-flex;
		gap: 4px;
	}

	.arrows button {
		min-height: 32px;
		padding: 0 10px;
	}

	.check {
		display: inline-flex;
		gap: 8px;
		align-items: baseline;
	}

	.actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
</style>
