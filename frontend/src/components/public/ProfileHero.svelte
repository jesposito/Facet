<script lang="ts">
	import type { Profile } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { parseMarkdown } from '$lib/utils';
	import { hexLuminance, DARK_TEXT_THRESHOLD, isValidHexColor } from '$lib/colors';

	type HeroLayout = 'standard' | 'centered' | 'split' | 'minimal' | 'stacked';
	type HeroSpacing = 'compact' | 'default' | 'spacious' | '';

	interface Props {
		profile: Profile | null;
		layout?: HeroLayout;
		spacing?: HeroSpacing;
		heroBgColor?: string;
		showAvatar?: boolean;
	}

	let { profile, layout = 'standard', spacing = '', heroBgColor = '', showAvatar = true }: Props = $props();

	let contactLinks = $derived(profile?.contact_links || []);
	let heroImageUrl = $derived(profile?.hero_image_url || null);
	let rawAvatarUrl = $derived((profile as unknown as Record<string, string>)?.avatar_url || null);
	// Respect site-wide show_avatar toggle (per cloud parity)
	let avatarUrl = $derived(showAvatar ? rawAvatarUrl : null);

	// Vertical breathing room — cloud parity (ProfileHero.svelte lines 28-34)
	let heroSpacingClass = $derived(
		spacing === 'compact'
			? 'py-10 sm:py-14'
			: spacing === 'spacious'
				? 'py-24 sm:py-36'
				: 'py-16 sm:py-24'
	);

	// Custom hero background color (cloud parity, hero_bg_color field).
	// When image is set, image wins — color is irrelevant.
	let effectiveBgColor = $derived(
		!heroImageUrl && heroBgColor && isValidHexColor(heroBgColor) ? heroBgColor : ''
	);
	// Use dark text when bg is light enough (WCAG 4.5:1 inflection at L≈0.179).
	let useDarkText = $derived(
		effectiveBgColor !== '' && hexLuminance(effectiveBgColor) > DARK_TEXT_THRESHOLD
	);
	// Per a11y review: stone-700 (not 600) and stone-200 (not 300) to keep AA on
	// arbitrary user-chosen backgrounds.
	let textColorClass = $derived(useDarkText ? 'text-stone-900' : 'text-white');
	let subTextColorClass = $derived(useDarkText ? 'text-stone-700' : 'text-stone-200');
	let mutedTextColorClass = $derived(useDarkText ? 'text-stone-700' : 'text-stone-200');
	let contactLinkClass = $derived(
		useDarkText
			? 'bg-stone-900/10 hover:bg-stone-900/20 text-stone-900'
			: 'bg-white/10 hover:bg-white/20'
	);
	let headerBgStyle = $derived(effectiveBgColor !== '' ? `background-color: ${effectiveBgColor};` : '');
	// When custom bg, drop the dark gradient class.
	let headerBgClass = $derived(effectiveBgColor !== '' ? '' : 'bg-gradient-to-br from-gray-900 to-gray-800');
</script>

{#if layout === 'centered'}
<!-- Centered: Bold headline centered, gradient/solid background, no image dependency -->
<header class="grain relative {headerBgClass} {textColorClass}" style={headerBgStyle} itemscope itemtype="https://schema.org/Person">
	<div class="relative max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 {heroSpacingClass} text-center">
		{#if profile?.name}
			<h1 class="hero-name text-4xl sm:text-5xl lg:text-6xl font-semibold mb-4 tracking-tight leading-[1.08]" itemprop="name">
				{profile.name}
			</h1>
		{/if}

		{#if profile?.headline}
			<p class="text-xl sm:text-2xl text-gray-300 mb-6 max-w-2xl mx-auto leading-relaxed" itemprop="jobTitle">
				{profile.headline}
			</p>
		{/if}

		{#if profile?.location}
			<p class="flex items-center justify-center gap-2 text-gray-400 mb-6" itemprop="address" itemscope itemtype="https://schema.org/PostalAddress">
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
				<span class="sr-only">{$t('public.hero.location_label')}</span>
				<span itemprop="addressLocality">{profile.location}</span>
			</p>
		{/if}

		{#if contactLinks.length > 0}
			<nav class="flex flex-wrap items-center justify-center gap-3" aria-label={$t('public.hero.contact_links_label')}>
				{#each contactLinks as link}
					<a
						href={link.url}
						target="_blank"
						rel="noopener noreferrer"
						class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
						aria-label={$t('public.hero.opens_new_tab', { values: { label: link.label || link.type } })}
						itemprop={link.type === 'email' ? 'email' : 'sameAs'}
					>
						{@render contactIcon(link.type)}
						<span>{link.label || link.type}</span>
					</a>
				{/each}
			</nav>
		{/if}

		{#if profile?.summary}
			<div class="mt-8 prose prose-invert prose-lg max-w-2xl mx-auto text-center" itemprop="description">
				{@html parseMarkdown(profile.summary)}
			</div>
		{/if}
	</div>
</header>

{:else if layout === 'split'}
<!-- Split: Text left, avatar/image right -->
<header class="grain relative {headerBgClass} {textColorClass}" style={headerBgStyle} itemscope itemtype="https://schema.org/Person">
	<div class="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 {heroSpacingClass}">
		<div class="grid grid-cols-1 md:grid-cols-5 gap-8 md:gap-12 items-center">
			<div class="md:col-span-3">
				{#if profile?.name}
					<h1 class="hero-name text-3xl sm:text-4xl lg:text-5xl font-semibold mb-3 tracking-tight leading-[1.08]" itemprop="name">
						{profile.name}
					</h1>
				{/if}

				{#if profile?.headline}
					<p class="text-xl sm:text-2xl text-gray-300 mb-4 leading-relaxed" itemprop="jobTitle">
						{profile.headline}
					</p>
				{/if}

				{#if profile?.location}
					<p class="flex items-center gap-2 text-gray-400 mb-4" itemprop="address" itemscope itemtype="https://schema.org/PostalAddress">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
						</svg>
						<span class="sr-only">{$t('public.hero.location_label')}</span>
						<span itemprop="addressLocality">{profile.location}</span>
					</p>
				{/if}

				{#if contactLinks.length > 0}
					<nav class="flex flex-wrap items-center gap-3 mb-6" aria-label={$t('public.hero.contact_links_label')}>
						{#each contactLinks as link}
							<a
								href={link.url}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
								aria-label={$t('public.hero.opens_new_tab', { values: { label: link.label || link.type } })}
								itemprop={link.type === 'email' ? 'email' : 'sameAs'}
							>
								{@render contactIcon(link.type)}
								<span>{link.label || link.type}</span>
							</a>
						{/each}
					</nav>
				{/if}

				{#if profile?.summary}
					<div class="prose prose-invert prose-lg max-w-none" itemprop="description">
						{@html parseMarkdown(profile.summary)}
					</div>
				{/if}
			</div>

			<div class="md:col-span-2 flex justify-center md:justify-end">
				{#if avatarUrl}
					<img
						src={avatarUrl}
						alt={profile?.name ? $t('public.hero.profile_photo_alt', { values: { name: profile.name } }) : $t('public.hero.profile_photo_generic')}
						class="w-48 h-48 sm:w-56 sm:h-56 rounded-2xl border-4 border-white/20 shadow-xl object-cover"
					/>
				{:else if heroImageUrl}
					<img
						src={heroImageUrl}
						alt=""
						class="w-full max-w-sm rounded-2xl border-4 border-white/10 shadow-xl object-cover aspect-[4/3]"
					/>
				{:else if profile?.name}
					<div class="w-48 h-48 sm:w-56 sm:h-56 rounded-2xl bg-primary-600 flex items-center justify-center text-6xl font-bold border-4 border-white/20" role="img" aria-label={$t('public.hero.profile_initial_alt', { values: { name: profile.name } })}>
						{profile.name.charAt(0)}
					</div>
				{/if}
			</div>
		</div>
	</div>
</header>

{:else if layout === 'minimal'}
<!-- Minimal: Large typography, no image, maximum whitespace -->
<header class="grain relative bg-white dark:bg-gray-900 text-gray-900 dark:text-white" itemscope itemtype="https://schema.org/Person">
	<div class="relative max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 {heroSpacingClass}">
		{#if profile?.name}
			<h1 class="hero-name text-display-lg sm:text-display-xl font-semibold mb-6 tracking-tight leading-[1.06]" itemprop="name">
				{profile.name}
			</h1>
		{/if}

		{#if profile?.headline}
			<p class="text-2xl sm:text-3xl text-gray-500 dark:text-gray-400 mb-6 leading-relaxed" itemprop="jobTitle">
				{profile.headline}
			</p>
		{/if}

		{#if profile?.location}
			<p class="flex items-center gap-2 text-gray-500 dark:text-gray-400 mb-8" itemprop="address" itemscope itemtype="https://schema.org/PostalAddress">
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
				<span class="sr-only">{$t('public.hero.location_label')}</span>
				<span itemprop="addressLocality">{profile.location}</span>
			</p>
		{/if}

		{#if contactLinks.length > 0}
			<nav class="flex flex-wrap items-center gap-3 mb-8" aria-label={$t('public.hero.contact_links_label')}>
				{#each contactLinks as link}
					<a
						href={link.url}
						target="_blank"
						rel="noopener noreferrer"
						class="inline-flex items-center gap-2 px-4 py-2 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors text-gray-600 dark:text-gray-400"
						aria-label={$t('public.hero.opens_new_tab', { values: { label: link.label || link.type } })}
						itemprop={link.type === 'email' ? 'email' : 'sameAs'}
					>
						{@render contactIcon(link.type)}
						<span>{link.label || link.type}</span>
					</a>
				{/each}
			</nav>
		{/if}

		{#if profile?.summary}
			<div class="prose prose-lg dark:prose-invert max-w-none" itemprop="description">
				{@html parseMarkdown(profile.summary)}
			</div>
		{/if}
	</div>
</header>

{:else if layout === 'stacked'}
<!-- Stacked: Headline at top, full-width image below -->
<header class="relative" itemscope itemtype="https://schema.org/Person">
	<div class="grain {headerBgClass} {textColorClass}" style={headerBgStyle}>
		<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 {heroSpacingClass}">
			<div class="flex flex-col sm:flex-row items-center sm:items-start gap-6 sm:gap-8">
				{#if avatarUrl}
					<img
						src={avatarUrl}
						alt={profile?.name ? $t('public.hero.profile_photo_alt', { values: { name: profile.name } }) : $t('public.hero.profile_photo_generic')}
						class="w-24 h-24 rounded-full border-4 border-white/20 shadow-xl object-cover"
					/>
				{:else if profile?.name}
					<div class="w-24 h-24 rounded-full bg-primary-600 flex items-center justify-center text-3xl font-bold border-4 border-white/20" role="img" aria-label={$t('public.hero.profile_initial_alt', { values: { name: profile.name } })}>
						{profile.name.charAt(0)}
					</div>
				{/if}

				<div class="text-center sm:text-left flex-1">
					{#if profile?.name}
						<h1 class="hero-name text-3xl sm:text-4xl lg:text-5xl font-semibold mb-2 tracking-tight leading-[1.06]" itemprop="name">
							{profile.name}
						</h1>
					{/if}

					{#if profile?.headline}
						<p class="text-xl sm:text-2xl text-gray-300 mb-4 leading-relaxed" itemprop="jobTitle">
							{profile.headline}
						</p>
					{/if}

					{#if profile?.location}
						<p class="flex items-center justify-center sm:justify-start gap-2 text-gray-400" itemprop="address" itemscope itemtype="https://schema.org/PostalAddress">
							<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
							</svg>
							<span class="sr-only">{$t('public.hero.location_label')}</span>
							<span itemprop="addressLocality">{profile.location}</span>
						</p>
					{/if}
				</div>
			</div>

			{#if contactLinks.length > 0}
				<nav class="flex flex-wrap items-center justify-center sm:justify-start gap-3 mt-6" aria-label={$t('public.hero.contact_links_label')}>
					{#each contactLinks as link}
						<a
							href={link.url}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
							aria-label={$t('public.hero.opens_new_tab', { values: { label: link.label || link.type } })}
							itemprop={link.type === 'email' ? 'email' : 'sameAs'}
						>
							{@render contactIcon(link.type)}
							<span>{link.label || link.type}</span>
						</a>
					{/each}
				</nav>
			{/if}

			{#if profile?.summary}
				<div class="mt-6 prose prose-invert prose-lg max-w-none" itemprop="description">
					{@html parseMarkdown(profile.summary)}
				</div>
			{/if}
		</div>
	</div>

	{#if heroImageUrl}
		<div class="w-full">
			<img
				src={heroImageUrl}
				alt=""
				class="w-full h-64 sm:h-80 lg:h-96 object-cover"
			/>
		</div>
	{/if}
</header>

{:else}
<!-- Standard (default): Image with gradient overlay, avatar left, text right -->
<header class="grain relative {headerBgClass} {textColorClass}" style={headerBgStyle} itemscope itemtype="https://schema.org/Person">
	{#if heroImageUrl}
		<div class="absolute inset-0" aria-hidden="true">
			<img
				src={heroImageUrl}
				alt=""
				class="w-full h-full object-cover object-top opacity-30"
			/>
			<div class="absolute inset-0 bg-gradient-to-t from-gray-900 via-gray-900/80 to-transparent"></div>
		</div>
	{/if}

	<div class="relative max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 {heroSpacingClass}">
		<div class="flex flex-col sm:flex-row items-center sm:items-start gap-6 sm:gap-8">
			{#if avatarUrl}
				<img
					src={avatarUrl}
					alt={profile?.name ? $t('public.hero.profile_photo_alt', { values: { name: profile.name } }) : $t('public.hero.profile_photo_generic')}
					class="w-32 h-32 sm:w-40 sm:h-40 rounded-full border-4 border-white/20 shadow-xl object-cover"
				/>
			{:else if profile?.name}
				<div class="w-32 h-32 sm:w-40 sm:h-40 rounded-full bg-primary-600 flex items-center justify-center text-4xl font-bold border-4 border-white/20" role="img" aria-label={$t('public.hero.profile_initial_alt', { values: { name: profile.name } })}>
					{profile.name.charAt(0)}
				</div>
			{/if}

			<div class="text-center sm:text-left flex-1">
				{#if profile?.name}
					<h1 class="hero-name text-3xl sm:text-4xl lg:text-5xl font-semibold mb-2 tracking-tight leading-[1.08]" itemprop="name">
						{profile.name}
					</h1>
				{/if}

				{#if profile?.headline}
					<p class="text-xl sm:text-2xl text-gray-300 mb-4 leading-relaxed" itemprop="jobTitle">
						{profile.headline}
					</p>
				{/if}

				{#if profile?.location}
					<p class="flex items-center justify-center sm:justify-start gap-2 text-gray-400 mb-4" itemprop="address" itemscope itemtype="https://schema.org/PostalAddress">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
						</svg>
						<span class="sr-only">{$t('public.hero.location_label')}</span>
						<span itemprop="addressLocality">{profile.location}</span>
					</p>
				{/if}

				{#if contactLinks.length > 0}
					<nav class="flex flex-wrap items-center justify-center sm:justify-start gap-3" aria-label={$t('public.hero.contact_links_label')}>
						{#each contactLinks as link}
							<a
								href={link.url}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
								aria-label={$t('public.hero.opens_new_tab', { values: { label: link.label || link.type } })}
								itemprop={link.type === 'email' ? 'email' : 'sameAs'}
							>
								{@render contactIcon(link.type)}
								<span>{link.label || link.type}</span>
							</a>
						{/each}
					</nav>
				{/if}
			</div>
		</div>

		{#if profile?.summary}
			<div class="mt-8 prose prose-invert prose-lg max-w-none" itemprop="description">
				{@html parseMarkdown(profile.summary)}
			</div>
		{/if}
	</div>
</header>
{/if}

{#snippet contactIcon(type: string)}
	{#if type === 'github'}
		<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
			<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
		</svg>
	{:else if type === 'linkedin'}
		<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
			<path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433c-1.144 0-2.063-.926-2.063-2.065 0-1.138.92-2.063 2.063-2.063 1.14 0 2.064.925 2.064 2.063 0 1.139-.925 2.065-2.064 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/>
		</svg>
	{:else if type === 'email'}
		<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
		</svg>
	{:else}
		<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
		</svg>
	{/if}
{/snippet}

<style>
	/*
	  Soft Premium editorial flourish — gated so classic mode is byte-identical.
	  The `.grain` paper texture is self-gating in app.css (only paints under
	  [data-design='soft-premium']) and is applied directly in the markup.

	  Here we give the hero name a Newsreader serif-accent treatment, but ONLY
	  under soft-premium. In classic mode `.hero-name` is an inert marker class
	  that carries no styles, so the heading renders exactly as before (Lora,
	  upright, via the existing Tailwind utilities).
	*/
	:global([data-design='soft-premium']) .hero-name {
		font-family: var(--font-accent);
		font-style: italic;
		font-weight: 500;
		letter-spacing: -0.01em;
	}
</style>
