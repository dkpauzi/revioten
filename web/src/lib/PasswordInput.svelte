<script lang="ts">
	// Kotak password dengan tombol mata (D-202): tekan untuk melihat ketikan,
	// tekan lagi untuk menyamarkannya. Tempel (paste) tetap boleh
	// (ASVS 6.2.6, 6.2.7). Nama tombol untuk pembaca layar diambil dari
	// katalog: "Show password" atau "Hide password".
	import { t } from '#lib/i18n/index.ts';

	let {
		id,
		value = $bindable(''),
		autocomplete,
		name,
		invalid = false,
		describedby
	}: {
		id: string;
		value?: string;
		autocomplete: 'current-password' | 'new-password';
		name?: string;
		invalid?: boolean;
		describedby?: string;
	} = $props();

	let shown = $state(false);
</script>

<div class="password">
	<input
		{id}
		{name}
		type={shown ? 'text' : 'password'}
		{autocomplete}
		aria-invalid={invalid ? 'true' : undefined}
		aria-describedby={describedby}
		bind:value
	/>
	<!-- type=button supaya tombol ini tidak mengirim formulir. -->
	<button
		type="button"
		class="eye"
		aria-label={shown ? t('loginHide') : t('loginShow')}
		aria-pressed={shown}
		aria-controls={id}
		title={shown ? t('loginHide') : t('loginShow')}
		onclick={() => (shown = !shown)}
	>
		{#if shown}
			<!-- Mata dicoret: password sedang terlihat, tekan untuk menyembunyikan. -->
			<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
				<path d="M3 3l18 18" />
				<path d="M10.6 5.1A10.5 10.5 0 0 1 12 5c5 0 9 4.5 10 7-0.4 1-1.3 2.4-2.6 3.7" />
				<path d="M6.6 6.6C4.4 8 2.8 10.1 2 12c1 2.5 5 7 10 7 1.9 0 3.6-0.6 5.1-1.5" />
				<path d="M9.9 9.9a3 3 0 0 0 4.2 4.2" />
			</svg>
		{:else}
			<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
				<path d="M2 12c1-2.5 5-7 10-7s9 4.5 10 7c-1 2.5-5 7-10 7S3 14.5 2 12z" />
				<circle cx="12" cy="12" r="3" />
			</svg>
		{/if}
	</button>
</div>

<style>
	.password {
		position: relative;
	}

	.password input {
		width: 100%;
		padding-right: 48px;
		box-sizing: border-box;
	}

	.eye {
		position: absolute;
		top: 50%;
		right: 4px;
		transform: translateY(-50%);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		padding: 0;
		border: none;
		background: transparent;
		color: var(--text-2);
		box-shadow: none;
	}

	.eye:hover {
		color: var(--text);
		background: transparent;
	}
</style>
