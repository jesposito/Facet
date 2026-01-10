<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import type { PageData } from './$types';
	import { parseMarkdown, formatDate } from '$lib/utils';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
import VisibilityBadge from '$components/shared/VisibilityBadge.svelte';
	import Footer from '$components/public/Footer.svelte';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

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

	let videoEmbed = $derived(data.talk.video_url ? getVideoEmbed(data.talk.video_url) : null);
	let formattedDate = $derived(data.talk.date ? formatDate(data.talk.date, { month: 'long', day: 'numeric', year: 'numeric' }) : null);
	const isYouTube = (url?: string) => !!url && getYouTubeId(url) !== null;
	const isVimeo = (url?: string) => !!url && getVimeoId(url) !== null;
	const isImage = (url?: string) => !!url && /\.(png|jpe?g|gif|webp|avif|svg)$/i.test(url);
	const isVideoFile = (url?: string) => !!url && /\.(mp4|mov|webm|mkv|avi)$/i.test(url);
	const getFileName = (url?: string) => {
		if (!url) return 'Media';
		try {
			const path = url.split('?')[0];
			const parts = path.split('/');
			return decodeURIComponent(parts[parts.length - 1] || url);
		} catch {
			return url;
		}
	};

	let referrerPath = $state('');

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
	let backUrl = $derived(data.fromView ? `/${data.fromView}` : referrerPath || '/talks');
	let backLabel = 'Back';

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
				<div class="absolute inset-0 bg-gradient-to-t from-gray-900 via-gray-900/80 to-transparent"></div>
			</div>
		{/if}

		<div class="relative max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16">
			<!-- Back navigation -->
			<a
				href={backUrl}
				onclick={preventDefault(handleBack)}
				class="inline-flex items-center gap-2 text-gray-300 hover:text-white mb-6 transition-colors"
			>
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
				</svg>
				<span>{backLabel}</span>
			</a>

			<!-- Talk title -->
			<div class="flex flex-wrap items-start gap-3">
				<h1 class="text-3xl sm:text-4xl lg:text-5xl font-bold leading-tight">
					{data.talk.title}
				</h1>
				{#if (data as any).isAuthenticated && ((data.talk as any).visibility !== 'public' || (data.talk as any).is_draft)}
					<VisibilityBadge visibility={(data.talk as any).visibility} isDraft={(data.talk as any).is_draft} />
				{/if}
			</div>

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

		{#if data.media_refs && data.media_refs.length > 0}
			<section class="mt-10 space-y-3">
				<h3 class="text-lg font-semibold text-gray-900 dark:text-white">Attached media</h3>
				<div class="grid gap-4 md:grid-cols-2">
					{#each data.media_refs as media}
						<div class="card p-4 space-y-3">
							<div class="flex items-start gap-3">
								<div class="w-10 h-10 rounded-lg bg-primary-50 dark:bg-primary-900/40 flex items-center justify-center text-primary-700 dark:text-primary-200">
									{#if isYouTube(media.url)}
										<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M21.6 7.2s-.2-1.5-.8-2.2c-.7-.8-1.5-.8-1.9-.8C15.7 4 12 4 12 4h-.1S8.3 4 5.1 4.2c-.4 0-1.2 0-1.9.8-.6.7-.8 2.2-.8 2.2S2 8.9 2 10.6v1.6c0 1.7.2 3.4.2 3.4s.2 1.5.8 2.2c.7.8 1.7.8 2.1.9 1.5.1 6.9.2 6.9.2s3.7 0 6.9-.2c.4 0 1.2 0 1.9-.9.6-.7.8-2.2.8-2.2s.2-1.7.2-3.4v-1.6c0-1.7-.2-3.4-.2-3.4Zm-12.7 6.8V8.8l5.2 2.6-5.2 2.6Z"/></svg>
									{:else if isVimeo(media.url)}
										<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M22.37 6.76c-.1 2.2-1.64 5.21-4.63 9.05-3.08 4-5.68 6-7.8 6-1.32 0-2.44-1.2-3.36-3.6l-1.84-6.6c-.68-2.4-1.4-3.6-2.16-3.6-.17 0-.78.36-1.82 1.08L0 7.38c1.15-1.01 2.29-2.02 3.43-3.03 1.54-1.33 2.7-2.03 3.5-2.1 1.84-.18 2.98 1.08 3.42 3.78.46 2.91.78 4.72.96 5.4.53 2.4 1.11 3.6 1.76 3.6.5 0 1.26-.79 2.28-2.36 1.01-1.58 1.55-2.79 1.62-3.64.14-1.38-.4-2.07-1.62-2.07-.58 0-1.18.12-1.8.36 1.2-3.9 3.47-5.79 6.8-5.68 2.48.06 3.64 1.68 3.48 4.86Z"/></svg>
									{:else if isImage(media.url)}
										<svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="8.5" cy="9.5" r="1.5"/><path d="M21 15 16 10 5 21"/></svg>
									{:else if isVideoFile(media.url)}
										<svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="5" width="18" height="14" rx="2"/><path d="m10 9 5 3-5 3V9Z"/></svg>
									{:else}
										<svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14m-5-5 5 5 5-5"/><path d="M5 9V7a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v2"/></svg>
									{/if}
								</div>
								<div class="min-w-0">
									<p class="text-sm font-semibold text-gray-900 dark:text-white line-clamp-2">
										{media.title || getFileName(media.url)}
									</p>
									{#if media.url}
										<p class="text-xs text-gray-500 dark:text-gray-400 break-all">{media.url}</p>
									{/if}
								</div>
							</div>

							{#if isYouTube(media.url)}
								<div class="aspect-video rounded-lg overflow-hidden bg-black/10">
									<iframe
										src={`https://www.youtube.com/embed/${getYouTubeId(media.url) ?? ''}`}
										title={media.title || 'YouTube'}
										allowfullscreen
										loading="lazy"
										class="w-full h-full"
									></iframe>
								</div>
							{:else if isVimeo(media.url)}
								<div class="aspect-video rounded-lg overflow-hidden bg-black/10">
									<iframe
										src={`https://player.vimeo.com/video/${getVimeoId(media.url) ?? ''}`}
										title={media.title || 'Vimeo'}
										allowfullscreen
										loading="lazy"
										class="w-full h-full"
									></iframe>
								</div>
							{:else if isImage(media.url)}
								<img src={media.url || ''} alt={media.title || ''} class="w-full rounded-lg" loading="lazy" />
							{:else if isVideoFile(media.url)}
								<video src={media.url || ''} controls class="w-full rounded-lg">
									<track kind="captions" srclang="en" label="captions" />
								</video>
							{:else if media.url}
								<a href={media.url} class="text-primary-600 dark:text-primary-300 hover:underline break-all text-sm" target="_blank" rel="noopener noreferrer">
									Open link
								</a>
							{/if}
						</div>
					{/each}
				</div>
			</section>
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
