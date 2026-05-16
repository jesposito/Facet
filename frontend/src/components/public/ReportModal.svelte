<script lang="ts">
	import { t } from 'svelte-i18n';
	import { tick } from 'svelte';
	import { toasts } from '$lib/stores';

	interface Props {
		commentId: string;
		open: boolean;
		onClose: () => void;
	}

	let { commentId, open, onClose }: Props = $props();

	type ReportReason = 'spam' | 'harassment' | 'off_topic' | 'other';

	let reason = $state<ReportReason>('spam');
	let detail = $state('');
	let submitting = $state(false);
	let submitError = $state('');

	let dialogEl: HTMLDivElement | undefined = $state();
	let firstFocusable: HTMLElement | undefined = $state();
	let previouslyFocused: HTMLElement | null = null;

	const titleId = `report-modal-title-${Math.random().toString(36).slice(2, 9)}`;

	// React to `open` flips so a re-opened modal restores focus correctly.
	// `onMount` only fires once per component lifetime, so a parent that
	// re-toggles `open` would lose focus restoration on subsequent opens.
	$effect(() => {
		if (open) {
			previouslyFocused = (document.activeElement as HTMLElement) ?? null;
			tick().then(() => {
				firstFocusable?.focus();
			});
		} else if (previouslyFocused) {
			// Defer so the dialog DOM unmounts first.
			const target = previouslyFocused;
			previouslyFocused = null;
			tick().then(() => {
				if (target && typeof target.focus === 'function' && document.contains(target)) {
					target.focus();
				}
			});
		}
	});

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			onClose();
			return;
		}
		if (e.key === 'Tab' && dialogEl) {
			// Focus trap
			const focusables = dialogEl.querySelectorAll<HTMLElement>(
				'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
			);
			if (focusables.length === 0) return;
			const first = focusables[0];
			const last = focusables[focusables.length - 1];
			const active = document.activeElement as HTMLElement | null;
			if (e.shiftKey) {
				if (active === first || !dialogEl.contains(active)) {
					e.preventDefault();
					last.focus();
				}
			} else {
				if (active === last) {
					e.preventDefault();
					first.focus();
				}
			}
		}
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (submitting) return;

		submitError = '';
		submitting = true;
		try {
			const resp = await fetch(`/api/comments/${commentId}/report`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					reason,
					detail: detail.trim() || undefined
				})
			});

			if (!resp.ok) {
				const err = await resp.json().catch(() => ({}));
				submitError = err.error || $t('comments.report');
				return;
			}

			toasts.add('success', $t('comments.report_submitted'));
			reason = 'spam';
			detail = '';
			onClose();
		} catch (err) {
			submitError = err instanceof Error ? err.message : $t('comments.report');
			toasts.add('error', submitError);
		} finally {
			submitting = false;
		}
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions, a11y_click_events_have_key_events -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
		onkeydown={handleKeydown}
		onclick={handleBackdropClick}
		role="presentation"
	>
		<div
			bind:this={dialogEl}
			class="bg-white dark:bg-stone-800 rounded-lg shadow-xl border border-stone-200 dark:border-stone-700 w-full max-w-md mx-4 p-6"
			role="dialog"
			aria-modal="true"
			aria-labelledby={titleId}
		>
			<div class="flex items-center justify-between mb-4">
				<h3 id={titleId} class="text-lg font-semibold text-stone-900 dark:text-white">
					{$t('comments.report_title')}
				</h3>
				<button
					bind:this={firstFocusable}
					type="button"
					onclick={onClose}
					class="text-stone-500 hover:text-stone-600 dark:hover:text-stone-300 transition-colors"
					aria-label={$t('comments.cancel')}
				>
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<form onsubmit={handleSubmit} class="space-y-4" novalidate>
				<fieldset>
					<legend class="text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">
						{$t('comments.report_reason')}
					</legend>
					<div class="space-y-2">
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="radio" bind:group={reason} value="spam" class="text-primary-600 focus:ring-primary-500" />
							<span class="text-sm text-stone-700 dark:text-stone-300">{$t('comments.report_spam')}</span>
						</label>
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="radio" bind:group={reason} value="harassment" class="text-primary-600 focus:ring-primary-500" />
							<span class="text-sm text-stone-700 dark:text-stone-300">{$t('comments.report_harassment')}</span>
						</label>
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="radio" bind:group={reason} value="off_topic" class="text-primary-600 focus:ring-primary-500" />
							<span class="text-sm text-stone-700 dark:text-stone-300">{$t('comments.report_off_topic')}</span>
						</label>
						<label class="flex items-center gap-2 cursor-pointer">
							<input type="radio" bind:group={reason} value="other" class="text-primary-600 focus:ring-primary-500" />
							<span class="text-sm text-stone-700 dark:text-stone-300">{$t('comments.report_other')}</span>
						</label>
					</div>
				</fieldset>

				<div>
					<label for="report-detail" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
						{$t('comments.report_detail')}
					</label>
					<textarea
						id="report-detail"
						bind:value={detail}
						rows={3}
						maxlength="1000"
						class="w-full px-3 py-2 text-sm border border-stone-300 dark:border-stone-600 rounded-lg bg-white dark:bg-stone-700 text-stone-900 dark:text-white placeholder-stone-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-y"
					></textarea>
				</div>

				{#if submitError}
					<p class="text-sm text-red-600 dark:text-red-400" role="alert">{submitError}</p>
				{/if}

				<div class="flex items-center justify-end gap-3">
					<button
						type="button"
						onclick={onClose}
						class="px-4 py-2 text-sm font-medium text-stone-600 dark:text-stone-400 hover:text-stone-800 dark:hover:text-stone-200 transition-colors"
					>
						{$t('comments.cancel')}
					</button>
					<button
						type="submit"
						disabled={submitting}
						aria-busy={submitting}
						class="px-4 py-2 text-sm font-medium bg-red-600 hover:bg-red-700 disabled:bg-red-400 text-white rounded-lg transition-colors disabled:cursor-not-allowed"
					>
						{#if submitting}
							<svg class="animate-spin inline-block h-4 w-4 me-1.5" fill="none" viewBox="0 0 24 24" aria-hidden="true">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
						{/if}
						{$t('comments.report')}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
