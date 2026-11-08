<script lang="ts">
	// Formulir rute approval (M1.8, UC-06): nama dan slot berurutan. Setiap slot
	// punya tahap, judul cetak (R-APR-17), dan jabatan yang boleh mengisinya.
	// Bentuk rute (Prepared, lalu Checked, lalu Approved) diperiksa server.
	import type { ApiResult } from '../api/client.ts';
	import type { ApprovalRoute, ApprovalRouteInput, PositionRef, RouteSlotInput } from '../api/gen/index.ts';
	import { failureText, problemText } from '../errors.ts';
	import { t } from '../i18n/index.ts';
	import { moveSlot, stageLabel } from '../routes.ts';

	let {
		initial,
		positions,
		save,
		oncancel,
		onsaved
	}: {
		initial: ApprovalRouteInput;
		positions: PositionRef[];
		// save menyimpan isian lewat API (tambah atau ubah).
		save: (input: ApprovalRouteInput) => Promise<ApiResult<ApprovalRoute>>;
		oncancel: () => void;
		onsaved: (route: ApprovalRoute) => void;
	} = $props();

	// Salinan isian, supaya perubahan di formulir tidak menyentuh daftar rute
	// sebelum disimpan. Hanya nilai awal yang dipakai.
	// svelte-ignore state_referenced_locally
	let name = $state(initial.name);
	// svelte-ignore state_referenced_locally
	let slots = $state<RouteSlotInput[]>(initial.slots.map((s) => ({ ...s, position_ids: [...s.position_ids] })));
	let fieldErrors = $state<Record<string, string>>({});
	let failure = $state('');

	const stages = ['prepared', 'checked', 'approved'] as const;

	function togglePosition(slot: RouteSlotInput, id: string) {
		slot.position_ids = slot.position_ids.includes(id)
			? slot.position_ids.filter((p) => p !== id)
			: [...slot.position_ids, id];
	}

	function addSlot() {
		slots = [...slots, { stage: 'approved', print_title: '', position_ids: [] }];
	}

	function removeSlot(index: number) {
		slots = slots.filter((_, i) => i !== index);
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		failure = '';
		const result = await save({ name, slots });
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d, 'route')]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		onsaved(result.data);
	}
</script>

<form onsubmit={submit} novalidate>
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}
	<div class="field">
		<label for="route-name">{t('routeName')}</label>
		<input id="route-name" bind:value={name} autocomplete="off" />
		{#if fieldErrors.name}<p class="error">{fieldErrors.name}</p>{/if}
	</div>

	<table>
		<thead>
			<tr>
				<th>{t('routeSlotOrder')}</th>
				<th>{t('routeSlotStage')}</th>
				<th>{t('routeSlotPrintTitle')}</th>
				<th>{t('routeSlotPositions')}</th>
				<th>{t('masterActions')}</th>
			</tr>
		</thead>
		<tbody>
			{#each slots as slot, index (slot)}
				{@const number = String(index + 1)}
				<tr>
					<td>{number}</td>
					<td>
						<select aria-label={`${t('routeSlotStage')} ${number}`} bind:value={slot.stage}>
							{#each stages as stage (stage)}
								<option value={stage}>{stageLabel(stage)}</option>
							{/each}
						</select>
					</td>
					<td>
						<input
							aria-label={`${t('routeSlotPrintTitle')} ${number}`}
							bind:value={slot.print_title}
							autocomplete="off"
						/>
					</td>
					<td>
						<fieldset>
							<legend class="hidden">{t('routeSlotPositions')} {number}</legend>
							{#each positions as position (position.id)}
								<label>
									<input
										type="checkbox"
										checked={slot.position_ids.includes(position.id)}
										onchange={() => togglePosition(slot, position.id)}
									/>
									{position.name}
								</label>
							{/each}
						</fieldset>
					</td>
					<td>
						<button
							type="button"
							aria-label={t('routeSlotUp', { number })}
							onclick={() => (slots = moveSlot(slots, index, -1))}>↑</button
						>
						<button
							type="button"
							aria-label={t('routeSlotDown', { number })}
							onclick={() => (slots = moveSlot(slots, index, 1))}>↓</button
						>
						<button type="button" aria-label={t('routeSlotRemove', { number })} onclick={() => removeSlot(index)}
							>✕</button
						>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
	{#each ['slots', 'print_title', 'position_ids'] as field (field)}
		{#if fieldErrors[field]}<p class="error">{fieldErrors[field]}</p>{/if}
	{/each}

	<div class="actions">
		<button type="button" onclick={addSlot}>{t('routeSlotAdd')}</button>
		<button type="submit">{t('masterSave')}</button>
		<button type="button" onclick={oncancel}>{t('masterCancel')}</button>
	</div>
</form>

<style>
	.field {
		max-width: 28rem;
		margin-bottom: 1rem;
	}

	fieldset {
		border: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
	}

	.hidden {
		position: absolute;
		left: -10000px;
	}

	.actions {
		display: flex;
		gap: 0.5rem;
		margin-top: 1rem;
	}
</style>
