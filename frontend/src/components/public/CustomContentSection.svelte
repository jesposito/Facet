<script lang="ts">
	import type { CustomContent } from '$lib/pocketbase';

	interface Props {
		item: CustomContent;
		layout?: string;
	}

	let { item, layout = 'default' }: Props = $props();

	// Process media URLs if available
	let mediaUrls = $derived(() => {
		if (item.media_urls && Array.isArray(item.media_urls)) {
			return item.media_urls;
		}
		return [];
	});
</script>

<section id="custom-{item.id}" class="mb-16" data-layout={layout}>
	<h2 class="section-title">{item.title}</h2>

	{#if layout === 'hero'}
		<!-- Hero layout: full-width cover image with content overlay -->
		<div class="relative animate-fade-in">
			{#if item.cover_image_url}
				<div class="relative w-full h-64 md:h-80 rounded-lg overflow-hidden">
					<img
						src={item.cover_image_url}
						alt={item.title}
						class="w-full h-full object-cover"
					/>
					<div class="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent"></div>
				</div>
			{/if}
			{#if item.content}
				<div class="card p-6 {item.cover_image_url ? '-mt-16 relative z-10 mx-4' : ''}">
					<div class="prose prose-gray dark:prose-invert max-w-none">
						{@html item.content}
					</div>
				</div>
			{/if}
		</div>
	{:else if layout === 'card'}
		<!-- Card layout: compact card with optional image -->
		<div class="card p-6 animate-fade-in">
			<div class="flex flex-col md:flex-row gap-6">
				{#if item.cover_image_url}
					<div class="md:w-1/3 flex-shrink-0">
						<img
							src={item.cover_image_url}
							alt={item.title}
							class="w-full h-48 md:h-full object-cover rounded-lg"
						/>
					</div>
				{/if}
				<div class="flex-1">
					{#if item.content}
						<div class="prose prose-gray dark:prose-invert max-w-none">
							{@html item.content}
						</div>
					{/if}
				</div>
			</div>
		</div>
	{:else}
		<!-- Default layout -->
		<div class="space-y-6 animate-fade-in">
			{#if item.cover_image_url}
				<div class="rounded-lg overflow-hidden">
					<img
						src={item.cover_image_url}
						alt={item.title}
						class="w-full h-auto max-h-96 object-cover"
					/>
				</div>
			{/if}

			{#if item.content}
				<div class="prose prose-gray dark:prose-invert max-w-none">
					{@html item.content}
				</div>
			{/if}

			{#if mediaUrls().length > 0}
				<div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
					{#each mediaUrls() as url (url)}
						<a href={url} target="_blank" rel="noopener noreferrer" class="block">
							<img
								src={url}
								alt="Media"
								class="w-full h-32 object-cover rounded-lg hover:opacity-90 transition-opacity"
							/>
						</a>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</section>
