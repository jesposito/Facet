<script lang="ts">
	import type { PageData } from './$types';
	import { parseMarkdown, formatDate } from '$lib/utils';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import Footer from '$components/public/Footer.svelte';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	export let data: PageData;

	// Extract YouTube video ID from various URL formats
	function getYouTubeId(url: string): string | null {
		const patterns = [
			/(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/)([^&?/]+)/,
			/youtube\.com\/v\/([^&?/]+)/
		];
		for (const pattern of patterns) {
			const match = url.match(pattern);
			if (match) return match[1];
		}
		return null;
	}

	// Extract Vimeo video ID from URL
	function getVimeoId(url: string): string | null {
		const match = url.match(/vimeo\.com\/(\d+)/);
		return match ? match[1] : null;
	}

	// Get embed type and ID
	function getVideoEmbed(url: string): { type: 'youtube' | 'vimeo'; id: string } | null {
		const youtubeId = getYouTubeId(url);
		if (youtubeId) return { type: 'youtube', id: youtubeId };

		const vimeoId = getVimeoId(url);
		if (vimeoId) return { type: 'vimeo', id: vimeoId };

		return null;
	}

	$: videoEmbed = data.talk.video_url ? getVideoEmbed(data.talk.video_url) : null;
	$: formattedDate = data.talk.date ? formatDate(data.talk.date, { month: 'long', day: 'numeric', year: 'numeric' }) : null;

	let referrerPath = '';

	onMount(() => {
		if (!browser) return;
		try {
			const ref = document.referrer;
			if (ref) {
				const refUrl = new URL(ref);
				if (refUrl.origin === window.location.origin && refUrl.pathname !== window.location.pathname) {
					referrerPath = refUrl.pathname + refUrl.search;
				}
			}
		} catch {
			// ignore
		}
	});

	// Determine back navigation URL and label
	// Prefer originating view, then referrer, otherwise talks index
	$: backUrl = data.fromView ? `/${data.fromView}` : referrerPath || '/talks';
	$: backLabel = 'Back';

	function handleBack(event: Event) {
		event.preventDefault();
		if (browser && window.history.length > 1) {
			window.history.back();
		} else {
			goto(backUrl, { replaceState: true });
		}
	}
</script>

<svelte:head>
	<title>{data.talk.title} | {data.profile?.name || 'Talks'}</title>
	<meta name="description" content="{data.talk.event ? `${data.talk.event} - ` : ''}{data.talk.title}" />
	<link rel="canonical" href="/talks/{data.talk.slug}" />
	<!-- Open Graph -->
	<meta property="og:title" content={data.talk.title} />
	<meta property="og:description" content="{data.talk.event ? `${data.talk.event} - ` : ''}{data.talk.title}" />
	<meta property="og:type" content="video.other" />
	{#if videoEmbed?.type === 'youtube'}
		<meta property="og:image" content="https://img.youtube.com/vi/{videoEmbed.id}/maxresdefault.jpg" />
		<meta property="og:video" content="https://www.youtube.com/embed/{videoEmbed.id}" />
	{/if}
</svelte:head>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<!-- Theme toggle -->
	<div class="fixed top-4 right-4 z-40 print:hidden">
		<ThemeToggle />
	</div>

	<!-- Hero section -->
	<header class="relative bg-gradient-to-br from-gray-900 to-gray-800 text-white">
		{#if videoEmbed?.type === 'youtube'}
			<div class="absolute inset-0">
				<img
					src="https://img.youtube.com/vi/{videoEmbed.id}/maxresdefault.jpg"
					alt=""
					class="w-full h-full object-cover opacity-20"
				/>
				<div class="absolute inset-0 bg-gradient-to-t from-gray-900 via-gray-900/80 to-transparent" />
			</div>
		{/if}

		<div class="relative max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16">
			<!-- Back navigation -->
			<a
				href={backUrl}
				on:click|preventDefault={handleBack}
				class="inline-flex items-center gap-2 text-gray-300 hover:text-white mb-6 transition-colors"
			>
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
				</svg>
				<span>{backLabel}</span>
			</a>

			<!-- Talk title -->
			<h1 class="text-3xl sm:text-4xl lg:text-5xl font-bold leading-tight">
				{data.talk.title}
			</h1>

			<!-- Meta information -->
			<div class="mt-6 flex flex-wrap items-center gap-4 text-gray-300">
				{#if data.talk.event}
					<span class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
						</svg>
						{#if data.talk.event_url}
							<a href={data.talk.event_url} target="_blank" rel="noopener noreferrer" class="hover:text-white transition-colors">
								{data.talk.event}
							</a>
						{:else}
							{data.talk.event}
						{/if}
					</span>
				{/if}

				{#if formattedDate}
					<time datetime={data.talk.date} class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
						{formattedDate}
					</time>
				{/if}

				{#if data.talk.location}
					<span class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
						</svg>
						{data.talk.location}
					</span>
				{/if}

				{#if data.profile?.name}
					<span class="flex items-center gap-2">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
						</svg>
						{data.profile.name}
					</span>
				{/if}
			</div>
		</div>
	</header>

	<!-- Main content -->
	<main id="main-content" class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
		<!-- Video embed -->
		{#if data.talk.video_url && videoEmbed}
			<div class="aspect-video w-full mb-8 rounded-xl overflow-hidden shadow-lg">
				{#if videoEmbed.type === 'youtube'}
					<iframe
						src="https://www.youtube.com/embed/{videoEmbed.id}"
						title={data.talk.title}
						class="w-full h-full"
						frameborder="0"
						allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
						allowfullscreen
					></iframe>
				{:else if videoEmbed.type === 'vimeo'}
					<iframe
						src="https://player.vimeo.com/video/{videoEmbed.id}"
						title={data.talk.title}
						class="w-full h-full"
						frameborder="0"
						allow="autoplay; fullscreen; picture-in-picture"
						allowfullscreen
					></iframe>
				{/if}
			</div>
		{:else if data.talk.video_url}
			<!-- Non-embeddable video URL -->
			<div class="mb-8">
				<a
					href={data.talk.video_url}
					target="_blank"
					rel="noopener noreferrer"
					class="inline-flex items-center gap-3 px-6 py-4 bg-purple-600 hover:bg-purple-700 text-white rounded-lg transition-colors"
				>
					<svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					<span class="text-lg font-medium">Watch Video</span>
				</a>
			</div>
		{/if}

		<!-- Action links -->
		{#if data.talk.slides_url || (data.talk.video_url && !videoEmbed)}
			<div class="flex flex-wrap gap-4 mb-8">
				{#if data.talk.slides_url}
					<a
						href={data.talk.slides_url}
						target="_blank"
						rel="noopener noreferrer"
						class="inline-flex items-center gap-2 px-4 py-2 bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
					>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
						</svg>
						View Slides
					</a>
				{/if}
			</div>
		{/if}

		<!-- Description (Markdown) -->
		{#if data.talk.description}
			<article class="prose prose-lg dark:prose-invert max-w-none prose-headings:scroll-mt-20 prose-a:text-primary-600 dark:prose-a:text-primary-400 prose-img:rounded-lg">
				{@html parseMarkdown(data.talk.description)}
			</article>
		{/if}

		<!-- Previous/Next navigation -->
		{#if data.prev_talk || data.next_talk}
			<nav class="mt-16 pt-8 border-t border-gray-200 dark:border-gray-700">
				<div class="flex flex-col sm:flex-row justify-between gap-4">
					{#if data.prev_talk}
						<a
							href="/talks/{data.prev_talk.slug}"
							class="group flex-1 p-4 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-500 dark:hover:border-primary-500 transition-colors"
						>
							<span class="text-sm text-gray-500 dark:text-gray-400 flex items-center gap-1">
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
								</svg>
								Previous Talk
							</span>
							<span class="mt-1 text-gray-900 dark:text-white font-medium group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors block">
								{data.prev_talk.title}
							</span>
						</a>
					{:else}
						<div class="flex-1"></div>
					{/if}

					{#if data.next_talk}
						<a
							href="/talks/{data.next_talk.slug}"
							class="group flex-1 p-4 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-primary-500 dark:hover:border-primary-500 transition-colors text-right"
						>
							<span class="text-sm text-gray-500 dark:text-gray-400 flex items-center justify-end gap-1">
								Next Talk
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
								</svg>
							</span>
							<span class="mt-1 text-gray-900 dark:text-white font-medium group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors block">
								{data.next_talk.title}
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
