<script lang="ts">
	/**
	 * Sign-up form for free courses.
	 *
	 * New learners get a confirmation email and nothing changes on this page
	 * until they follow its link. A visitor whose learner cookie already
	 * proves the same address is sent straight to the course by the backend.
	 */
	import { t } from 'svelte-i18n';
	import { startEnrollment } from '$lib/commerce';

	interface Props {
		slug: string;
		title: string;
		heading?: string;
		intro?: string;
	}

	let { slug, title, heading = '', intro = '' }: Props = $props();

	const uid = $props.id();
	let email = $state('');
	let name = $state('');
	let busy = $state(false);
	let errorMessage = $state('');
	let sentTo = $state('');

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (busy) return;
		busy = true;
		errorMessage = '';
		const result = await startEnrollment(slug, email, name);
		if (result.kind === 'redirect') {
			window.location.assign(result.url);
			return;
		}
		if (result.kind === 'confirmation_sent') {
			sentTo = email.trim();
		} else {
			errorMessage = result.message;
		}
		busy = false;
	}
</script>

<section class="card p-6" aria-labelledby="{uid}-heading" data-testid="enrollment-form">
	<h2 id="{uid}-heading" class="text-lg font-semibold text-gray-900 dark:text-white">
		{heading || $t('public.commerce.enrollment.heading')}
	</h2>

	<div role="status" aria-live="polite">
		{#if sentTo}
			<div class="mt-2 text-sm text-gray-700 dark:text-gray-200" data-testid="enrollment-sent">
				<p>
					{$t('public.commerce.enrollment.sent_before')}<strong>{sentTo}</strong>{$t(
						'public.commerce.enrollment.sent_after',
						{ values: { title } }
					)}
				</p>
				<p class="mt-2 text-gray-600 dark:text-gray-300">
					{$t('public.commerce.enrollment.sent_help')}
				</p>
			</div>
		{/if}
	</div>

	{#if !sentTo}
		<p class="mt-2 text-sm text-gray-600 dark:text-gray-300">
			{intro || $t('public.commerce.enrollment.intro')}
		</p>
	{/if}

	<form
		class="mt-4 space-y-3"
		onsubmit={submit}
		aria-describedby={errorMessage ? `${uid}-error` : undefined}
	>
		<div>
			<label for="{uid}-email" class="block text-sm font-medium text-gray-700 dark:text-gray-200"
				>{$t('public.commerce.enrollment.email')}</label
			>
			<input
				id="{uid}-email"
				type="email"
				bind:value={email}
				required
				autocomplete="email"
				maxlength="254"
				class="input mt-1 w-full sm:w-80"
				disabled={busy}
				aria-invalid={errorMessage ? 'true' : undefined}
			/>
		</div>
		<div>
			<label for="{uid}-name" class="block text-sm font-medium text-gray-700 dark:text-gray-200">
				{$t('public.commerce.enrollment.name')}
				<span class="font-normal text-gray-500 dark:text-gray-400"
					>{$t('public.commerce.enrollment.name_hint')}</span
				>
			</label>
			<input
				id="{uid}-name"
				type="text"
				bind:value={name}
				autocomplete="name"
				maxlength="200"
				class="input mt-1 w-full sm:w-80"
				disabled={busy}
			/>
		</div>
		<button type="submit" class="btn btn-primary" disabled={busy} aria-busy={busy}>
			{busy
				? $t('public.commerce.common.sending')
				: sentTo
					? $t('public.commerce.enrollment.send_again')
					: $t('public.commerce.enrollment.send')}
		</button>
		{#if errorMessage}
			<p id="{uid}-error" role="alert" class="text-sm text-red-700 dark:text-red-400">
				{errorMessage}
			</p>
		{/if}
	</form>
</section>
