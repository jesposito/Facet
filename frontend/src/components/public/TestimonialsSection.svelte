<script lang="ts">
	import { browser } from '$app/environment';

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
	}

	let { items, layout = 'wall' }: Props = $props();

	// Carousel state
	let carouselContainer: HTMLDivElement | null = $state(null);
	let currentIndex = $state(0);

	function scrollToIndex(index: number) {
		if (!carouselContainer || !browser) return;
		const children = carouselContainer.children;
		if (children[index]) {
			children[index].scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'start' });
			currentIndex = index;
		}
	}

	function scrollPrev() {
		const newIndex = Math.max(0, currentIndex - 1);
		scrollToIndex(newIndex);
	}

	function scrollNext(maxIndex: number) {
		const newIndex = Math.min(maxIndex - 1, currentIndex + 1);
		scrollToIndex(newIndex);
	}

	function getVerificationLabel(method: string, identifier: string): string | null {
		if (method === 'email') return 'Verified';
		if (method === 'github') return `@${identifier}`;
		if (method === 'twitter') return `@${identifier}`;
		if (method === 'linkedin') return 'LinkedIn';
		return null;
	}

	function getRelationshipLabel(rel: string): string {
		const labels: Record<string, string> = {
			client: 'Client',
			colleague: 'Colleague',
			manager: 'Manager',
			report: 'Direct Report',
			mentor: 'Mentor',
			other: ''
		};
		return labels[rel] || '';
	}
</script>

<section id="testimonials" class="mb-16">
	<h2 class="section-title">Testimonials</h2>

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
								alt=""
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
									{item.author_title}{item.author_title && item.author_company ? ' at ' : ''}{item.author_company}
								</p>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if layout === 'carousel'}
		{@const featuredItems = items.filter(t => t.featured)}
		{@const displayItems = featuredItems.length > 0 ? featuredItems : items}
		<div class="relative">
			<!-- Navigation buttons -->
			{#if displayItems.length > 1}
				<button
					onclick={scrollPrev}
					class="absolute left-0 top-1/2 -translate-y-1/2 z-10 p-2 rounded-full bg-white dark:bg-gray-800 shadow-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
					disabled={currentIndex === 0}
					aria-label="Previous testimonial"
				>
					<svg class="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
					</svg>
				</button>
				<button
					onclick={() => scrollNext(displayItems.length)}
					class="absolute right-0 top-1/2 -translate-y-1/2 z-10 p-2 rounded-full bg-white dark:bg-gray-800 shadow-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
					disabled={currentIndex === displayItems.length - 1}
					aria-label="Next testimonial"
				>
					<svg class="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
					</svg>
				</button>
			{/if}

			<!-- Carousel container -->
			<div
				bind:this={carouselContainer}
				class="flex gap-6 overflow-x-auto snap-x snap-mandatory pb-4 px-8 scrollbar-hide"
			>
				{#each displayItems as item, i (item.id)}
					<div class="snap-center shrink-0 w-full md:w-[calc(50%-12px)] lg:w-[calc(33.333%-16px)] first:ml-0">
						<div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm h-full">
							<blockquote class="text-gray-700 dark:text-gray-300 mb-4 line-clamp-4">
								"{item.content}"
							</blockquote>
							<div class="flex items-center gap-3">
								{#if item.author_photo}
									<img src={item.author_photo} alt="" class="w-10 h-10 rounded-full object-cover" />
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

			<!-- Dot indicators -->
			{#if displayItems.length > 1}
				<div class="flex justify-center gap-2 mt-2">
					{#each displayItems as _, i}
						<button
							onclick={() => scrollToIndex(i)}
							class="w-2 h-2 rounded-full transition-colors {i === currentIndex ? 'bg-primary-600 dark:bg-primary-400' : 'bg-gray-300 dark:bg-gray-600'}"
							aria-label="Go to testimonial {i + 1}"
						></button>
					{/each}
				</div>
			{/if}
		</div>
	{:else if layout === 'featured'}
		{@const featured = items.find(t => t.featured) || items[0]}
		{#if featured}
			<div class="bg-gradient-to-br from-primary-50 to-primary-100 dark:from-gray-800 dark:to-gray-900 dark:border dark:border-primary-800 rounded-2xl p-6 sm:p-8 md:p-12 text-center">
				<svg class="w-10 h-10 sm:w-12 sm:h-12 mx-auto text-primary-300 dark:text-primary-500 mb-4 sm:mb-6" fill="currentColor" viewBox="0 0 24 24">
					<path d="M14.017 21v-7.391c0-5.704 3.731-9.57 8.983-10.609l.995 2.151c-2.432.917-3.995 3.638-3.995 5.849h4v10h-9.983zm-14.017 0v-7.391c0-5.704 3.748-9.57 9-10.609l.996 2.151c-2.433.917-3.996 3.638-3.996 5.849h3.983v10h-9.983z" />
				</svg>
				<blockquote class="text-lg sm:text-xl md:text-2xl text-gray-800 dark:text-gray-100 font-medium mb-6 sm:mb-8 max-w-3xl mx-auto">
					"{featured.content}"
				</blockquote>
				<div class="flex flex-col sm:flex-row items-center justify-center gap-2 sm:gap-3">
					{#if featured.author_photo}
						<img src={featured.author_photo} alt="" class="w-12 h-12 rounded-full object-cover" />
					{:else}
						<div class="w-12 h-12 rounded-full bg-primary-200 dark:bg-primary-900 flex items-center justify-center">
							<span class="text-lg font-medium text-primary-700 dark:text-primary-300">
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
								{featured.author_title}{featured.author_title && featured.author_company ? ' at ' : ''}{featured.author_company}
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
