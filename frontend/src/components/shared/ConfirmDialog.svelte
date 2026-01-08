<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { confirmDialog } from '$lib/stores';
	import { icon } from '$lib/icons';

	let dialogEl: HTMLDivElement;
	let cancelBtnEl: HTMLButtonElement;
	let previousActiveElement: HTMLElement | null = null;

	$: isOpen = $confirmDialog.open;
	$: options = $confirmDialog.options;

	// Focus trap and keyboard handling
	function handleKeydown(event: KeyboardEvent) {
		if (!isOpen) return;

		if (event.key === 'Escape') {
			event.preventDefault();
			confirmDialog.respond(false);
			return;
		}

		// Focus trap
		if (event.key === 'Tab') {
			const focusableElements = dialogEl?.querySelectorAll<HTMLElement>(
				'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
			);
			
			if (!focusableElements?.length) return;

			const firstElement = focusableElements[0];
			const lastElement = focusableElements[focusableElements.length - 1];

			if (event.shiftKey && document.activeElement === firstElement) {
				event.preventDefault();
				lastElement.focus();
			} else if (!event.shiftKey && document.activeElement === lastElement) {
				event.preventDefault();
				firstElement.focus();
			}
		}
	}

	// Handle overlay click
	function handleOverlayClick(event: MouseEvent) {
		if (event.target === event.currentTarget) {
			confirmDialog.respond(false);
		}
	}

	// Lock body scroll when open
	$: if (typeof document !== 'undefined') {
		if (isOpen) {
			previousActiveElement = document.activeElement as HTMLElement;
			document.body.style.overflow = 'hidden';
			// Focus cancel button after dialog renders
			requestAnimationFrame(() => {
				cancelBtnEl?.focus();
			});
		} else {
			document.body.style.overflow = '';
			// Restore focus to previous element
			if (previousActiveElement && typeof previousActiveElement.focus === 'function') {
				previousActiveElement.focus();
			}
		}
	}

	onDestroy(() => {
		if (typeof document !== 'undefined') {
			document.body.style.overflow = '';
		}
	});
</script>

<svelte:window on:keydown={handleKeydown} />

{#if isOpen && options}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in"
		on:click={handleOverlayClick}
		on:keydown={handleKeydown}
		role="presentation"
	>
		<!-- Dialog -->
		<div
			bind:this={dialogEl}
			role="alertdialog"
			aria-modal="true"
			aria-labelledby="confirm-dialog-title"
			aria-describedby="confirm-dialog-message"
			class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-md w-full p-6 animate-fade-in"
		>
			<!-- Icon and Title -->
			<div class="flex items-start gap-4">
				{#if options.danger}
					<div class="flex-shrink-0 w-10 h-10 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center text-red-600 dark:text-red-400">
						{@html icon('warning')}
					</div>
				{:else}
					<div class="flex-shrink-0 w-10 h-10 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center text-primary-600 dark:text-primary-400">
						{@html icon('info')}
					</div>
				{/if}
				<div class="flex-1 min-w-0">
					<h2
						id="confirm-dialog-title"
						class="text-lg font-semibold text-gray-900 dark:text-white"
					>
						{options.title}
					</h2>
					<p
						id="confirm-dialog-message"
						class="mt-2 text-sm text-gray-600 dark:text-gray-300 whitespace-pre-wrap"
					>
						{options.message}
					</p>
				</div>
			</div>

			<!-- Buttons -->
			<div class="mt-6 flex justify-end gap-3">
				<button
					bind:this={cancelBtnEl}
					type="button"
					class="btn btn-secondary"
					on:click={() => confirmDialog.respond(false)}
				>
					{options.cancelText || 'Cancel'}
				</button>
				<button
					type="button"
					class="btn {options.danger ? 'btn-danger' : 'btn-primary'}"
					on:click={() => confirmDialog.respond(true)}
				>
					{options.confirmText || 'Confirm'}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	/* Ensure dialog animations respect reduced motion preference */
	@media (prefers-reduced-motion: reduce) {
		.animate-fade-in {
			animation: none;
		}
	}
</style>
