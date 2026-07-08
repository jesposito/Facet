<script lang="ts">
	import { t } from 'svelte-i18n';
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';
	import { toasts } from '$lib/stores';

	interface Props {
		contentType: string;
		contentId: string;
		parentId?: string;
		onSubmit?: () => void;
		onCancel?: () => void;
	}

	let { contentType, contentId, parentId, onSubmit, onCancel }: Props = $props();

	const MAX_BODY_LENGTH = 5000;
	const STORAGE_KEY_NAME = 'facet_comment_author_name';
	const STORAGE_KEY_EMAIL = 'facet_comment_author_email';

	let authorName = $state('');
	let authorEmail = $state('');
	let body = $state('');
	let submitting = $state(false);
	let submitError = $state('');

	let charsRemaining = $derived(MAX_BODY_LENGTH - body.length);
	let suffix = $derived(parentId ?? '');

	onMount(() => {
		if (!browser) return;
		authorName = localStorage.getItem(STORAGE_KEY_NAME) || '';
		authorEmail = localStorage.getItem(STORAGE_KEY_EMAIL) || '';
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (submitting) return;

		submitError = '';

		if (!authorName.trim()) {
			submitError = $t('comments.name') + ' *';
			return;
		}
		if (!body.trim()) {
			submitError = $t('comments.body') + ' *';
			return;
		}

		submitting = true;
		try {
			if (browser) {
				localStorage.setItem(STORAGE_KEY_NAME, authorName.trim());
				if (authorEmail.trim()) {
					localStorage.setItem(STORAGE_KEY_EMAIL, authorEmail.trim());
				}
			}

			const payload: Record<string, string> = {
				author_name: authorName.trim(),
				body: body.trim()
			};
			if (authorEmail.trim()) {
				payload.author_email = authorEmail.trim();
			}
			if (parentId) {
				payload.parent_id = parentId;
			}

			const resp = await fetch(`/api/comments/${contentType}/${contentId}`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});

			if (!resp.ok) {
				const err = await resp.json().catch(() => ({}));
				submitError = err.error || $t('comments.submit');
				return;
			}

			const data = await resp.json().catch(() => ({}));

			if (data.status === 'pending' || data.awaiting_moderation) {
				toasts.add('success', $t('comments.awaiting'));
			} else {
				toasts.add('success', $t('comments.submitted'));
			}

			body = '';
			onSubmit?.();
		} catch (err) {
			submitError = err instanceof Error ? err.message : $t('comments.submit');
			toasts.add('error', submitError);
		} finally {
			submitting = false;
		}
	}
</script>

<form
	onsubmit={handleSubmit}
	class="rounded-lg border border-stone-200 dark:border-stone-700 bg-white dark:bg-stone-800 p-4 space-y-4"
	novalidate
>
	<h4 class="text-sm font-medium text-stone-700 dark:text-stone-300">
		{$t('comments.write')}
	</h4>

	<!-- Honeypot field - hidden from real users -->
	<div style="position: absolute; left: -9999px; top: -9999px;" aria-hidden="true">
		<label for="website_url_hp{suffix}">Website</label>
		<input type="text" id="website_url_hp{suffix}" name="website_url" tabindex="-1" autocomplete="off" />
	</div>

	<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
		<div>
			<label for="comment-name{suffix}" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
				{$t('comments.name')} <span class="text-red-500" aria-hidden="true">*</span>
			</label>
			<input
				id="comment-name{suffix}"
				type="text"
				bind:value={authorName}
				required
				maxlength="100"
				aria-required="true"
				class="w-full px-3 py-2 text-sm border border-stone-300 dark:border-stone-600 rounded-lg bg-white dark:bg-stone-700 text-stone-900 dark:text-white placeholder-stone-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
				placeholder={$t('comments.name')}
			/>
		</div>
		<div>
			<label for="comment-email{suffix}" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
				{$t('comments.email')}
			</label>
			<input
				id="comment-email{suffix}"
				type="email"
				bind:value={authorEmail}
				aria-describedby="comment-email-hint{suffix}"
				class="w-full px-3 py-2 text-sm border border-stone-300 dark:border-stone-600 rounded-lg bg-white dark:bg-stone-700 text-stone-900 dark:text-white placeholder-stone-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
				placeholder={$t('comments.email')}
			/>
			<p id="comment-email-hint{suffix}" class="mt-1 text-xs text-stone-600 dark:text-stone-300">{$t('comments.email_hint')}</p>
		</div>
	</div>

	<div>
		<label for="comment-body{suffix}" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
			{$t('comments.body')} <span class="text-red-500" aria-hidden="true">*</span>
		</label>
		<textarea
			id="comment-body{suffix}"
			bind:value={body}
			required
			maxlength={MAX_BODY_LENGTH}
			rows={parentId ? 3 : 4}
			aria-required="true"
			aria-describedby="comment-chars{suffix}"
			class="w-full px-3 py-2 text-sm border border-stone-300 dark:border-stone-600 rounded-lg bg-white dark:bg-stone-700 text-stone-900 dark:text-white placeholder-stone-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-y"
			placeholder={$t('comments.body')}
		></textarea>
		<p
			id="comment-chars{suffix}"
			class="mt-1 text-xs {charsRemaining < 200 ? 'text-amber-500' : 'text-stone-600 dark:text-stone-300'}"
			aria-live="polite"
		>
			{$t('comments.chars_remaining', { values: { count: charsRemaining } })}
		</p>
	</div>

	{#if submitError}
		<p class="text-sm text-red-600 dark:text-red-400" role="alert">{submitError}</p>
	{/if}

	<div class="flex items-center gap-3">
		<button
			type="submit"
			disabled={submitting || !body.trim() || !authorName.trim()}
			aria-busy={submitting}
			class="px-4 py-2 text-sm font-medium bg-primary-600 hover:bg-primary-700 disabled:bg-primary-400 text-white rounded-lg transition-colors disabled:cursor-not-allowed"
		>
			{#if submitting}
				<svg class="animate-spin inline-block h-4 w-4 me-1.5" fill="none" viewBox="0 0 24 24" aria-hidden="true">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
			{/if}
			{$t('comments.submit')}
		</button>
		{#if onCancel}
			<button
				type="button"
				onclick={onCancel}
				class="px-4 py-2 text-sm font-medium text-stone-600 dark:text-stone-400 hover:text-stone-800 dark:hover:text-stone-200 transition-colors"
			>
				{$t('comments.cancel')}
			</button>
		{/if}
	</div>
</form>
