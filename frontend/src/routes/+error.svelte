<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { dev, browser } from '$app/environment';

	onMount(() => {
		console.error('[ERROR PAGE] Displayed error:', {
			status: $page.status,
			message: $page.error?.message,
			url: $page.url.href
		});
	});

	let is404 = $derived($page.status === 404);
	let is500 = $derived($page.status >= 500);

	let referrerPath = '';

	onMount(() => {
		if (!browser) return;
		try {
			const ref = document.referrer;
			if (ref) {
				const refUrl = new URL(ref);
				// Only use same-origin referrers, and avoid self-loops
				if (refUrl.origin === window.location.origin && refUrl.pathname !== window.location.pathname) {
					referrerPath = refUrl.pathname + refUrl.search;
				}
			}
		} catch {
			// ignore
		}
	});

	const goHome = () => {
		if (!browser) return;
		// If we have a usable referrer, prefer it
		if (referrerPath && referrerPath !== $page.url.pathname) {
			window.location.href = referrerPath;
		} else {
			window.location.href = '/';
		}
	};

	const goBack = () => {
		if (!browser) return;
		if (referrerPath && referrerPath !== $page.url.pathname) {
			window.location.href = referrerPath;
		} else {
			window.history.back();
		}
	};

	const hardReload = () => {
		if (!browser) return;
		window.location.reload();
	};
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
	<div class="text-center max-w-2xl">

		{#if is404}
			<!-- 404: I Didn't Build That Facet -->
			<div class="mb-8">
				<!-- SVG Illustration: Developer with incomplete diamond -->
				<svg class="w-64 h-64 mx-auto mb-6" viewBox="0 0 200 200" fill="none" xmlns="http://www.w3.org/2000/svg">
					<!-- Incomplete diamond (missing a facet) -->
					<g class="text-primary-500" stroke="currentColor" stroke-width="2" fill="none">
						<path d="M 100 30 L 150 80 L 130 150 L 100 170 L 70 150 L 50 80 Z" opacity="0.3"/>
						<line x1="100" y1="30" x2="150" y2="80"/>
						<line x1="150" y1="80" x2="130" y2="150"/>
						<line x1="130" y1="150" x2="100" y2="170"/>
						<line x1="100" y1="170" x2="70" y2="150"/>
						<line x1="70" y1="150" x2="50" y2="80"/>
						<!-- Missing facet - dashed line -->
						<line x1="50" y1="80" x2="100" y2="30" stroke-dasharray="5,5" opacity="0.3"/>
					</g>

					<!-- Stick figure developer -->
					<g class="text-gray-700 dark:text-gray-300" stroke="currentColor" stroke-width="2" fill="none">
						<!-- Head -->
						<circle cx="40" cy="120" r="8"/>
						<!-- Body -->
						<line x1="40" y1="128" x2="40" y2="155"/>
						<!-- Arms - scratching head -->
						<line x1="40" y1="135" x2="35" y2="145"/>
						<line x1="40" y1="135" x2="38" y2="115"/>
						<!-- Legs -->
						<line x1="40" y1="155" x2="33" y2="170"/>
						<line x1="40" y1="155" x2="47" y2="170"/>
					</g>

					<!-- TODO sticky note -->
					<g class="text-yellow-400" fill="currentColor" opacity="0.9">
						<rect x="145" y="40" width="45" height="35" rx="2"/>
						<text x="150" y="52" font-size="6" class="fill-gray-800" font-family="monospace">TODO:</text>
						<text x="150" y="60" font-size="6" class="fill-gray-800" font-family="monospace">Build</text>
						<text x="150" y="68" font-size="6" class="fill-gray-800" font-family="monospace">this!</text>
					</g>

					<!-- Question marks -->
					<text x="33" y="110" font-size="12" class="fill-gray-400">?</text>
				</svg>
			</div>

			<h1 class="text-5xl font-bold text-gray-900 dark:text-white mb-4">
				404: I Didn't Build That Facet
			</h1>
			<p class="text-xl text-gray-600 dark:text-gray-400 mb-8">
				Turns out I didn't cut this part of the diamond.<br/>
				Maybe it's on the roadmap... maybe I forgot.
			</p>

		{:else if is500}
			<!-- 500: I Should Have Added More Tests -->
			<div class="mb-8">
				<!-- SVG Illustration: Developer with broken diamond -->
				<svg class="w-64 h-64 mx-auto mb-6" viewBox="0 0 200 200" fill="none" xmlns="http://www.w3.org/2000/svg">
					<!-- Broken diamond pieces -->
					<g class="text-primary-500" stroke="currentColor" stroke-width="2" fill="none">
						<!-- Main broken piece -->
						<path d="M 95 40 L 130 70 L 115 120" transform="rotate(-5 110 80)"/>
						<!-- Fragment 1 -->
						<path d="M 120 130 L 135 140 L 125 155" transform="rotate(15 127 142)"/>
						<!-- Fragment 2 -->
						<path d="M 80 125 L 70 140 L 85 145" transform="rotate(-10 77 135)"/>
						<!-- Crack lines -->
						<line x1="100" y1="100" x2="110" y2="120" stroke-dasharray="3,3" class="text-red-500"/>
						<line x1="105" y1="110" x2="95" y2="125" stroke-dasharray="3,3" class="text-red-500"/>
					</g>

					<!-- Stick figure developer looking sheepish -->
					<g class="text-gray-700 dark:text-gray-300" stroke="currentColor" stroke-width="2" fill="none">
						<!-- Head -->
						<circle cx="45" cy="130" r="8"/>
						<!-- Worried face -->
						<circle cx="43" cy="128" r="1" class="fill-current"/>
						<circle cx="47" cy="128" r="1" class="fill-current"/>
						<path d="M 42 134 Q 45 132 48 134" stroke-linecap="round"/>
						<!-- Body -->
						<line x1="45" y1="138" x2="45" y2="165"/>
						<!-- Arms - holding hammer -->
						<line x1="45" y1="145" x2="55" y2="140"/>
						<rect x="54" y="136" width="8" height="4" rx="1" class="fill-gray-600"/>
						<line x1="45" y1="145" x2="35" y2="150"/>
						<!-- Legs -->
						<line x1="45" y1="165" x2="38" y2="180"/>
						<line x1="45" y1="165" x2="52" y2="180"/>
					</g>

					<!-- Thought bubble -->
					<g class="text-gray-400" fill="currentColor">
						<circle cx="20" cy="115" r="2"/>
						<circle cx="24" cy="108" r="3"/>
						<ellipse cx="30" cy="90" rx="20" ry="15" class="fill-white dark:fill-gray-800 stroke-current" stroke-width="1"/>
						<text x="18" y="93" font-size="6" class="fill-gray-700 dark:fill-gray-300" font-family="monospace">Should've</text>
						<text x="15" y="99" font-size="6" class="fill-gray-700 dark:fill-gray-300" font-family="monospace">added tests</text>
					</g>
				</svg>
			</div>

			<h1 class="text-5xl font-bold text-gray-900 dark:text-white mb-4">
				500: I Should Have Added More Tests
			</h1>
			<p class="text-xl text-gray-600 dark:text-gray-400 mb-8">
				Something I wrote is broken. Shocking, I know.<br/>
				Check back in 5 minutes after I frantically Google this.
			</p>

		{:else}
			<!-- Other errors - generic but still themed -->
			<div class="mb-8">
				<div class="text-6xl mb-4">💎</div>
			</div>
			<h1 class="text-5xl font-bold text-gray-900 dark:text-white mb-4">
				{$page.status}: Something's Off
			</h1>
			<p class="text-xl text-gray-600 dark:text-gray-400 mb-8">
				{$page.error?.message || "This facet isn't quite right."}
			</p>
		{/if}

		<!-- Debug info for development only -->
		{#if dev}
			<div class="text-left text-sm bg-gray-100 dark:bg-gray-800 p-4 rounded-lg mb-8 max-w-md mx-auto">
				<p class="text-gray-500 dark:text-gray-400 mb-2 font-mono">Debug info:</p>
				<ul class="text-gray-600 dark:text-gray-400 space-y-1 font-mono text-xs">
					<li><strong>URL:</strong> {$page.url.pathname}</li>
					<li><strong>Status:</strong> {$page.status}</li>
					<li><strong>Error:</strong> {$page.error?.message || 'None'}</li>
				</ul>
			</div>
		{/if}

		<!-- Action buttons -->
		<div class="flex flex-wrap gap-4 justify-center">
			<button
				type="button"
				onclick={goHome}
				data-testid="go-home-button"
				class="inline-flex items-center gap-2 px-6 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
			>
				<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
				</svg>
				{is404 ? 'Back to What Actually Works' : 'Go Home'}
			</button>

			{#if is404}
				<button
					type="button"
					onclick={goBack}
					class="inline-flex items-center gap-2 px-6 py-3 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
					</svg>
					Go Back
				</button>
			{:else}
				<button
					type="button"
					onclick={hardReload}
					class="inline-flex items-center gap-2 px-6 py-3 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
					</svg>
					{is500 ? 'Refresh and Pray' : 'Try Again'}
				</button>
			{/if}
		</div>

		<!-- Subtle signature -->
		<p class="mt-12 text-sm text-gray-400 dark:text-gray-500 italic">
			— Error pages crafted with self-awareness
		</p>
	</div>
</div>
