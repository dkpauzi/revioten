<script lang="ts">
	// Halaman nomor dokumen (Tahap 2, UC-10): mendaftarkan nomor PFD, PFMEA,
	// dan CP. Nomor dialokasikan server di dalam transaksi (R-NUM-03); layar
	// hanya mengirim permintaan beserta Idempotency-Key (M2.2). Menu dan tombol
	// mengikuti hak akun; server tetap memeriksa setiap operasi.
	import { onMount } from 'svelte';
	import {
		getNextDocumentNumber,
		listCustomers,
		listDocumentTypes,
		registerDocumentNumber,
		registerDocumentNumberSet
	} from '#lib/api/client.ts';
	import type { Customer, DocumentType } from '#lib/api/gen/index.ts';
	import { failureText, problemText } from '#lib/errors.ts';
	import { t } from '#lib/i18n/index.ts';
	import { registeredText, skippedText } from '#lib/numbers.ts';
	import RegisterSection from '#lib/numbers/RegisterSection.svelte';
	import { can } from '#lib/session.svelte.ts';

	// Data dari server. null berarti belum selesai dimuat.
	let customers = $state<Customer[] | null>(null);
	let types = $state<DocumentType[]>([]);
	let loadError = $state('');
	let canRegister = $derived(can('E-1'));
	let canManage = $derived(can('E-8'));
	// Dinaikkan setiap ada nomor baru, supaya register dimuat ulang.
	let refresh = $state(0);
	// Hanya customer aktif yang mendapat nomor baru (ASUMSI E-67).
	let activeCustomers = $derived((customers ?? []).filter((c) => c.active));

	// Formulir pendaftaran. Daftar sebagai set dicentang dulu, karena sebagian
	// besar dokumen satu model berbagi satu nomor (R-NUM-05, M2.4).
	let asSet = $state(true);
	let customerId = $state('');
	let typeId = $state('');
	// Kotak angka memberi null bila dikosongkan; null dikirim sebagai 0.
	let initialRevision = $state<number | null>(0);
	let next = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let message = $state('');
	let skipped = $state('');
	let failure = $state('');
	// Idempotency-Key untuk pendaftaran yang sedang diisi. Kunci yang sama
	// dipakai lagi bila pendaftaran ini dikirim ulang (misalnya setelah jaringan
	// putus), dan diganti setelah berhasil, supaya tidak ada nomor ganda.
	let key = $state(crypto.randomUUID());

	async function load() {
		// Dua permintaan dijalankan bersamaan, lalu ditunggu keduanya.
		const [customerResult, typeResult] = await Promise.all([listCustomers(), listDocumentTypes()]);
		if (!customerResult.ok) {
			loadError = failureText(customerResult.error);
			return;
		}
		if (!typeResult.ok) {
			loadError = failureText(typeResult.error);
			return;
		}
		customers = customerResult.data.items;
		// Sistem hanya menomori PFD, PFMEA, dan CP (R-NUM-01).
		types = typeResult.data.items.filter((d) => d.key !== undefined);
		loadError = '';
	}

	// showNext menampilkan nomor yang akan didapat bila customer dan jenis
	// dokumen sudah dipilih. Nomor sebenarnya ditentukan saat mendaftar.
	async function showNext() {
		next = '';
		if (asSet || customerId === '' || typeId === '') {
			return;
		}
		const result = await getNextDocumentNumber(customerId, typeId);
		if (result.ok) {
			next = result.data.number;
		}
	}

	async function register(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		message = '';
		skipped = '';
		failure = '';
		const revision = initialRevision ?? 0;
		const result = asSet
			? await registerDocumentNumberSet({ customer_id: customerId, initial_revision: revision }, key)
			: await registerDocumentNumber({ customer_id: customerId, document_type_id: typeId, initial_revision: revision }, key);
		if (!result.ok) {
			const details = result.error.body?.details ?? [];
			fieldErrors = Object.fromEntries(details.map((d) => [d.field, problemText(d)]));
			if (details.length === 0) {
				failure = failureText(result.error);
			}
			return;
		}
		message = t('numberRegistered', { numbers: registeredText(result.data.items) });
		skipped = skippedText(result.data.items);
		key = crypto.randomUUID();
		refresh += 1;
		await showNext();
	}

	onMount(() => {
		load();
	});
</script>

<h1>{t('numbersTitle')}</h1>
<p>{t('numbersIntro')}</p>
{#if canManage}
	<!-- Impor nomor lama untuk dokumen kontrol (E-8, M2.5). -->
	<p><a href="/numbers/import">{t('numberImportLink')}</a></p>
{/if}

{#if loadError}
	<p class="error" role="alert">{loadError}</p>
{:else if customers === null}
	<p>{t('numbersLoading')}</p>
{:else if canRegister}
	<section>
		<h2>{t('numberRegisterTitle')}</h2>
		{#if message}<p role="status">{message}</p>{/if}
		{#if skipped}<p>{t('numberSkipped', { numbers: skipped })}</p>{/if}
		{#if failure}<p class="error" role="alert">{failure}</p>{/if}
		{#if fieldErrors.idempotency_key}<p class="error" role="alert">{fieldErrors.idempotency_key}</p>{/if}
		<form class="add" onsubmit={register} novalidate>
			<div class="field">
				<label for="number-customer">{t('numberCustomer')}</label>
				<select id="number-customer" bind:value={customerId} onchange={showNext}>
					<option value="">{t('numberCustomerChoose')}</option>
					{#each activeCustomers as c (c.id)}
						<option value={c.id}>{c.code} · {c.name}</option>
					{/each}
				</select>
				{#if fieldErrors.customer_id}<p class="error">{fieldErrors.customer_id}</p>{/if}
			</div>
			{#if !asSet}
				<div class="field">
					<label for="number-type">{t('numberType')}</label>
					<select id="number-type" bind:value={typeId} onchange={showNext}>
						<option value="">{t('numberTypeChoose')}</option>
						{#each types as d (d.id)}
							<option value={d.id}>{d.name} ({d.prefix})</option>
						{/each}
					</select>
					{#if fieldErrors.document_type_id}<p class="error">{fieldErrors.document_type_id}</p>{/if}
				</div>
			{/if}
			<div class="field">
				<label for="number-revision">{t('numberInitialRevision')}</label>
				<input id="number-revision" type="number" min="0" bind:value={initialRevision} />
				{#if fieldErrors.initial_revision}<p class="error">{fieldErrors.initial_revision}</p>{/if}
			</div>
			<button type="submit">{t('numberRegister')}</button>
		</form>
		<label class="check">
			<input type="checkbox" bind:checked={asSet} onchange={showNext} />
			{t('numberAsSet')}
		</label>
		<p class="hint">{asSet ? t('numberSetHint') : t('numberInitialRevisionHint')}</p>
		{#if next && !asSet}<p>{t('numberNext', { number: next })}</p>{/if}
	</section>
{/if}

{#if customers !== null}
	<RegisterSection {customers} {types} {canManage} {refresh} />
{/if}

<style>
	section {
		margin-bottom: 2rem;
	}

	.add {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	/* Field dirapikan ke atas; tombol diturunkan setinggi label supaya sejajar
	   dengan kotak isian, juga bila ada teks bantuan atau pesan masalah. */
	.add > button {
		margin-top: var(--label-space);
	}

	.check {
		display: block;
		margin-top: 0.5rem;
	}
</style>
