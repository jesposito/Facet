<script lang="ts">
	import { page } from '$app/stores';
	import { afterNavigate } from '$app/navigation';
	import { t } from 'svelte-i18n';
	import { siteNavStore } from '$lib/stores/siteNav';

	interface NavItem {
		viewId: string;
		slug: string;
		label: string;
		name: string;
	}

	interface Props {
		ctaUrl?: string;
		ctaButtonText?: string;
		ctaText?: string;
		ctaEnabled?: boolean;
		ssrNavEnabled?: boolean;
		ssrNavItems?: NavItem[];
	}

	let {
		ctaUrl = '',
		ctaButtonText = 'Learn More',
		ctaText = '',
		ctaEnabled = true,
		ssrNavEnabled = false,
		ssrNavItems = []
	}: Props = $props();

	// Use SSR props directly for initial render (no store delay).
	// Store syncs later for client-side navigation updates.
	let navEnabled = $derived(ssrNavEnabled || $siteNavStore.enabled);
	let navMode = $derived($siteNavStore.mode || 'bar');
	let navItems = $derived(ssrNavEnabled ? ssrNavItems : $siteNavStore.items);
	let mobileMenuOpen = $derived($siteNavStore.mobileMenuOpen);

	// Close mobile menu on navigation
	afterNavigate(() => {
		siteNavStore.closeMobileMenu();
	});

	// Get current path to highlight active nav item
	let currentPath = $derived($page.url.pathname);

	function isActive(slug: string): boolean {
		return currentPath === `/${slug}`;
	}

	function isHome(): boolean {
		return currentPath === '/';
	}

	function toggleMobileMenu() {
		siteNavStore.toggleMobileMenu();
	}

	function closeMobileMenu() {
		siteNavStore.closeMobileMenu();
	}
</script>

{#if navEnabled && navItems.length > 0}
	{#if navMode === 'chips'}
		<!-- Chips mode: light surface with rounded-full pill links -->
		<nav aria-label="Site navigation" class="bg-white border-b border-stone-200 dark:bg-stone-900 dark:border-stone-700">
			<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
				<!-- Desktop Navigation -->
				<div class="hidden md:flex items-center justify-between py-3">
					<div class="flex items-center gap-1 flex-wrap">
						<!-- Home button -->
						<a
							href="/"
							aria-current={isHome() ? 'page' : undefined}
							class="inline-flex items-center min-h-11 px-4 rounded-full text-sm font-medium transition-colors
								focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2
								focus-visible:ring-primary-600 dark:focus-visible:ring-primary-400
								focus-visible:ring-offset-white dark:focus-visible:ring-offset-stone-900
								{isHome()
									? 'bg-primary-600 text-white shadow-sm'
									: 'text-stone-600 hover:bg-stone-100 hover:text-stone-900 dark:text-stone-300 dark:hover:bg-stone-800 dark:hover:text-white'}"
						>
							{$t('public.nav.home')}
						</a>

						<!-- Nav items -->
						{#each navItems as item (item.viewId)}
							<a
								href="/{item.slug}"
								aria-current={isActive(item.slug) ? 'page' : undefined}
								class="inline-flex items-center min-h-11 px-4 rounded-full text-sm font-medium transition-colors
									focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2
									focus-visible:ring-primary-600 dark:focus-visible:ring-primary-400
									focus-visible:ring-offset-white dark:focus-visible:ring-offset-stone-900
									{isActive(item.slug)
										? 'bg-primary-600 text-white shadow-sm'
										: 'text-stone-600 hover:bg-stone-100 hover:text-stone-900 dark:text-stone-300 dark:hover:bg-stone-800 dark:hover:text-white'}"
							>
								{item.label}
							</a>
						{/each}
					</div>

					<!-- CTA Button -->
					{#if ctaUrl && ctaEnabled}
						<a
							href={ctaUrl}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center min-h-11 px-4 rounded-full bg-primary-600 text-white hover:bg-primary-700 text-sm font-medium
								focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2
								focus-visible:ring-primary-600 focus-visible:ring-offset-white dark:focus-visible:ring-offset-stone-900"
						>
							{ctaButtonText}
						</a>
					{/if}
				</div>

				<!-- Mobile Navigation -->
				<div class="md:hidden flex items-center justify-between py-3">
					<button
						type="button"
						class="p-2 rounded-md text-stone-600 hover:bg-stone-100 hover:text-stone-900 dark:text-stone-300 dark:hover:bg-stone-800 dark:hover:text-white
							focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-600 dark:focus-visible:ring-primary-400"
						onclick={toggleMobileMenu}
						aria-expanded={mobileMenuOpen}
						aria-controls="mobile-menu-chips"
					>
						<span class="sr-only">{$t('public.nav.open_main_menu')}</span>
						{#if mobileMenuOpen}
							<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
							</svg>
						{:else}
							<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
							</svg>
						{/if}
					</button>

					{#if ctaUrl && ctaEnabled}
						<a
							href={ctaUrl}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center min-h-11 px-4 rounded-full bg-primary-600 text-white hover:bg-primary-700 text-sm font-medium"
						>
							{ctaButtonText}
						</a>
					{/if}
				</div>
			</div>

			<!-- Mobile menu dropdown (chips) -->
			{#if mobileMenuOpen}
				<div class="md:hidden border-t border-stone-200 dark:border-stone-700" id="mobile-menu-chips">
					<div class="px-2 pt-2 pb-3 space-y-1">
						<a
							href="/"
							onclick={closeMobileMenu}
							aria-current={isHome() ? 'page' : undefined}
							class="block px-3 py-2 rounded-md text-base font-medium
								{isHome()
									? 'bg-primary-600 text-white'
									: 'text-stone-700 hover:bg-stone-100 dark:text-stone-200 dark:hover:bg-stone-800'}"
						>
							{$t('public.nav.home')}
						</a>
						{#each navItems as item (item.viewId)}
							<a
								href="/{item.slug}"
								onclick={closeMobileMenu}
								aria-current={isActive(item.slug) ? 'page' : undefined}
								class="block px-3 py-2 rounded-md text-base font-medium
									{isActive(item.slug)
										? 'bg-primary-600 text-white'
										: 'text-stone-700 hover:bg-stone-100 dark:text-stone-200 dark:hover:bg-stone-800'}"
							>
								{item.label}
							</a>
						{/each}
					</div>
				</div>
			{/if}
		</nav>
	{:else}
		<!-- Bar mode (default): full-width colored bar with pill links -->
		<nav aria-label="Site navigation" class="bg-primary-600 text-white">
			<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
				<!-- Desktop Navigation -->
				<div class="hidden md:flex items-center justify-between py-3">
					<div class="flex items-center gap-1">
						<!-- Home button -->
						<a
							href="/"
							aria-current={isHome() ? 'page' : undefined}
							class="px-3 py-2 rounded-md text-sm font-medium transition-colors
								focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-primary-600
								{isHome()
									? 'bg-primary-700 text-white'
									: 'text-primary-100 hover:bg-primary-500 hover:text-white'}"
						>
							{$t('public.nav.home')}
						</a>

						<!-- Nav items -->
						{#each navItems as item (item.viewId)}
							<a
								href="/{item.slug}"
								aria-current={isActive(item.slug) ? 'page' : undefined}
								class="px-3 py-2 rounded-md text-sm font-medium transition-colors
									focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-primary-600
									{isActive(item.slug)
										? 'bg-primary-700 text-white'
										: 'text-primary-100 hover:bg-primary-500 hover:text-white'}"
							>
								{item.label}
							</a>
						{/each}
					</div>

					<!-- CTA Button (if configured and enabled) -->
					{#if ctaUrl && ctaEnabled}
						<a
							href={ctaUrl}
							target="_blank"
							rel="noopener noreferrer"
							class="btn bg-white text-primary-600 hover:bg-stone-100 text-sm"
						>
							{ctaButtonText}
						</a>
					{/if}
				</div>

				<!-- Mobile Navigation -->
				<div class="md:hidden flex items-center justify-between py-3">
					<!-- Hamburger button -->
					<button
						type="button"
						class="p-2 rounded-md text-primary-100 hover:bg-primary-500 hover:text-white focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white"
						onclick={toggleMobileMenu}
						aria-expanded={mobileMenuOpen}
						aria-controls="mobile-menu"
					>
						<span class="sr-only">{$t('public.nav.open_main_menu')}</span>
						{#if mobileMenuOpen}
							<!-- Close icon -->
							<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
							</svg>
						{:else}
							<!-- Menu icon -->
							<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
							</svg>
						{/if}
					</button>

					<!-- CTA Button on mobile (if configured and enabled) -->
					{#if ctaUrl && ctaEnabled}
						<a
							href={ctaUrl}
							target="_blank"
							rel="noopener noreferrer"
							class="btn bg-white text-primary-600 hover:bg-stone-100 text-sm"
						>
							{ctaButtonText}
						</a>
					{/if}
				</div>
			</div>

			<!-- Mobile menu dropdown -->
			{#if mobileMenuOpen}
				<div class="md:hidden border-t border-primary-500" id="mobile-menu">
					<div class="px-2 pt-2 pb-3 space-y-1">
						<!-- Home button -->
						<a
							href="/"
							onclick={closeMobileMenu}
							aria-current={isHome() ? 'page' : undefined}
							class="block px-3 py-2 rounded-md text-base font-medium
								{isHome()
									? 'bg-primary-700 text-white'
									: 'text-primary-100 hover:bg-primary-500 hover:text-white'}"
						>
							{$t('public.nav.home')}
						</a>

						<!-- Nav items -->
						{#each navItems as item (item.viewId)}
							<a
								href="/{item.slug}"
								onclick={closeMobileMenu}
								aria-current={isActive(item.slug) ? 'page' : undefined}
								class="block px-3 py-2 rounded-md text-base font-medium
									{isActive(item.slug)
										? 'bg-primary-700 text-white'
										: 'text-primary-100 hover:bg-primary-500 hover:text-white'}"
							>
								{item.label}
							</a>
						{/each}

						</div>
				</div>
			{/if}
		</nav>
	{/if}
{:else if ($siteNavStore.loaded || ssrNavEnabled === false) && ctaUrl && ctaText && ctaEnabled}
	<!-- Fallback to traditional CTA banner when nav is disabled -->
	<div class="bg-primary-600 text-white py-4">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center justify-between">
			<span class="font-medium">{ctaText}</span>
			<a
				href={ctaUrl}
				target="_blank"
				rel="noopener noreferrer"
				class="btn bg-white text-primary-600 hover:bg-stone-100"
			>
				{ctaButtonText}
			</a>
		</div>
	</div>
{/if}
