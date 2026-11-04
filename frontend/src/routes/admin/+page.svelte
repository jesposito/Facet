<script lang="ts">
	import type { ListResult } from 'pocketbase';
	import { pb, type Experience, type Project, type View } from '$lib/pocketbase';
	import { collection, demoMode } from '$lib/stores/demo';
	import { onMount, onDestroy, tick } from 'svelte';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import FacetUseCases from '$components/admin/FacetUseCases.svelte';

	type Timestamped = { updated?: string };
	type DashboardView = View & Timestamped & { view_count?: number; last_viewed_at?: string };

	interface SiteMetrics {
		homepage_view_count?: number;
		homepage_last_viewed_at?: string;
	}

	// The SDK cancels by method and path unless given a key, so without these the
	// layout's and sidebar's calls to the same collections cancel each other's.
	const REQUEST_KEYS = {
		projects: 'admin-dashboard-projects',
		experience: 'admin-dashboard-experience',
		views: 'admin-dashboard-views',
		proposals: 'admin-dashboard-proposals'
	};

	// null means "not known yet", which is different from a confirmed zero.
	let projects = $state<ListResult<Project & Timestamped> | null>(null);
	let experience = $state<ListResult<Experience & Timestamped> | null>(null);
	let views = $state<DashboardView[] | null>(null);
	let pendingProposals = $state<number | null>(null);
	let pendingProposalId = $state('');
	let siteMetrics = $state<SiteMetrics | null>(null);
	let loading = $state(true);
	let loadFailed = $state(false);
	let heading: HTMLHeadingElement | null = $state(null);

	let loadSequence = 0;
	let settingsRequest: AbortController | null = null;

	// Simple pattern - admin layout handles auth
	onMount(() => {
		loadDashboard();
	});

	onDestroy(() => {
		loadSequence += 1;
		settingsRequest?.abort();
		Object.values(REQUEST_KEYS).forEach((key) => pb.cancelRequest(key));
	});

	async function fetchSiteMetrics(signal: AbortSignal): Promise<SiteMetrics> {
		const response = await fetch('/api/site-settings', { signal });
		if (!response.ok) {
			throw new Error(`Site settings request failed with status ${response.status}`);
		}
		return response.json();
	}

	async function loadDashboard() {
		const sequence = ++loadSequence;
		settingsRequest?.abort();
		settingsRequest = new AbortController();
		loading = true;
		const timeout = setTimeout(() => {
			if (sequence !== loadSequence) return;
			settingsRequest?.abort();
			Object.values(REQUEST_KEYS).forEach((key) => pb.cancelRequest(key));
		}, 20_000);

		// Demo collections were cloned before created/updated existed.
		const recentSort = $demoMode ? {} : { sort: '-updated' };

		const results = await Promise.allSettled([
			collection('projects').getList<Project & Timestamped>(1, 3, {
				...recentSort,
				requestKey: REQUEST_KEYS.projects
			}),
			collection('experience').getList<Experience & Timestamped>(1, 3, {
				...recentSort,
				requestKey: REQUEST_KEYS.experience
			}),
			collection('views').getFullList<DashboardView>({
				sort: '-view_count',
				requestKey: REQUEST_KEYS.views
			}),
			pb.collection('import_proposals').getList(1, 1, {
				filter: "status = 'pending'",
				requestKey: REQUEST_KEYS.proposals
			}),
			fetchSiteMetrics(settingsRequest.signal)
		]);
		clearTimeout(timeout);

		// Left the page, or a newer load took over.
		if (sequence !== loadSequence) return;

		const [projectsResult, experienceResult, viewsResult, proposalsResult, settingsResult] =
			results;

		// Each source updates on success only, so a failed retry keeps what's known.
		if (projectsResult.status === 'fulfilled') projects = projectsResult.value;
		if (experienceResult.status === 'fulfilled') experience = experienceResult.value;
		if (viewsResult.status === 'fulfilled') views = viewsResult.value;
		if (proposalsResult.status === 'fulfilled') {
			pendingProposals = proposalsResult.value.totalItems;
			pendingProposalId = proposalsResult.value.items[0]?.id ?? '';
		}
		if (settingsResult.status === 'fulfilled') siteMetrics = settingsResult.value;

		const failures = results.filter((result) => result.status === 'rejected');
		failures.forEach((failure) => console.error('Failed to load dashboard data:', failure.reason));
		loadFailed = failures.length > 0;
		loading = false;
	}

	async function retryDashboard() {
		if (loading) return;
		await loadDashboard();
		if (!loadFailed) {
			// The Retry button is gone, so keep keyboard focus somewhere useful.
			await tick();
			heading?.focus();
		}
	}

	let isEmpty = $derived(
		!loadFailed && projects?.totalItems === 0 && experience?.totalItems === 0 && views?.length === 0
	);

	let showSkeleton = $derived(
		loading &&
			!loadFailed &&
			projects === null &&
			experience === null &&
			views === null &&
			pendingProposals === null &&
			siteMetrics === null
	);

	let homepageViewCount = $derived(siteMetrics?.homepage_view_count ?? 0);

	let metrics = $derived([
		{ label: $t('admin.dashboard.stats_projects'), value: projects?.totalItems ?? null },
		{ label: $t('admin.dashboard.stats_experience'), value: experience?.totalItems ?? null },
		{
			label: $t('admin.dashboard.stats_total_visitors'),
			value:
				views && siteMetrics
					? views.reduce((sum, view) => sum + (view.view_count ?? 0), homepageViewCount)
					: null
		},
		{ label: $t('admin.dashboard.pending_reviews'), value: pendingProposals }
	]);

	let recentFacets = $derived(
		[...(views ?? [])]
			.sort((a, b) => parseTimestamp(b.updated) - parseTimestamp(a.updated))
			.slice(0, 4)
	);

	let recentActivity = $derived(
		[
			...(projects?.items ?? []).map((p) => ({
				type: 'project',
				title: p.title,
				id: p.id,
				updated: p.updated
			})),
			...(experience?.items ?? []).map((e) => ({
				type: 'experience',
				title: `${e.title} at ${e.company}`,
				id: e.id,
				updated: e.updated
			}))
		]
			.sort((a, b) => parseTimestamp(b.updated) - parseTimestamp(a.updated))
			.slice(0, 5)
	);

	// PocketBase separates date and time with a space, which only some browsers parse.
	function parseTimestamp(dateStr?: string): number {
		if (!dateStr) return 0;
		const parsed = Date.parse(dateStr.replace(' ', 'T'));
		return Number.isNaN(parsed) ? 0 : parsed;
	}

	function formatRelativeDate(dateStr?: string): string {
		const timestamp = parseTimestamp(dateStr);
		if (!timestamp) return '';
		const date = new Date(timestamp);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMs / 3600000);
		const diffDays = Math.floor(diffMs / 86400000);

		if (diffMins < 1) return $t('admin.dashboard.relative_time.just_now');
		if (diffMins < 60)
			return $t('admin.dashboard.relative_time.minutes_ago', { values: { count: diffMins } });
		if (diffHours < 24)
			return $t('admin.dashboard.relative_time.hours_ago', { values: { count: diffHours } });
		if (diffDays < 30)
			return $t('admin.dashboard.relative_time.days_ago', { values: { count: diffDays } });
		return date.toLocaleDateString();
	}
</script>

<svelte:head>
	<title>{$t('admin.dashboard.title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-6xl mx-auto">
	<header class="flex flex-wrap items-center justify-between gap-3 mb-4">
		<h1
			bind:this={heading}
			tabindex="-1"
			class="text-2xl font-bold text-gray-900 dark:text-white outline-none"
		>
			{isEmpty ? $t('admin.dashboard.welcome_title') : $t('admin.dashboard.title')}
		</h1>
		<a href="/admin/views/new" class="btn btn-primary">{$t('admin.views.create_facet')}</a>
	</header>

	{#if loadFailed}
		<div
			class="mb-4 card p-4 border-l-4 border-red-500 bg-red-50 dark:bg-red-900/20"
			role="alert"
			aria-busy={loading}
		>
			<div class="flex flex-wrap items-center gap-3">
				<div class="flex-1 min-w-[12rem]">
					<p class="font-medium text-red-800 dark:text-red-200">
						{$t('admin.dashboard.load_error_title')}
					</p>
					<p class="text-sm text-red-700 dark:text-red-300">
						{$t('admin.dashboard.load_error_message')}
					</p>
				</div>
				<!-- aria-disabled, not disabled: a disabled button would drop keyboard focus. -->
				<button
					type="button"
					class="btn btn-sm btn-secondary"
					aria-disabled={loading}
					onclick={retryDashboard}
				>
					{loading ? $t('admin.dashboard.retrying') : $t('admin.dashboard.retry')}
				</button>
			</div>
		</div>
	{/if}

	{#if isEmpty}
		<!-- Welcome copy for first-time users -->
		<div class="mb-6">
			<p class="text-gray-600 dark:text-gray-400">
				{@html $t('admin.dashboard.welcome_description', {
					values: {
						profile_link: `<a href="/admin/homepage" class="text-primary-600 dark:text-primary-400 hover:underline">${$t('admin.dashboard.adding_profile')}</a>`,
						import_link: `<a href="/admin/import" class="text-primary-600 dark:text-primary-400 hover:underline">${$t('admin.dashboard.import_project')}</a>`
					}
				})}
			</p>
			<p class="text-gray-500 dark:text-gray-300 text-sm mt-2">
				{$t('admin.dashboard.no_rush')}
			</p>
		</div>
	{/if}

	<!-- Pending proposals alert -->
	{#if pendingProposals !== null && pendingProposals > 0}
		<div class="mb-4 card p-4 border-l-4 border-yellow-500 bg-yellow-50 dark:bg-yellow-900/20">
			<div class="flex flex-wrap items-center gap-3">
				<svg
					class="w-6 h-6 text-yellow-600 dark:text-yellow-400"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
					/>
				</svg>
				<div class="flex-1 min-w-[12rem]">
					<p class="font-medium text-yellow-800 dark:text-yellow-200">
						{$t('admin.dashboard.pending_alert', { values: { count: pendingProposals } })}
					</p>
				</div>
				<a
					href="/admin/review/{pendingProposalId}"
					class="btn btn-sm bg-yellow-700 text-white hover:bg-yellow-800"
				>
					{$t('admin.dashboard.review_now')}
				</a>
			</div>
		</div>
	{/if}

	{#if showSkeleton}
		<!-- Loading skeleton that matches content structure -->
		<div
			class="animate-fade-in space-y-4 mb-6"
			role="status"
			aria-label={$t('admin.layout.loading')}
		>
			<div class="card p-4 space-y-3">
				{#each Array(3) as _}
					<div class="h-10 bg-gray-200 dark:bg-gray-700 rounded relative overflow-hidden">
						<div class="absolute inset-0 animate-shimmer"></div>
					</div>
				{/each}
			</div>
			<div class="card p-4 grid grid-cols-2 lg:grid-cols-4 gap-4">
				{#each Array(4) as _}
					<div class="h-12 bg-gray-200 dark:bg-gray-700 rounded relative overflow-hidden">
						<div class="absolute inset-0 animate-shimmer"></div>
					</div>
				{/each}
			</div>
		</div>
	{:else}
		<!-- Existing Facets: the owner's work, ahead of the numbers -->
		{#if recentFacets.length > 0}
			<section class="card mb-4" aria-labelledby="dashboard-facets-heading">
				<div class="flex items-center justify-between gap-3 px-4 pt-3 pb-2">
					<h2
						id="dashboard-facets-heading"
						class="text-lg font-semibold text-gray-900 dark:text-white"
					>
						{$t('admin.views.title')}
					</h2>
					<a
						href="/admin/views"
						class="text-sm text-primary-700 dark:text-primary-400 hover:underline"
					>
						{$t('admin.dashboard.manage_facets')}
					</a>
				</div>
				<ul class="divide-y divide-gray-100 dark:divide-gray-700">
					{#each recentFacets as facet (facet.id)}
						<li class="px-4 py-2 min-w-0">
							<a
								href="/admin/views/{facet.id}"
								class="block truncate text-sm font-medium text-gray-900 dark:text-white hover:text-primary-600 dark:hover:text-primary-400"
							>
								{facet.name}
							</a>
							<p class="truncate text-xs text-gray-500 dark:text-gray-400">
								/{facet.slug} · {$t(
									`admin.views.filter_visibility_${facet.visibility}`
								)}{facet.is_active ? '' : ` · ${$t('admin.views.pill_inactive')}`}
							</p>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		<!-- Stats (overview band under soft-premium: flat, typographic) -->
		{#if metrics.some((metric) => metric.value !== null)}
			<dl class="stat-band card grid grid-cols-2 lg:grid-cols-4 gap-x-4 gap-y-3 p-4 mb-6">
				{#each metrics as metric (metric.label)}
					<div class="stat-cell min-w-0">
						<dt class="stat-label text-sm text-gray-500 dark:text-gray-400">{metric.label}</dt>
						<dd class="stat-figure text-xl font-bold tabular-nums text-gray-900 dark:text-white">
							{#if metric.value === null}
								<span aria-hidden="true">—</span>
								<span class="sr-only">{$t('admin.dashboard.metric_unavailable')}</span>
							{:else}
								{metric.value.toLocaleString()}
							{/if}
						</dd>
					</div>
				{/each}
			</dl>
		{/if}

		<FacetUseCases
			variant={views?.length === 0 ? 'full' : 'compact'}
			showCreate={views?.length === 0}
		/>
	{/if}

	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
		<!-- Quick actions -->
		<div class="card p-4 sm:p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
				{$t('admin.dashboard.quick_actions')}
			</h2>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
				<a href="/admin/projects?new=true" class="group btn btn-secondary justify-start">
					<svg
						class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:scale-110"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 4v16m8-8H4"
						/>
					</svg>
					{$t('admin.dashboard.add_project')}
				</a>
				<a href="/admin/experience?new=true" class="group btn btn-secondary justify-start">
					<svg
						class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:scale-110"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 4v16m8-8H4"
						/>
					</svg>
					{$t('admin.dashboard.add_experience')}
				</a>
				<a href="/admin/import" class="group btn btn-secondary justify-start">
					<svg
						class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:-translate-y-0.5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
						/>
					</svg>
					{$t('admin.dashboard.import_github')}
				</a>
				<a href="/rss.xml" target="_blank" class="group btn btn-secondary justify-start">
					<svg
						class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:scale-110"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 5c7.18 0 13 5.82 13 13M6 11a7 7 0 017 7M6 17a1 1 0 11-2 0 1 1 0 012 0z"
						/>
					</svg>
					{$t('admin.dashboard.rss_feed')}
				</a>
				<a href="/talks.ics" class="group btn btn-secondary justify-start">
					<svg
						class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:scale-110"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
						/>
					</svg>
					{$t('admin.dashboard.talks_calendar')}
				</a>
			</div>
		</div>

		<!-- Recent activity -->
		{#if projects || experience}
			<div class="card p-4 sm:p-6">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
					{$t('admin.dashboard.recent_activity')}
				</h2>
				{#if recentActivity.length === 0}
					{#if projects && experience}
						<p class="text-gray-500 dark:text-gray-400 text-sm">
							{$t('admin.dashboard.nothing_yet')}
						</p>
					{/if}
				{:else}
					<ul class="divide-y divide-gray-100 dark:divide-gray-700">
						{#each recentActivity as activity (`${activity.type}-${activity.id}`)}
							<li class="py-2 min-w-0">
								<p class="text-sm font-medium text-gray-900 dark:text-white truncate">
									{activity.title}
								</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">
									{activity.type === 'project'
										? $t('admin.dashboard.activity_project')
										: $t('admin.dashboard.activity_experience')}{activity.updated
										? ` · ${formatRelativeDate(activity.updated)}`
										: ''}
								</p>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Visitor stats -->
	{#if views && siteMetrics && (views.length > 0 || homepageViewCount > 0)}
		<div class="mt-6 card p-4 sm:p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
				{$t('admin.dashboard.visitor_stats')}
			</h2>
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gray-200 dark:border-gray-700">
							<th
								scope="col"
								class="text-left py-2 pr-4 text-gray-500 dark:text-gray-400 font-medium"
								>{$t('admin.dashboard.visitor_view_name')}</th
							>
							<th
								scope="col"
								class="text-right py-2 px-4 text-gray-500 dark:text-gray-400 font-medium"
								>{$t('admin.dashboard.visitor_count')}</th
							>
							<th
								scope="col"
								class="text-right py-2 pl-4 text-gray-500 dark:text-gray-400 font-medium"
								>{$t('admin.dashboard.visitor_last_visited')}</th
							>
						</tr>
					</thead>
					<tbody>
						<tr class="border-b border-gray-100 dark:border-gray-800">
							<td class="py-2.5 pr-4">
								<a
									href="/admin/homepage"
									class="text-gray-900 dark:text-white hover:text-primary-600 dark:hover:text-primary-400 flex items-center gap-1.5"
								>
									<svg
										class="w-4 h-4 text-gray-400"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										aria-hidden="true"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
										/>
									</svg>
									{$t('admin.dashboard.visitor_homepage')}
								</a>
							</td>
							<td class="text-right py-2.5 px-4 font-medium text-gray-900 dark:text-white">
								{homepageViewCount.toLocaleString()}
							</td>
							<td class="text-right py-2.5 pl-4 text-gray-500 dark:text-gray-400">
								{siteMetrics.homepage_last_viewed_at
									? formatRelativeDate(siteMetrics.homepage_last_viewed_at)
									: $t('admin.dashboard.visitor_never')}
							</td>
						</tr>
						{#each views as view (view.id)}
							<tr class="border-b border-gray-100 dark:border-gray-800 last:border-0">
								<td class="py-2.5 pr-4">
									<a
										href="/admin/views/{view.id}"
										class="text-gray-900 dark:text-white hover:text-primary-600 dark:hover:text-primary-400"
									>
										{view.name}
									</a>
								</td>
								<td class="text-right py-2.5 px-4 font-medium text-gray-900 dark:text-white">
									{(view.view_count ?? 0).toLocaleString()}
								</td>
								<td class="text-right py-2.5 pl-4 text-gray-500 dark:text-gray-400">
									{view.last_viewed_at
										? formatRelativeDate(view.last_viewed_at)
										: $t('admin.dashboard.visitor_never')}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<style>
	/* ── Soft Premium dashboard composition ──────────────────────────────────
	   Composition principle: chrome is an affordance, not a grouping device.
	   The stats are display-only data, so under soft-premium they read as a flat
	   "overview band" (typography + whitespace + a hairline) instead of a card.
	   The clickable quick-action / activity cards keep their card chrome (they
	   ARE affordances). Every rule is gated on [data-design='soft-premium'].

	   :global(...) is required: .card / .stat-* live on elements Svelte does not
	   scope-hash (utility classes), and the gate lives on <html>. */

	/* Flatten the band: strip card border/background/shadow/padding so the figures
	   sit in open space. Specificity (attribute + two classes) beats the global
	   .card utility without !important. */
	:global([data-design='soft-premium'] .stat-band.card) {
		background: transparent;
		border: 0;
		box-shadow: none;
		padding: 0.25rem 0;
	}

	/* Hairline rhythm between cells on the wide (single-row) layout: one quiet
	   divider, not four boxes. The two-column layout groups by alignment alone. */
	@media (min-width: 1024px) {
		:global([data-design='soft-premium'] .stat-band .stat-cell + .stat-cell) {
			border-left: 1px solid var(--border-hairline);
			padding-left: 1.5rem;
		}
	}

	/* Warm editorial typography for label + figure. */
	:global([data-design='soft-premium'] .stat-band .stat-label) {
		color: var(--muted);
		letter-spacing: 0.02em;
	}
	:global([data-design='soft-premium'] .stat-band .stat-figure) {
		color: var(--ink);
		font-family: var(--font-accent);
		font-variant-numeric: tabular-nums;
		font-weight: 600;
		font-size: 1.5rem;
		line-height: 1.1;
		margin-top: 0.125rem;
	}
</style>
