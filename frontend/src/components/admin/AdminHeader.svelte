<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { pb, currentUser } from '$lib/pocketbase';
	import { adminSidebarOpen } from '$lib/stores';
	import { demoMode as demoModeStore } from '$lib/stores/demo';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';

	let demoMode = false;
	let toggleLoading = false;
	let showDemoAnimation = false;

	// Subscribe to demo mode store
	demoModeStore.subscribe(value => {
		demoMode = value;
	});

	onMount(() => {
		// Always show animation for now (TODO: add first-time detection later)
		showDemoAnimation = true;
		// Auto-dismiss after 10 seconds
		setTimeout(() => {
			showDemoAnimation = false;
		}, 10000);
	});

	function dismissDemoAnimation() {
		showDemoAnimation = false;
		try {
			localStorage.setItem('hasSeenDemoToggle', 'true');
		} catch (err) {
			console.warn('Failed to save demo toggle state', err);
		}
	}

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

	async function toggleDemoMode() {
		// Dismiss animation when user interacts with toggle
		dismissDemoAnimation();

		console.log('[TOGGLE] toggleDemoMode() called, current demoMode:', demoMode);
		if (!demoMode) {
			// Turning on demo mode
			if (!confirm('This will replace your current profile data with sample data. Your original data will be backed up and can be restored when you toggle off demo mode.\n\nNote: If you currently have no profile data, toggling demo OFF later will keep the demo data as your starting profile. Continue?')) {
				console.log('[TOGGLE] User cancelled');
				return;
			}
		}

		toggleLoading = true;
		try {
			const endpoint = demoMode ? '/api/demo/restore' : '/api/demo/enable';
			console.log('[TOGGLE] Calling endpoint:', endpoint);
			const response = await fetch(endpoint, {
				method: 'POST',
				headers: { Authorization: pb.authStore.token }
			});
			console.log('[TOGGLE] Response:', response.status, response.ok);

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Failed to toggle demo mode');
			}

			const newDemoMode = !demoMode;
			console.log('[TOGGLE] Updating store to:', newDemoMode);
			// Update the store
			demoModeStore.set(newDemoMode);
			console.log('[TOGGLE] Store updated, reloading page...');

			// Refresh the page to show updated data
			window.location.reload();
		} catch (err) {
			console.error('[TOGGLE] Failed to toggle demo mode:', err);
			alert(err instanceof Error ? err.message : 'Failed to toggle demo mode');
		} finally {
			toggleLoading = false;
		}
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
			<!-- Demo Mode Toggle -->
			<div class="relative flex items-center gap-2 px-3 py-1.5 rounded-lg bg-gray-100 dark:bg-gray-700 {showDemoAnimation ? 'ring-2 ring-primary-500 animate-pulse' : ''}">
				<span class="text-xs font-medium text-gray-700 dark:text-gray-300 hidden sm:inline">
					Demo
				</span>
				<button
					on:click={toggleDemoMode}
					disabled={toggleLoading}
					class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed
						{demoMode ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'}"
					role="switch"
					aria-checked={demoMode}
					aria-label="Toggle demo mode"
				>
					<span
						class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform
							{demoMode ? 'translate-x-5' : 'translate-x-0.5'}"
					/>
				</button>
				{#if demoMode}
					<span class="text-xs text-primary-600 dark:text-primary-400 font-medium hidden md:inline">
						ON
					</span>
				{/if}
				{#if showDemoAnimation}
					<span class="absolute -top-2 -right-2 flex h-3 w-3">
						<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary-400 opacity-75"></span>
						<span class="relative inline-flex rounded-full h-3 w-3 bg-primary-500"></span>
					</span>
				{/if}
			</div>

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
