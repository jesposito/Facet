<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';
	import { pb } from '$lib/pocketbase';

	interface AuditLogEntry {
		id: string;
		action: string;
		resource_type: string;
		resource_id: string;
		user_email: string;
		ip_address: string;
		created: string;
	}

	let logs: AuditLogEntry[] = $state([]);
	let loading = $state(true);
	let page = $state(1);
	let totalPages = $state(1);
	let totalItems = $state(0);
	let filterAction = $state('');
	let filterResource = $state('');
	const perPage = 50;

	onMount(() => {
		loadLogs();
	});

	async function loadLogs() {
		loading = true;
		try {
			const filterParts: string[] = [];
			if (filterAction) filterParts.push(`action = "${filterAction}"`);
			if (filterResource) filterParts.push(`resource_type = "${filterResource}"`);

			const result = await pb.collection('audit_logs').getList(page, perPage, {
				sort: '-created',
				filter: filterParts.join(' && ')
			});

			logs = result.items as unknown as AuditLogEntry[];
			totalPages = result.totalPages;
			totalItems = result.totalItems;
		} catch (err) {
			console.error('Failed to load audit logs:', err);
		} finally {
			loading = false;
		}
	}

	function applyFilters() {
		page = 1;
		loadLogs();
	}

	function clearFilters() {
		filterAction = '';
		filterResource = '';
		page = 1;
		loadLogs();
	}

	function goToPage(p: number) {
		page = p;
		loadLogs();
	}

	function formatDate(dateStr: string): string {
		if (!dateStr) return '';
		const d = new Date(dateStr);
		return d.toLocaleString();
	}

	function actionBadgeClass(action: string): string {
		switch (action) {
			case 'create':
				return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400';
			case 'update':
				return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400';
			case 'delete':
				return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400';
			case 'login':
				return 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400';
			default:
				return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400';
		}
	}
</script>

<div class="max-w-6xl mx-auto">
	<div class="mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">
			{$t('admin.audit.title')}
		</h1>
		<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
			{$t('admin.audit.description')}
		</p>
	</div>

	<!-- Filters -->
	<div class="mb-4 flex flex-wrap gap-3 items-end">
		<div>
			<label for="filter-action" class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
				{$t('admin.audit.filter_action')}
			</label>
			<select
				id="filter-action"
				bind:value={filterAction}
				onchange={applyFilters}
				class="block w-36 rounded-md border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm shadow-sm focus:border-primary-500 focus:ring-primary-500"
			>
				<option value="">{$t('admin.audit.all_actions')}</option>
				<option value="create">{$t('admin.audit.action_create')}</option>
				<option value="update">{$t('admin.audit.action_update')}</option>
				<option value="delete">{$t('admin.audit.action_delete')}</option>
				<option value="login">{$t('admin.audit.action_login')}</option>
			</select>
		</div>

		<div>
			<label for="filter-resource" class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
				{$t('admin.audit.filter_resource')}
			</label>
			<select
				id="filter-resource"
				bind:value={filterResource}
				onchange={applyFilters}
				class="block w-44 rounded-md border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm shadow-sm focus:border-primary-500 focus:ring-primary-500"
			>
				<option value="">{$t('admin.audit.all_resources')}</option>
				<option value="auth">auth</option>
				<option value="profile">profile</option>
				<option value="experience">experience</option>
				<option value="education">education</option>
				<option value="certifications">certifications</option>
				<option value="awards">awards</option>
				<option value="skills">skills</option>
				<option value="projects">projects</option>
				<option value="posts">posts</option>
				<option value="talks">talks</option>
				<option value="testimonials">testimonials</option>
				<option value="contact_methods">contact_methods</option>
				<option value="custom_content">custom_content</option>
				<option value="views">views</option>
				<option value="share_tokens">share_tokens</option>
				<option value="admin_tags">admin_tags</option>
				<option value="ai_providers">ai_providers</option>
				<option value="site_settings">site_settings</option>
				<option value="uploads">uploads</option>
				<option value="external_media">external_media</option>
			</select>
		</div>

		{#if filterAction || filterResource}
			<button
				type="button"
				onclick={clearFilters}
				class="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 underline"
			>
				{$t('admin.audit.clear_filters')}
			</button>
		{/if}

		<div class="ml-auto text-sm text-gray-500 dark:text-gray-400">
			{$t('admin.audit.total_entries', { values: { count: totalItems } })}
		</div>
	</div>

	<!-- Table -->
	{#if loading}
		<div class="flex justify-center py-12">
			<svg class="w-6 h-6 animate-spin text-primary-600" fill="none" viewBox="0 0 24 24" aria-hidden="true">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
			</svg>
		</div>
	{:else if logs.length === 0}
		<div class="text-center py-12 text-gray-500 dark:text-gray-400">
			<svg class="w-12 h-12 mx-auto mb-3 text-gray-300 dark:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
			</svg>
			<p>{$t('admin.audit.no_entries')}</p>
		</div>
	{:else}
		<div class="overflow-x-auto bg-white dark:bg-gray-800 rounded-lg shadow">
			<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
				<thead class="bg-gray-50 dark:bg-gray-900/50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							{$t('admin.audit.col_time')}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							{$t('admin.audit.col_action')}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							{$t('admin.audit.col_resource')}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							{$t('admin.audit.col_user')}
						</th>
						<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
							{$t('admin.audit.col_ip')}
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-200 dark:divide-gray-700">
					{#each logs as log (log.id)}
						<tr class="hover:bg-gray-50 dark:hover:bg-gray-700/50">
							<td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300 whitespace-nowrap">
								{formatDate(log.created)}
							</td>
							<td class="px-4 py-3 whitespace-nowrap">
								<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium {actionBadgeClass(log.action)}">
									{log.action}
								</span>
							</td>
							<td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300 whitespace-nowrap">
								<span class="font-medium">{log.resource_type}</span>
								{#if log.resource_id}
									<span class="text-gray-400 dark:text-gray-500 ml-1 text-xs">{log.resource_id}</span>
								{/if}
							</td>
							<td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300 whitespace-nowrap">
								{log.user_email || '—'}
							</td>
							<td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap font-mono text-xs">
								{log.ip_address || '—'}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<!-- Pagination -->
		{#if totalPages > 1}
			<div class="mt-4 flex justify-between items-center">
				<button
					type="button"
					onclick={() => goToPage(page - 1)}
					disabled={page <= 1}
					class="px-3 py-1.5 text-sm rounded-md border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{$t('shared.actions.previous')}
				</button>
				<span class="text-sm text-gray-600 dark:text-gray-400">
					{$t('admin.audit.page_info', { values: { page, totalPages } })}
				</span>
				<button
					type="button"
					onclick={() => goToPage(page + 1)}
					disabled={page >= totalPages}
					class="px-3 py-1.5 text-sm rounded-md border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{$t('shared.actions.next')}
				</button>
			</div>
		{/if}
	{/if}
</div>
