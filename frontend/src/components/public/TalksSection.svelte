<script lang="ts">
	import type { Talk } from '$lib/pocketbase';
	import { formatDate, parseMarkdown } from '$lib/utils';

	export let items: Talk[];
	export let layout: string = 'default';
	export let viewSlug: string = '';

	// Build the talk URL with optional from parameter for back navigation
	function getTalkUrl(slug: string): string {
		if (viewSlug) {
			return `/talks/${slug}?from=${encodeURIComponent(viewSlug)}`;
		}
		return `/talks/${slug}`;
	}

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
</script>

<section id="talks" class="mb-16">
	<h2 class="section-title">Talks & Presentations</h2>

	<div class="space-y-8">
		{#each items as talk (talk.id)}
			<article class="card overflow-hidden animate-fade-in">
				<div class="flex flex-col lg:flex-row">
					<!-- Video embed or placeholder -->
					{#if talk.video_url}
						{@const embed = getVideoEmbed(talk.video_url)}
						{#if embed}
							<div class="lg:w-2/5 aspect-video bg-gray-900 flex-shrink-0">
								{#if embed.type === 'youtube'}
									<iframe
										src="https://www.youtube.com/embed/{embed.id}"
										title={talk.title}
										class="w-full h-full"
										frameborder="0"
										allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
										allowfullscreen
									></iframe>
								{:else if embed.type === 'vimeo'}
									<iframe
										src="https://player.vimeo.com/video/{embed.id}"
										title={talk.title}
										class="w-full h-full"
										frameborder="0"
										allow="autoplay; fullscreen; picture-in-picture"
										allowfullscreen
									></iframe>
								{/if}
							</div>
						{:else}
							<!-- Non-embeddable video URL - show link instead -->
							<div class="lg:w-2/5 aspect-video bg-gradient-to-br from-purple-600 to-indigo-800 flex-shrink-0 flex items-center justify-center">
								<a href={talk.video_url} target="_blank" rel="noopener noreferrer" class="flex flex-col items-center gap-3 text-white hover:opacity-80 transition-opacity">
									<svg class="w-16 h-16" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
									<span class="text-sm font-medium">Watch Video</span>
								</a>
							</div>
						{/if}
					{:else}
						<!-- No video - show presentation icon -->
						<div class="lg:w-2/5 aspect-video bg-gradient-to-br from-indigo-500 to-purple-700 flex-shrink-0 flex items-center justify-center">
							<svg class="w-20 h-20 text-white/50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
							</svg>
						</div>
					{/if}

					<!-- Content -->
					<div class="flex-1 p-6">
						<h3 class="text-xl font-semibold text-gray-900 dark:text-white">
							{talk.title}
						</h3>

						<!-- Event and meta info -->
						<div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-600 dark:text-gray-400">
							{#if talk.event}
								<span class="flex items-center gap-1.5">
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
									</svg>
									{#if talk.event_url}
										<a href={talk.event_url} target="_blank" rel="noopener noreferrer" class="hover:text-primary-600 dark:hover:text-primary-400">
											{talk.event}
										</a>
									{:else}
										{talk.event}
									{/if}
								</span>
							{/if}

							{#if talk.date}
								<span class="flex items-center gap-1.5">
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
									</svg>
									{formatDate(talk.date, { month: 'long', day: 'numeric', year: 'numeric' })}
								</span>
							{/if}

							{#if talk.location}
								<span class="flex items-center gap-1.5">
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
									</svg>
									{talk.location}
								</span>
							{/if}
						</div>

						<!-- Description -->
						{#if talk.description}
							<div class="mt-4 text-gray-600 dark:text-gray-400 prose prose-sm dark:prose-invert max-w-none">
								{@html parseMarkdown(talk.description)}
							</div>
						{/if}

						<!-- Action links -->
						<div class="mt-4 flex flex-wrap gap-3">
							{#if talk.video_url}
								<a
									href={talk.video_url}
									target="_blank"
									rel="noopener noreferrer"
									class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
								>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
									Watch Video
								</a>
							{/if}

							{#if talk.slides_url}
								<a
									href={talk.slides_url}
									target="_blank"
									rel="noopener noreferrer"
									class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
								>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
									</svg>
									View Slides
								</a>
							{/if}
						</div>
					</div>
				</div>
			</article>
		{/each}
	</div>
</section>
