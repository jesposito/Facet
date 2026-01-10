<script lang="ts">
	import type { Post } from '$lib/pocketbase';
	import { formatDate, truncate } from '$lib/utils';

	interface Props {
		items: Post[];
		layout?: string;
		viewSlug?: string;
	}

	let { items, layout = 'grid-3', viewSlug = '' }: Props = $props();

	// Build the post URL with optional from parameter for back navigation
	function getPostUrl(slug: string): string {
		if (viewSlug) {
			return `/posts/${slug}?from=${encodeURIComponent(viewSlug)}`;
		}
		return `/posts/${slug}`;
	}

	const thumbSrc = (post: Post) =>
		(post as unknown as Record<string, string>).cover_image_thumb_url ||
		(post as unknown as Record<string, string>).cover_image_url ||
		'';

	const largeSrc = (post: Post) =>
		(post as unknown as Record<string, string>).cover_image_large_url ||
		(post as unknown as Record<string, string>).cover_image_url ||
		'';
</script>

<section id="posts" class="mb-16">
	<h2 class="section-title">Posts</h2>

	<div class="grid grid-cols-1 {layout === 'grid-2' ? 'md:grid-cols-2' : layout === 'list' ? '' : 'md:grid-cols-2 lg:grid-cols-3'} gap-6">
		{#each items as post (post.id)}
			<article class="card overflow-hidden group animate-fade-in">
				{#if thumbSrc(post)}
					<div class="aspect-video overflow-hidden bg-gray-100 dark:bg-gray-700">
						<img
							src={largeSrc(post)}
							srcset={`${thumbSrc(post)} 640w, ${largeSrc(post)} 1280w`}
							sizes="(max-width: 768px) 100vw, (max-width: 1024px) 50vw, 33vw"
							alt={post.title}
							class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
						/>
					</div>
				{:else}
					<div class="aspect-video bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
						<svg class="w-12 h-12 text-white/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
						</svg>
					</div>
				{/if}

				<div class="p-5">
					<div class="flex items-start justify-between gap-2">
						<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
							{#if post.slug}
								<a href={getPostUrl(post.slug)} class="hover:text-primary-600 dark:hover:text-primary-400">
									{post.title}
								</a>
							{:else}
								{post.title}
							{/if}
						</h3>
					</div>

					{#if post.published_at}
						<time datetime={post.published_at} class="mt-1 text-sm text-gray-500 dark:text-gray-400 block">
							{formatDate(post.published_at, { month: 'long', day: 'numeric', year: 'numeric' })}
						</time>
					{/if}

					{#if post.excerpt}
						<p class="mt-2 text-gray-600 dark:text-gray-400 text-sm">
							{truncate(post.excerpt, 120)}
						</p>
					{/if}

					{#if post.tags && post.tags.length > 0}
						<div class="mt-3 flex flex-wrap gap-1.5">
							{#each post.tags.slice(0, 4) as tag}
								<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
									{tag}
								</span>
							{/each}
							{#if post.tags.length > 4}
								<span class="px-2 py-0.5 text-xs text-gray-500">
									+{post.tags.length - 4}
								</span>
							{/if}
						</div>
					{/if}

					{#if post.slug}
						<div class="mt-4">
							<a
								href={getPostUrl(post.slug)}
								class="inline-flex items-center gap-1.5 text-sm text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
							>
								Read more
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
								</svg>
							</a>
						</div>
					{/if}
				</div>
			</article>
		{/each}
	</div>
</section>
