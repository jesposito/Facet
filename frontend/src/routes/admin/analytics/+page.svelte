<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { hasFeature } from '$lib/stores/plan';

	const hasAdvancedAnalytics = hasFeature('analytics');

	let loading = $state(true);
	let error = $state(false);
	let period = $state('30d');
	let totalViews = $state(0);
	let uniqueVisitors = $state(0);
	let viewsOverTime: Array<{ date: string; count: number }> = $state([]);
	let popularViews: Array<{ view_id: string; view_slug: string; count: number }> = $state([]);
	let topReferrers: Array<{ referrer: string; count: number }> = $state([]);
	let countries: Array<{ country_code: string; count: number }> = $state([]);
	let utmStats: Array<{ source: string; medium: string; campaign: string; count: number }> = $state([]);

	onMount(() => {
		loadAnalytics();
	});

	async function loadAnalytics() {
		loading = true;
		error = false;
		try {
			const response = await fetch(`/api/admin/analytics?period=${period}`, {
				headers: { Authorization: `Bearer ${pb.authStore.token}` }
			});
			if (response.ok) {
				const data = await response.json();
				totalViews = data.total_views || 0;
				uniqueVisitors = data.unique_visitors || 0;
				viewsOverTime = data.views_over_time || [];
				popularViews = data.popular_views || [];
				topReferrers = data.top_referrers || [];
				countries = data.countries || [];
				utmStats = data.utm_stats || [];
			} else {
				error = true;
			}
		} catch (err) {
			console.error('Failed to load analytics:', err);
			error = true;
		} finally {
			loading = false;
		}
	}

	function changePeriod(newPeriod: string) {
		period = newPeriod;
		loadAnalytics();
	}

	function periodLabel(p: string): string {
		if (p === '7d') return $t('admin.analytics_page.period_7d');
		if (p === '30d') return $t('admin.analytics_page.period_30d');
		return $t('admin.analytics_page.period_90d');
	}

	// Calculate max for bar chart scaling
	let maxDailyViews = $derived(
		viewsOverTime.length > 0
			? Math.max(...viewsOverTime.map((d) => d.count))
			: 1
	);

	let maxViewCount = $derived(
		popularViews.length > 0
			? Math.max(...popularViews.map((v) => v.count))
			: 1
	);

	let maxReferrerCount = $derived(
		topReferrers.length > 0
			? Math.max(...topReferrers.map((r) => r.count))
			: 1
	);
</script>

<svelte:head>
	<title>{$t('admin.analytics_page.title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.analytics_page.title')}</h1>
			<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.analytics_page.description')}</p>
		</div>

		<div class="flex gap-1 bg-gray-100 dark:bg-gray-800 rounded-lg p-1" role="group" aria-label={$t('admin.analytics_page.title')}>
			{#each ['7d', '30d', '90d'] as p}
				<button
					type="button"
					class="px-3 py-1.5 text-sm rounded-md transition-colors {period === p
						? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
						: 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'}"
					aria-pressed={period === p}
					onclick={() => changePeriod(p)}
				>
					{periodLabel(p)}
				</button>
			{/each}
		</div>
	</div>

	{#if loading}
		<div class="grid grid-cols-1 gap-6" aria-hidden="true">
			{#each [1, 2, 3] as _}
				<div class="card p-6 animate-pulse">
					<div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/4 mb-4"></div>
					<div class="h-32 bg-gray-200 dark:bg-gray-700 rounded"></div>
				</div>
			{/each}
		</div>
		<div class="sr-only" role="status">{$t('admin.analytics_page.loading')}</div>
	{:else if error}
		<div class="card p-6 text-center" role="alert">
			<p class="text-red-600 dark:text-red-400 mb-3">{$t('admin.analytics_page.error_loading')}</p>
			<button
				type="button"
				class="btn btn-primary btn-sm"
				onclick={() => loadAnalytics()}
			>
				{$t('admin.analytics_page.retry')}
			</button>
		</div>
	{:else}
		<!-- Headline stats -->
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-6">
			<div class="card p-6">
				<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.analytics_page.total_views')}</p>
				<p
					class="text-3xl font-bold text-gray-900 dark:text-white"
					aria-describedby="total-views-help"
				>
					{totalViews.toLocaleString()}
				</p>
				<p id="total-views-help" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					{$t('admin.analytics_page.total_views_help')}
				</p>
			</div>
			<div class="card p-6">
				<p class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.analytics_page.unique_visitors')}</p>
				<p
					class="text-3xl font-bold text-gray-900 dark:text-white"
					aria-describedby="unique-visitors-help"
				>
					{uniqueVisitors.toLocaleString()}
				</p>
				<p id="unique-visitors-help" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					{$t('admin.analytics_page.unique_visitors_help')}
				</p>
			</div>
		</div>

		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- Views over time -->
			<div class="card p-6 lg:col-span-2">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.analytics_page.views_over_time')}</h2>
				{#if viewsOverTime.length === 0}
					<p class="text-gray-500 dark:text-gray-400 text-sm">{$t('admin.analytics_page.no_view_data')}</p>
				{:else}
					<div
						class="flex items-end gap-[2px] h-40"
						role="img"
						aria-label={$t('admin.analytics_page.views_over_time')}
					>
						{#each viewsOverTime as day}
							<div
								class="flex-1 bg-primary-500 dark:bg-primary-400 rounded-t min-h-[2px] transition-all hover:bg-primary-600 dark:hover:bg-primary-300"
								style="height: {(day.count / maxDailyViews) * 100}%"
								title="{day.count} - {day.date}"
							></div>
						{/each}
					</div>
					<div class="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-2">
						<span>{viewsOverTime[0]?.date}</span>
						<span>{viewsOverTime[viewsOverTime.length - 1]?.date}</span>
					</div>
				{/if}
			</div>

			<!-- Popular views -->
			<div class="card p-6">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.analytics_page.popular_views')}</h2>
				{#if popularViews.length === 0}
					<p class="text-gray-500 dark:text-gray-400 text-sm">{$t('admin.analytics_page.no_view_data')}</p>
				{:else}
					<div class="space-y-3">
						{#each popularViews as view}
							<div>
								<div class="flex justify-between text-sm mb-1">
									<span class="text-gray-700 dark:text-gray-300 truncate">{view.view_slug || view.view_id}</span>
									<span class="text-gray-500 dark:text-gray-400 ml-2 shrink-0">{view.count}</span>
								</div>
								<div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
									<div
										class="bg-primary-500 dark:bg-primary-400 h-2 rounded-full transition-all"
										style="width: {(view.count / maxViewCount) * 100}%"
									></div>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Top referrers (advanced analytics) -->
			{#if $hasAdvancedAnalytics}
				<div class="card p-6">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.analytics_page.top_referrers')}</h2>
					{#if topReferrers.length === 0}
						<p class="text-gray-500 dark:text-gray-400 text-sm">{$t('admin.analytics_page.no_referrer_data')}</p>
					{:else}
						<div class="space-y-3">
							{#each topReferrers as ref}
								<div>
									<div class="flex justify-between text-sm mb-1">
										<span class="text-gray-700 dark:text-gray-300 truncate">{ref.referrer}</span>
										<span class="text-gray-500 dark:text-gray-400 ml-2 shrink-0">{ref.count}</span>
									</div>
									<div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
										<div
											class="bg-blue-500 dark:bg-blue-400 h-2 rounded-full transition-all"
											style="width: {(ref.count / maxReferrerCount) * 100}%"
										></div>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Countries -->
				{#if countries.length > 0}
					<div class="card p-6 lg:col-span-2">
						<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.analytics_page.top_countries')}</h2>
						<div class="flex flex-wrap gap-3">
							{#each countries as country}
								<div class="flex items-center gap-2 px-3 py-1.5 bg-gray-100 dark:bg-gray-800 rounded-lg text-sm">
									<span class="text-gray-700 dark:text-gray-300">{country.country_code || '??'}</span>
									<span class="text-gray-500 dark:text-gray-400">{country.count}</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- UTM Campaigns -->
				{#if utmStats.length > 0}
					<div class="card p-6 lg:col-span-2">
						<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">UTM Campaigns</h2>
						<div class="overflow-x-auto">
							<table class="w-full text-sm">
								<thead>
									<tr class="text-left text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-700">
										<th class="pb-2 font-medium">Source</th>
										<th class="pb-2 font-medium">Medium</th>
										<th class="pb-2 font-medium">Campaign</th>
										<th class="pb-2 font-medium text-right">Visits</th>
									</tr>
								</thead>
								<tbody>
									{#each utmStats as utm}
										<tr class="border-b border-gray-100 dark:border-gray-800">
											<td class="py-2 text-gray-700 dark:text-gray-300">{utm.source}</td>
											<td class="py-2 text-gray-600 dark:text-gray-400">{utm.medium}</td>
											<td class="py-2 text-gray-600 dark:text-gray-400">{utm.campaign}</td>
											<td class="py-2 text-right font-medium text-gray-900 dark:text-white">{utm.count}</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</div>
