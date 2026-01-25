<script lang="ts">
	import { page } from '$app/stores';

	interface NavItem {
		slug: string;
		label: string;
		url: string;
		is_home: boolean;
	}

	interface CTAButton {
		url: string;
		label: string;
	}

	interface Props {
		items: NavItem[];
		ctaButton?: CTAButton | null;
	}

	let { items, ctaButton = null }: Props = $props();

	let currentPath = $derived($page.url.pathname);

	function isActive(item: NavItem): boolean {
		if (item.is_home) {
			return currentPath === '/';
		}
		return currentPath === item.url || currentPath.startsWith(item.url + '/');
	}

	let mobileMenuOpen = $state(false);

	function closeMobileMenu() {
		mobileMenuOpen = false;
	}
</script>

{#if items.length > 0}
	<nav class="bg-primary-600 text-white print:hidden">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex items-center justify-between py-3">
				<button
					type="button"
					class="sm:hidden p-2 -ml-2 rounded-md hover:bg-primary-700 transition-colors"
					onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
					aria-expanded={mobileMenuOpen}
					aria-controls="mobile-nav-menu"
					aria-label={mobileMenuOpen ? 'Close navigation menu' : 'Open navigation menu'}
				>
					{#if mobileMenuOpen}
						<svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					{:else}
						<svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
						</svg>
					{/if}
				</button>

				<div class="hidden sm:flex items-center gap-1 overflow-x-auto scrollbar-hide">
					{#each items as item (item.url)}
						<a
							href={item.url}
							class="flex-shrink-0 px-3 py-1.5 text-sm font-medium rounded-full transition-colors
								{isActive(item)
									? 'bg-white/20'
									: 'hover:bg-white/10'}"
							aria-current={isActive(item) ? 'page' : undefined}
						>
							{item.label}
						</a>
					{/each}
				</div>

				{#if ctaButton}
					<a
						href={ctaButton.url}
						target="_blank"
						rel="noopener noreferrer"
						class="flex-shrink-0 px-4 py-1.5 text-sm font-medium rounded-full bg-white text-primary-600 hover:bg-gray-100 transition-colors"
					>
						{ctaButton.label}
					</a>
				{:else}
					<div class="hidden sm:block"></div>
				{/if}
			</div>
		</div>

		{#if mobileMenuOpen}
			<div
				id="mobile-nav-menu"
				class="sm:hidden border-t border-primary-500 bg-primary-700"
			>
				<div class="px-4 py-3 space-y-1">
					{#each items as item (item.url)}
						<a
							href={item.url}
							onclick={closeMobileMenu}
							class="block px-3 py-2 text-base font-medium rounded-md transition-colors
								{isActive(item)
									? 'bg-white/20'
									: 'hover:bg-white/10'}"
							aria-current={isActive(item) ? 'page' : undefined}
						>
							{item.label}
						</a>
					{/each}
				</div>
			</div>
		{/if}
	</nav>
{/if}

<style>
	.scrollbar-hide {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
	.scrollbar-hide::-webkit-scrollbar {
		display: none;
	}
</style>
