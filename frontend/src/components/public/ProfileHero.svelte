<script lang="ts">
	import type { Profile } from '$lib/pocketbase';
	import { parseMarkdown } from '$lib/utils';

	interface Props {
		profile: Profile | null;
	}

	let { profile }: Props = $props();

	let contactLinks = $derived(profile?.contact_links || []);
	let heroImageUrl = $derived(profile?.hero_image_url || null);
	let avatarUrl = $derived((profile as unknown as Record<string, string>)?.avatar_url || null);
</script>

<header class="relative bg-gradient-to-br from-gray-900 to-gray-800 text-white">
	{#if heroImageUrl}
		<div class="absolute inset-0" aria-hidden="true">
			<img
				src={heroImageUrl}
				alt=""
				class="w-full h-full object-cover opacity-30"
			/>
			<div class="absolute inset-0 bg-gradient-to-t from-gray-900 via-gray-900/80 to-transparent"></div>
		</div>
	{/if}

	<div class="relative max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-16 sm:py-24">
		<div class="flex flex-col sm:flex-row items-center sm:items-start gap-6 sm:gap-8">
			{#if avatarUrl}
				<img
					src={avatarUrl}
					alt={profile?.name ? `${profile.name}'s profile photo` : 'Profile photo'}
					class="w-32 h-32 sm:w-40 sm:h-40 rounded-full border-4 border-white/20 shadow-xl object-cover"
				/>
			{:else if profile?.name}
				<div class="w-32 h-32 sm:w-40 sm:h-40 rounded-full bg-primary-600 flex items-center justify-center text-4xl font-bold border-4 border-white/20" role="img" aria-label={`${profile.name}'s profile initial`}>
					{profile.name.charAt(0)}
				</div>
			{/if}

			<div class="text-center sm:text-left flex-1">
				{#if profile?.name}
					<h1 class="text-3xl sm:text-4xl lg:text-5xl font-bold mb-2">
						{profile.name}
					</h1>
				{/if}

				{#if profile?.headline}
					<p class="text-xl sm:text-2xl text-gray-300 mb-4">
						{profile.headline}
					</p>
				{/if}

				{#if profile?.location}
					<p class="flex items-center justify-center sm:justify-start gap-2 text-gray-400 mb-4">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
						</svg>
						<span class="sr-only">Location:</span>
						{profile.location}
					</p>
				{/if}

				{#if contactLinks.length > 0}
					<nav class="flex flex-wrap items-center justify-center sm:justify-start gap-3" aria-label="Contact links">
						{#each contactLinks as link}
							<a
								href={link.url}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
								aria-label={`${link.label || link.type} (opens in new tab)`}
							>
								{#if link.type === 'github'}
									<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
										<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
									</svg>
								{:else if link.type === 'linkedin'}
									<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
										<path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433c-1.144 0-2.063-.926-2.063-2.065 0-1.138.92-2.063 2.063-2.063 1.14 0 2.064.925 2.064 2.063 0 1.139-.925 2.065-2.064 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/>
									</svg>
								{:else if link.type === 'email'}
									<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
									</svg>
								{:else}
									<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
									</svg>
								{/if}
								<span>{link.label || link.type}</span>
							</a>
						{/each}
					</nav>
				{/if}
			</div>
		</div>

		{#if profile?.summary}
			<div class="mt-8 prose prose-invert prose-lg max-w-none">
				{@html parseMarkdown(profile.summary)}
			</div>
		{/if}
	</div>
</header>
