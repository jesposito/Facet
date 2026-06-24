<script lang="ts">
	import { goto } from '$app/navigation';
	import { t } from 'svelte-i18n';
	import { pb, currentUser, performLogout } from '$lib/pocketbase';
	import { adminSidebarOpen } from '$lib/stores';
	import { userPreferences } from '$lib/stores/userPreferences';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';

	function toggleLivePreview() {
		userPreferences.setPref('livePreviewEnabled', !$userPreferences.livePreviewEnabled);
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

	async function logout() {
		await performLogout();
		goto('/admin/login');
	}
</script>

<header
	class="admin-header fixed top-0 left-0 right-0 h-16 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 z-40"
>
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
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
				</svg>
				<span class="hidden sm:inline">{$t('admin.header.view_site')}</span>
				<span class="sr-only">(opens in new tab)</span>
			</a>

			<!-- Live site preview toggle. Hidden on screens <lg because the
			     right-column iframe is hidden at that breakpoint anyway —
			     no value in toggling something invisible. -->
			<button
				type="button"
				role="switch"
				aria-checked={$userPreferences.livePreviewEnabled}
				aria-label={$t('admin.header.live_preview_toggle')}
				onclick={toggleLivePreview}
				class="hidden lg:inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {$userPreferences.livePreviewEnabled
					? 'bg-blue-100 dark:bg-blue-900/40 text-blue-800 dark:text-blue-200 hover:bg-blue-200 dark:hover:bg-blue-900/60'
					: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'}"
				title={$t('admin.header.live_preview_toggle')}
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
				</svg>
				<span class="hidden xl:inline">{$t('admin.header.live_preview_label')}</span>
				<span aria-hidden="true" class="hidden xl:inline text-xs"
					>{$userPreferences.livePreviewEnabled
						? $t('admin.header.live_preview_on')
						: $t('admin.header.live_preview_off')}</span
				>
			</button>

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

<style>
	/* Soft Premium ONLY: turn the admin top bar into a refined translucent glass
	   surface. Classic receives no rule here, so its bg-white/dark:bg-gray-800 +
	   gray borders render byte-identical to today. The [data-design] attribute
	   lives on <html>, outside this component, so the selector is wrapped in
	   :global(). The translucent fill is derived from the warm --surface token
	   (auto light/dark) via color-mix; --border-hairline supplies a soft warm
	   bottom edge. Contrast of header text/icons is unchanged: --ink (#2a2522 on
	   light, #f4eee4 on dark) stays well above 4.5:1 against the ~80%-opaque warm
	   surface, and a hard fallback keeps a solid surface where backdrop-filter is
	   unsupported. */
	:global([data-design='soft-premium']) .admin-header {
		background-color: var(--surface);
		background-color: color-mix(in srgb, var(--surface) 80%, transparent);
		border-bottom-color: var(--border-hairline);
		-webkit-backdrop-filter: blur(20px) saturate(1.4);
		backdrop-filter: blur(20px) saturate(1.4);
	}
</style>
