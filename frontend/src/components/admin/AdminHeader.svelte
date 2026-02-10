<script lang="ts">
	import { goto } from '$app/navigation';
	import { t } from 'svelte-i18n';
	import { pb, currentUser } from '$lib/pocketbase';
	import { adminSidebarOpen } from '$lib/stores';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';

	function toggleSidebar() {
		adminSidebarOpen.update((v) => {
			const next = !v;
			try {
				localStorage.setItem('adminSidebarOpen', next ? 'true' : 'false');
			} catch (err) {
				console.warn('Failed to persist sidebar state', err);
			}
			return next;
		});
	}

	async function logout() {
		pb.authStore.clear();
		goto('/admin/login');
	}
</script>

<header class="fixed top-0 left-0 right-0 h-16 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 z-40">
	<div class="flex items-center justify-between h-full px-4">
		<div class="flex items-center gap-4">
			<button
				class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
				onclick={toggleSidebar}
				aria-label={$adminSidebarOpen ? $t('admin.header.collapse_sidebar') : $t('admin.header.expand_sidebar')}
				aria-expanded={$adminSidebarOpen}
				aria-controls="admin-sidebar"
			>
				<svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
				</svg>
			</button>

			<a href="/admin" class="flex items-center gap-2">
				<span class="text-xl font-bold text-gray-900 dark:text-white">Facet</span>
			</a>
		</div>

		<div class="flex items-center gap-3">
			<!-- View Site Link -->
			<a
				href="/"
				target="_blank"
				rel="noopener"
				class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
				title={$t('admin.header.view_site_title')}
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
				</svg>
				<span class="hidden sm:inline">{$t('admin.header.view_site')}</span>
			</a>

			<ThemeToggle />

			{#if $currentUser}
				<div class="flex items-center gap-2">
					<span class="text-sm text-gray-600 dark:text-gray-400 hidden sm:inline">
						{$currentUser.email || $currentUser.username || 'Admin'}
					</span>
					<button
						onclick={logout}
						class="btn btn-ghost btn-sm"
						aria-label={$t('admin.header.sign_out')}
					>
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
						</svg>
						<span class="hidden sm:inline ml-1">{$t('admin.header.logout')}</span>
					</button>
				</div>
			{/if}
		</div>
	</div>
</header>
