<script lang="ts">
	// Baris versi satu template (M3.4, R-TPL-05): memilih versi, menerbitkan
	// versi draft, dan membuat versi baru. Dipakai halaman format template
	// (A-2) dan process template (E-6); halaman itulah yang memanggil API.
	import { formatDateTime } from '#lib/format.ts';
	import { t } from '#lib/i18n/index.ts';
	import { canStartNewVersion, isDraft, versionOptionText, type VersionInfo } from '#lib/versions.ts';

	let {
		id,
		versions,
		current,
		canEdit,
		canPublish = undefined,
		onselect,
		onpublish,
		onnew
	}: {
		id: string; // awalan id elemen, supaya label unik di halaman
		versions: VersionInfo[]; // semua versi template, menurut nomor
		current: VersionInfo; // versi yang sedang ditampilkan
		canEdit: boolean; // akun ini boleh mengubah template
		// Akun ini boleh menerbitkan. Biasanya sama dengan canEdit; versi bermetode
		// AP diterbitkan penyetuju form (D-139).
		canPublish?: boolean;
		onselect: (versionId: string) => void;
		onpublish: () => void;
		onnew: () => void;
	} = $props();

	// publish meminta konfirmasi dulu, karena versi terbit tidak bisa diubah lagi.
	function publish() {
		if (confirm(t('versionPublishConfirm', { version: String(current.version) }))) {
			onpublish();
		}
	}
</script>

<div class="bar">
	<div class="field">
		<label for="{id}-version">{t('versionChoose')}</label>
		<select id="{id}-version" value={current.id} onchange={(event) => onselect(event.currentTarget.value)}>
			{#each versions as v (v.id)}
				<option value={v.id}>{versionOptionText(v)}</option>
			{/each}
		</select>
	</div>
	<span class="badge" data-status={current.status}>
		{isDraft(current) ? t('versionStatusDraft') : t('versionStatusPublished')}
	</span>
	{#if (canPublish ?? canEdit) && isDraft(current)}
		<button type="button" class="primary" onclick={publish}>
			{t('versionPublish', { version: String(current.version) })}
		</button>
	{/if}
	{#if canEdit && canStartNewVersion(versions)}
		<button type="button" onclick={onnew}>{t('versionNew')}</button>
	{/if}
</div>
{#if current.published_at}
	<p class="hint">{t('versionPublishedOn', { time: formatDateTime(current.published_at) })}</p>
{:else if canEdit}
	<p class="hint">{t('versionDraftHint')}</p>
{/if}

<style>
	.bar {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: flex-start;
	}

	/* Label status dan tombol diturunkan setinggi label field, supaya sejajar
	   dengan kotak pilihan versi. */
	.bar > button {
		margin-top: var(--label-space);
	}

	.bar > .badge {
		margin-top: calc(var(--label-space) + 8px);
	}
</style>
