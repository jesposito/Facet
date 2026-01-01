<script lang="ts">
	import type { Education } from '$lib/pocketbase';
	import { formatDateRange, parseMarkdown } from '$lib/utils';

	export let items: Education[];
	export let layout: string = 'default';
</script>

<section id="education" class="mb-16">
	<h2 class="section-title">Education</h2>

	{#if layout === 'timeline'}
		<!-- Timeline Layout -->
		<div class="relative">
			<div class="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-700"></div>

			<div class="space-y-6 pl-12">
				{#each items as item (item.id)}
					<article class="relative animate-fade-in">
						<div class="absolute -left-12 w-8 h-8 bg-primary-600 rounded-full flex items-center justify-center z-10">
							<svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path d="M12 14l9-5-9-5-9 5 9 5z" />
							</svg>
						</div>

						<div>
							<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
								{item.institution}
							</h3>
							{#if item.degree || item.field}
								<p class="text-primary-600 dark:text-primary-400">
									{[item.degree, item.field].filter(Boolean).join(' in ')}
								</p>
							{/if}
							<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
								{formatDateRange(item.start_date, item.end_date)}
							</p>

							{#if item.description}
								<div class="mt-2 prose-custom text-gray-600 dark:text-gray-300 text-sm">
									{@html parseMarkdown(item.description)}
								</div>
							{/if}
						</div>
					</article>
				{/each}
			</div>
		</div>

	{:else}
		<!-- Default Layout (Cards with icons) -->
		<div class="space-y-6">
			{#each items as item (item.id)}
				<article class="card p-6 animate-fade-in">
					<div class="flex items-start gap-4">
						<div class="shrink-0 w-12 h-12 rounded-lg bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
							<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path d="M12 14l9-5-9-5-9 5 9 5z" />
								<path d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5zm0 0l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14zm-4 6v-7.5l4-2.222" />
							</svg>
						</div>

						<div class="flex-1">
							<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
								{item.institution}
							</h3>
							{#if item.degree || item.field}
								<p class="text-primary-600 dark:text-primary-400">
									{[item.degree, item.field].filter(Boolean).join(' in ')}
								</p>
							{/if}
							<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
								{formatDateRange(item.start_date, item.end_date)}
							</p>

							{#if item.description}
								<div class="mt-3 prose-custom text-gray-600 dark:text-gray-300 text-sm">
									{@html parseMarkdown(item.description)}
								</div>
							{/if}
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</section>
