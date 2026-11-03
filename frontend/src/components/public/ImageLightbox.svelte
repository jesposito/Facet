<script lang="ts">
	import { t } from 'svelte-i18n';
	import { focusTrap } from '$lib/actions/focusTrap';

	export interface LightboxImage {
		url?: string;
		title?: string;
		alt_text?: string;
		description?: string;
	}

	interface Props {
		images: LightboxImage[];
		index?: number;
		onClose: () => void;
	}

	let { images, index = $bindable(0), onClose }: Props = $props();

	function nextImage() {
		if (images.length > 0) index = (index + 1) % images.length;
	}

	function prevImage() {
		if (images.length > 0) index = (index - 1 + images.length) % images.length;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowRight') nextImage();
		if (event.key === 'ArrowLeft') prevImage();
	}

	function handleBackdropClick(event: MouseEvent) {
		if (event.target === event.currentTarget) onClose();
	}

	function handleBackdropKeydown(event: KeyboardEvent) {
		if (event.target === event.currentTarget && (event.key === 'Enter' || event.key === ' ')) {
			event.preventDefault();
			onClose();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if images[index]}
	<div
		class="fixed inset-0 z-50 bg-black/90 flex items-center justify-center"
		role="dialog"
		aria-modal="true"
		aria-label={$t('public.lightbox.label')}
		tabindex="-1"
		onclick={handleBackdropClick}
		onkeydown={handleBackdropKeydown}
		use:focusTrap={{ onEscape: onClose }}
	>
		<button
			type="button"
			onclick={onClose}
			class="absolute top-4 right-4 p-2 text-white/70 hover:text-white transition-colors"
			aria-label={$t('public.lightbox.close')}
		>
			<svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>

		{#if images.length > 1}
			<button
				type="button"
				onclick={prevImage}
				class="absolute left-4 p-2 text-white/70 hover:text-white transition-colors"
				aria-label={$t('public.lightbox.previous_image')}
			>
				<svg
					class="w-10 h-10"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
				</svg>
			</button>
			<button
				type="button"
				onclick={nextImage}
				class="absolute right-4 p-2 text-white/70 hover:text-white transition-colors"
				aria-label={$t('public.lightbox.next_image')}
			>
				<svg
					class="w-10 h-10"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		{/if}

		<img
			src={images[index].url || ''}
			alt={images[index].alt_text || images[index].title || ''}
			class="max-h-[90vh] max-w-[90vw] object-contain"
		/>

		<div class="absolute bottom-4 left-1/2 -translate-x-1/2 text-center text-white">
			{#if images[index].title}
				<p class="text-lg font-medium mb-1">{images[index].title}</p>
			{/if}
			{#if images[index].description}
				<p class="text-sm text-white/80 mb-1">{images[index].description}</p>
			{/if}
			{#if images.length > 1}
				<p class="text-sm text-white/70">{index + 1} / {images.length}</p>
			{/if}
		</div>
	</div>
{/if}
