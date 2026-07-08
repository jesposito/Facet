<script lang="ts">
	import { t } from 'svelte-i18n';
	import type { Post } from '$lib/pocketbase';
	import { formatDate, truncate } from '$lib/utils';

	interface Props {
		items: Post[];
		layout?: string;
		viewSlug?: string;
		showHeader?: boolean;
	}

	let { items, layout = 'grid-3', viewSlug = '', showHeader = true }: Props = $props();

	// Featured layout: first post large, rest in 3-col grid
	let featuredPost = $derived(items[0]);
	let remainingPosts = $derived(items.slice(1));

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
	{#if showHeader}
		<h2 class="section-title">{$t('public.sections.posts')}</h2>
	{/if}

	{#if layout === 'featured' && featuredPost}
		<!-- Featured Layout: hero card for the first post, 3-col grid for the rest -->
		<div class="space-y-8">
			<article class="card p-0 overflow-hidden group animate-fade-in">
				<div class="grid grid-cols-1 lg:grid-cols-2 gap-0">
					<!-- Image -->
					<div class="relative aspect-video lg:aspect-auto lg:min-h-[20rem] bg-stone-100 dark:bg-stone-800">
						{#if thumbSrc(featuredPost)}
							{#if featuredPost.slug}
								<a href={getPostUrl(featuredPost.slug)} class="absolute inset-0 block focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-600 focus-visible:ring-inset">
									<img
										src={largeSrc(featuredPost)}
										srcset={`${thumbSrc(featuredPost)} 640w, ${largeSrc(featuredPost)} 1280w`}
										sizes="(max-width: 1024px) 100vw, 50vw"
										alt={featuredPost.title}
										class="w-full h-full object-cover group-hover:scale-[1.02] transition-transform duration-500"
									/>
								</a>
							{:else}
								<img
									src={largeSrc(featuredPost)}
									alt={featuredPost.title}
									class="absolute inset-0 w-full h-full object-cover"
								/>
							{/if}
						{:else}
							<div class="absolute inset-0 bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
								<svg class="w-16 h-16 text-white/50" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
								</svg>
							</div>
						{/if}
						<span class="absolute top-3 left-3 inline-flex items-center px-2.5 py-1 rounded-full bg-primary-600 text-white text-xs font-semibold uppercase tracking-wide shadow-sm">
							{$t('public.posts.featured')}
						</span>
					</div>

					<!-- Metadata -->
					<div class="p-6 sm:p-8 flex flex-col justify-center">
						{#if featuredPost.published_at}
							<time datetime={featuredPost.published_at} class="text-sm text-stone-600 dark:text-stone-300">
								{formatDate(featuredPost.published_at, { month: 'long', day: 'numeric', year: 'numeric' })}
							</time>
						{/if}
						<h3 class="mt-2 text-2xl sm:text-3xl font-bold text-stone-900 dark:text-white leading-tight">
							{#if featuredPost.slug}
								<a href={getPostUrl(featuredPost.slug)} class="hover:text-primary-600 dark:hover:text-primary-400 focus:outline-none focus-visible:underline">
									{featuredPost.title}
								</a>
							{:else}
								{featuredPost.title}
							{/if}
						</h3>
						{#if featuredPost.excerpt}
							<p class="mt-3 text-stone-600 dark:text-stone-300">
								{truncate(featuredPost.excerpt, 200)}
							</p>
						{/if}
						{#if featuredPost.tags && featuredPost.tags.length > 0}
							<div class="mt-4 flex flex-wrap gap-1.5">
								{#each featuredPost.tags.slice(0, 5) as tag}
									<span class="px-2 py-0.5 text-xs bg-stone-100 dark:bg-stone-700 text-stone-600 dark:text-stone-300 rounded">
										{tag}
									</span>
								{/each}
							</div>
						{/if}
						{#if featuredPost.slug}
							<div class="mt-5">
								<a
									href={getPostUrl(featuredPost.slug)}
									class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 focus:outline-none focus-visible:underline"
								>
									{$t('public.posts.read_more')}
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
									</svg>
								</a>
							</div>
						{/if}
					</div>
				</div>
			</article>

			{#if remainingPosts.length > 0}
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
					{#each remainingPosts as post (post.id)}
						{@render postCard(post)}
					{/each}
				</div>
			{/if}
		</div>
	{:else}
		<div class="grid grid-cols-1 {layout === 'grid-2' ? 'md:grid-cols-2' : layout === 'list' ? '' : 'md:grid-cols-2 lg:grid-cols-3'} gap-6">
			{#each items as post (post.id)}
				{@render postCard(post)}
			{/each}
		</div>
	{/if}
</section>

{#snippet postCard(post: Post)}
	<article class="card overflow-hidden group animate-fade-in">
		{#if thumbSrc(post)}
			{#if post.slug}
				<a href={getPostUrl(post.slug)} class="block aspect-video overflow-hidden bg-gray-100 dark:bg-gray-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-600 focus-visible:ring-inset">
					<img
						src={largeSrc(post)}
						srcset={`${thumbSrc(post)} 640w, ${largeSrc(post)} 1280w`}
						sizes="(max-width: 768px) 100vw, (max-width: 1024px) 50vw, 33vw"
						alt={post.title}
						class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
					/>
				</a>
			{:else}
				<div class="aspect-video overflow-hidden bg-gray-100 dark:bg-gray-700">
					<img
						src={largeSrc(post)}
						srcset={`${thumbSrc(post)} 640w, ${largeSrc(post)} 1280w`}
						sizes="(max-width: 768px) 100vw, (max-width: 1024px) 50vw, 33vw"
						alt={post.title}
						class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
					/>
				</div>
			{/if}
		{:else}
			<div class="aspect-video bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
				<svg class="w-12 h-12 text-white/50" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
				</svg>
			</div>
		{/if}

		<div class="p-5">
			<div class="flex items-start justify-between gap-2">
				<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
					{#if post.slug}
						<a href={getPostUrl(post.slug)} class="hover:text-primary-600 dark:hover:text-primary-400 focus:outline-none focus-visible:underline">
							{post.title}
						</a>
					{:else}
						{post.title}
					{/if}
				</h3>
			</div>

			{#if post.published_at}
				<time datetime={post.published_at} class="mt-1 text-sm text-gray-600 dark:text-gray-300 block">
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
						class="inline-flex items-center gap-1.5 text-sm text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 focus:outline-none focus-visible:underline"
					>
						{$t('public.posts.read_more')}
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
						</svg>
					</a>
				</div>
			{/if}
		</div>
	</article>
{/snippet}
