<script lang="ts">
	import type { PageData } from './$types';
	import { parseMarkdown, formatDate } from '$lib/utils';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import Footer from '$components/public/Footer.svelte';

	export let data: PageData;

	// Format the published date
	$: publishedDate = data.post.published_at ? formatDate(data.post.published_at) : null;

	// Determine back navigation URL and label
	// If coming from a view (profile page), go back to that view
	// Otherwise, go back to the posts index page
	$: backUrl = data.fromView ? `/${data.fromView}` : '/posts';
	$: backLabel = data.fromView ? 'Back to Profile' : 'Back to Posts';
</script>

<svelte:head>
	<title>{data.post.title} | {data.profile?.name || 'Blog'}</title>
	<meta name="description" content={data.post.excerpt || ''} />
	<link rel="canonical" href="/posts/{data.post.slug}" />
	<!-- Open Graph -->
	<meta property="og:title" content={data.post.title} />
	<meta property="og:description" content={data.post.excerpt || ''} />
	<meta property="og:type" content="article" />
	{#if data.post.cover_image_url}
		<meta property="og:image" content={data.post.cover_image_url} />
	{/if}
	{#if publishedDate}
		<meta property="article:published_time" content={data.post.published_at} />
	{/if}
</svelte:head>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<!-- Theme toggle -->
	<div class="fixed top-4 right-4 z-40">
		<ThemeToggle />
	</div>

	<!-- Hero section with cover image -->
	<header class="relative bg-gradient-to-br from-gray-900 to-gray-800 text-white">
		{#if data.post.cover_image_url}
			<div class="absolute inset-0">
				<img
					src={data.post.cover_image_url}
					alt=""
					class="w-full h-full object-cover opacity-30"
				/>
				<div class="absolute inset-0 bg-gradient-to-t from-gray-900 via-gray-900/80 to-transparent" />
			</div>
		{/if}

		<div class="relative max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16">
			<!-- Back navigation -->
			<a
				href={backUrl}
				class="inline-flex items-center gap-2 text-gray-300 hover:text-white mb-6 transition-colors"
			>
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
				</svg>
				<span>{backLabel}</span>
			</a>

			<!-- Post title -->
			<h1 class="text-3xl sm:text-4xl lg:text-5xl font-bold leading-tight">
				{data.post.title}
			</h1>

			<!-- Meta information -->
			<div class="mt-6 flex flex-wrap items-center gap-4 text-gray-300">
				{#if publishedDate}
					<time datetime={data.post.published_at} class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
						{publishedDate}
					</time>
				{/if}

				{#if data.profile?.name}
					<span class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
						</svg>
						{data.profile.name}
					</span>
				{/if}
			</div>

			<!-- Tags -->
			{#if data.post.tags && data.post.tags.length > 0}
				<div class="mt-4 flex flex-wrap gap-2">
					{#each data.post.tags as tag}
						<span class="px-3 py-1 text-sm bg-white/10 rounded-full">
							{tag}
						</span>
					{/each}
				</div>
			{/if}
		</div>
	</header>

	<!-- Main content -->
	<main class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
		<!-- Excerpt -->
		{#if data.post.excerpt}
			<p class="text-xl text-gray-600 dark:text-gray-400 mb-8 italic border-l-4 border-primary-500 pl-4">
				{data.post.excerpt}
			</p>
		{/if}

		<!-- Content (Markdown) -->
		{#if data.post.content}
			<article class="prose prose-lg dark:prose-invert max-w-none prose-headings:scroll-mt-20 prose-a:text-primary-600 dark:prose-a:text-primary-400 prose-img:rounded-lg prose-pre:bg-gray-800 dark:prose-pre:bg-gray-950">
				{@html parseMarkdown(data.post.content)}
			</article>
		{/if}

		<!-- Previous/Next navigation -->
		{#if data.prev_post || data.next_post}
			<nav class="mt-16 pt-8 border-t border-gray-200 dark:border-gray-700">
				<div class="flex flex-col sm:flex-row justify-between gap-4">
					{#if data.prev_post}
						<a
							href="/posts/{data.prev_post.slug}"
							class="group flex-1 p-4 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-500 dark:hover:border-primary-500 transition-colors"
						>
							<span class="text-sm text-gray-500 dark:text-gray-400 flex items-center gap-1">
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
								</svg>
								Previous
							</span>
							<span class="mt-1 text-gray-900 dark:text-white font-medium group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors block">
								{data.prev_post.title}
							</span>
						</a>
					{:else}
						<div class="flex-1"></div>
					{/if}

					{#if data.next_post}
						<a
							href="/posts/{data.next_post.slug}"
							class="group flex-1 p-4 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-500 dark:hover:border-primary-500 transition-colors text-right"
						>
							<span class="text-sm text-gray-500 dark:text-gray-400 flex items-center justify-end gap-1">
								Next
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
								</svg>
							</span>
							<span class="mt-1 text-gray-900 dark:text-white font-medium group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors block">
								{data.next_post.title}
							</span>
						</a>
					{/if}
				</div>
			</nav>
		{/if}
	</main>

	<!-- Footer -->
	<Footer profile={data.profile} />
</div>
