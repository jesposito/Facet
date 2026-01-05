<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { pb, currentUser } from '$lib/pocketbase';
	import { adminSidebarOpen } from '$lib/stores';
	import { initDemoMode } from '$lib/stores/demo';
	import AdminSidebar from '$components/admin/AdminSidebar.svelte';
	import AdminHeader from '$components/admin/AdminHeader.svelte';

	let loading = true;
	let authorized = false;
	let mounted = false;

	// Check if we're on the login page (don't require auth there)
	$: isLoginPage = $page.url.pathname === '/admin/login';

	// Handle login page - always stop loading immediately
	$: if (isLoginPage) {
		loading = false;
		authorized = false;
	}

	// Watch for logout (user becomes unauthenticated after being authorized)
	$: if (mounted && !isLoginPage && authorized && !$currentUser && !pb.authStore.isValid) {
		console.log('[LAYOUT] User logged out, redirecting to login');
		authorized = false;
		goto('/admin/login');
	}

	onMount(async () => {
		mounted = true;

		// Restore sidebar state from localStorage
		try {
			const saved = localStorage.getItem('adminSidebarOpen');
			if (saved === 'false') {
				adminSidebarOpen.set(false);
			} else if (saved === 'true') {
				adminSidebarOpen.set(true);
			} else if (window.innerWidth < 1024) {
				// default to collapsed on small screens
				adminSidebarOpen.set(false);
			}
		} catch (err) {
			console.warn('Failed to restore sidebar state', err);
		}

		// Check path directly
		const onLoginPage = window.location.pathname === '/admin/login';

		// Login page doesn't require authentication
		if (onLoginPage) {
			loading = false;
			return;
		}

		// Initialize demo mode state BEFORE child pages render
		console.log('[LAYOUT] About to call initDemoMode()');
		await initDemoMode();
		console.log('[LAYOUT] initDemoMode() completed');

		// Check auth state directly after demo mode is initialized
		console.log('[LAYOUT] Checking auth state - currentUser:', $currentUser, 'authStore.isValid:', pb.authStore.isValid);

		if ($currentUser && pb.authStore.isValid) {
			console.log('[LAYOUT] User authenticated, showing admin');
			authorized = true;
			loading = false;
		} else if (pb.authStore.isValid && !$currentUser) {
			// Auth store is valid but store not updated yet - wait briefly
			console.log('[LAYOUT] Auth store valid but $currentUser not set, waiting...');
			await new Promise(resolve => setTimeout(resolve, 150));

			// Re-check after delay
			if ($currentUser && pb.authStore.isValid) {
				console.log('[LAYOUT] User authenticated after delay, showing admin');
				authorized = true;
				loading = false;
			} else {
				console.log('[LAYOUT] Still not authenticated after delay, redirecting to login');
				goto('/admin/login');
			}
		} else {
			// Not authenticated - redirect to login
			console.log('[LAYOUT] Not authenticated, redirecting to login');
			goto('/admin/login');
		}
	});

	onDestroy(() => {
		mounted = false;
	});
</script>

{#if loading}
	<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900" role="status" aria-label="Loading admin dashboard">
		<div class="animate-pulse text-center">
			<div class="w-12 h-12 mx-auto mb-4 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
				<svg class="w-6 h-6 text-primary-600 animate-spin" fill="none" viewBox="0 0 24 24" aria-hidden="true">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
			</div>
			<p class="text-gray-600 dark:text-gray-400">Loading admin...</p>
		</div>
	</div>
{:else if isLoginPage}
	<!-- Login page renders without admin chrome -->
	<slot />
{:else if authorized}
	<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
		<AdminHeader />

		<div class="flex">
			<AdminSidebar />

			<main id="main-content" class="flex-1 p-6 {$adminSidebarOpen ? 'ml-64' : 'ml-16'} transition-all duration-200 mt-16">
				<slot />
			</main>
		</div>
	</div>
{/if}
