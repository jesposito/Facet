<script lang="ts">
	import { t } from 'svelte-i18n';
	import type { CustomContent } from '$lib/pocketbase';
	import { parseMarkdown } from '$lib/utils';

	interface Props {
		item: CustomContent;
		layout?: string;
	}

	let { item, layout = 'default' }: Props = $props();

	// Process library media refs (attached media/embeds)
	let mediaRefs = $derived(() => {
		if (item.media_refs_expand && Array.isArray(item.media_refs_expand)) {
			return item.media_refs_expand;
		}
		return [];
	});

	// Filter for images only (for lightbox)
	let imageMedia = $derived(mediaRefs().filter(m => m.url && /\.(png|jpe?g|webp|gif|avif)$/i.test(m.url)));

	// Lightbox state
	let lightboxOpen = $state(false);
	let lightboxIndex = $state(0);

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
	// Check if a title is just a filename (e.g. "IMG_1234.jpg") rather than a meaningful caption
	const isFilename = (title?: string) => !!title && /\.\w{2,5}$/.test(title) && !title.includes(' ');

	// Type for media ref items
	type MediaRef = {
		id: string;
		url: string;
		title?: string;
		mime?: string;
		thumbnail_url?: string;
		description?: string;
		alt_text?: string;
		provider?: string;
	};
</script>

<svelte:window onkeydown={handleLightboxKeydown} />

{#snippet mediaRefCard(media: MediaRef)}
	{#if isImage(media.url)}
		{@const displayTitle = media.title && !isFilename(media.title) ? media.title : ''}
		<!-- Image: clean gallery card without decoration -->
		<div class="card overflow-hidden">
			<button type="button" onclick={() => openLightbox(imageMedia.findIndex(m => m.url === media.url))} class="w-full cursor-zoom-in gallery-thumb bg-gray-100 dark:bg-gray-800 overflow-hidden">
				<img src={media.url || ''} alt={media.alt_text || media.title || ''} class="w-full h-full object-cover" loading="lazy" />
			</button>
			{#if displayTitle || media.description}
				<div class="p-3 space-y-1">
					{#if displayTitle}
						<p class="text-sm font-medium text-gray-900 dark:text-white">{displayTitle}</p>
					{/if}
					{#if media.description}
						<p class="text-xs text-gray-600 dark:text-gray-300">{media.description}</p>
					{/if}
				</div>
			{/if}
		</div>
	{:else}
		<!-- Non-image media: keep current card layout with icon header -->
		<div class="card p-4 space-y-3">
			<div class="flex items-start gap-3">
				<div class="w-10 h-10 rounded-lg bg-primary-50 dark:bg-primary-900/40 flex items-center justify-center text-primary-700 dark:text-primary-200 shrink-0">
					{#if isYouTube(media.url)}
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M21.6 7.2s-.2-1.5-.8-2.2c-.7-.8-1.5-.8-1.9-.8C15.7 4 12 4 12 4h-.1S8.3 4 5.1 4.2c-.4 0-1.2 0-1.9.8-.6.7-.8 2.2-.8 2.2S2 8.9 2 10.6v1.6c0 1.7.2 3.4.2 3.4s.2 1.5.8 2.2c.7.8 1.7.8 2.1.9 1.5.1 6.9.2 6.9.2s3.7 0 6.9-.2c.4 0 1.2 0 1.9-.9.6-.7.8-2.2.8-2.2s.2-1.7.2-3.4v-1.6c0-1.7-.2-3.4-.2-3.4Zm-12.7 6.8V8.8l5.2 2.6-5.2 2.6Z"/></svg>
					{:else if isVimeo(media.url)}
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M22.37 6.76c-.1 2.2-1.64 5.21-4.63 9.05-3.08 4-5.68 6-7.8 6-1.32 0-2.44-1.2-3.36-3.6l-1.84-6.6c-.68-2.4-1.4-3.6-2.16-3.6-.17 0-.78.36-1.82 1.08L0 7.38c1.15-1.01 2.29-2.02 3.43-3.03 1.54-1.33 2.7-2.03 3.5-2.1 1.84-.18 2.98 1.08 3.42 3.78.46 2.91.78 4.72.96 5.4.53 2.4 1.11 3.6 1.76 3.6.5 0 1.26-.79 2.28-2.36 1.01-1.58 1.55-2.79 1.62-3.64.14-1.38-.4-2.07-1.62-2.07-.58 0-1.18.12-1.8.36 1.2-3.9 3.47-5.79 6.8-5.68 2.48.06 3.64 1.68 3.48 4.86Z"/></svg>
					{:else if isVideoFile(media.url)}
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="5" width="18" height="14" rx="2"/><path d="m10 9 5 3-5 3V9Z"/></svg>
					{:else}
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14m-5-5 5 5 5-5"/><path d="M5 9V7a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v2"/></svg>
					{/if}
				</div>
				<div class="min-w-0 flex-1">
					<p class="text-sm font-semibold text-gray-900 dark:text-white line-clamp-1">
						{media.title || media.url}
					</p>
					{#if media.description}
						<p class="text-xs text-gray-600 dark:text-gray-400 line-clamp-2 mt-0.5">
							{media.description}
						</p>
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
			{:else if isVideoFile(media.url)}
				<video src={media.url || ''} controls class="w-full rounded-lg">
					<track kind="captions" srclang="en" label="captions" />
				</video>
			{:else if media.url}
				<a href={media.url} class="text-primary-600 dark:text-primary-300 hover:underline text-sm inline-flex items-center gap-1" target="_blank" rel="noopener noreferrer">
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 6h8m0 0v8m0-8-9.5 9.5a3 3 0 0 1-4.243 0l-.757-.757a3 3 0 0 1 0-4.243L12 6Z"/></svg>
					{$t('public.custom_content.open_link')}
				</a>
			{/if}
		</div>
	{/if}
{/snippet}

<section id="custom-{item.id}" class="mb-16" data-layout={layout}>
	<h2 class="section-title">{item.title}</h2>

	{#if layout === 'hero'}
		<!-- Hero layout: full-width cover image with content overlay -->
		<div class="space-y-6 animate-fade-in">
			<div class="relative">
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
							{@html parseMarkdown(item.content || '')}
						</div>
					</div>
				{/if}
			</div>

			<!-- Attached media/embeds for hero layout -->
			{#if mediaRefs().length > 0}
				<div class="grid gap-4 md:grid-cols-2">
					{#each mediaRefs() as media}
						{@render mediaRefCard(media)}
					{/each}
				</div>
			{/if}
		</div>
	{:else if layout === 'card'}
		<!-- Card layout: compact card with optional image -->
		<div class="space-y-6 animate-fade-in">
			<div class="card p-6">
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
								{@html parseMarkdown(item.content || '')}
							</div>
						{/if}
					</div>
				</div>
			</div>

			<!-- Attached media/embeds for card layout -->
			{#if mediaRefs().length > 0}
				<div class="grid gap-4 md:grid-cols-2">
					{#each mediaRefs() as media}
						{@render mediaRefCard(media)}
					{/each}
				</div>
			{/if}
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
					{@html parseMarkdown(item.content || '')}
				</div>
			{/if}

			<!-- Attached media/embeds for default layout -->
			{#if mediaRefs().length > 0}
				<div class="grid gap-4 md:grid-cols-2">
					{#each mediaRefs() as media}
						{@render mediaRefCard(media)}
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</section>

<!-- Lightbox modal for images -->
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
				<svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
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
				<svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		{/if}

		<!-- Image -->
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
		<img
			src={imageMedia[lightboxIndex].url || ''}
			alt={imageMedia[lightboxIndex].title || imageMedia[lightboxIndex].alt_text || ''}
			class="max-h-[90vh] max-w-[90vw] object-contain"
			onclick={(e) => e.stopPropagation()}
			oncontextmenu={(e) => e.preventDefault()}
		/>

		<!-- Image counter and caption -->
		<div class="absolute bottom-4 left-1/2 -translate-x-1/2 text-center text-white">
			{#if imageMedia[lightboxIndex].title}
				<p class="text-lg font-medium mb-1">{imageMedia[lightboxIndex].title}</p>
			{/if}
			{#if imageMedia[lightboxIndex].description}
				<p class="text-sm text-white/80 mb-1">{imageMedia[lightboxIndex].description}</p>
			{/if}
			{#if imageMedia.length > 1}
				<p class="text-sm text-white/70">{lightboxIndex + 1} / {imageMedia.length}</p>
			{/if}
		</div>
	</div>
{/if}
