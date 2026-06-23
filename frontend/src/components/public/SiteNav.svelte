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
		ssrNavMode?: string;
		ssrNavPosition?: string;
		ssrNavItems?: NavItem[];
		/** Where this instance is placed in the page. Renders only when slot matches configured position. */
		slot?: 'above' | 'below';
	}

	let {
		ctaUrl = '',
		ctaButtonText = 'Learn More',
		ctaText = '',
		ctaEnabled = true,
		ssrNavEnabled = false,
		ssrNavMode = '',
		ssrNavPosition = '',
		ssrNavItems = [],
		slot = 'below'
	}: Props = $props();

	// Use SSR props directly for initial render (no store delay).
	// Store syncs later for client-side navigation updates.
	let navEnabled = $derived(ssrNavEnabled || $siteNavStore.enabled);
	let navMode = $derived(($siteNavStore.loaded ? $siteNavStore.mode : ssrNavMode) || ssrNavMode || 'bar');
	let navPosition = $derived<'above' | 'below'>(
		(($siteNavStore.loaded ? $siteNavStore.position : ssrNavPosition) === 'above' ? 'above' : 'below')
	);
	let navItems = $derived(ssrNavEnabled ? ssrNavItems : $siteNavStore.items);
	let mobileMenuOpen = $derived($siteNavStore.mobileMenuOpen);
	// Only the slot matching the configured position renders.
	let slotMatches = $derived(slot === navPosition);

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

{#if slotMatches && navEnabled && navItems.length > 0}
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
		<nav aria-label="Site navigation" class="sp-masthead grain bg-primary-600 text-white">
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
{:else if slot === 'below' && ($siteNavStore.loaded || ssrNavEnabled === false) && ctaUrl && ctaText && ctaEnabled}
	<!-- Fallback to traditional CTA banner when nav is disabled. Only render in 'below' slot to avoid double rendering. -->
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

<style>
	/* ──────────────────────────────────────────────────────────────────────
	   Soft Premium "grain masthead" treatment for the bar-mode site nav.
	   ENTIRELY gated behind :global([data-design='soft-premium']) so the
	   classic design renders byte-identical (no overrides emitted).
	   Visual-only: structure, behavior, a11y, and i18n are untouched.

	   Contrast: bar background is --ink (#2a2522). White link text on that
	   ground is ~14.5:1 (AAA); --gray-200 (#ece6e0) inactive text is ~12:1.
	   Hover/active overlays are translucent white over the dark ground so
	   text contrast only ever increases. Focus ring is handled globally by
	   --focus-ring (amber) under soft-premium.
	   ────────────────────────────────────────────────────────────────────── */

	/* Warm the bar to the editorial ink ground and let .grain paint over it.
	   The grain texture (.grain::after) is defined in app.css and only renders
	   under soft-premium, so this stays in lockstep with classic = no-op. */
	:global([data-design='soft-premium']) .sp-masthead {
		background-color: var(--ink);
		/* Hairline keeps the masthead anchored against the page below it. */
		border-bottom: 1px solid var(--border-hairline);
	}

	/* Keep the nav content above the grain texture overlay (which is an
	   absolutely-positioned ::after with no z-index of its own). */
	:global([data-design='soft-premium']) .sp-masthead > div {
		position: relative;
		z-index: 1;
	}

	/* Links + hamburger: AAA white on the ink ground. Covers home, items,
	   and the mobile dropdown links in both rest and active states. */
	:global([data-design='soft-premium']) .sp-masthead a:not(.btn),
	:global([data-design='soft-premium']) .sp-masthead button {
		color: #ffffff;
	}

	/* Inactive items get a touch of warmth but stay >=12:1 (well past AAA). */
	:global([data-design='soft-premium']) .sp-masthead a:not(.btn):not([aria-current='page']) {
		color: rgb(var(--gray-200-rgb));
	}

	/* Hover/focus on inactive links: translucent warm wash (raises contrast). */
	:global([data-design='soft-premium']) .sp-masthead a:not(.btn):not([aria-current='page']):hover,
	:global([data-design='soft-premium']) .sp-masthead button:hover {
		background-color: rgb(255 255 255 / 0.1);
		color: #ffffff;
	}

	/* Active (aria-current) link reads as a solid accent-warm chip. */
	:global([data-design='soft-premium']) .sp-masthead a:not(.btn)[aria-current='page'] {
		background-color: rgb(255 255 255 / 0.16);
		color: #ffffff;
	}

	/* CTA pill: invert to the ink-on-cream editorial button. */
	:global([data-design='soft-premium']) .sp-masthead a.btn {
		background-color: rgb(var(--gray-50-rgb));
		color: var(--ink);
	}
	:global([data-design='soft-premium']) .sp-masthead a.btn:hover {
		background-color: rgb(var(--gray-200-rgb));
	}

	/* Mobile dropdown divider → warm hairline to match the masthead. */
	:global([data-design='soft-premium']) .sp-masthead #mobile-menu {
		border-top-color: var(--border-hairline);
	}
</style>
