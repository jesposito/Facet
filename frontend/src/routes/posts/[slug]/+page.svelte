<script lang="ts">
	import type { PageData } from './$types';
	import { parseMarkdown, formatDate } from '$lib/utils';
import ThemeToggle from '$components/shared/ThemeToggle.svelte';
import Footer from '$components/public/Footer.svelte';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { onMount } from 'svelte';
	import type { RecordModel } from 'pocketbase';
import { getCanonicalUrl, generateOpenGraphTags } from '$lib/seo';

	export let data: PageData;

	// SEO metadata
	$: baseUrl = browser ? window.location.origin : 'http://localhost:8080';
	$: canonicalUrl = getCanonicalUrl(baseUrl, `posts/${data.post.slug}`);
	$: ogTags = generateOpenGraphTags({
		title: data.post.title,
		description: data.post.excerpt || '',
		image: data.post.cover_image_url || undefined,
		url: canonicalUrl,
		type: 'article',
		siteName: data.profile?.name || 'Facet',
		publishedTime: data.post.published_at || undefined,
		modifiedTime: data.post.updated || undefined
	});

	// Format the published date
	$: publishedDate = data.post.published_at ? formatDate(data.post.published_at) : null;
	$: postThumb = (data.post as Record<string, string>).cover_image_thumb_url ?? data.post.cover_image_url;
	$: postLarge = (data.post as Record<string, string>).cover_image_large_url ?? data.post.cover_image_url;
	let mediaRefs: Array<RecordModel & { url?: string; title?: string; mime?: string }> = (data.media_refs as any) || [];

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
	// Prefer originating view, then referrer, otherwise posts index
	$: backUrl = data.fromView ? `/${data.fromView}` : referrerPath || '/posts';
	$: backLabel = 'Back';

	function isYouTube(url?: string): string | null {
		if (!url) return null;
		try {
			const u = new URL(url);
			if (u.hostname.includes('youtu.be')) return u.pathname.replace('/', '');
			if (u.searchParams.get('v')) return u.searchParams.get('v');
			if (u.pathname.startsWith('/embed/')) return u.pathname.replace('/embed/', '');
			return null;
		} catch {
			return null;
		}
	}

	function isVimeo(url?: string): string | null {
		if (!url) return null;
		try {
			const u = new URL(url);
			if (u.hostname.includes('vimeo.com')) {
				const parts = u.pathname.split('/').filter(Boolean);
				return parts.pop() || null;
			}
			return null;
		} catch {
			return null;
		}
	}

const isImage = (url?: string) => !!url && /\.(png|jpe?g|webp|gif|avif)$/i.test(url);
const isVideoFile = (url?: string) => !!url && /\.(mp4|webm|mov)$/i.test(url);
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
const getHost = (url?: string) => {
	if (!url) return '';
	try {
		return new URL(url, 'http://localhost').host || '';
	} catch {
		return '';
	}
};

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
	<title>{data.post.title} | {data.profile?.name || 'Blog'}</title>
	<meta name="description" content={data.post.excerpt || ''} />

	<!-- Canonical URL -->
	<link rel="canonical" href={canonicalUrl} />

	<!-- Open Graph and Twitter Card meta tags -->
	{#each Object.entries(ogTags) as [property, content]}
		<meta property={property} content={content} />
	{/each}
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
					src={postLarge}
					srcset={`${postThumb ?? data.post.cover_image_url} 640w, ${postLarge} 1280w, ${data.post.cover_image_url} 1600w`}
					sizes="100vw"
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
				on:click|preventDefault={handleBack}
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

		{#if mediaRefs && mediaRefs.length > 0}
			<section class="mt-10 space-y-3">
				<h3 class="text-lg font-semibold text-gray-900 dark:text-white">Attached media</h3>
				<div class="grid gap-4 md:grid-cols-2">
					{#each mediaRefs as media}
						<div class="card p-4 space-y-3">
							<div class="flex items-start gap-3">
								<div class="w-10 h-10 rounded-lg bg-primary-50 dark:bg-primary-900/40 flex items-center justify-center text-primary-700 dark:text-primary-200 shrink-0">
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
										<p class="text-xs text-gray-500 dark:text-gray-400">{getHost(media.url)}</p>
									{/if}
								</div>
							</div>

							{#if isYouTube(media.url)}
								<div class="aspect-video rounded-lg overflow-hidden bg-black/10">
									<iframe
										src={`https://www.youtube.com/embed/${isYouTube(media.url) ?? ''}`}
										title={media.title || 'YouTube'}
										allowfullscreen
										loading="lazy"
										class="w-full h-full"
									></iframe>
								</div>
							{:else if isVimeo(media.url)}
								<div class="aspect-video rounded-lg overflow-hidden bg-black/10">
									<iframe
										src={`https://player.vimeo.com/video/${isVimeo(media.url) ?? ''}`}
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
								<a href={media.url} class="text-primary-600 dark:text-primary-300 hover:underline text-sm inline-flex items-center gap-1" target="_blank" rel="noopener noreferrer">
									<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 6h8m0 0v8m0-8-9.5 9.5a3 3 0 0 1-4.243 0l-.757-.757a3 3 0 0 1 0-4.243L12 6Z"/></svg>
									Open link
								</a>
							{/if}
						</div>
					{/each}
				</div>
			</section>
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
