<script lang="ts">
	// Halaman rute approval (M1.8, UC-06). Konsep: alur kerja sebagai data.
	// Rute berisi slot berurutan (D-45 sampai D-49); setiap customer punya rute
	// default (D-47), dan setiap jenis perubahan boleh punya rute sendiri
	// (R-TPL-10, ASUMSI E-35). Semua akun bisa melihat; mengubah hanya untuk
	// pemegang A-2.
	import { onMount } from 'svelte';
	import {
		createApprovalRoute,
		listApprovalRoutes,
		listChangeTypes,
		listCustomers,
		listPositions,
		setChangeTypeRoute,
		setCustomerDefaultRoute,
		updateApprovalRoute
	} from '#lib/api/client.ts';
	import type {
		ApprovalRoute,
		ApprovalRouteInput,
		ChangeType,
		Customer,
		PositionRef
	} from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import RouteEditor from '#lib/master/RouteEditor.svelte';
	import { positionNames, routeInputOf, stageLabel } from '#lib/routes.ts';
	import { can } from '#lib/session.svelte.ts';

	// Daftar dari server. null berarti belum selesai dimuat.
	let routes = $state<ApprovalRoute[] | null>(null);
	let positions = $state<PositionRef[]>([]);
	let customers = $state<Customer[]>([]);
	let changeTypes = $state<ChangeType[]>([]);
	let loadError = $state('');
	let canEdit = $derived(can('A-2'));

	// Rute yang sedang diubah ('new' untuk rute baru), atau null.
	let editing = $state<string | null>(null);
	let message = $state('');
	let failure = $state('');

	// Isian awal rute baru: tiga slot sesuai bentuk rute.
	const newRoute: ApprovalRouteInput = {
		name: '',
		slots: [
			{ stage: 'prepared', print_title: '', position_ids: [] },
			{ stage: 'checked', print_title: '', position_ids: [] },
			{ stage: 'approved', print_title: '', position_ids: [] }
		]
	};

	async function load() {
		const [routeResult, positionResult, customerResult, typeResult] = await Promise.all([
			listApprovalRoutes(),
			listPositions(),
			listCustomers(),
			listChangeTypes()
		]);
		if (!routeResult.ok) {
			loadError = failureText(routeResult.error);
			return;
		}
		if (!positionResult.ok) {
			loadError = failureText(positionResult.error);
			return;
		}
		if (!customerResult.ok) {
			loadError = failureText(customerResult.error);
			return;
		}
		if (!typeResult.ok) {
			loadError = failureText(typeResult.error);
			return;
		}
		routes = routeResult.data.items;
		positions = positionResult.data.items.map((p) => ({ id: p.id, name: p.name }));
		customers = customerResult.data.items;
		changeTypes = typeResult.data.items;
		loadError = '';
	}

	async function saved(route: ApprovalRoute) {
		message = t('routeSaved', { name: route.name });
		editing = null;
		await load();
	}

	// setDefaultRoute menyimpan rute default customer begitu pilihan diganti.
	async function setDefaultRoute(customer: Customer, routeId: string) {
		message = '';
		failure = '';
		const result = await setCustomerDefaultRoute(customer.id, routeId);
		if (!result.ok) {
			const detail = result.error.body?.details?.[0];
			failure = detail ? problemText(detail) : failureText(result.error);
			return;
		}
		message = t('defaultRouteSaved', { code: result.data.code, route: result.data.default_route?.name ?? '' });
		await load();
	}

	// setTypeRoute menyimpan rute satu jenis perubahan. '' berarti rute default customer.
	async function setTypeRoute(changeType: ChangeType, routeId: string) {
		message = '';
		failure = '';
		const result = await setChangeTypeRoute(changeType.id, routeId === '' ? null : routeId);
		if (!result.ok) {
			const detail = result.error.body?.details?.[0];
			failure = detail ? problemText(detail) : failureText(result.error);
			return;
		}
		message = t('changeTypeRouteSaved', { name: result.data.name });
		await load();
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('routesTitle')}</h1>
<p>{t('routesIntro')}</p>

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if routes === null}
	<p>{t('routesLoading')}</p>
{:else}
	{#if message}<p role="status">{message}</p>{/if}
	{#if failure}<p class="error" role="alert">{failure}</p>{/if}

	{#each routes as route (route.id)}
		<section>
			<h2>{route.name}</h2>
			{#if editing === route.id}
				<RouteEditor
					initial={routeInputOf(route)}
					{positions}
					save={(input) => updateApprovalRoute(route.id, input)}
					oncancel={() => (editing = null)}
					onsaved={saved}
				/>
			{:else}
				<table>
					<thead>
						<tr>
							<th>{t('routeSlotOrder')}</th>
							<th>{t('routeSlotStage')}</th>
							<th>{t('routeSlotPrintTitle')}</th>
							<th>{t('routeSlotPositions')}</th>
						</tr>
					</thead>
					<tbody>
						{#each route.slots as slot, index (index)}
							<tr>
								<td>{index + 1}</td>
								<td>{stageLabel(slot.stage)}</td>
								<td>{slot.print_title}</td>
								<td>{positionNames(slot)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
				{#if canEdit && editing === null}
					<button type="button" onclick={() => (editing = route.id)}>{t('routeEdit')}</button>
				{/if}
			{/if}
		</section>
	{/each}

	{#if canEdit}
		{#if editing === 'new'}
			<section>
				<h2>{t('routeAddTitle')}</h2>
				<RouteEditor
					initial={newRoute}
					{positions}
					save={(input) => createApprovalRoute(input)}
					oncancel={() => (editing = null)}
					onsaved={saved}
				/>
			</section>
		{:else if editing === null}
			<button type="button" onclick={() => (editing = 'new')}>{t('routeAdd')}</button>
		{/if}
	{/if}

	<section>
		<h2>{t('defaultRoutesTitle')}</h2>
		<p>{t('defaultRoutesIntro')}</p>
		<table>
			<thead>
				<tr>
					<th>{t('customerCode')}</th>
					<th>{t('customerName')}</th>
					<th>{t('defaultRoutesTitle')}</th>
				</tr>
			</thead>
			<tbody>
				{#each customers as customer (customer.id)}
					<tr>
						<td>{customer.code}</td>
						<td>{customer.name}</td>
						<td>
							{#if canEdit}
								<select
									aria-label={t('defaultRouteChoose', { code: customer.code })}
									value={customer.default_route?.id ?? ''}
									onchange={(e) => setDefaultRoute(customer, e.currentTarget.value)}
								>
									{#if !customer.default_route}
										<option value="">{t('defaultRouteNone')}</option>
									{/if}
									{#each routes as route (route.id)}
										<option value={route.id}>{route.name}</option>
									{/each}
								</select>
							{:else}
								{customer.default_route?.name ?? t('defaultRouteNone')}
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>

	<section>
		<h2>{t('changeTypeRoutesTitle')}</h2>
		<p>{t('changeTypeRoutesIntro')}</p>
		<table>
			<thead>
				<tr>
					<th>{t('changeTypesListTitle')}</th>
					<th>{t('routesTitle')}</th>
				</tr>
			</thead>
			<tbody>
				{#each changeTypes as changeType (changeType.id)}
					<tr>
						<td>{changeType.name}</td>
						<td>
							{#if canEdit}
								<select
									aria-label={t('changeTypeRouteChoose', { name: changeType.name })}
									value={changeType.approval_route?.id ?? ''}
									onchange={(e) => setTypeRoute(changeType, e.currentTarget.value)}
								>
									<option value="">{t('changeTypeRouteDefault')}</option>
									{#each routes as route (route.id)}
										<option value={route.id}>{route.name}</option>
									{/each}
								</select>
							{:else}
								{changeType.approval_route?.name ?? t('changeTypeRouteDefault')}
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>
{/if}

<style>
	section {
		margin-bottom: 2rem;
	}

	.error {
		color: #b00020;
		margin: 0;
	}

	table {
		border-collapse: collapse;
		margin-bottom: 0.5rem;
	}

	th,
	td {
		border: 1px solid #ccc;
		padding: 0.25rem 0.75rem;
		text-align: left;
		vertical-align: top;
	}
</style>
