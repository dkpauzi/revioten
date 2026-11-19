<script lang="ts">
	// Home (D-202, desain "Home v2"): sapaan, peringatan review 3 tahun, empat
	// pekerjaan yang paling sering, pekerjaan yang menunggu orang ini, dan
	// dokumen yang terakhir dibuka. Semua tautan menuju halaman yang memeriksa
	// hak sendiri; kartu dan daftar hanya menampilkan yang boleh dibuka.
	import { onMount } from 'svelte';
	import { getInbox, listReviewSets } from '#lib/api/client.ts';
	import type { Inbox, InboxTask, ReviewSetList } from '#lib/api/gen/index.ts';
	import HealthStatus from '#lib/HealthStatus.svelte';
	import { t, type MessageKey } from '#lib/i18n/index.ts';
	import { readRecent, type RecentDocument } from '#lib/recent.ts';
	import { can, session } from '#lib/session.svelte.ts';

	let review = $state<ReviewSetList | null>(null);
	let inbox = $state<Inbox | null>(null);
	let recent = $state<RecentDocument[]>([]);

	let firstName = $derived((session.current?.person.name ?? '').split(' ')[0]);
	let greeting = $derived.by(() => {
		const hour = new Date().getHours();
		return hour < 11 ? 'homeMorning' : hour < 15 ? 'homeAfternoon' : 'homeEvening';
	});

	type Action = { href: string; title: MessageKey; text: MessageKey; icon: string; tone: string; show: boolean };
	let actions = $derived<Action[]>(
		[
			{ href: '/numbers', title: 'homeNewSet', text: 'homeNewSetHint', icon: 'plus', tone: 'accent', show: can('E-1') },
			{ href: '/documents', title: 'homeRevise', text: 'homeReviseHint', icon: 'pen', tone: 'success', show: can('L-4') },
			{ href: '/documents/released', title: 'homeFind', text: 'homeFindHint', icon: 'search', tone: 'warning', show: can('L-1') },
			{ href: '/documents', title: 'homePrint', text: 'homePrintHint', icon: 'print', tone: 'muted', show: can('E-5') }
		].filter((a) => a.show) as Action[]
	);

	const kindLabel: Record<InboxTask['kind'], MessageKey> = {
		review: 'homeKindReview',
		verify: 'homeKindVerify',
		confirm_review: 'homeKindConfirm',
		release: 'homeKindRelease',
		yokoten: 'homeKindYokoten'
	};

	function taskText(task: InboxTask): string {
		switch (task.kind) {
			case 'review':
				return task.days > 0 ? t('homeTaskReviewLate', { days: String(task.days) }) : t('homeTaskReviewSoon', { days: String(-task.days) });
			case 'verify':
				return t('homeTaskVerify', { detail: task.detail ?? '' });
			case 'confirm_review':
				return t('homeTaskConfirm', { by: task.detail ?? '' });
			case 'release':
				return t('homeTaskRelease');
			case 'yokoten':
				return t('homeTaskYokoten', { decided: task.detail ?? '' });
		}
	}

	onMount(async () => {
		recent = readRecent();
		const [sets, tasks] = await Promise.all([listReviewSets({}), getInbox()]);
		review = sets.ok ? sets.data : null;
		inbox = tasks.ok ? tasks.data : null;
	});
</script>

<h1>{t(greeting as MessageKey, { name: firstName })}</h1>
<p class="lead">{t('homeLead')}</p>

{#if review && review.overdue + review.due_soon > 0}
	<section class="warning" aria-labelledby="home-review" data-testid="review-summary">
		<span class="icon" aria-hidden="true">
			<svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3l9.5 17h-19z" /><path d="M12 10v4" /><path d="M12 17.5v.01" /></svg>
		</span>
		<div class="text">
			<h2 id="home-review">{t('homeReviewTitle', { count: String(review.overdue + review.due_soon) })}</h2>
			<p>{t('homeReviewText', { overdue: String(review.overdue), soon: String(review.due_soon), days: String(review.warning_days) })}</p>
		</div>
		<span class="badge" data-review="overdue">{t('reviewCountOverdue', { count: String(review.overdue) })}</span>
		<span class="badge" data-review="due_soon">{t('reviewCountDueSoon', { count: String(review.due_soon) })}</span>
		<a class="button danger" href="/periodic-reviews">{t('homeOpenReviewList')}</a>
	</section>
{/if}

{#if actions.length > 0}
	<div class="actions">
		{#each actions as a (a.title)}
			<a class="action" href={a.href}>
				<span class="action-icon" data-tone={a.tone} aria-hidden="true">
					<svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						{#if a.icon === 'plus'}<path d="M12 5v14M5 12h14" />
						{:else if a.icon === 'pen'}<path d="M4 20h4L19 9l-4-4L4 16z" /><path d="M14 6l4 4" />
						{:else if a.icon === 'search'}<circle cx="11" cy="11" r="6.5" /><path d="M16 16l4.5 4.5" />
						{:else}<path d="M7 9V4h10v5" /><rect x="4" y="9" width="16" height="7" rx="2" /><path d="M7 14h10v6H7z" />{/if}
					</svg>
				</span>
				<span class="action-title">{t(a.title)}</span>
				<span class="action-text">{t(a.text)}</span>
			</a>
		{/each}
	</div>
{/if}

<div class="pair">
	<section class="panel" aria-labelledby="home-waiting">
		<h2 id="home-waiting">{t('homeWaiting', { count: String(inbox?.total ?? 0) })}</h2>
		{#if !inbox || inbox.items.length === 0}
			<p class="empty">{t('homeWaitingNone')}</p>
		{:else}
			<ul class="tasks">
				{#each inbox.items as task, i (task.kind + task.link + i)}
					<li>
						<span class="badge kind" data-kind={task.kind}>{t(kindLabel[task.kind])}</span>
						<span class="task">
							<strong>{task.title}</strong>
							<span class="sub">{taskText(task)}</span>
						</span>
						<a class="button" class:primary={task.kind === 'review' || task.kind === 'verify'} href={task.link}>{t('homeOpen')}</a>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
	<section class="panel" aria-labelledby="home-recent">
		<h2 id="home-recent">{t('homeRecent')}</h2>
		{#if recent.length === 0}
			<p class="empty">{t('homeRecentNone')}</p>
		{:else}
			<ul class="recent">
				{#each recent as d (d.id)}
					<li>
						<a class="mono" href={d.view === 'released' ? `/documents/released/view?id=${d.id}` : `/documents/view?id=${d.id}`}>{d.number}</a>
						<span>{d.title}</span>
						<span class="badge" data-status={d.status === 'released' ? 'released' : 'in_use'}>
							{t(d.status === 'released' ? 'homeRecentReleased' : 'homeRecentEditing', { revision: String(d.revision).padStart(2, '0') })}
						</span>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>

<section class="guide" aria-label={t('homeGuideTitle')}>
	<span class="icon" aria-hidden="true">
		<svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9.5" /><path d="M9.5 9.5a2.5 2.5 0 1 1 3.5 2.3c-.6.3-1 .8-1 1.5v.7" /><path d="M12 17v.01" /></svg>
	</span>
	<p><strong>{t('homeGuideTitle')}</strong> {t('homeGuideText')}</p>
	<a class="button" href="/help">{t('homeGuideOpen')}</a>
</section>

<div class="health">
	<HealthStatus />
</div>

<style>
	.lead {
		margin-top: -8px;
		color: var(--text-3);
		font-size: 18px;
	}

	.warning {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px 16px;
		margin: 24px 0;
		padding: 18px 24px;
		background: var(--danger-weak);
		border: 1.5px solid var(--danger-ring);
		border-radius: var(--radius-lg);
	}

	.warning .icon {
		display: inline-flex;
		padding: 10px;
		border-radius: var(--radius);
		background: var(--danger-ring);
		color: var(--danger);
	}

	.warning .text {
		flex: 1 1 320px;
	}

	.warning h2 {
		margin: 0;
		font-size: 19px;
	}

	.warning p {
		margin: 4px 0 0;
		color: var(--danger);
	}

	a.button.danger {
		background: var(--danger);
		border-color: var(--danger);
		color: #ffffff;
	}

	.actions {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 16px;
		margin-bottom: 24px;
	}

	.action {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 22px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
		color: inherit;
		text-decoration: none;
	}

	.action:hover {
		border-color: var(--accent);
		box-shadow: 0 6px 20px rgba(67, 56, 202, 0.1);
	}

	.action-icon {
		display: inline-flex;
		width: fit-content;
		padding: 12px;
		border-radius: var(--radius);
	}

	.action-icon[data-tone='accent'] {
		background: var(--accent-weak);
		color: var(--accent);
	}

	.action-icon[data-tone='success'] {
		background: var(--success-weak);
		color: var(--success);
	}

	.action-icon[data-tone='warning'] {
		background: var(--warning-weak);
		color: var(--warning);
	}

	.action-icon[data-tone='muted'] {
		background: var(--surface-2);
		color: var(--text-2);
	}

	.action-title {
		font-size: 19px;
		font-weight: 600;
		color: var(--ink);
	}

	.action-text {
		color: var(--text-3);
	}

	.pair {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
		gap: 16px;
		margin-bottom: 24px;
	}

	.panel {
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
		min-width: 0;
	}

	.panel h2 {
		margin: 0;
		padding: 18px 24px;
		font-size: 19px;
		border-bottom: 1px solid var(--border);
	}

	.empty {
		padding: 18px 24px;
		color: var(--text-3);
	}

	.tasks,
	.recent {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.tasks li,
	.recent li {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px;
		padding: 14px 24px;
		border-bottom: 1px solid var(--border-soft);
	}

	.task {
		flex: 1 1 200px;
		display: flex;
		flex-direction: column;
	}

	.sub {
		color: var(--text-3);
		font-size: 14px;
	}

	.recent li span:nth-child(2) {
		flex: 1 1 160px;
	}

	.badge.kind[data-kind='review'],
	.badge.kind[data-kind='verify'] {
		background: var(--danger-weak);
		color: var(--danger);
	}

	.badge.kind[data-kind='yokoten'] {
		background: var(--warning-weak);
		color: var(--warning);
	}

	.badge.kind[data-kind='confirm_review'],
	.badge.kind[data-kind='release'] {
		background: var(--accent-weak);
		color: var(--accent-ink);
	}

	.guide {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px 16px;
		margin-bottom: 24px;
		padding: 18px 24px;
		background: var(--surface);
		border: 1.5px solid var(--border);
		border-radius: var(--radius-lg);
	}

	.guide .icon {
		color: var(--accent);
		display: inline-flex;
	}

	.guide p {
		flex: 1 1 320px;
		margin: 0;
		color: var(--text-3);
	}

	.guide strong {
		color: var(--ink);
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
