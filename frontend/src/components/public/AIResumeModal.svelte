<script lang="ts">
	import { t } from 'svelte-i18n';
	import type { AIResumeController } from '$lib/ai-resume/controller.svelte';

	/**
	 * AI Resume generation modal. Renders the form (or success download CTA)
	 * and delegates state to the controller passed in by the page.
	 *
	 * Built on the native `<dialog>` element:
	 * - Browser owns the focus trap, Escape handling, inert backdrop, and the
	 *   top-layer stacking. ~60% less hand-rolled lifecycle than a
	 *   `role="dialog"` implementation, with the same a11y contract.
	 * - `<dialog>.showModal()` already sets `role="dialog"` + `aria-modal="true"`.
	 * - `aria-labelledby` points at the visible heading so SRs announce the
	 *   dialog name on open.
	 * - First focus lands on the dialog element (browser default for
	 *   `showModal`), then Tab walks the focusable children.
	 * - We track the previously-focused element and restore it on close so
	 *   keyboard users return to the trigger button, not document.body.
	 *
	 * SvelteKit SSR safety: `bind:this` and `$effect` only run client-side,
	 * so `showModal()` won't be invoked during render. The closed `<dialog>`
	 * is inert visually by default — safe to ship in the SSR HTML.
	 */
	interface Props {
		/** Controller returned by createAIResumeController — owns state + actions. */
		controller: AIResumeController;
	}
	let { controller }: Props = $props();

	let dialogEl: HTMLDialogElement | undefined = $state();
	let panelEl: HTMLDivElement | undefined = $state();

	// Track whether the pointer-down originated inside the panel, so a
	// drag-select that releases on the backdrop doesn't fire a dismiss.
	let pointerDownInsidePanel = $state(false);

	let wasOpen = false;
	$effect(() => {
		if (!dialogEl) return;
		const isOpen = controller.open;
		if (isOpen && !dialogEl.open) {
			dialogEl.showModal();
		} else if (!isOpen && dialogEl.open) {
			dialogEl.close();
			// Native <dialog> close returns focus to document.body; we want it
			// back on the trigger button for keyboard parity. The trigger was
			// captured by the controller at show()-time so it survives any
			// synchronous unmount of a parent popover that fired alongside.
			const triggerEl = controller.trigger;
			if (
				triggerEl &&
				typeof triggerEl.focus === 'function' &&
				document.contains(triggerEl)
			) {
				triggerEl.focus();
			}
		}
		wasOpen = isOpen;
	});

	// Browser fires `cancel` for Escape (and the native close button if any).
	// We preventDefault so the parent's `open` state stays the authoritative
	// source — closing happens via controller.dismiss(), not the dialog's
	// internal toggle.
	function handleCancel(event: Event) {
		event.preventDefault();
		controller.dismiss();
	}

	function handlePointerDown(event: PointerEvent) {
		pointerDownInsidePanel = !!panelEl && panelEl.contains(event.target as Node);
	}

	function handleClick(event: MouseEvent) {
		// Clicks bubble up from the panel content; only treat clicks whose
		// target is the dialog element itself as backdrop clicks.
		if (event.target !== dialogEl) return;
		if (pointerDownInsidePanel) {
			// The drag started inside the panel — don't dismiss.
			pointerDownInsidePanel = false;
			return;
		}
		controller.dismiss();
	}
</script>

<dialog
	bind:this={dialogEl}
	aria-labelledby="ai-resume-modal-title"
	aria-describedby={controller.downloadUrl ? undefined : 'ai-resume-modal-desc'}
	onclick={handleClick}
	onpointerdown={handlePointerDown}
	oncancel={handleCancel}
	class="w-full max-w-md rounded-lg bg-white p-0 text-stone-900 shadow-xl backdrop:bg-black/50 dark:bg-stone-800 dark:text-stone-100 focus-visible:outline-none print:hidden"
>
	<!-- Panel wraps body content so backdrop-vs-panel click detection works. -->
	<div bind:this={panelEl}>
		<div class="p-4 border-b border-stone-200 dark:border-stone-700">
			<h2
				id="ai-resume-modal-title"
				class="text-lg font-semibold text-stone-900 dark:text-white"
			>
				{$t('public.resume_modal.title')}
			</h2>
			<p id="ai-resume-modal-desc" class="text-sm text-stone-500 dark:text-stone-400 mt-1">
				{$t('public.resume_modal.description')}
			</p>
		</div>

		<div class="p-4 space-y-4">
			{#if controller.downloadUrl}
				<!-- aria-live so a screen reader user knows generation completed
					 even if focus stayed on the now-disabled Generate button before
					 the download CTA mounted. -->
				<div
					role="status"
					aria-live="polite"
					class="bg-green-50 dark:bg-green-900/30 border border-green-200 dark:border-green-800 rounded-lg p-4 text-center"
				>
					<p class="text-green-700 dark:text-green-300 mb-3">
						{$t('public.resume_modal.success')}
					</p>
					<a href={controller.downloadUrl} download class="btn btn-primary">
						{$t('public.resume_modal.download')}
					</a>
				</div>
			{:else}
				<div>
					<label
						for="ai-resume-format"
						class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1"
					>
						{$t('public.resume_modal.format')}
					</label>
					<select id="ai-resume-format" bind:value={controller.config.format} class="input">
						<option value="pdf">{$t('public.resume_modal.format_pdf')}</option>
						<option value="docx">{$t('public.resume_modal.format_word')}</option>
					</select>
				</div>

				<div>
					<label
						for="ai-resume-style"
						class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1"
					>
						{$t('public.resume_modal.style')}
					</label>
					<select id="ai-resume-style" bind:value={controller.config.style} class="input">
						<option value="chronological">{$t('public.resume_modal.style_chronological')}</option>
						<option value="functional">{$t('public.resume_modal.style_functional')}</option>
						<option value="hybrid">{$t('public.resume_modal.style_hybrid')}</option>
					</select>
				</div>

				<div>
					<label
						for="ai-resume-length"
						class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1"
					>
						{$t('public.resume_modal.length')}
					</label>
					<select id="ai-resume-length" bind:value={controller.config.length} class="input">
						<option value="one-page">{$t('public.resume_modal.length_one')}</option>
						<option value="two-page">{$t('public.resume_modal.length_two')}</option>
						<option value="full">{$t('public.resume_modal.length_full')}</option>
					</select>
				</div>
			{/if}
		</div>

		<div
			class="p-4 border-t border-stone-200 dark:border-stone-700 flex justify-end gap-2"
		>
			<button type="button" class="btn btn-ghost" onclick={() => controller.dismiss()}>
				{controller.downloadUrl
					? $t('public.resume_modal.close')
					: $t('public.resume_modal.cancel')}
			</button>
			{#if !controller.downloadUrl}
				<button
					type="button"
					class="btn btn-primary"
					onclick={() => controller.generate()}
					disabled={controller.generating}
					aria-busy={controller.generating}
				>
					{#if controller.generating}
						<svg
							class="animate-spin -ml-1 mr-2 h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
						{$t('public.resume_modal.generating')}
					{:else}
						{$t('public.resume_modal.generate')}
					{/if}
				</button>
			{/if}
		</div>
	</div>
</dialog>
