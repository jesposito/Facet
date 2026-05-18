<script lang="ts">
	import { t } from 'svelte-i18n';
	import { isFilename } from '$lib/utils';

	interface LightboxImage {
		url?: string;
		title?: string;
		alt_text?: string;
	}

	interface Props {
		/** Two-way bound open state. Set to false to close. */
		open: boolean;
		/** Images to display. Lightbox renders nothing when empty. */
		images: LightboxImage[];
		/** Index to show when opening; reset each time `open` transitions to true. */
		startIndex?: number;
	}

	let { open = $bindable(), images, startIndex = 0 }: Props = $props();

	let index = $state(startIndex);
	let dialogEl: HTMLDivElement | undefined = $state();
	let closeBtnEl: HTMLButtonElement | undefined = $state();
	let prevFocus: HTMLElement | null = null;
	const titleId = `lightbox-title-${Math.random().toString(36).slice(2, 10)}`;

	$effect(() => {
		if (!open) return;
		prevFocus = (document.activeElement as HTMLElement | null) ?? null;
		index = startIndex;
		queueMicrotask(() => closeBtnEl?.focus());
		const prevOverflow = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			document.body.style.overflow = prevOverflow;
			// Restore focus to the element that opened the lightbox (WCAG 2.4.3)
			if (prevFocus && typeof prevFocus.focus === 'function') {
				prevFocus.focus();
			}
			prevFocus = null;
		};
	});

	function close() {
		open = false;
	}

	function next() {
		if (images.length > 0) index = (index + 1) % images.length;
	}

	function prev() {
		if (images.length > 0) index = (index - 1 + images.length) % images.length;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!open) return;
		if (e.key === 'Escape') {
			e.preventDefault();
			close();
		} else if (e.key === 'ArrowRight') {
			e.preventDefault();
			next();
		} else if (e.key === 'ArrowLeft') {
			e.preventDefault();
			prev();
		} else if (e.key === 'Tab') {
			// Focus trap inside the dialog
			const focusables = dialogEl?.querySelectorAll<HTMLElement>('button:not([disabled])');
			if (!focusables || focusables.length === 0) return;
			const first = focusables[0];
			const last = focusables[focusables.length - 1];
			const active = document.activeElement;
			if (e.shiftKey && active === first) {
				e.preventDefault();
				last.focus();
			} else if (!e.shiftKey && active === last) {
				e.preventDefault();
				first.focus();
			}
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) close();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open && images[index]}
	{@const current = images[index]}
	{@const showTitle = !!current.title && !isFilename(current.title)}
	<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
	<div
		bind:this={dialogEl}
		role="dialog"
		aria-modal="true"
		aria-label={$t('public.lightbox.title')}
		aria-labelledby={showTitle ? titleId : undefined}
		class="fixed inset-0 z-50 bg-black/90 flex items-center justify-center"
		onclick={handleBackdropClick}
		oncontextmenu={(e) => e.preventDefault()}
	>
		<!-- Close button -->
		<button
			bind:this={closeBtnEl}
			type="button"
			onclick={close}
			class="absolute top-4 right-4 min-w-11 min-h-11 inline-flex items-center justify-center p-2 text-white/70 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-white rounded-md transition-colors"
			aria-label={$t('public.lightbox.close')}
		>
			<svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>

		{#if images.length > 1}
			<button
				type="button"
				onclick={(e) => { e.stopPropagation(); prev(); }}
				class="absolute left-4 min-w-11 min-h-11 inline-flex items-center justify-center p-2 text-white/70 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-white rounded-md transition-colors"
				aria-label={$t('public.lightbox.previous_image')}
			>
				<svg class="w-10 h-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
				</svg>
			</button>

			<button
				type="button"
				onclick={(e) => { e.stopPropagation(); next(); }}
				class="absolute right-4 min-w-11 min-h-11 inline-flex items-center justify-center p-2 text-white/70 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-white rounded-md transition-colors"
				aria-label={$t('public.lightbox.next_image')}
			>
				<svg class="w-10 h-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		{/if}

		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
		<img
			src={current.url || ''}
			alt={current.title || current.alt_text || ''}
			class="max-h-[90vh] max-w-[90vw] object-contain"
			onclick={(e) => e.stopPropagation()}
			oncontextmenu={(e) => e.preventDefault()}
		/>

		<div class="absolute bottom-4 left-1/2 -translate-x-1/2 text-center text-white">
			{#if showTitle}
				<p id={titleId} class="text-lg font-medium mb-1">{current.title}</p>
			{/if}
			{#if images.length > 1}
				<p class="text-sm text-white/70">{index + 1} / {images.length}</p>
			{/if}
		</div>
	</div>
{/if}
