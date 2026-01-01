<script lang="ts">
	import type { PageData } from './$types';
	import { formatDate } from '$lib/utils';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import Footer from '$components/public/Footer.svelte';

	export let data: PageData;

	// Compute back navigation URL based on where user came from
	$: backUrl = data.fromView ? `/${data.fromView}` : '/';
</script>

<svelte:head>
	<title>Posts{data.selectedTag ? ` tagged "${data.selectedTag}"` : ''} | {data.profile?.name || 'Blog'}</title>
	<meta name="description" content="Blog posts and articles{data.selectedTag ? ` about ${data.selectedTag}` : ''}" />
	<link rel="canonical" href="/posts{data.selectedTag ? `?tag=${data.selectedTag}` : ''}" />
	<meta property="og:title" content="Posts | {data.profile?.name || 'Blog'}" />
	<meta property="og:type" content="website" />
</svelte:head>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<!-- Theme toggle -->
	<div class="fixed top-4 right-4 z-40 print:hidden">
		<ThemeToggle />
	</div>

	<!-- Header -->
	<header class="bg-gradient-to-br from-gray-900 to-gray-800 text-white">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
			<!-- Back navigation -->
			<a
				href={backUrl}
				class="inline-flex items-center gap-2 text-gray-300 hover:text-white mb-6 transition-colors"
			>
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
				</svg>
				<span>Back to Profile</span>
			</a>

			<h1 class="text-3xl sm:text-4xl font-bold">
				{#if data.selectedTag}
					Posts tagged "{data.selectedTag}"
				{:else}
					All Posts
				{/if}
			</h1>

			{#if data.profile?.name}
				<p class="mt-2 text-gray-300">
					by {data.profile.name}
				</p>
			{/if}
		</div>
	</header>

	<!-- Main content -->
	<main id="main-content" class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
		<!-- Tag filter -->
		{#if data.allTags.length > 0}
			<div class="mb-8">
				<h2 class="sr-only">Filter by tag</h2>
				<div class="flex flex-wrap gap-2">
					<a
						href="/posts"
						class="px-3 py-1.5 text-sm rounded-full transition-colors {!data.selectedTag
							? 'bg-primary-600 text-white'
							: 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'}"
					>
						All
					</a>
					{#each data.allTags as tag}
						<a
							href="/posts?tag={encodeURIComponent(tag)}"
							class="px-3 py-1.5 text-sm rounded-full transition-colors {data.selectedTag === tag
								? 'bg-primary-600 text-white'
								: 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'}"
						>
							{tag}
						</a>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Posts grid -->
		{#if data.posts.length > 0}
			<div class="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
				{#each data.posts as post (post.id)}
					<article class="group bg-white dark:bg-gray-800 rounded-xl shadow-sm overflow-hidden hover:shadow-md transition-shadow">
						<!-- Cover image -->
						{#if post.cover_image_url}
							<a href="/posts/{post.slug}" class="block aspect-video overflow-hidden">
								<img
									src={post.cover_image_url}
									alt=""
									class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
								/>
							</a>
						{:else}
							<a href="/posts/{post.slug}" class="block aspect-video bg-gradient-to-br from-primary-500 to-purple-600 flex items-center justify-center">
								<svg class="w-12 h-12 text-white/50" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
								</svg>
							</a>
						{/if}

						<!-- Content -->
						<div class="p-5">
							<a href="/posts/{post.slug}" class="block">
								<h2 class="text-lg font-semibold text-gray-900 dark:text-white group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors line-clamp-2">
									{post.title}
								</h2>
							</a>

							{#if post.published_at}
								<time
									datetime={post.published_at}
									class="block mt-2 text-sm text-gray-500 dark:text-gray-400"
								>
									{formatDate(post.published_at, { month: 'long', day: 'numeric', year: 'numeric' })}
								</time>
							{/if}

							{#if post.excerpt}
								<p class="mt-3 text-gray-600 dark:text-gray-400 text-sm line-clamp-3">
									{post.excerpt}
								</p>
							{/if}

							{#if post.tags && post.tags.length > 0}
								<div class="mt-4 flex flex-wrap gap-1.5">
									{#each post.tags.slice(0, 3) as tag}
										<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
											{tag}
										</span>
									{/each}
									{#if post.tags.length > 3}
										<span class="px-2 py-0.5 text-xs text-gray-500 dark:text-gray-500">
											+{post.tags.length - 3}
										</span>
									{/if}
								</div>
							{/if}
						</div>
					</article>
				{/each}
			</div>
		{:else}
			<!-- Empty state -->
			<div class="text-center py-16">
				<svg class="w-16 h-16 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
				</svg>
				<p class="text-gray-500 dark:text-gray-400 text-lg">
					{#if data.selectedTag}
						No posts found with tag "{data.selectedTag}"
					{:else}
						No posts yet
					{/if}
				</p>
				{#if data.selectedTag}
					<a href="/posts" class="mt-4 inline-block text-primary-600 dark:text-primary-400 hover:underline">
						View all posts
					</a>
				{/if}
			</div>
		{/if}
	</main>

	<!-- Footer -->
	<Footer profile={data.profile} />
</div>
