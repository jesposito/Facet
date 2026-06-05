<script lang="ts">
	/**
	 * ReorderAnnouncer - DRY screen-reader announcer for reorderable lists.
	 *
	 * Renders a single always-mounted polite `aria-live` region. Mounting it
	 * unconditionally (and only toggling the text) avoids the double/zero
	 * announcement bugs that occur when a live region is added to or removed
	 * from the DOM. Consumers grab a reference with `bind:this` and call
	 * `announce(msg)` after BOTH keyboard moves (Move up/down buttons) and drag
	 * finalize so position changes reach assistive tech regardless of input.
	 *
	 * Usage: declare `let announcer = $state<ReorderAnnouncer>();`, render
	 * `<ReorderAnnouncer bind:this={announcer} />`, then call
	 * `announcer?.announce(translatedMessage)`.
	 *
	 * The component is i18n-agnostic: pass an already-translated string so the
	 * caller controls the message and its placeholders.
	 */
	let message = $state('');

	/**
	 * Announce a reorder change. Re-announces even when the text is identical
	 * (e.g. swapping the same item back and forth) by briefly clearing the
	 * region first, which forces screen readers to re-read it.
	 */
	export function announce(msg: string) {
		if (msg === message) {
			message = '';
			queueMicrotask(() => {
				message = msg;
			});
			return;
		}
		message = msg;
	}
</script>

<div class="sr-only" role="status" aria-live="polite" aria-atomic="true">{message}</div>
