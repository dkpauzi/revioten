<script lang="ts">
	import { t } from '#lib/i18n/index.ts';
	import { language, languages, setLanguage } from '#lib/i18n/language.svelte.ts';

	// Pengalih bahasa EN/ID (D-128), tampil di bar atas di semua halaman,
	// termasuk halaman login. aria-pressed memberi tahu pembaca layar bahasa
	// mana yang aktif.
</script>

<div class="switch" role="group" aria-label={t('languageLabel')}>
	{#each languages as code (code)}
		<button
			type="button"
			lang={code}
			aria-pressed={language.current === code}
			onclick={() => setLanguage(code)}
		>
			{code.toUpperCase()}
		</button>
	{/each}
</div>

<style>
	.switch {
		display: inline-flex;
		border: 1px solid var(--border-strong);
		border-radius: var(--radius);
		overflow: hidden;
	}

	button {
		min-height: 36px;
		padding: 0 12px;
		border: 0;
		border-radius: 0;
		background: var(--surface);
		color: var(--text-3);
		font-size: 13px;
		font-weight: 600;
	}

	button[aria-pressed='true'] {
		background: var(--ink);
		color: #ffffff;
	}
</style>
