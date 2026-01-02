<script lang="ts">
	import type { PageData } from './$types';
	import { parseMarkdown } from '$lib/utils';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import Footer from '$components/public/Footer.svelte';

	export let data: PageData;

	// Back navigation: return to originating view if provided, otherwise home
	$: backUrl = data.fromView ? `/${data.fromView}` : '/';
	$: backLabel = data.fromView ? 'Back to Profile' : 'Back to Profile';

	function getLinkIcon(type: string) {
		switch (type.toLowerCase()) {
			case 'github':
				return `<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/></svg>`;
			case 'website':
			case 'demo':
				return `<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" /></svg>`;
			case 'docs':
			case 'documentation':
				return `<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>`;
			case 'npm':
				return `<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M0 7.334v8h6.666v1.332H12v-1.332h12v-8H0zm6.666 6.664H5.334v-4H3.999v4H1.335V8.667h5.331v5.331zm4 0v1.336H8.001V8.667h5.334v5.332h-2.669v-.001zm12.001 0h-1.33v-4h-1.336v4h-1.335v-4h-1.33v4h-2.671V8.667h8.002v5.331z"/></svg>`;
			default:
				return `<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" /></svg>`;
		}
	}
</script>

<svelte:head>
	<title>{data.project.title} | {data.profile?.name || 'Projects'}</title>
	<meta name="description" content={data.project.summary || ''} />
	<link rel="canonical" href="/projects/{data.project.slug}" />
	<!-- Open Graph -->
	<meta property="og:title" content={data.project.title} />
	<meta property="og:description" content={data.project.summary || ''} />
	<meta property="og:type" content="article" />
	{#if data.project.cover_image_url}
		<meta property="og:image" content={data.project.cover_image_url} />
	{/if}
</svelte:head>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<!-- Theme toggle -->
	<div class="fixed top-4 right-4 z-40">
		<ThemeToggle />
	</div>

	<!-- Hero section with cover image -->
	<header class="relative bg-gradient-to-br from-gray-900 to-gray-800 text-white">
		{#if data.project.cover_image_url}
			<div class="absolute inset-0">
				<img
					src={data.project.cover_image_url}
					alt=""
					class="w-full h-full object-cover opacity-30"
				/>
				<div class="absolute inset-0 bg-gradient-to-t from-gray-900 via-gray-900/80 to-transparent" />
			</div>
		{/if}

		<div class="relative max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16">
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

			<!-- Project title and badges -->
			<div class="flex flex-wrap items-start gap-3 mb-4">
				<h1 class="text-3xl sm:text-4xl lg:text-5xl font-bold">
					{data.project.title}
				</h1>
				{#if data.project.is_featured}
					<span class="px-3 py-1 text-sm font-medium bg-yellow-500 text-yellow-900 rounded-full">
						Featured
					</span>
				{/if}
			</div>

			<!-- Summary -->
			{#if data.project.summary}
				<p class="text-xl text-gray-300 max-w-3xl">
					{data.project.summary}
				</p>
			{/if}

			<!-- Links -->
			{#if data.project.links && data.project.links.length > 0}
				<div class="flex flex-wrap items-center gap-3 mt-6">
					{#each data.project.links as link}
						<a
							href={link.url}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
						>
							{@html getLinkIcon(link.type)}
							<span class="capitalize">{link.type}</span>
						</a>
					{/each}
				</div>
			{/if}
		</div>
	</header>

	<!-- Main content -->
	<main class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
		<!-- Tech stack -->
		{#if data.project.tech_stack && data.project.tech_stack.length > 0}
			<section class="mb-10">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Tech Stack</h2>
				<div class="flex flex-wrap gap-2">
					{#each data.project.tech_stack as tech}
						<span class="px-3 py-1.5 text-sm bg-primary-100 dark:bg-primary-900/30 text-primary-800 dark:text-primary-200 rounded-full">
							{tech}
						</span>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Categories -->
		{#if data.project.categories && data.project.categories.length > 0}
			<section class="mb-10">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Categories</h2>
				<div class="flex flex-wrap gap-2">
					{#each data.project.categories as category}
						<span class="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-full">
							{category}
						</span>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Description (Markdown) -->
		{#if data.project.description}
			<section class="mb-10">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">About</h2>
				<div class="prose prose-lg dark:prose-invert max-w-none">
					{@html parseMarkdown(data.project.description)}
				</div>
			</section>
		{/if}

		<!-- Media gallery -->
		{#if data.project.media_urls && data.project.media_urls.length > 0}
			<section class="mb-10">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Gallery</h2>
				<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
					{#each data.project.media_urls as mediaUrl}
						<a
							href={mediaUrl}
							target="_blank"
							rel="noopener noreferrer"
							class="block aspect-video overflow-hidden rounded-lg bg-gray-100 dark:bg-gray-800 hover:opacity-90 transition-opacity"
						>
							<img
								src={mediaUrl}
								alt="Project media"
								class="w-full h-full object-cover"
							/>
						</a>
					{/each}
				</div>
			</section>
		{/if}
	</main>

	<!-- Footer -->
	<Footer profile={data.profile} />
</div>
