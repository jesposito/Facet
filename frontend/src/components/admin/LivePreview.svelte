<!--
  Live site preview iframe.

  Renders the public profile homepage in a sandboxed iframe with a
  debounced reload on PocketBase realtime updates to public content,
  site settings, and navigation.

  Behaviour:
  - Public content collections use wildcard topics. Profile wildcard
    subscriptions also cover a profile created after the preview mounts.
  - Site settings use an owner-only topic with no record data, because the
    settings record stays private.
  - Kill-switch via `VITE_DISABLE_LIVE_PREVIEW=1` at build time disables
    the feature without touching source. Mounting is skipped entirely
    when set.
  - Iframe error events announce failure via the same polite live region
    used for the ready announcement.
  - Subscription failures leave the iframe available for manual reload.

  A11y design:
  - iframe is tabindex=-1 (sandbox restricts interaction; sealing focus
    out matches actual behavior and avoids a focus trap inside).
  - Single sr-only note explains the preview is read-only.
  - aria-busy on container during reload — no aria-live wrapper (would
    spam SR users on every edit).
  - One-time "ready" announcement on first load; one-time "unavailable"
    on iframe error, both routed through a polite role="status" region.

  Mounted only when the layout decides — see +layout.svelte. Default-OFF
  means existing installs see zero behaviour change until the operator
  opts in via the header toggle.
-->
<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { t } from 'svelte-i18n';
	import { pb } from '$lib/pocketbase';
	import { browser } from '$app/env';
	import type { UnsubscribeFunc } from 'pocketbase';

	const RELOAD_DEBOUNCE_MS = 400;
	const SITE_CONFIG_TOPIC = 'public_site_config';
	const CONTENT_COLLECTIONS = [
		'profile',
		'experience',
		'projects',
		'education',
		'certifications',
		'awards',
		'skills',
		'posts',
		'talks',
		'courses',
		'testimonials',
		'contact_methods',
		'custom_content'
	];

	const KILL_SWITCH =
		typeof import.meta.env !== 'undefined' &&
		(import.meta.env.VITE_DISABLE_LIVE_PREVIEW === '1' ||
			import.meta.env.VITE_DISABLE_LIVE_PREVIEW === 'true');

	let iframeEl: HTMLIFrameElement | undefined = $state();
	let busy = $state(false);
	let firstLoadAnnounced = $state(false);
	let errorAnnounced = $state(false);
	let liveStatus = $state('');
	let reloadCounter = $state(0);
	let src = $derived(`/?preview=1&_=${reloadCounter}`);

	let reloadTimer: ReturnType<typeof setTimeout> | null = null;
	let announcementTimer: ReturnType<typeof setTimeout> | null = null;
	let unsubscribeFns: UnsubscribeFunc[] = [];
	let destroyed = false;

	function scheduleReload() {
		if (destroyed) return;
		if (reloadTimer) clearTimeout(reloadTimer);
		busy = true;
		reloadTimer = setTimeout(() => {
			reloadTimer = null;
			if (!destroyed) reloadCounter += 1;
		}, RELOAD_DEBOUNCE_MS);
	}

	function announce(message: string) {
		if (destroyed) return;
		liveStatus = message;
		if (announcementTimer) clearTimeout(announcementTimer);
		announcementTimer = setTimeout(() => {
			announcementTimer = null;
			if (!destroyed) liveStatus = '';
		}, 2000);
	}

	function handleIframeLoad() {
		if (destroyed) return;
		busy = false;
		errorAnnounced = false;
		if (!firstLoadAnnounced) {
			announce($t('admin.site_preview.ready_announce'));
			firstLoadAnnounced = true;
		}
	}

	function handleIframeError() {
		if (destroyed) return;
		busy = false;
		if (!errorAnnounced) {
			announce($t('admin.site_preview.error_announce'));
			errorAnnounced = true;
		}
	}

	function unsubscribe(fn: UnsubscribeFunc) {
		void fn().catch((err) => {
			console.warn('[live-preview] unsubscribe failed:', err);
		});
	}

	async function subscribe(name: string, start: () => Promise<UnsubscribeFunc>) {
		try {
			const unsub = await start();
			if (destroyed) {
				unsubscribe(unsub);
				return;
			}
			unsubscribeFns.push(unsub);
		} catch (err) {
			if (!destroyed) console.warn(`[live-preview] ${name} subscribe failed:`, err);
		}
	}

	onMount(async () => {
		if (!browser || KILL_SWITCH) return;
		await Promise.all([
			...CONTENT_COLLECTIONS.map((collection) =>
				subscribe(collection, () => pb.collection(collection).subscribe('*', scheduleReload))
			),
			subscribe(SITE_CONFIG_TOPIC, () => pb.realtime.subscribe(SITE_CONFIG_TOPIC, scheduleReload))
		]);
	});

	onDestroy(() => {
		destroyed = true;
		if (reloadTimer) clearTimeout(reloadTimer);
		if (announcementTimer) clearTimeout(announcementTimer);
		for (const unsub of unsubscribeFns.splice(0)) unsubscribe(unsub);
	});
</script>

{#if !KILL_SWITCH}
	<aside class="live-preview-column min-w-0" aria-labelledby="site-preview-heading">
		<h2 id="site-preview-heading" class="sr-only">{$t('admin.site_preview.heading')}</h2>
		<span id="site-preview-note" class="sr-only">{$t('admin.site_preview.readonly_note')}</span>

		<div
			class="sticky top-20 h-[calc(100vh-6rem)] rounded-xl overflow-hidden border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-sm"
			aria-busy={busy}
		>
			<iframe
				bind:this={iframeEl}
				title={$t('admin.site_preview.iframe_title')}
				aria-describedby="site-preview-note"
				{src}
				onload={handleIframeLoad}
				onerror={handleIframeError}
				loading="lazy"
				sandbox="allow-same-origin allow-scripts"
				referrerpolicy="same-origin"
				tabindex="-1"
				class="w-full h-full border-0 bg-white dark:bg-gray-800"
			></iframe>
		</div>

		<!-- Polite, throttled status for screen readers. Fires once on first
		     load and once on first error. -->
		<div class="sr-only" role="status" aria-live="polite">{liveStatus}</div>
	</aside>
{/if}

<style>
	/* Opacity-only transition: width/transform shifts cause layout reflow
	   that can trigger vestibular issues. */
	.live-preview-column {
		transition: opacity 150ms ease;
	}
	@media (prefers-reduced-motion: reduce) {
		.live-preview-column {
			transition: none;
		}
	}
</style>
