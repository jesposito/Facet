<script lang="ts">
	import type { PageData } from './$types';
	import { formatDate } from '$lib/utils';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import Footer from '$components/public/Footer.svelte';
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { browser } from '$app/environment';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	// Compute back navigation URL based on where user came from
	let backUrl = $derived(data.fromView ? `/${data.fromView}` : '/');
	let landingMessage = $derived(data.landingPageMessage || 'This profile is being set up.');
	let icalUrl = $derived(browser ? new URL('/talks.ics', pb.baseUrl).toString() : '/talks.ics');

	onMount(() => {
		console.log('[TALKS PAGE CLIENT] Page mounted, backUrl:', backUrl, 'fromView:', data.fromView);
	});

	function handleBackClick() {
		console.log('[TALKS PAGE CLIENT] Back button clicked, navigating to:', backUrl);
	}

	// Extract YouTube/Vimeo thumbnail
	function getVideoThumbnail(url: string): string | null {
		// YouTube
		const ytMatch = url.match(/(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/)([^&?/]+)/);
		if (ytMatch) {
			return `https://img.youtube.com/vi/${ytMatch[1]}/mqdefault.jpg`;
		}
		// Vimeo - would need API call, so skip for now
		return null;
	}
</script>

<svelte:head>
	<title>Talks{data.selectedYear ? ` from ${data.selectedYear}` : ''} | {data.profile?.name || 'Talks'}</title>
	<meta name="description" content="Talks and presentations{data.selectedYear ? ` from ${data.selectedYear}` : ''}" />
	<link rel="canonical" href="/talks{data.selectedYear ? `?year=${data.selectedYear}` : ''}" />
	<meta property="og:title" content="Talks | {data.profile?.name || 'Talks'}" />
	<meta property="og:type" content="website" />
</svelte:head>

{#if data.homepageDisabled}
	<div class="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center px-4">
		<div class="fixed top-4 right-4 z-40 flex items-center gap-2 print:hidden">
			<ThemeToggle />
		</div>
		<div class="max-w-2xl w-full bg-white dark:bg-gray-800 rounded-2xl shadow-lg border border-gray-200 dark:border-gray-700 p-8 text-center space-y-4">
			<h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Talks are hidden right now</h1>
			<p class="text-gray-600 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{landingMessage}</p>
		</div>
	</div>
{:else}
<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<!-- Theme toggle -->
	<div class="fixed top-4 right-4 z-40 print:hidden">
		<ThemeToggle />
	</div>

	<!-- Header -->
	<header class="bg-gradient-to-br from-gray-900 to-gray-800 text-white">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
			<!-- Back navigation -->
			<!-- Using data-sveltekit-reload to force full page load - workaround for client-side nav issue -->
			<a
				href={backUrl}
				onclick={handleBackClick}
				data-sveltekit-reload
				class="inline-flex items-center gap-2 text-gray-300 hover:text-white mb-6 transition-colors"
			>
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
				</svg>
				<span>Back to Profile</span>
			</a>

			<h1 class="text-3xl sm:text-4xl font-bold">
				{#if data.selectedYear}
					Talks from {data.selectedYear}
				{:else}
					All Talks
				{/if}
			</h1>

			{#if data.profile?.name}
				<p class="mt-2 text-gray-300">
					by {data.profile.name}
				</p>
			{/if}

			<div class="mt-6 flex flex-wrap gap-3">
				<a
					href={icalUrl}
					class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 text-white transition-colors border border-white/20"
				>
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 15h.01M12 15h.01M16 15h.01" />
					</svg>
					<span class="text-sm font-medium">Subscribe to calendar (.ics)</span>
				</a>
			</div>
		</div>
	</header>

	<!-- Main content -->
	<main id="main-content" class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
		<!-- Year filter -->
		{#if data.allYears.length > 0}
			<div class="mb-8">
				<h2 class="sr-only">Filter by year</h2>
				<div class="flex flex-wrap gap-2">
					<a
						href="/talks"
						class="px-3 py-1.5 text-sm rounded-full transition-colors {!data.selectedYear
							? 'bg-primary-600 text-white'
							: 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'}"
					>
						All
					</a>
					{#each data.allYears as year}
						<a
							href="/talks?year={year}"
							class="px-3 py-1.5 text-sm rounded-full transition-colors {data.selectedYear === year
								? 'bg-primary-600 text-white'
								: 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'}"
						>
							{year}
						</a>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Talks list -->
		{#if data.talks.length > 0}
			<div class="space-y-6">
				{#each data.talks as talk (talk.id)}
					{@const thumbnail = talk.video_url ? getVideoThumbnail(talk.video_url) : null}
					<article class="group bg-white dark:bg-gray-800 rounded-xl shadow-sm overflow-hidden hover:shadow-md transition-shadow">
						<div class="flex flex-col sm:flex-row">
							<!-- Thumbnail or placeholder -->
							<div class="sm:w-64 flex-shrink-0">
								{#if talk.slug}
									<a href="/talks/{talk.slug}" class="block aspect-video sm:h-full">
										{#if thumbnail}
											<img
												src={thumbnail}
												alt=""
												class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
											/>
										{:else if talk.video_url}
											<div class="w-full h-full bg-gradient-to-br from-purple-600 to-indigo-800 flex items-center justify-center">
												<svg class="w-12 h-12 text-white/50" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
												</svg>
											</div>
										{:else}
											<div class="w-full h-full bg-gradient-to-br from-indigo-500 to-purple-700 flex items-center justify-center">
												<svg class="w-12 h-12 text-white/50" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
												</svg>
											</div>
										{/if}
									</a>
								{:else}
									<div class="aspect-video sm:h-full">
										{#if thumbnail}
											<img
												src={thumbnail}
												alt=""
												class="w-full h-full object-cover"
											/>
										{:else if talk.video_url}
											<div class="w-full h-full bg-gradient-to-br from-purple-600 to-indigo-800 flex items-center justify-center">
												<svg class="w-12 h-12 text-white/50" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
												</svg>
											</div>
										{:else}
											<div class="w-full h-full bg-gradient-to-br from-indigo-500 to-purple-700 flex items-center justify-center">
												<svg class="w-12 h-12 text-white/50" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
												</svg>
											</div>
										{/if}
									</div>
								{/if}
							</div>

							<!-- Content -->
							<div class="flex-1 p-5">
								{#if talk.slug}
									<a href="/talks/{talk.slug}" class="block">
										<h2 class="text-lg font-semibold text-gray-900 dark:text-white group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
											{talk.title}
										</h2>
									</a>
								{:else}
									<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
										{talk.title}
									</h2>
								{/if}

								<!-- Event and meta -->
								<div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-500 dark:text-gray-400">
									{#if talk.event}
										<span class="flex items-center gap-1.5">
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
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
										<time datetime={talk.date} class="flex items-center gap-1.5">
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
											</svg>
											{formatDate(talk.date, { month: 'long', day: 'numeric', year: 'numeric' })}
										</time>
									{/if}

									{#if talk.location}
										<span class="flex items-center gap-1.5">
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
											</svg>
											{talk.location}
										</span>
									{/if}
								</div>

								<!-- Action links -->
								<div class="mt-4 flex flex-wrap gap-3">
									{#if talk.slug}
										<a
											href="/talks/{talk.slug}"
											class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
										>
											View Details
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
											</svg>
										</a>
									{/if}

									{#if talk.video_url}
										<a
											href={talk.video_url}
											target="_blank"
											rel="noopener noreferrer"
											class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
										>
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
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
											class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
										>
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
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
		{:else}
			<!-- Empty state -->
			<div class="text-center py-16">
				<svg class="w-16 h-16 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z" />
				</svg>
				<p class="text-gray-500 dark:text-gray-400 text-lg">
					{#if data.selectedYear}
						No talks found from {data.selectedYear}
					{:else}
						No talks yet
					{/if}
				</p>
				{#if data.selectedYear}
					<a href="/talks" class="mt-4 inline-block text-primary-600 dark:text-primary-400 hover:underline">
						View all talks
					</a>
				{/if}
			</div>
		{/if}
	</main>

	<!-- Footer -->
	<Footer profile={data.profile} />
</div>
{/if}
