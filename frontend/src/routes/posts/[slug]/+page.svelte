<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import type { PageData } from './$types';
	import { parseMarkdown, formatDate, isFilename } from '$lib/utils';
import ThemeToggle from '$components/shared/ThemeToggle.svelte';
import ShareButton from '$components/shared/ShareButton.svelte';
import VisibilityBadge from '$components/shared/VisibilityBadge.svelte';
import Footer from '$components/public/Footer.svelte';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { onMount } from 'svelte';
	import type { RecordModel } from 'pocketbase';
import { getCanonicalUrl, generateOpenGraphTags } from '$lib/seo';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	// SEO metadata
	let baseUrl = $derived(browser ? window.location.origin : 'http://localhost:8080');
	let canonicalUrl = $derived(getCanonicalUrl(baseUrl, `posts/${data.post.slug}`));
	let ogTags = $derived(generateOpenGraphTags({
		title: data.post.title,
		description: data.post.excerpt || '',
		image: data.post.cover_image_url || undefined,
		url: canonicalUrl,
		type: 'article',
		siteName: data.profile?.name || 'Facet',
		publishedTime: data.post.published_at || undefined,
		modifiedTime: data.post.updated || undefined
	}));

	// Format the published date
	let publishedDate = $derived(data.post.published_at ? formatDate(data.post.published_at) : null);
	let postThumb = $derived((data.post as Record<string, string>).cover_image_thumb_url ?? data.post.cover_image_url);
	let postLarge = $derived((data.post as Record<string, string>).cover_image_large_url ?? data.post.cover_image_url);
	let mediaRefs = $derived((data.media_refs as Array<RecordModel & {
	url?: string;
	title?: string;
	mime?: string;
	description?: string;
	alt_text?: string;
	type?: string;
	provider?: string;
}>) || []);

	// Lightbox state
	let lightboxOpen = $state(false);
	let lightboxIndex = $state(0);
	let imageMedia = $derived(mediaRefs.filter(m => m.url && /\.(png|jpe?g|webp|gif|avif)$/i.test(m.url)));

	function openLightbox(index: number) {
		lightboxIndex = index;
		lightboxOpen = true;
	}

	function closeLightbox() {
		lightboxOpen = false;
	}

	function nextImage() {
		if (imageMedia.length > 0) {
			lightboxIndex = (lightboxIndex + 1) % imageMedia.length;
		}
	}

	function prevImage() {
		if (imageMedia.length > 0) {
			lightboxIndex = (lightboxIndex - 1 + imageMedia.length) % imageMedia.length;
		}
	}

	function handleLightboxKeydown(e: KeyboardEvent) {
		if (!lightboxOpen) return;
		if (e.key === 'Escape') closeLightbox();
		if (e.key === 'ArrowRight') nextImage();
		if (e.key === 'ArrowLeft') prevImage();
	}

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
	// Prefer originating view, then referrer, otherwise posts index
	let backUrl = $derived(data.fromView ? `/${data.fromView}` : referrerPath || '/posts');
	let backLabel = 'Back';

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
	<div class="fixed top-4 right-4 z-40 flex items-center gap-2 print:hidden">
		{#if data.post.visibility === 'public'}
			<ShareButton 
				url={browser ? window.location.href : `/posts/${data.post.slug}`}
				title={`${data.post.title} | ${data.profile?.name || 'Blog'}`}
				text={data.post.excerpt || data.post.title}
			/>
		{/if}
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
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
				</svg>
				<span>{backLabel}</span>
			</a>

			<!-- Post title -->
			<div class="flex flex-wrap items-start gap-3">
				<h1 class="text-3xl sm:text-4xl lg:text-5xl font-bold leading-tight">
					{data.post.title}
				</h1>
				{#if (data as any).isAuthenticated && ((data.post as any).visibility !== 'public' || (data.post as any).is_draft)}
					<VisibilityBadge visibility={(data.post as any).visibility} isDraft={(data.post as any).is_draft} />
				{/if}
			</div>

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
				<h3 class="text-lg font-semibold text-gray-900 dark:text-white">Image Gallery</h3>
				<div class="grid gap-4 md:grid-cols-2">
					{#each mediaRefs as media}
						{@const displayTitle = media.title && !isFilename(media.title) ? media.title : ''}
						<div class="card overflow-hidden">
							{#if isYouTube(media.url)}
								<div class="aspect-video bg-black/10">
									<iframe
										src={`https://www.youtube.com/embed/${isYouTube(media.url) ?? ''}`}
										title={media.title || 'YouTube'}
										allowfullscreen
										loading="lazy"
										class="w-full h-full"
									></iframe>
								</div>
							{:else if isVimeo(media.url)}
								<div class="aspect-video bg-black/10">
									<iframe
										src={`https://player.vimeo.com/video/${isVimeo(media.url) ?? ''}`}
										title={media.title || 'Vimeo'}
										allowfullscreen
										loading="lazy"
										class="w-full h-full"
									></iframe>
								</div>
							{:else if isImage(media.url)}
								<button
									type="button"
									onclick={() => openLightbox(imageMedia.findIndex(m => m.url === media.url))}
									class="w-full cursor-zoom-in gallery-thumb bg-gray-100 dark:bg-gray-800 overflow-hidden"
								>
									<img src={media.url || ''} alt={media.alt_text || media.title || ''} class="w-full h-full object-cover" loading="lazy" />
								</button>
							{:else if isVideoFile(media.url)}
								<video src={media.url || ''} controls class="w-full">
									<track kind="captions" srclang="en" label="captions" />
								</video>
							{:else if media.url}
								<div class="p-4">
									<a href={media.url} class="text-primary-600 dark:text-primary-300 hover:underline text-sm inline-flex items-center gap-1" target="_blank" rel="noopener noreferrer">
										<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 6h8m0 0v8m0-8-9.5 9.5a3 3 0 0 1-4.243 0l-.757-.757a3 3 0 0 1 0-4.243L12 6Z"/></svg>
										{media.title || getFileName(media.url)}
									</a>
								</div>
							{/if}
							{#if displayTitle || media.description || (media.provider && media.provider !== 'upload')}
								<div class="p-3 space-y-1">
									{#if displayTitle}
										<p class="text-sm font-medium text-gray-900 dark:text-white">{displayTitle}</p>
									{/if}
									{#if media.description}
										<p class="text-xs text-gray-600 dark:text-gray-300">{media.description}</p>
									{/if}
									{#if media.url && media.provider && media.provider !== 'upload'}
										<p class="text-xs text-gray-500 dark:text-gray-400">Source: {getHost(media.url)}</p>
									{/if}
								</div>
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

<!-- Lightbox Modal -->
<svelte:window onkeydown={handleLightboxKeydown} />

{#if lightboxOpen && imageMedia[lightboxIndex]}
	<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
	<div
		class="fixed inset-0 z-50 bg-black/90 flex items-center justify-center"
		onclick={closeLightbox}
		oncontextmenu={(e) => e.preventDefault()}
	>
		<!-- Close button -->
		<button
			type="button"
			onclick={closeLightbox}
			class="absolute top-4 right-4 p-2 text-white/70 hover:text-white transition-colors"
			aria-label="Close lightbox"
		>
			<svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>

		<!-- Previous button -->
		{#if imageMedia.length > 1}
			<button
				type="button"
				onclick={(e) => { e.stopPropagation(); prevImage(); }}
				class="absolute left-4 p-2 text-white/70 hover:text-white transition-colors"
				aria-label="Previous image"
			>
				<svg class="w-10 h-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
				</svg>
			</button>

			<!-- Next button -->
			<button
				type="button"
				onclick={(e) => { e.stopPropagation(); nextImage(); }}
				class="absolute right-4 p-2 text-white/70 hover:text-white transition-colors"
				aria-label="Next image"
			>
				<svg class="w-10 h-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		{/if}

		<!-- Image container -->
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
		<img
			src={imageMedia[lightboxIndex].url || ''}
			alt={imageMedia[lightboxIndex].title || imageMedia[lightboxIndex].alt_text || ''}
			class="max-h-[90vh] max-w-[90vw] object-contain"
			onclick={(e) => e.stopPropagation()}
			oncontextmenu={(e) => e.preventDefault()}
		/>

		<!-- Image counter and title -->
		<div class="absolute bottom-4 left-1/2 -translate-x-1/2 text-center text-white">
			{#if imageMedia[lightboxIndex].title && !isFilename(imageMedia[lightboxIndex].title)}
				<p class="text-lg font-medium mb-1">{imageMedia[lightboxIndex].title}</p>
			{/if}
			{#if imageMedia.length > 1}
				<p class="text-sm text-white/70">{lightboxIndex + 1} / {imageMedia.length}</p>
			{/if}
		</div>
	</div>
{/if}
