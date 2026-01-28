<script lang="ts">
	import { t } from 'svelte-i18n';
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';

	interface Testimonial {
		id: string;
		content: string;
		relationship: string;
		author_name: string;
		author_title: string;
		author_company: string;
		author_photo?: string;
		verification_method: string;
		verification_identifier: string;
		featured: boolean;
	}

	interface Props {
		items: Testimonial[];
		layout?: 'wall' | 'carousel' | 'featured';
		featuredId?: string;
	}

	let { items, layout = 'wall', featuredId }: Props = $props();

	// Carousel state
	let carouselContainer: HTMLDivElement | null = $state(null);
	let currentIndex = $state(0);
	let isScrolling = $state(false);
	let scrollTimeout: ReturnType<typeof setTimeout> | null = null;

	// Handle manual scroll to sync currentIndex with scroll position
	function handleScroll() {
		if (!carouselContainer || !browser || isScrolling) return;

		// Debounce scroll handling
		if (scrollTimeout) clearTimeout(scrollTimeout);
		scrollTimeout = setTimeout(() => {
			updateCurrentIndexFromScroll();
		}, 50);
	}

	function updateCurrentIndexFromScroll() {
		if (!carouselContainer) return;

		const children = Array.from(carouselContainer.children) as HTMLElement[];
		if (children.length === 0) return;

		const containerRect = carouselContainer.getBoundingClientRect();
		const containerCenter = containerRect.left + containerRect.width / 2;

		// Find the child closest to the center of the container
		let closestIndex = 0;
		let closestDistance = Infinity;

		children.forEach((child, index) => {
			const childRect = child.getBoundingClientRect();
			const childCenter = childRect.left + childRect.width / 2;
			const distance = Math.abs(childCenter - containerCenter);

			if (distance < closestDistance) {
				closestDistance = distance;
				closestIndex = index;
			}
		});

		if (currentIndex !== closestIndex) {
			currentIndex = closestIndex;
		}
	}

	function scrollToIndex(index: number) {
		if (!carouselContainer || !browser) return;
		const children = carouselContainer.children;
		if (children[index]) {
			isScrolling = true;
			children[index].scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'center' });
			currentIndex = index;
			// Reset scrolling flag after animation completes
			setTimeout(() => {
				isScrolling = false;
			}, 500);
		}
	}

	function scrollPrev(maxIndex: number) {
		// Loop to end if at beginning
		const newIndex = currentIndex === 0 ? maxIndex - 1 : currentIndex - 1;
		scrollToIndex(newIndex);
	}

	function scrollNext(maxIndex: number) {
		// Loop to beginning if at end
		const newIndex = currentIndex === maxIndex - 1 ? 0 : currentIndex + 1;
		scrollToIndex(newIndex);
	}


	onMount(() => {
		return () => {
			if (scrollTimeout) clearTimeout(scrollTimeout);
		};
	});
</script>

<section id="testimonials" class="mb-16">
	<h2 class="section-title">{$t('public.sections.testimonials')}</h2>

	{#if layout === 'wall'}
		<div class="columns-1 md:columns-2 gap-4 space-y-4">
			{#each items as item (item.id)}
				<div class="break-inside-avoid bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
					<blockquote class="text-gray-700 dark:text-gray-300 mb-4">
						"{item.content}"
					</blockquote>
					<div class="flex items-center gap-3">
						{#if item.author_photo}
							<img
								src={item.author_photo}
								alt={$t('public.testimonials.photo_alt', { values: { name: item.author_name } })}
								class="w-10 h-10 rounded-full object-cover"
							/>
						{:else}
							<div class="w-10 h-10 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
								<span class="text-sm font-medium text-primary-600 dark:text-primary-400">
									{item.author_name.charAt(0).toUpperCase()}
								</span>
							</div>
						{/if}
						<div class="min-w-0">
							<div class="flex items-center gap-2">
								<span class="font-medium text-gray-900 dark:text-white truncate">{item.author_name}</span>
								{#if item.verification_method && item.verification_method !== 'none'}
									<svg class="w-4 h-4 text-green-500 shrink-0" fill="currentColor" viewBox="0 0 20 20">
										<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
									</svg>
								{/if}
							</div>
							{#if item.author_title || item.author_company}
								<p class="text-sm text-gray-500 dark:text-gray-400 truncate">
									{item.author_title}{item.author_title && item.author_company ? ` ${$t('public.testimonials.at')} ` : ''}{item.author_company}
								</p>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if layout === 'carousel'}
		{@const featuredItems = featuredId 
			? items.filter(t => t.id === featuredId)
			: items.filter(t => t.featured)}
		{@const displayItems = featuredItems.length > 0 ? featuredItems : items}
		<div class="relative">
			<!-- Navigation buttons (endless loop, no disabled state) -->
			{#if displayItems.length > 1}
				<button
					onclick={() => scrollPrev(displayItems.length)}
					class="absolute left-0 top-1/2 -translate-y-1/2 z-10 p-2 rounded-full bg-white dark:bg-gray-800 shadow-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
					aria-label={$t('public.testimonials.previous')}
				>
					<svg class="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
					</svg>
				</button>
				<button
					onclick={() => scrollNext(displayItems.length)}
					class="absolute right-0 top-1/2 -translate-y-1/2 z-10 p-2 rounded-full bg-white dark:bg-gray-800 shadow-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
					aria-label={$t('public.testimonials.next')}
				>
					<svg class="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
					</svg>
				</button>
			{/if}

			<!-- Carousel container -->
			<div
				bind:this={carouselContainer}
				onscroll={handleScroll}
				class="flex gap-6 overflow-x-auto snap-x snap-mandatory pb-4 px-8 scrollbar-hide"
				role="region"
				aria-label={$t('public.testimonials.carousel_label')}
			>
				{#each displayItems as item, i (item.id)}
					<div class="snap-center shrink-0 w-full md:w-[calc(50%-12px)] lg:w-[calc(33.333%-16px)] first:ml-0">
						<div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm h-full">
							<blockquote class="text-gray-700 dark:text-gray-300 mb-4 line-clamp-4">
								"{item.content}"
							</blockquote>
							<div class="flex items-center gap-3">
								{#if item.author_photo}
									<img src={item.author_photo} alt={$t('public.testimonials.photo_alt', { values: { name: item.author_name } })} class="w-10 h-10 rounded-full object-cover" />
								{:else}
									<div class="w-10 h-10 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
										<span class="text-sm font-medium text-primary-600 dark:text-primary-400">
											{item.author_name.charAt(0).toUpperCase()}
										</span>
									</div>
								{/if}
								<div>
									<span class="font-medium text-gray-900 dark:text-white">{item.author_name}</span>
									{#if item.author_title || item.author_company}
										<p class="text-sm text-gray-500 dark:text-gray-400">
											{item.author_title}{item.author_title && item.author_company ? ', ' : ''}{item.author_company}
										</p>
									{/if}
								</div>
							</div>
						</div>
					</div>
				{/each}
			</div>

		</div>
	{:else if layout === 'featured'}
		{@const featured = (featuredId ? items.find(t => t.id === featuredId) : items.find(t => t.featured)) || items[0]}
		{#if featured}
			<div class="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 p-6 sm:p-8 md:p-12 text-center shadow-lg">
				<svg class="w-10 h-10 sm:w-12 sm:h-12 mx-auto text-primary-400 dark:text-primary-500 mb-4 sm:mb-6" fill="currentColor" viewBox="0 0 24 24">
					<path d="M14.017 21v-7.391c0-5.704 3.731-9.57 8.983-10.609l.995 2.151c-2.432.917-3.995 3.638-3.995 5.849h4v10h-9.983zm-14.017 0v-7.391c0-5.704 3.748-9.57 9-10.609l.996 2.151c-2.433.917-3.996 3.638-3.996 5.849h3.983v10h-9.983z" />
				</svg>
				<blockquote class="text-lg sm:text-xl md:text-2xl text-gray-800 dark:text-gray-100 font-medium mb-6 sm:mb-8 max-w-3xl mx-auto">
					"{featured.content}"
				</blockquote>
				<div class="flex flex-col sm:flex-row items-center justify-center gap-2 sm:gap-3">
					{#if featured.author_photo}
						<img src={featured.author_photo} alt={$t('public.testimonials.photo_alt', { values: { name: featured.author_name } })} class="w-12 h-12 rounded-full object-cover" />
					{:else}
						<div class="w-12 h-12 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
							<span class="text-lg font-medium text-primary-600 dark:text-primary-400">
								{featured.author_name.charAt(0).toUpperCase()}
							</span>
						</div>
					{/if}
					<div class="text-center sm:text-left">
						<div class="flex items-center justify-center sm:justify-start gap-2">
							<span class="font-semibold text-gray-900 dark:text-white">{featured.author_name}</span>
							{#if featured.verification_method && featured.verification_method !== 'none'}
								<svg class="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20">
									<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
								</svg>
							{/if}
						</div>
						{#if featured.author_title || featured.author_company}
							<p class="text-gray-600 dark:text-gray-300">
								{featured.author_title}{featured.author_title && featured.author_company ? ` ${$t('public.testimonials.at')} ` : ''}{featured.author_company}
							</p>
						{/if}
					</div>
				</div>
			</div>
		{/if}
	{/if}
</section>

<style>
	.scrollbar-hide {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
	.scrollbar-hide::-webkit-scrollbar {
		display: none;
	}
</style>
