<script lang="ts">
	import { run } from 'svelte/legacy';

	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/stores';
	import { pb, currentUser } from '$lib/pocketbase';
	import { adminSidebarOpen } from '$lib/stores';
	import { demoMode, initDemoMode } from '$lib/stores/demo';
	import AdminSidebar from '$components/admin/AdminSidebar.svelte';
	import AdminHeader from '$components/admin/AdminHeader.svelte';
	import PasswordChangeModal from '$components/admin/PasswordChangeModal.svelte';
	interface Props {
		children?: import('svelte').Snippet;
	}

	let { children }: Props = $props();

	let loading = $state(true);
	let authorized = $state(false);
	let mounted = $state(false);
	let showPasswordChangeModal = $state(false);
	
	// Mobile detection for responsive sidebar behavior
	// Mobile: sidebar is overlay drawer (hidden by default)
	// Desktop: sidebar is persistent (current behavior)
	let isMobile = $state(false);
	const MOBILE_BREAKPOINT = 1024; // lg breakpoint




	async function checkDefaultPassword() {
		try {
			const response = await fetch('/api/auth/check-default-password', {
				headers: {
					Authorization: `Bearer ${pb.authStore.token}`
				}
			});

			if (response.ok) {
				const data = await response.json();
				if (data.has_default_password) {
					showPasswordChangeModal = true;
				}
			}
		} catch (err) {
			console.error('Failed to check default password:', err);
		}
	}

	// Media query listener cleanup
	let mediaQueryCleanup: (() => void) | null = null;

	onMount(async () => {
		mounted = true;

		// Set up mobile detection with media query
		const mediaQuery = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
		isMobile = mediaQuery.matches;
		
		const handleMediaChange = (e: MediaQueryListEvent) => {
			isMobile = e.matches;
			// When switching to mobile, close sidebar by default
			if (e.matches) {
				adminSidebarOpen.set(false);
			}
		};
		
		mediaQuery.addEventListener('change', handleMediaChange);
		mediaQueryCleanup = () => mediaQuery.removeEventListener('change', handleMediaChange);

		// Restore sidebar state from localStorage (desktop only)
		// On mobile, sidebar is always closed by default
		try {
			if (isMobile) {
				adminSidebarOpen.set(false);
			} else {
				const saved = localStorage.getItem('adminSidebarOpen');
				if (saved === 'false') {
					adminSidebarOpen.set(false);
				} else if (saved === 'true') {
					adminSidebarOpen.set(true);
				}
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

		// Initialize demo mode state BEFORE child pages render (with timeout protection)
		try {
			await initDemoMode();
		} catch (err) {
			console.error('[LAYOUT] initDemoMode() failed:', err);
			// Continue - demo mode failure shouldn't block login
		}

		// CRITICAL: Check auth state - MUST be authenticated to proceed
		const isAuthenticated = $currentUser && pb.authStore.isValid;

		if (isAuthenticated) {
			// User is fully authenticated - check if they have default password
			checkDefaultPassword();
			authorized = true;
			loading = false;
		} else if (pb.authStore.isValid && !$currentUser) {
			// Auth store is valid but $currentUser store not updated yet - wait briefly
			await new Promise(resolve => setTimeout(resolve, 150));

			// Re-check after delay
			const stillAuthenticated = $currentUser && pb.authStore.isValid;
			if (stillAuthenticated) {
				// Check if user has default password
				checkDefaultPassword();
				authorized = true;
				loading = false;
			} else {
				// Auth check failed - redirect to login
				loading = false; // Stop loading before redirect
				goto('/admin/login');
			}
		} else {
			// Not authenticated at all - redirect to login immediately
			loading = false; // Stop loading before redirect
			goto('/admin/login');
		}
	});

	onDestroy(() => {
		mounted = false;
		// Clean up media query listener
		if (mediaQueryCleanup) {
			mediaQueryCleanup();
		}
	});

	// Close sidebar on navigation (mobile only)
	afterNavigate(() => {
		if (isMobile && $adminSidebarOpen) {
			adminSidebarOpen.set(false);
		}
	});

	function handlePasswordChanged() {
		// Password was successfully changed - hide modal and reload user data
		showPasswordChangeModal = false;

		// Refresh user data to get updated password_changed_from_default field
		if ($currentUser) {
			pb.collection('users')
				.getOne($currentUser.id)
				.then((updatedUser) => {
					// Update the currentUser store
					currentUser.set(updatedUser);
				})
				.catch((err) => {
					console.error('Failed to refresh user data:', err);
				});
		}
	}
	// Check if we're on the login page (don't require auth there)
	let isLoginPage = $derived($page.url.pathname === '/admin/login');
	// Handle login page - always stop loading immediately
	run(() => {
		if (isLoginPage) {
			loading = false;
			authorized = false;
		}
	});
	// Reactive auth check - update authorized when currentUser changes
	run(() => {
		if (mounted && !isLoginPage) {
			const isAuth = $currentUser && pb.authStore.isValid;
			if (isAuth && !authorized) {
				// User just logged in - check for default password
				checkDefaultPassword();
				authorized = true;
				loading = false;
			} else if (!isAuth && authorized) {
				// User just logged out - redirect
				authorized = false;
				goto('/admin/login');
			}
		}
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
	{@render children?.()}
{:else if authorized}
	<div class="min-h-screen bg-gray-50 dark:bg-gray-900 overflow-x-hidden">
		<AdminHeader />

		<div class="flex min-w-0 overflow-hidden">
			<!-- Mobile backdrop overlay - closes sidebar when clicked -->
			{#if isMobile && $adminSidebarOpen}
				<button
					type="button"
					class="fixed inset-0 bg-black/50 z-20 lg:hidden"
					onclick={() => adminSidebarOpen.set(false)}
					aria-label="Close navigation menu"
				></button>
			{/if}

			<AdminSidebar {isMobile} />

			<!-- Main content: 
				- Mobile: full width (no margin)
				- Desktop: margin-left based on sidebar state (preserves current behavior)
			-->
			<main 
				id="main-content" 
				class="flex-1 min-w-0 p-4 lg:p-6 mt-16 transition-all duration-200 overflow-x-hidden w-full max-w-full
					{isMobile ? '' : ($adminSidebarOpen ? 'lg:ml-64' : 'lg:ml-16')}"
			>
				{#key $demoMode}
					{@render children?.()}
				{/key}
			</main>
		</div>

		<!-- Password change modal (blocks all access until password is changed) -->
		{#if showPasswordChangeModal}
			<PasswordChangeModal onPasswordChanged={handlePasswordChanged} />
		{/if}
	</div>
{:else}
	<!-- CRITICAL SECURITY: Fallback for any edge case where user is not authenticated -->
	<!-- This prevents blank pages AND unauthorized access to admin panel -->
	<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
		<div class="text-center">
			<div class="w-16 h-16 mx-auto mb-4 rounded-full bg-red-100 dark:bg-red-900 flex items-center justify-center">
				<svg class="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
				</svg>
			</div>
			<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">Authentication Required</h2>
			<p class="text-gray-600 dark:text-gray-400 mb-6">You must be logged in to access the admin panel.</p>
			<a href="/admin/login" class="inline-block px-6 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors">
				Go to Login
			</a>
		</div>
	</div>
{/if}
