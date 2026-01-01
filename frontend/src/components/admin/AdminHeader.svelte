<script lang="ts">
	import { goto } from '$app/navigation';
	import { pb, currentUser } from '$lib/pocketbase';
	import { adminSidebarOpen } from '$lib/stores';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';

	function toggleSidebar() {
		adminSidebarOpen.update((v) => !v);
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
				on:click={toggleSidebar}
				aria-label={$adminSidebarOpen ? 'Collapse sidebar' : 'Expand sidebar'}
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
			<ThemeToggle />

			{#if $currentUser}
				<div class="flex items-center gap-2">
					<span class="text-sm text-gray-600 dark:text-gray-400 hidden sm:inline">
						{$currentUser.email || $currentUser.username || 'Admin'}
					</span>
					<button
						on:click={logout}
						class="btn btn-ghost btn-sm"
						aria-label="Sign out"
					>
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
						</svg>
						<span class="hidden sm:inline ml-1">Logout</span>
					</button>
				</div>
			{/if}
		</div>
	</div>
</header>
