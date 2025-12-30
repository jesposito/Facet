<script lang="ts">
	import type { Experience } from '$lib/pocketbase';
	import { formatDateRange, parseMarkdown } from '$lib/utils';

	export let items: Experience[];
</script>

<section id="experience" class="mb-16">
	<h2 class="section-title">Experience</h2>

	<div class="space-y-8">
		{#each items as item (item.id)}
			<article class="card p-6 animate-fade-in">
				<div class="flex flex-col sm:flex-row sm:items-start gap-4">
					<div class="flex-1">
						<h3 class="text-xl font-semibold text-gray-900 dark:text-white">
							{item.title}
						</h3>
						<p class="text-lg text-primary-600 dark:text-primary-400 font-medium">
							{item.company}
						</p>
						<div class="flex flex-wrap items-center gap-x-4 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
							<span>{formatDateRange(item.start_date, item.end_date)}</span>
							{#if item.location}
								<span class="flex items-center gap-1">
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
									</svg>
									{item.location}
								</span>
							{/if}
						</div>
					</div>
				</div>

				{#if item.description}
					<div class="mt-4 prose-custom text-gray-600 dark:text-gray-300">
						{@html parseMarkdown(item.description)}
					</div>
				{/if}

				{#if item.bullets && item.bullets.length > 0}
					<ul class="mt-4 space-y-2">
						{#each item.bullets as bullet}
							<li class="flex items-start gap-2 text-gray-600 dark:text-gray-300">
								<span class="text-primary-500 mt-1">•</span>
								<span>{bullet}</span>
							</li>
						{/each}
					</ul>
				{/if}

				{#if item.skills && item.skills.length > 0}
					<div class="mt-4 flex flex-wrap gap-2">
						{#each item.skills as skill}
							<span class="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-full">
								{skill}
							</span>
						{/each}
					</div>
				{/if}
			</article>
		{/each}
	</div>
</section>
