<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts } from '$lib/stores';
	import { onMount, onDestroy } from 'svelte';
	import { t } from 'svelte-i18n';

	let stats = $state({
		projects: 0,
		experience: 0,
		totalVisitors: 0,
		pendingProposals: 0
	});

	let viewMetrics: Array<{ id: string; name: string; slug: string; view_count: number; last_viewed_at: string }> = $state([]);
	let homepageMetrics = $state({ view_count: 0, last_viewed_at: '' });
	let recentActivity: Array<{ type: string; title: string }> = $state([]);
	let loading = $state(true);
	let mounted = false;

	// Simple pattern - admin layout handles auth
	onMount(() => {
		mounted = true;
		loadDashboard();
	});

	onDestroy(() => {
		mounted = false;
	});

	async function loadDashboard() {
		if (!mounted) return;

		try {
			const [projectsRes, experienceRes, viewsRes, proposalsRes, settingsRes] = await Promise.all([
				collection('projects').getList(1, 1),
				collection('experience').getList(1, 1),
				collection('views').getFullList({ sort: '-view_count' }),
				pb.collection('import_proposals').getList(1, 1, { filter: "status = 'pending'" }),
				fetch('/api/site-settings').then(r => r.ok ? r.json() : null).catch(() => null)
			]);

			if (!mounted) return;

			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			const views = viewsRes as any[];

			const hpViewCount = settingsRes?.homepage_view_count || 0;
			homepageMetrics = {
				view_count: hpViewCount,
				last_viewed_at: settingsRes?.homepage_last_viewed_at || ''
			};

			stats = {
				projects: projectsRes.totalItems,
				experience: experienceRes.totalItems,
				totalVisitors: views.reduce((sum, v) => sum + (v.view_count || 0), 0) + hpViewCount,
				pendingProposals: proposalsRes.totalItems
			};

			viewMetrics = views.map((v) => ({
				id: v.id,
				name: v.name || '',
				slug: v.slug || '',
				view_count: v.view_count || 0,
				last_viewed_at: v.last_viewed_at || ''
			}));

			// Get recent projects and experience for activity feed
			const [recentProjects, recentExperience] = await Promise.all([
				collection('projects').getList(1, 3, { sort: '-id' }),
				collection('experience').getList(1, 3, { sort: '-id' })
			]);

			if (!mounted) return;

			recentActivity = [
				...recentProjects.items.map((p) => ({
					type: 'project',
					title: p.title,
					id: p.id
				})),
				...recentExperience.items.map((e) => ({
					type: 'experience',
					title: `${e.title} at ${e.company}`,
					id: e.id
				}))
			]
				.sort((a, b) => b.id.localeCompare(a.id))
				.slice(0, 5);
		} catch (err) {
			if (err instanceof Error && err.message.includes('autocancelled')) {
				return;
			}
			console.error('Failed to load dashboard stats:', err);
			if (mounted) {
				toasts.error('Failed to load dashboard stats');
			}
		} finally {
			if (mounted) {
				loading = false;
			}
		}
	}

	let isEmpty = $derived(!loading && stats.projects === 0 && stats.experience === 0 && viewMetrics.length === 0);

	function formatRelativeDate(dateStr: string): string {
		if (!dateStr) return '';
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMs / 3600000);
		const diffDays = Math.floor(diffMs / 86400000);

		if (diffMins < 1) return $t('admin.dashboard.relative_time.just_now');
		if (diffMins < 60) return $t('admin.dashboard.relative_time.minutes_ago', { values: { count: diffMins } });
		if (diffHours < 24) return $t('admin.dashboard.relative_time.hours_ago', { values: { count: diffHours } });
		if (diffDays < 30) return $t('admin.dashboard.relative_time.days_ago', { values: { count: diffDays } });
		return date.toLocaleDateString();
	}
</script>

<svelte:head>
	<title>{$t('admin.dashboard.title')} | Facet</title>
</svelte:head>

<div class="max-w-6xl mx-auto">
	{#if loading}
		<!-- Loading skeleton that matches content structure -->
		<div class="animate-fade-in">
			<!-- Title skeleton -->
			<div class="h-8 bg-gray-200 dark:bg-gray-700 rounded w-48 mb-6 relative overflow-hidden">
				<div class="absolute inset-0 animate-shimmer"></div>
			</div>

			<!-- Stats grid skeleton -->
			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
				{#each Array(4) as _}
					<div class="card p-6">
						<div class="flex items-center justify-between">
							<div class="space-y-2">
								<div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-20 relative overflow-hidden">
									<div class="absolute inset-0 animate-shimmer"></div>
								</div>
								<div class="h-8 bg-gray-200 dark:bg-gray-700 rounded w-12 relative overflow-hidden">
									<div class="absolute inset-0 animate-shimmer"></div>
								</div>
							</div>
							<div class="w-12 h-12 rounded-lg bg-gray-200 dark:bg-gray-700 relative overflow-hidden">
								<div class="absolute inset-0 animate-shimmer"></div>
							</div>
						</div>
					</div>
				{/each}
			</div>

			<!-- Quick actions and activity skeleton -->
			<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
				<div class="card p-6">
					<div class="h-6 bg-gray-200 dark:bg-gray-700 rounded w-32 mb-4 relative overflow-hidden">
						<div class="absolute inset-0 animate-shimmer"></div>
					</div>
					<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
						{#each Array(6) as _}
							<div class="h-10 bg-gray-200 dark:bg-gray-700 rounded-lg relative overflow-hidden">
								<div class="absolute inset-0 animate-shimmer"></div>
							</div>
						{/each}
					</div>
				</div>
				<div class="card p-6">
					<div class="h-6 bg-gray-200 dark:bg-gray-700 rounded w-32 mb-4 relative overflow-hidden">
						<div class="absolute inset-0 animate-shimmer"></div>
					</div>
					<div class="space-y-3">
						{#each Array(3) as _}
							<div class="flex items-center gap-3">
								<div class="w-10 h-10 rounded-lg bg-gray-200 dark:bg-gray-700 relative overflow-hidden">
									<div class="absolute inset-0 animate-shimmer"></div>
								</div>
								<div class="flex-1 h-4 bg-gray-200 dark:bg-gray-700 rounded relative overflow-hidden">
									<div class="absolute inset-0 animate-shimmer"></div>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	{:else if isEmpty}
		<!-- Welcome state for first-time users -->
		<div class="card p-8 mb-8">
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-3">{$t('admin.dashboard.welcome_title')}</h1>
			<p class="text-gray-600 dark:text-gray-400">
				{@html $t('admin.dashboard.welcome_description', {
					values: {
						profile_link: `<a href="/admin/homepage" class="text-primary-600 dark:text-primary-400 hover:underline">${$t('admin.dashboard.adding_profile')}</a>`,
						import_link: `<a href="/admin/import" class="text-primary-600 dark:text-primary-400 hover:underline">${$t('admin.dashboard.import_project')}</a>`
					}
				})}
			</p>
			<p class="text-gray-500 dark:text-gray-500 text-sm mt-2">
				{$t('admin.dashboard.no_rush')}
			</p>
		</div>
	{:else}
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">{$t('admin.dashboard.title')}</h1>

		<!-- Stats grid -->
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
			<div class="card p-6">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm text-gray-500 dark:text-gray-400">{$t('admin.dashboard.stats_projects')}</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.projects}</p>
					</div>
					<div class="w-12 h-12 rounded-lg bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
						<svg class="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
						</svg>
					</div>
				</div>
			</div>

			<div class="card p-6">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm text-gray-500 dark:text-gray-400">{$t('admin.dashboard.stats_experience')}</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.experience}</p>
					</div>
					<div class="w-12 h-12 rounded-lg bg-green-100 dark:bg-green-900 flex items-center justify-center">
						<svg class="w-6 h-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
						</svg>
					</div>
				</div>
			</div>

			<div class="card p-6">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm text-gray-500 dark:text-gray-400">{$t('admin.dashboard.stats_total_visitors')}</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.totalVisitors.toLocaleString()}</p>
					</div>
					<div class="w-12 h-12 rounded-lg bg-purple-100 dark:bg-purple-900 flex items-center justify-center">
						<svg class="w-6 h-6 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
						</svg>
					</div>
				</div>
			</div>

			<div class="card p-6">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm text-gray-500 dark:text-gray-400">{$t('admin.dashboard.pending_reviews')}</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.pendingProposals}</p>
					</div>
					<div class="w-12 h-12 rounded-lg bg-yellow-100 dark:bg-yellow-900 flex items-center justify-center">
						<svg class="w-6 h-6 text-yellow-600 dark:text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
						</svg>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
		<!-- Quick actions -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.dashboard.quick_actions')}</h2>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
				<a href="/admin/projects?new=true" class="group btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:scale-110" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					{$t('admin.dashboard.add_project')}
				</a>
				<a href="/admin/experience?new=true" class="group btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:scale-110" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					{$t('admin.dashboard.add_experience')}
				</a>
				<a href="/admin/import" class="group btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:-translate-y-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
					</svg>
					{$t('admin.dashboard.import_github')}
				</a>
				<a href="/admin/views/new" class="group btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:scale-110" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
					</svg>
					{$t('admin.dashboard.create_view')}
				</a>
				<a href="/rss.xml" target="_blank" class="group btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:scale-110" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 5c7.18 0 13 5.82 13 13M6 11a7 7 0 017 7M6 17a1 1 0 11-2 0 1 1 0 012 0z" />
					</svg>
					{$t('admin.dashboard.rss_feed')}
				</a>
				<a href="/talks.ics" class="group btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2 transition-transform duration-200 group-hover:scale-110" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
					</svg>
					{$t('admin.dashboard.talks_calendar')}
				</a>
			</div>
		</div>

		<!-- Recent activity -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.dashboard.recent_activity')}</h2>
			{#if loading}
				<div class="space-y-3">
					{#each Array(3) as _}
						<div class="animate-pulse flex items-center gap-3">
							<div class="w-10 h-10 rounded-lg bg-gray-200 dark:bg-gray-700"></div>
							<div class="flex-1">
								<div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
							</div>
						</div>
					{/each}
				</div>
			{:else if recentActivity.length === 0}
				<p class="text-gray-500 dark:text-gray-400 text-sm">{$t('admin.dashboard.nothing_yet')}</p>
			{:else}
				<div class="space-y-3">
					{#each recentActivity as activity}
						<div class="flex items-center gap-3">
							<div class="w-10 h-10 rounded-lg {activity.type === 'project' ? 'bg-blue-100 dark:bg-blue-900' : 'bg-green-100 dark:bg-green-900'} flex items-center justify-center">
								{#if activity.type === 'project'}
									<svg class="w-5 h-5 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
									</svg>
								{:else}
									<svg class="w-5 h-5 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
									</svg>
								{/if}
							</div>
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium text-gray-900 dark:text-white truncate">
									{activity.title}
								</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">
									{activity.type === 'project' ? $t('admin.dashboard.activity_project') : $t('admin.dashboard.activity_experience')}
								</p>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Visitor stats -->
	{#if !loading && (viewMetrics.length > 0 || homepageMetrics.view_count > 0)}
		<div class="mt-6 card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.dashboard.visitor_stats')}</h2>
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gray-200 dark:border-gray-700">
							<th scope="col" class="text-left py-2 pr-4 text-gray-500 dark:text-gray-400 font-medium">{$t('admin.dashboard.visitor_view_name')}</th>
							<th scope="col" class="text-right py-2 px-4 text-gray-500 dark:text-gray-400 font-medium">{$t('admin.dashboard.visitor_count')}</th>
							<th scope="col" class="text-right py-2 pl-4 text-gray-500 dark:text-gray-400 font-medium">{$t('admin.dashboard.visitor_last_visited')}</th>
						</tr>
					</thead>
					<tbody>
						<tr class="border-b border-gray-100 dark:border-gray-800">
							<td class="py-2.5 pr-4">
								<a href="/admin/homepage" class="text-gray-900 dark:text-white hover:text-primary-600 dark:hover:text-primary-400 flex items-center gap-1.5">
									<svg class="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
									</svg>
									{$t('admin.dashboard.visitor_homepage')}
								</a>
							</td>
							<td class="text-right py-2.5 px-4 font-medium text-gray-900 dark:text-white">
								{homepageMetrics.view_count.toLocaleString()}
							</td>
							<td class="text-right py-2.5 pl-4 text-gray-500 dark:text-gray-400">
								{homepageMetrics.last_viewed_at ? formatRelativeDate(homepageMetrics.last_viewed_at) : $t('admin.dashboard.visitor_never')}
							</td>
						</tr>
						{#each viewMetrics as view}
							<tr class="border-b border-gray-100 dark:border-gray-800 last:border-0">
								<td class="py-2.5 pr-4">
									<a href="/admin/views/{view.id}" class="text-gray-900 dark:text-white hover:text-primary-600 dark:hover:text-primary-400">
										{view.name}
									</a>
								</td>
								<td class="text-right py-2.5 px-4 font-medium text-gray-900 dark:text-white">
									{view.view_count.toLocaleString()}
								</td>
								<td class="text-right py-2.5 pl-4 text-gray-500 dark:text-gray-400">
									{view.last_viewed_at ? formatRelativeDate(view.last_viewed_at) : $t('admin.dashboard.visitor_never')}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- Pending proposals alert -->
	{#if stats.pendingProposals > 0}
		<div class="mt-6 card p-4 border-l-4 border-yellow-500 bg-yellow-50 dark:bg-yellow-900/20">
			<div class="flex items-center gap-3">
				<svg class="w-6 h-6 text-yellow-600 dark:text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
				</svg>
				<div class="flex-1">
					<p class="font-medium text-yellow-800 dark:text-yellow-200">
						{$t('admin.dashboard.pending_alert', { values: { count: stats.pendingProposals } })}
					</p>
				</div>
				<a href="/admin/proposals" class="btn btn-sm bg-yellow-600 text-white hover:bg-yellow-700">
					{$t('admin.dashboard.review_now')}
				</a>
			</div>
		</div>
	{/if}
</div>
