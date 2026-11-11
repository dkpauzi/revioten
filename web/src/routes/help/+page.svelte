<script lang="ts">
	// Halaman Help: buku panduan pengguna di dalam aplikasi (D-141), dalam dua
	// bahasa. Isinya ada di #lib/help/en.ts dan id.ts. Nama tombol diambil
	// dari katalog teks layar, jadi selalu sama dengan layarnya. Gambar
	// dipotret otomatis: npm --prefix web run help:shots.
	import { parts, type HelpContent } from '#lib/help/content.ts';
	import { helpEn } from '#lib/help/en.ts';
	import { helpId } from '#lib/help/id.ts';
	import { t } from '#lib/i18n/index.ts';
	import { language } from '#lib/i18n/language.svelte.ts';

	const contents: Record<string, HelpContent> = { en: helpEn, id: helpId };
	let help = $derived(contents[language.current] ?? helpEn);
</script>

{#snippet rich(text: string)}
	{#each parts(text, (key) => t(key)) as part, i (i)}{#if part.strong}<strong>{part.text}</strong>{:else}{part.text}{/if}{/each}
{/snippet}

<h1>{help.title}</h1>
<p>{@render rich(help.intro)}</p>
<p class="no-print"><button type="button" onclick={() => window.print()}>{help.print}</button></p>

<section class="contents">
	<h2>{help.contentsTitle}</h2>
	<ol>
		{#each help.chapters as chapter (chapter.id)}
			<li><a href="#{chapter.id}">{chapter.title}</a> <span class="who">{chapter.who}</span></li>
		{/each}
	</ol>
</section>

{#each help.chapters as chapter, n (chapter.id)}
	<section id={chapter.id} class="chapter">
		<h2>{n + 1}. {chapter.title}</h2>
		<p class="who"><span class="badge">{help.whoLabel}: {chapter.who}</span></p>
		<p>{@render rich(chapter.intro)}</p>
		<ol>
			{#each chapter.steps as step, i (i)}
				<li>{@render rich(step)}</li>
			{/each}
		</ol>
		{#if chapter.image}
			<figure>
				<img src="/help/{language.current}/{chapter.image}.png" alt={chapter.imageAlt} loading="lazy" />
			</figure>
		{/if}
		{#if chapter.tips}
			<h3>{help.tipsTitle}</h3>
			<ul>
				{#each chapter.tips as tip, i (i)}
					<li>{@render rich(tip)}</li>
				{/each}
			</ul>
		{/if}
	</section>
{/each}

<style>
	.contents ol {
		margin: 0;
		padding-left: 1.5rem;
		line-height: 1.9;
	}

	.who {
		color: var(--muted);
		font-size: 13px;
	}

	.chapter ol,
	.chapter ul {
		padding-left: 1.5rem;
		line-height: 1.7;
		max-width: 60rem;
	}

	/* Jangkar tidak tertutup bilah atas saat dituju dari daftar isi. */
	.chapter {
		scroll-margin-top: 80px;
	}

	figure {
		margin: 1rem 0;
	}

	img {
		display: block;
		max-width: 100%;
		height: auto;
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}

	/* Cetak: setiap bab tidak terbelah bila masih muat di satu halaman. */
	@media print {
		.chapter {
			break-inside: avoid-page;
			border: none;
		}
	}
</style>
