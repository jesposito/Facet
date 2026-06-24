<!--
  Live site preview iframe.

  Renders the public profile homepage in a sandboxed iframe with a
  debounced reload on PocketBase realtime updates to the profile record,
  so the operator watches the public site update as they edit.

  Behaviour:
  - Realtime subscription is scoped to the specific profile record id
    (not the wildcard `*` topic), keeping the subscription tight + cheap.
  - Kill-switch via `VITE_DISABLE_LIVE_PREVIEW=1` at build time disables
    the feature without touching source. Mounting is skipped entirely
    when set.
  - Iframe error events announce failure via the same polite live region
    used for the ready announcement.
  - If we can't resolve the profile record id (e.g. pre-seed), the
    component still renders the iframe but skips the realtime
    subscription — preview is static until manual reload, never crashes.

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
	import { browser } from '$app/environment';

	// Reload debounce: 400ms after PB realtime fires. Tuned so a fast typer
	// saving every keystroke (autosave debounces internally) gets exactly
	// one preview reload per coherent edit, not per keystroke.
	const RELOAD_DEBOUNCE_MS = 400;

	// Build-time kill switch. Set VITE_DISABLE_LIVE_PREVIEW=1 in .env to
	// fully disable the feature without touching code. Default empty →
	// enabled (subject to userPreferences toggle).
	const KILL_SWITCH =
		typeof import.meta.env !== 'undefined' &&
		(import.meta.env.VITE_DISABLE_LIVE_PREVIEW === '1' ||
			import.meta.env.VITE_DISABLE_LIVE_PREVIEW === 'true');

	let iframeEl: HTMLIFrameElement | undefined = $state();
	let busy = $state(false);
	let firstLoadAnnounced = $state(false);
	let errorAnnounced = $state(false);
	let liveStatus = $state('');

	// Bust the iframe cache on reload — appending a counter to the src
	// forces a fresh fetch instead of relying on the browser's HTTP cache.
	// The `preview=1` marker is ignored server-side but kept for parity.
	let reloadCounter = $state(0);
	let src = $derived(`/?preview=1&_=${reloadCounter}`);

	let reloadTimer: ReturnType<typeof setTimeout> | null = null;
	let unsubscribeFn: (() => void) | null = null;

	function scheduleReload() {
		if (reloadTimer) clearTimeout(reloadTimer);
		busy = true;
		reloadTimer = setTimeout(() => {
			reloadCounter += 1;
			reloadTimer = null;
		}, RELOAD_DEBOUNCE_MS);
	}

	function announce(message: string) {
		liveStatus = message;
		setTimeout(() => {
			liveStatus = '';
		}, 2000);
	}

	function handleIframeLoad() {
		busy = false;
		// Clear the error latch on a successful load so a later break→recover→break
		// cycle re-announces the failure (WCAG 4.1.3) instead of going silent.
		errorAnnounced = false;
		if (!firstLoadAnnounced) {
			// Throttled status: only the first successful load. Reloads
			// after the user just typed don't need an announcement.
			announce($t('admin.site_preview.ready_announce'));
			firstLoadAnnounced = true;
		}
	}

	function handleIframeError() {
		busy = false;
		if (!errorAnnounced) {
			// One-shot error announcement so AT users hear when the preview
			// breaks (e.g., CSP block, network failure, backend 503).
			announce($t('admin.site_preview.error_announce'));
			errorAnnounced = true;
		}
	}

	onMount(async () => {
		if (!browser || KILL_SWITCH) return;

		// Narrow the realtime subscription to the single profile record.
		// Subscribing to the specific record id (rather than the wildcard
		// `*` topic) keeps the subscription tight + avoids fan-out from
		// unrelated record-level writes.
		//
		// `requestKey: null` opts out of the PocketBase SDK's per-collection
		// auto-cancellation. Other admin pages also fetch the profile
		// collection at mount time, and the SDK cancels earlier in-flight
		// calls to the same collection when a later one fires. Without
		// `requestKey: null` our getList could lose the race and the
		// subscribe path would never establish. One-shot retry on
		// cancellation as belt-and-suspenders.
		async function subscribeWithRetry(attempt = 0): Promise<void> {
			try {
				const records = await pb.collection('profile').getList(1, 1, { requestKey: null });
				const profileId = records.items[0]?.id;
				if (!profileId) {
					// No profile record yet (rare — pre-seed). Render the
					// iframe statically; operator can refresh manually.
					return;
				}
				const unsub = await pb.collection('profile').subscribe(profileId, () => {
					scheduleReload();
				});
				unsubscribeFn = unsub;
			} catch (err) {
				const msg = String(err);
				const isAutoCancel = msg.includes('autocancelled') || msg.includes('aborted');
				if (isAutoCancel && attempt < 2) {
					// One backoff then retry. Auto-cancel happens when another
					// page's same-collection fetch raced ahead of ours.
					await new Promise((r) => setTimeout(r, 250 * (attempt + 1)));
					return subscribeWithRetry(attempt + 1);
				}
				console.warn('[live-preview] realtime subscribe failed:', err);
			}
		}
		await subscribeWithRetry();
	});

	onDestroy(() => {
		if (reloadTimer) clearTimeout(reloadTimer);
		if (unsubscribeFn) {
			try {
				unsubscribeFn();
			} catch (err) {
				console.warn('[live-preview] unsubscribe failed:', err);
			}
		}
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
