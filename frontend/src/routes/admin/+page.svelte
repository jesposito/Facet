<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { onMount } from 'svelte';

	let stats = {
		projects: 0,
		experience: 0,
		views: 0,
		pendingProposals: 0
	};

	let recentActivity: Array<{ type: string; title: string; date: string }> = [];
	let loading = true;

	// Simple pattern - admin layout handles auth
	onMount(loadDashboard);

	async function loadDashboard() {
		try {
			const [projectsRes, experienceRes, viewsRes, proposalsRes] = await Promise.all([
				pb.collection('projects').getList(1, 1),
				pb.collection('experience').getList(1, 1),
				pb.collection('views').getList(1, 1),
				pb.collection('import_proposals').getList(1, 1, { filter: "status = 'pending'" })
			]);

			stats = {
				projects: projectsRes.totalItems,
				experience: experienceRes.totalItems,
				views: viewsRes.totalItems,
				pendingProposals: proposalsRes.totalItems
			};

			// Get recent projects and experience for activity feed
			const [recentProjects, recentExperience] = await Promise.all([
				pb.collection('projects').getList(1, 3, { sort: '-id' }),
				pb.collection('experience').getList(1, 3, { sort: '-id' })
			]);

			recentActivity = [
				...recentProjects.items.map((p) => ({
					type: 'project',
					title: p.title,
					date: p.id
				})),
				...recentExperience.items.map((e) => ({
					type: 'experience',
					title: `${e.title} at ${e.company}`,
					date: e.id
				}))
			]
				.sort((a, b) => b.date.localeCompare(a.date))
				.slice(0, 5);
		} catch (err) {
			console.error('Failed to load dashboard stats:', err);
		} finally {
			loading = false;
		}
	}

	function formatDate(dateString: string): string {
		return new Date(dateString).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	$: isEmpty = !loading && stats.projects === 0 && stats.experience === 0 && stats.views === 0;
</script>

<svelte:head>
	<title>Dashboard | Facet</title>
</svelte:head>

<div class="max-w-6xl mx-auto">
	{#if loading}
		<!-- Loading state -->
		<div class="card p-8 mb-8 animate-pulse">
			<div class="h-8 bg-gray-200 dark:bg-gray-700 rounded w-48 mb-4"></div>
			<div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-96"></div>
		</div>
	{:else if isEmpty}
		<!-- Welcome state for first-time users -->
		<div class="card p-8 mb-8">
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-3">This is your space.</h1>
			<p class="text-gray-600 dark:text-gray-400">
				You might start by <a href="/admin/profile" class="text-primary-600 dark:text-primary-400 hover:underline">adding your profile</a>,
				or <a href="/admin/import" class="text-primary-600 dark:text-primary-400 hover:underline">import a project from GitHub</a>.
			</p>
			<p class="text-gray-500 dark:text-gray-500 text-sm mt-2">
				There's no rush — add things as you're ready.
			</p>
		</div>
	{:else}
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Dashboard</h1>

		<!-- Stats grid -->
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
			<div class="card p-6">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm text-gray-500 dark:text-gray-400">Projects</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.projects}</p>
					</div>
					<div class="w-12 h-12 rounded-lg bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
						<svg class="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
						</svg>
					</div>
				</div>
			</div>

			<div class="card p-6">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm text-gray-500 dark:text-gray-400">Experience</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.experience}</p>
					</div>
					<div class="w-12 h-12 rounded-lg bg-green-100 dark:bg-green-900 flex items-center justify-center">
						<svg class="w-6 h-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
						</svg>
					</div>
				</div>
			</div>

			<div class="card p-6">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm text-gray-500 dark:text-gray-400">Views</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.views}</p>
					</div>
					<div class="w-12 h-12 rounded-lg bg-purple-100 dark:bg-purple-900 flex items-center justify-center">
						<svg class="w-6 h-6 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
						</svg>
					</div>
				</div>
			</div>

			<div class="card p-6">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm text-gray-500 dark:text-gray-400">Pending Reviews</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.pendingProposals}</p>
					</div>
					<div class="w-12 h-12 rounded-lg bg-yellow-100 dark:bg-yellow-900 flex items-center justify-center">
						<svg class="w-6 h-6 text-yellow-600 dark:text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Quick Actions</h2>
			<div class="grid grid-cols-2 gap-3">
				<a href="/admin/projects/new" class="btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					Add Project
				</a>
				<a href="/admin/experience/new" class="btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					Add Experience
				</a>
				<a href="/admin/import" class="btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
					</svg>
					Import from GitHub
				</a>
				<a href="/admin/views/new" class="btn btn-secondary justify-start">
					<svg class="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
					</svg>
					Create View
				</a>
			</div>
		</div>

		<!-- Recent activity -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Recent Activity</h2>
			{#if loading}
				<div class="space-y-3">
					{#each Array(3) as _}
						<div class="animate-pulse flex items-center gap-3">
							<div class="w-10 h-10 rounded-lg bg-gray-200 dark:bg-gray-700"></div>
							<div class="flex-1">
								<div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
								<div class="h-3 bg-gray-200 dark:bg-gray-700 rounded w-1/2 mt-1"></div>
							</div>
						</div>
					{/each}
				</div>
			{:else if recentActivity.length === 0}
				<p class="text-gray-500 dark:text-gray-400 text-sm">Nothing here yet — and that's okay.</p>
			{:else}
				<div class="space-y-3">
					{#each recentActivity as activity}
						<div class="flex items-center gap-3">
							<div class="w-10 h-10 rounded-lg {activity.type === 'project' ? 'bg-blue-100 dark:bg-blue-900' : 'bg-green-100 dark:bg-green-900'} flex items-center justify-center">
								{#if activity.type === 'project'}
									<svg class="w-5 h-5 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
									</svg>
								{:else}
									<svg class="w-5 h-5 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
									</svg>
								{/if}
							</div>
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium text-gray-900 dark:text-white truncate">
									{activity.title}
								</p>
								<p class="text-xs text-gray-500 dark:text-gray-400">
									{formatDate(activity.date)}
								</p>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Pending proposals alert -->
	{#if stats.pendingProposals > 0}
		<div class="mt-6 card p-4 border-l-4 border-yellow-500 bg-yellow-50 dark:bg-yellow-900/20">
			<div class="flex items-center gap-3">
				<svg class="w-6 h-6 text-yellow-600 dark:text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
				</svg>
				<div class="flex-1">
					<p class="font-medium text-yellow-800 dark:text-yellow-200">
						You have {stats.pendingProposals} pending import proposal{stats.pendingProposals > 1 ? 's' : ''} to review
					</p>
				</div>
				<a href="/admin/proposals" class="btn btn-sm bg-yellow-600 text-white hover:bg-yellow-700">
					Review Now
				</a>
			</div>
		</div>
	{/if}
</div>
