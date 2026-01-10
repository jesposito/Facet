<script lang="ts">
	import type { Award } from '$lib/pocketbase';
	import { formatDate, truncate } from '$lib/utils';

	interface Props {
		items: Award[];
		layout?: string;
	}

	let { items, layout = 'grouped' }: Props = $props();


	function groupByIssuer(awards: Award[]): Map<string, Award[]> {
		const groups = new Map<string, Award[]>();
		for (const award of awards) {
			const key = award.issuer || 'Other';
			if (!groups.has(key)) groups.set(key, []);
			groups.get(key)!.push(award);
		}
		return groups;
	}
	// Group awards by issuer for grouped layout
	let groupedAwards = $derived(layout === 'grouped' ? groupByIssuer(items) : null);
</script>

<section id="awards" class="mb-16" data-layout={layout}>
	<h2 class="section-title">Awards & Honors</h2>

	{#if layout === 'timeline'}
		<div class="space-y-6">
			{#each items as award (award.id)}
				<article class="card p-5 flex gap-4 items-start animate-fade-in">
					<div class="flex-shrink-0 w-10 h-10 rounded-full bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-200 flex items-center justify-center font-semibold">
						{award.awarded_at ? formatDate(award.awarded_at, { year: 'numeric' }) : ''}
					</div>
					<div class="flex-1 space-y-1">
						<p class="text-sm text-gray-500 dark:text-gray-400">
							{award.awarded_at ? formatDate(award.awarded_at, { month: 'short', year: 'numeric' }) : 'Date not provided'}
						</p>
						<h3 class="text-lg font-semibold text-gray-900 dark:text-white">{award.title}</h3>
						{#if award.issuer}
							<p class="text-sm text-gray-600 dark:text-gray-400">{award.issuer}</p>
						{/if}
						{#if award.description}
							<p class="text-gray-700 dark:text-gray-300 text-sm">{truncate(award.description, 220)}</p>
						{/if}
						{#if award.url}
							<a
								href={award.url}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
								</svg>
								View details
							</a>
						{/if}
					</div>
				</article>
			{/each}
		</div>
	{:else if layout === 'grid'}
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			{#each items as award (award.id)}
				<article class="card p-5 space-y-2 animate-fade-in">
					<div class="flex items-start justify-between gap-2">
						<h3 class="text-lg font-semibold text-gray-900 dark:text-white">{award.title}</h3>
						{#if award.awarded_at}
							<span class="text-xs px-2 py-1 rounded bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-200">
								{formatDate(award.awarded_at, { month: 'short', year: 'numeric' })}
							</span>
						{/if}
					</div>
					{#if award.issuer}
						<p class="text-sm text-gray-600 dark:text-gray-400">{award.issuer}</p>
					{/if}
					{#if award.description}
						<p class="text-gray-700 dark:text-gray-300 text-sm line-clamp-3">{award.description}</p>
					{/if}
					{#if award.url}
						<a
							href={award.url}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
						>
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
							</svg>
							View details
						</a>
					{/if}
				</article>
			{/each}
		</div>
	{:else}
		<!-- Grouped by issuer -->
		<div class="space-y-8">
			{#each [...(groupedAwards || [])] as [issuerName, awards] (issuerName)}
				<div class="animate-fade-in">
					<h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-4">
						{issuerName}
					</h3>
					<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
						{#each awards as award (award.id)}
							<article class="card p-5 space-y-2">
								<div class="flex items-start justify-between gap-2">
									<h4 class="font-semibold text-gray-900 dark:text-white">
										{award.title}
									</h4>
									{#if award.awarded_at}
										<span class="text-xs px-2 py-1 rounded bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-200">
											{formatDate(award.awarded_at, { month: 'short', year: 'numeric' })}
										</span>
									{/if}
								</div>
								{#if award.description}
									<p class="text-sm text-gray-700 dark:text-gray-300 line-clamp-3">
										{award.description}
									</p>
								{/if}
								{#if award.url}
									<a
										href={award.url}
										target="_blank"
										rel="noopener noreferrer"
										class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
									>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
										</svg>
										View details
									</a>
								{/if}
							</article>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</section>
