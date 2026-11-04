<script lang="ts">
	import { run } from 'svelte/legacy';

	import { onMount, onDestroy, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import { t } from 'svelte-i18n';
	import { pb, currentUser, performLogout, freshLogin, clearFreshLogin } from '$lib/pocketbase';
	import { adminSidebarOpen } from '$lib/stores';
	import { demoMode, initDemoMode, collection } from '$lib/stores/demo';
	import AdminSidebar from '$components/admin/AdminSidebar.svelte';
	import AdminHeader from '$components/admin/AdminHeader.svelte';
	import LivePreview from '$components/admin/LivePreview.svelte';
	import { userPreferences } from '$lib/stores/userPreferences';
	import PasswordChangeModal from '$components/admin/PasswordChangeModal.svelte';
	import TwoFactorModal from '$components/admin/TwoFactorModal.svelte';
	import SetupWizard from '$components/admin/SetupWizard.svelte';
	import { setupWizard, shouldShowWizard, isSetupIncomplete } from '$lib/stores/setupWizard';
	import { get } from 'svelte/store';
	import type { Profile, View } from '$lib/pocketbase';

	interface Props {
		children?: import('svelte').Snippet;
	}

	let { children }: Props = $props();
	let loading = $state(true);
	let authorized = $state(false);
	let mounted = $state(false);
	let showPasswordChangeModal = $state(false);
	let showTwoFactorModal = $state(false);
	let twoFactorError = $state(false);
	let authCheckError = $state(false);

	// Mobile detection for responsive sidebar behavior
	// Mobile: sidebar is overlay drawer (hidden by default)
	// Desktop: sidebar is persistent (current behavior)
	let isMobile = $state(false);
	const MOBILE_BREAKPOINT = 1024;
	// Below the breakpoint the open sidebar is a modal drawer. The layout owns it because
	// only the layout can reach the header, the sidebar and the main content together.
	const drawerOpen = $derived(isMobile && $adminSidebarOpen);
	const DRAWER_FOCUSABLE = 'a[href], button:not(:disabled)';
	let drawerEl: HTMLDivElement | undefined = $state();
	let menuButton: HTMLButtonElement | undefined = $state();

	let profileData: Profile | null = $state(null);
	let viewsData: View[] = $state([]);
	let showEncryptionWarning = $state(false);
	let encryptionKeySource = $state('');
	let checkingSetupWizard = $state(false);
	let setupCheckFailed = $state(false);
	let setupIncomplete = $state(false);
	let wizardCheckedThisSession = false;

	async function check2FAStatus(): Promise<boolean> {
		try {
			const response = await fetch('/api/totp/status', {
				headers: { Authorization: `Bearer ${pb.authStore.token}` }
			});
			if (!response.ok) {
				console.error('2FA status check failed:', response.status);
				showTwoFactorModal = false;
				twoFactorError = true;
				return true;
			}
			const data = await response.json();
			if (data.enabled && !data.verified) {
				twoFactorError = false;
				showTwoFactorModal = true;
				return true;
			}
		} catch (err) {
			console.error('Failed to check 2FA status:', err);
			showTwoFactorModal = false;
			twoFactorError = true;
			return true;
		}
		return false;
	}

	function handleTwoFactorVerified() {
		showTwoFactorModal = false;
		twoFactorError = false;
		authorized = true;
		checkSetupWizard();
		checkEncryptionKeyStatus();
	}

	async function handleTwoFactorLogout() {
		showTwoFactorModal = false;
		twoFactorError = false;
		await performLogout();
		goto('/admin/login');
	}

	async function checkEncryptionKeyStatus() {
		try {
			const dismissed = localStorage.getItem('facet_encryption_warning_dismissed');
			if (dismissed === 'true') return;

			const response = await fetch('/api/system/encryption-status', {
				headers: { Authorization: `Bearer ${pb.authStore.token}` }
			});
			if (response.ok) {
				const data = await response.json();
				if (data.source === 'auto' || data.source === 'file') {
					encryptionKeySource = data.source;
					showEncryptionWarning = true;
				}
			}
		} catch (err) {
			// Silently ignore — non-critical check
		}
	}

	function dismissEncryptionWarning() {
		showEncryptionWarning = false;
		try {
			localStorage.setItem('facet_encryption_warning_dismissed', 'true');
		} catch {
			// localStorage unavailable
		}
	}

	async function checkDefaultPassword(): Promise<boolean> {
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
					return true;
				}
			}
		} catch (err) {
			console.error('Failed to check default password:', err);
		}
		return false;
	}

	// Auto-open must not steal focus from someone already filling in a form
	function isEditingForm(): boolean {
		const active = document.activeElement;
		return (
			active instanceof HTMLElement &&
			active.matches('input, textarea, select, [contenteditable="true"]')
		);
	}

	async function checkSetupWizard() {
		// Only check wizard once per page load — prevents nagging on every auth state change
		if (wizardCheckedThisSession) return;

		// Early exit: a finished wizard needs no API calls. A dismissed one still
		// does, because the Resume action depends on whether setup is incomplete.
		if (get(setupWizard).completedWizard) {
			wizardCheckedThisSession = true;
			return;
		}

		if (checkingSetupWizard) return;
		checkingSetupWizard = true;
		try {
			// The sidebar and dashboard load these collections at the same moment. Each
			// request here needs its own key or the SDK cancels one in favour of the other.
			const [profileRes, viewsRes] = await Promise.all([
				collection('profile').getList(1, 1, { requestKey: 'setup-wizard-profile' }),
				collection('views').getList(1, 100, { requestKey: 'setup-wizard-views' })
			]);

			profileData = (profileRes.items[0] as unknown as Profile) || null;
			viewsData = viewsRes.items as unknown as View[];
			setupIncomplete = isSetupIncomplete(profileData, viewsData, $demoMode);
			setupCheckFailed = false;
			wizardCheckedThisSession = true;

			if (setupIncomplete) {
				setupWizard.prefillProfile(profileData);
			}
			// An open drawer also holds the keyboard, and the Resume banner covers a skipped open
			if (shouldShowWizard(profileData, viewsData, $demoMode) && !isEditingForm() && !drawerOpen) {
				setupWizard.open();
			}
		} catch (err) {
			// A failed or cancelled check says nothing about setup, so keep the wizard
			// closed, leave this page load unchecked and offer a retry.
			if (!(err as { isAbort?: boolean }).isAbort) {
				console.error('Failed to check setup wizard:', err);
			}
			setupCheckFailed = true;
		} finally {
			checkingSetupWizard = false;
		}
	}

	// Media query listener cleanup
	let mediaQueryCleanup: (() => void) | null = null;

	onMount(async () => {
		mounted = true;

		// Set up mobile detection with media query
		const mediaQuery = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
		isMobile = mediaQuery.matches;

		const handleMediaChange = async (e: MediaQueryListEvent) => {
			const drawerWasOpen = drawerOpen;
			isMobile = e.matches;
			// When switching to mobile, close sidebar by default
			if (e.matches) {
				adminSidebarOpen.set(false);
			} else if (drawerWasOpen) {
				// The close control does not exist on desktop, so focus may need a new home
				await tick();
				if (document.activeElement === document.body) menuButton?.focus();
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
		// Use authRefresh to validate token server-side and prevent redirect loops
		if (pb.authStore.isValid) {
			try {
				// Skip authRefresh if we just logged in — the token is already valid
				// and re-validating immediately can cause a 401 race condition
				if (!freshLogin) {
					await pb.collection('users').authRefresh();
				}
				clearFreshLogin();
				// Token is valid - proceed
				const needsPasswordChange = await checkDefaultPassword();
				if (!needsPasswordChange) {
					const needs2FA = await check2FAStatus();
					if (!needs2FA) {
						checkSetupWizard();
						checkEncryptionKeyStatus();
						authorized = true;
					}
				}
				loading = false;
			} catch (err) {
				const authError = err as { status?: number; response?: { status?: number } };
				const status = authError.status || authError.response?.status || 0;
				if (status === 0 || status >= 500) {
					console.warn('[LAYOUT] Unable to verify auth state; keeping admin content locked');
					authorized = false;
					authCheckError = true;
				} else {
					// Token validation failed (401/403) - clear stale auth and redirect to login
					console.log(
						'[LAYOUT] Auth validation failed (status:',
						status,
						'), clearing stale auth state'
					);
					pb.authStore.clear();
					// Clear the cookie to prevent redirect loops
					document.cookie = 'pb_auth=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';
					goto('/admin/login');
				}
				loading = false;
			}
		} else {
			// Not authenticated at all - redirect to login immediately
			loading = false;
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
	afterNavigate(({ shallow, type }) => {
		if (shallow && type === 'goto') return;

		if (isMobile && $adminSidebarOpen) {
			adminSidebarOpen.set(false);
		}
	});

	// Explicit closes return focus to the trigger. A navigation does not call this, so
	// SvelteKit's focus handling for the destination page stays in charge.
	async function closeDrawer() {
		adminSidebarOpen.set(false);
		// The trigger stays inert until the layout has re-rendered
		await tick();
		menuButton?.focus();
	}

	function handleDrawerKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			// An open quick-create menu takes this Escape through its own listener
			if (!document.querySelector('[data-quick-create-menu]')) closeDrawer();
		} else if (e.key === 'Tab' && drawerEl) {
			const focusable = drawerEl.querySelectorAll<HTMLElement>(DRAWER_FOCUSABLE);
			const first = focusable[0];
			const last = focusable[focusable.length - 1];
			const current = document.activeElement;
			if (!drawerEl.contains(current) || current === (e.shiftKey ? first : last)) {
				e.preventDefault();
				(e.shiftKey ? last : first)?.focus();
			}
		}
	}

	// While the drawer is open, hold the page still and keep the keyboard inside it
	$effect(() => {
		if (!drawerOpen || !drawerEl) return;
		const bodyOverflow = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		document.addEventListener('keydown', handleDrawerKeydown);
		drawerEl.querySelector<HTMLElement>(DRAWER_FOCUSABLE)?.focus();

		return () => {
			document.removeEventListener('keydown', handleDrawerKeydown);
			document.body.style.overflow = bodyOverflow;
		};
	});

	async function handlePasswordChanged() {
		// Password was successfully changed - hide modal and check 2FA before authorizing
		showPasswordChangeModal = false;

		// Refresh user data to get updated password_changed_from_default field
		if ($currentUser) {
			try {
				const updatedUser = await pb.collection('users').getOne($currentUser.id);
				currentUser.set(updatedUser);
			} catch (err) {
				console.error('Failed to refresh user data:', err);
			}
		}

		// Now check 2FA — password change must NOT bypass the 2FA gate
		const needs2FA = await check2FAStatus();
		if (!needs2FA) {
			authorized = true;
			checkSetupWizard();
			checkEncryptionKeyStatus();
		}
		// If needs2FA, the modal is now showing and authorized stays false
	}
	// Check if we're on the login page (don't require auth there)
	let isLoginPage = $derived(page.url.pathname === '/admin/login');
	// Handle login page - always stop loading immediately
	run(() => {
		if (isLoginPage) {
			loading = false;
			authorized = false;
		}
	});
	run(() => {
		if (mounted && !isLoginPage) {
			const isAuth = $currentUser && pb.authStore.isValid;
			if (
				isAuth &&
				!authorized &&
				!showTwoFactorModal &&
				!showPasswordChangeModal &&
				!twoFactorError &&
				!authCheckError &&
				!loading
			) {
				// Show loading spinner during auth checks to prevent "Unauthorized" flash
				loading = true;
				(async () => {
					try {
						// Initialize demo mode if not done yet (skipped when entering from login page)
						await initDemoMode();

						const needsPasswordChange = await checkDefaultPassword();
						if (!needsPasswordChange) {
							const needs2FA = await check2FAStatus();
							if (!needs2FA) {
								authorized = true;
								checkSetupWizard();
								checkEncryptionKeyStatus();
							}
						}
						// If needsPasswordChange: showPasswordChangeModal is now true.
						// authorized stays false — the modal renders outside the authorized guard.
					} finally {
						loading = false;
					}
				})();
			} else if (!isAuth && authorized) {
				// User is no longer authenticated (passive token expiry or active logout)
				// Set loading=true BEFORE clearing authorized to prevent "Unauthorized" flash
				loading = true;
				authorized = false;
				if (pb.authStore.isValid) {
					pb.authStore.clear();
					document.cookie = 'pb_auth=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';
				}
				goto('/admin/login', { replaceState: true });
			}
		}
	});
</script>

{#if loading}
	<div
		class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900"
		role="status"
		aria-label={$t('admin.layout.loading')}
	>
		<div class="animate-pulse text-center">
			<div
				class="w-12 h-12 mx-auto mb-4 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center"
			>
				<svg
					class="w-6 h-6 text-primary-600 animate-spin"
					fill="none"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
					></circle>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					></path>
				</svg>
			</div>
			<p class="text-gray-600 dark:text-gray-400">{$t('admin.layout.loading')}</p>
		</div>
	</div>
{:else if isLoginPage}
	<!-- Login page renders without admin chrome -->
	{@render children?.()}
{:else if authorized}
	<div class="admin-shell min-h-screen bg-gray-50 dark:bg-gray-900 overflow-x-clip">
		<AdminHeader bind:menuButton inert={drawerOpen} />

		<div class="flex min-w-0 overflow-x-clip">
			<!-- Mobile backdrop overlay - closes sidebar when clicked -->
			{#if isMobile && $adminSidebarOpen}
				<button
					type="button"
					tabindex="-1"
					class="fixed inset-0 bg-black/50 z-20 lg:hidden animate-fade-in"
					onclick={closeDrawer}
					aria-label={$t('admin.layout.close_navigation')}
				>
					<!-- Visible tap-to-close affordance -->

					<span
						class="absolute right-4 top-20 flex items-center gap-2 bg-white/90 dark:bg-gray-800/90 backdrop-blur-sm rounded-full px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 shadow-lg"
					>
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M6 18L18 6M6 6l12 12"
							></path>
						</svg>
						{$t('shared.actions.close')}
					</span>
				</button>
			{/if}

			<!-- Dialog semantics sit on this wrapper because <aside> cannot take role="dialog" -->
			<div
				bind:this={drawerEl}
				class={drawerOpen ? 'fixed inset-y-0 left-0 z-30 w-64' : undefined}
				role={drawerOpen ? 'dialog' : undefined}
				aria-modal={drawerOpen ? 'true' : undefined}
				aria-label={drawerOpen ? $t('shared.aria.admin_navigation') : undefined}
				inert={isMobile && !$adminSidebarOpen}
			>
				<AdminSidebar {isMobile} onClose={closeDrawer} />
			</div>

			<!-- Main content:
				- Mobile: full width (no margin)
				- Desktop: margin-left based on sidebar state (preserves current behavior)
				- When live preview is ON, allow main to shrink so the preview
				  column gets reserved space at lg+. Below lg the preview is
				  hidden via its own `lg:` gate so main stays full.
			-->
			<main
				id="main-content"
				inert={drawerOpen}
				class="flex-1 min-w-0 p-4 lg:p-6 mt-16 transition-all duration-200 overflow-x-clip w-full max-w-full
					{isMobile ? '' : $adminSidebarOpen ? 'lg:ml-64' : 'lg:ml-16'}
					{!isMobile && $userPreferences.livePreviewEnabled
					? 'lg:max-w-[calc(50%-1rem)] xl:max-w-[calc(50%-1.5rem)]'
					: ''}"
			>
				{#if showEncryptionWarning}
					<div
						class="mb-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4"
						role="status"
					>
						<div class="flex items-start gap-3">
							<svg
								class="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								aria-hidden="true"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
								></path>
							</svg>
							<div class="flex-1">
								<h3 class="text-sm font-medium text-blue-800 dark:text-blue-200">
									{$t('admin.layout.encryption_info_title')}
								</h3>
								<p class="text-sm text-blue-700 dark:text-blue-300 mt-1">
									{$t('admin.layout.encryption_info_message')}
								</p>
							</div>
							<button
								type="button"
								onclick={dismissEncryptionWarning}
								class="flex-shrink-0 text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-200"
								aria-label={$t('admin.layout.encryption_warning_dismiss')}
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M6 18L18 6M6 6l12 12"
									></path>
								</svg>
							</button>
						</div>
					</div>
				{/if}
				{#if setupCheckFailed}
					<div
						class="mb-4 flex flex-wrap items-center justify-between gap-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4"
						role="alert"
					>
						<p class="text-sm text-amber-800 dark:text-amber-200">
							{$t('admin.wizard.check_failed')}
						</p>
						<button
							type="button"
							class="btn btn-secondary"
							disabled={checkingSetupWizard}
							onclick={checkSetupWizard}
						>
							{$t('admin.wizard.check_retry')}
						</button>
					</div>
				{:else if setupIncomplete}
					<div
						class="mb-4 flex flex-wrap items-center justify-between gap-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4"
						role="status"
					>
						<p class="text-sm text-blue-800 dark:text-blue-200">
							{$t('admin.wizard.resume_message')}
						</p>
						<button
							id="resume-setup-button"
							type="button"
							class="btn btn-primary"
							onclick={() => setupWizard.resume()}
						>
							{$t('admin.wizard.resume_button')}
						</button>
					</div>
				{/if}
				{@render children?.()}
			</main>

			<!-- Live site preview column.
			     - Hidden via Tailwind `hidden lg:block` so mobile + tablet
			       never render the iframe (no value on small screens; saves
			       bandwidth + CPU on touch devices).
			     - Mounted only when the toggle is ON. When OFF nothing renders
			       (regression-safe). -->
			{#if !isMobile && $userPreferences.livePreviewEnabled}
				<div class="hidden lg:block flex-1 min-w-0 p-4 lg:p-6 mt-16">
					<LivePreview />
				</div>
			{/if}
		</div>

		<SetupWizard
			blocked={showPasswordChangeModal || showTwoFactorModal}
			onComplete={() => (setupIncomplete = false)}
		/>
	</div>
{:else if authCheckError}
	<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
		<div class="text-center max-w-md mx-auto px-4" role="alert">
			<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
				Couldn't verify your admin session
			</h2>
			<p class="text-gray-600 dark:text-gray-400 mb-6">
				The server could not confirm your session. Admin content stays hidden until verification
				succeeds.
			</p>
			<div class="flex gap-3 justify-center">
				<button type="button" class="btn btn-primary" onclick={() => window.location.reload()}>
					{$t('admin.two_factor.retry_button')}
				</button>
				<button
					type="button"
					class="btn btn-secondary"
					onclick={async () => {
						authCheckError = false;
						await performLogout();
						goto('/admin/login');
					}}
				>
					{$t('admin.two_factor.logout_link')}
				</button>
			</div>
		</div>
	</div>
{:else if twoFactorError}
	<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
		<div class="text-center max-w-md mx-auto px-4">
			<div
				class="w-16 h-16 mx-auto mb-4 rounded-full bg-amber-100 dark:bg-amber-900 flex items-center justify-center"
			>
				<svg
					class="w-8 h-8 text-amber-600 dark:text-amber-400"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
					></path>
				</svg>
			</div>
			<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
				{$t('admin.two_factor.status_check_failed_title')}
			</h2>
			<p class="text-gray-600 dark:text-gray-400 mb-6">
				{$t('admin.two_factor.status_check_failed_message')}
			</p>
			<div class="flex gap-3 justify-center">
				<button
					type="button"
					class="btn btn-primary"
					onclick={async () => {
						twoFactorError = false;
						const needs2FA = await check2FAStatus();
						if (!needs2FA) {
							authorized = true;
							checkSetupWizard();
							checkEncryptionKeyStatus();
						}
					}}
				>
					{$t('admin.two_factor.retry_button')}
				</button>
				<button
					type="button"
					class="btn btn-secondary"
					onclick={async () => {
						twoFactorError = false;
						await performLogout();
						goto('/admin/login');
					}}
				>
					{$t('admin.two_factor.logout_link')}
				</button>
			</div>
		</div>
	</div>
{:else if showPasswordChangeModal || showTwoFactorModal}
	<div class="min-h-screen bg-gray-50 dark:bg-gray-900"></div>
{:else}
	<!-- CRITICAL SECURITY: Fallback for any edge case where user is not authenticated -->
	<!-- This prevents blank pages AND unauthorized access to admin panel -->
	<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
		<div class="text-center">
			<div
				class="w-16 h-16 mx-auto mb-4 rounded-full bg-red-100 dark:bg-red-900 flex items-center justify-center"
			>
				<svg class="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
					></path>
				</svg>
			</div>
			<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
				{$t('errors.unauthorized')}
			</h2>
			<p class="text-gray-600 dark:text-gray-400 mb-6">{$t('errors.unauthorized_message')}</p>
			<a
				href="/admin/login"
				class="inline-block px-6 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
			>
				{$t('errors.go_to_login')}
			</a>
		</div>
	</div>
{/if}

<!-- Blocking modals render outside ALL layout branches — they overlay everything -->
{#if showTwoFactorModal}
	<TwoFactorModal onVerified={handleTwoFactorVerified} onLogout={handleTwoFactorLogout} />
{/if}

{#if showPasswordChangeModal}
	<PasswordChangeModal onPasswordChanged={handlePasswordChanged} />
{/if}
