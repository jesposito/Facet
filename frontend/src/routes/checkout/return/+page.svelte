<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';
	import { page } from '$app/state';
	import {
		ACCESS_EMAIL_RESEND_COOLDOWN,
		confirmCheckout,
		confirmPollDelay,
		isCheckoutSessionId,
		type ConfirmResult
	} from '$lib/commerce';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	type View = 'checking' | ConfirmResult['kind'];

	// A session is only confirmed when the visitor isn't already unlocked.
	// Whatever the confirm endpoint says, this page grants nothing: the access
	// link is emailed to the buyer.
	let view = $state<View>('checking');
	let emailHint = $state('');
	let busy = $state(false);
	let polling = $state(false);
	let waitSeconds = $state(0);
	let notice = $state('');

	const confirmFlow = $derived(
		data.hasSession && (data.status === 'locked' || data.status === 'unknown')
	);
	const status = $derived(confirmFlow ? view : data.status);
	const itemName = $derived(
		data.title ? `“${data.title}”` : $t('public.commerce.checkout.your_purchase')
	);

	let sessionId = '';
	let pollAttempt = 0;
	let destroyed = false;
	let controller: AbortController | null = null;
	let pollTimer: ReturnType<typeof setTimeout> | undefined;
	let tickTimer: ReturnType<typeof setInterval> | undefined;

	const errorMessages = $derived<Partial<Record<View, string>>>({
		failed: $t('public.commerce.checkout.error_failed'),
		refunded: $t('public.commerce.checkout.error_refunded'),
		delivery_failed: $t('public.commerce.checkout.error_delivery_failed'),
		delivery_unavailable: $t('public.commerce.checkout.error_delivery_unavailable'),
		not_found: $t('public.commerce.checkout.error_not_found'),
		mismatch: $t('public.commerce.checkout.error_mismatch'),
		rate_limited: $t('public.commerce.checkout.error_rate_limited'),
		error: $t('public.commerce.checkout.error_generic')
	});
	const errorMessage = $derived(confirmFlow ? (errorMessages[view] ?? '') : '');

	function formatWait(total: number): string {
		const minutes = Math.floor(total / 60);
		return `${minutes}:${String(total % 60).padStart(2, '0')}`;
	}

	function startWait(total: number) {
		if (destroyed) return;
		clearInterval(tickTimer);
		waitSeconds = total;
		if (total <= 0) return;
		tickTimer = setInterval(() => {
			waitSeconds = Math.max(0, waitSeconds - 1);
			if (waitSeconds === 0) clearInterval(tickTimer);
		}, 1000);
	}

	function schedulePoll() {
		if (destroyed) return;
		const delay = confirmPollDelay(pollAttempt++);
		polling = delay !== null;
		if (delay !== null) pollTimer = setTimeout(() => void confirm(false), delay * 1000);
	}

	async function confirm(resend: boolean) {
		if (!data.type || !data.slug || !sessionId || busy || destroyed) return;
		clearTimeout(pollTimer);
		polling = false;
		busy = true;
		notice = '';
		controller = new AbortController();

		let result: ConfirmResult;
		try {
			result = await confirmCheckout(
				{ sessionId, contentType: data.type, slug: data.slug, resend },
				fetch,
				controller.signal
			);
		} catch {
			// Only an abort throws, and that means the page has gone.
			return;
		} finally {
			busy = false;
		}
		if (destroyed) return;

		// A refused or cooled-down resend keeps the "check your email" view.
		if (resend && view === 'sent' && result.kind === 'rate_limited') {
			notice = $t('public.commerce.checkout.notice_rate_limited');
			startWait(result.retryAfter);
			return;
		}

		view = result.kind;
		switch (result.kind) {
			case 'sent':
				if (result.emailHint) emailHint = result.emailHint;
				if (resend) {
					notice = result.retryAfter
						? 'An access email went out a moment ago. Give it a few minutes, and check your spam folder.'
						: 'Sent again. It can take a few minutes to arrive, so check your spam folder too.';
					startWait(result.retryAfter || ACCESS_EMAIL_RESEND_COOLDOWN);
				}
				break;
			case 'pending':
				schedulePoll();
				break;
			case 'rate_limited':
				startWait(result.retryAfter);
				break;
			case 'delivery_failed':
				startWait(10);
				break;
			case 'delivery_unavailable':
				startWait(60);
				break;
			case 'error':
				startWait(5);
				break;
		}
	}

	function onAction(resend: boolean) {
		if (busy || waitSeconds > 0) return;
		pollAttempt = 0;
		void confirm(resend);
	}

	onMount(() => {
		const id = page.url.searchParams.get('session_id');
		if (confirmFlow && isCheckoutSessionId(id)) {
			sessionId = id;
			void confirm(false);
		}
		return () => {
			destroyed = true;
			controller?.abort();
			clearTimeout(pollTimer);
			clearInterval(tickTimer);
		};
	});
</script>

<svelte:head>
	<title>{$t('public.commerce.checkout.title')}</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

{#snippet actionButton(label: string, resend: boolean, primary: boolean)}
	{@const waiting = waitSeconds > 0}
	<button
		type="button"
		class="btn {primary ? 'btn-primary' : 'btn-secondary'}"
		aria-disabled={busy || waiting}
		aria-describedby="return-status"
		data-testid="confirm-action"
		onclick={() => onAction(resend)}
	>
		{#if busy}{$t('public.commerce.common.checking')}{:else if waiting}{$t(
				'public.commerce.checkout.action_in',
				{ values: { label, time: formatWait(waitSeconds) } }
			)}{:else}{label}{/if}
	</button>
{/snippet}

<main
	id="main-content"
	class="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center px-4 py-16"
>
	<section
		class="card max-w-lg w-full p-8"
		aria-labelledby="return-heading"
		aria-busy={busy}
		data-testid="checkout-return"
		data-status={status}
	>
		{#if confirmFlow}
			<h1 id="return-heading" class="text-2xl font-semibold text-gray-900 dark:text-white">
				{#if view === 'checking'}{$t('public.commerce.checkout.checking')}
				{:else if view === 'sent'}{$t('public.commerce.checkout.check_email')}
				{:else if view === 'pending'}{$t('public.commerce.checkout.pending')}
				{:else if view === 'failed'}{$t('public.commerce.checkout.failed')}
				{:else if view === 'refunded'}{$t('public.commerce.checkout.refunded')}
				{:else if view === 'delivery_failed' || view === 'delivery_unavailable'}{$t(
						'public.commerce.checkout.paid'
					)}
				{:else if view === 'not_found'}{$t('public.commerce.checkout.not_found')}
				{:else if view === 'mismatch'}{$t('public.commerce.checkout.mismatch')}
				{:else if view === 'rate_limited'}{$t('public.commerce.checkout.rate_limited')}
				{:else}{$t('public.commerce.checkout.error')}{/if}
			</h1>

			<div
				id="return-status"
				role="status"
				aria-live="polite"
				aria-atomic="true"
				class="mt-3 text-gray-600 dark:text-gray-300"
			>
				{#if view === 'checking'}
					<p>{$t('public.commerce.checkout.checking_stripe')}</p>
				{:else if view === 'sent'}
					<p>
						{$t('public.commerce.checkout.sent_before', {
							values: { item: itemName }
						})}{#if emailHint}<strong data-testid="email-hint">{emailHint}</strong>{:else}{$t(
								'public.commerce.checkout.sent_address'
							)}{/if}{$t('public.commerce.checkout.sent_after')}
					</p>
					{#if notice}<p class="mt-3" data-testid="confirm-notice">{notice}</p>{/if}
				{:else if view === 'pending'}
					{#if polling || busy}
						<p>{$t('public.commerce.checkout.pending_polling')}</p>
					{:else}
						<p>{$t('public.commerce.checkout.pending_stopped')}</p>
					{/if}
				{/if}
			</div>

			{#if errorMessage}
				<p role="alert" class="mt-3 text-red-700 dark:text-red-300" data-testid="confirm-error">
					{errorMessage}
				</p>
			{/if}

			<p class="mt-3 text-sm text-gray-500 dark:text-gray-400">
				{$t('public.commerce.checkout.no_unlock')}
			</p>

			<noscript>
				<p class="mt-3 text-gray-600 dark:text-gray-300">
					{$t('public.commerce.checkout.noscript')}
				</p>
			</noscript>

			<div class="mt-6 flex flex-wrap gap-3">
				{#if view === 'sent'}
					{@render actionButton($t('public.commerce.checkout.send_again'), true, false)}
				{:else if view === 'pending' && !polling && !busy}
					{@render actionButton($t('public.commerce.common.check_again'), false, true)}
				{:else if view === 'delivery_failed' || view === 'delivery_unavailable'}
					{@render actionButton($t('public.commerce.checkout.send_retry'), false, true)}
				{:else if view === 'rate_limited' || view === 'error'}
					{@render actionButton($t('public.commerce.common.check_again'), false, true)}
				{/if}
				<a href={data.href} class="btn btn-secondary"
					>{$t('public.commerce.checkout.back_to', {
						values: { item: data.title || $t('public.commerce.checkout.item') }
					})}</a
				>
			</div>
		{:else if data.status === 'unlocked'}
			<h1 id="return-heading" class="text-2xl font-semibold text-gray-900 dark:text-white">
				{$t('public.commerce.checkout.all_set')}
			</h1>
			<p class="mt-3 text-gray-600 dark:text-gray-300">
				{$t('public.commerce.checkout.unlocked', {
					values: {
						item: data.title ? `“${data.title}”` : $t('public.commerce.checkout.your_purchase')
					}
				})}
			</p>
			<a href={data.href} class="btn btn-primary mt-6"
				>{$t('public.commerce.checkout.go_to', {
					values: { item: data.title || $t('public.commerce.checkout.your_purchase') }
				})}</a
			>
		{:else if data.status === 'locked'}
			<h1 id="return-heading" class="text-2xl font-semibold text-gray-900 dark:text-white">
				{$t('public.commerce.checkout.confirming')}
			</h1>
			<p class="mt-3 text-gray-600 dark:text-gray-300">
				{$t('public.commerce.checkout.locked_help')}
			</p>
			<p class="mt-3 text-gray-600 dark:text-gray-300">
				{$t('public.commerce.checkout.locked_contact')}
			</p>
			<a href={data.href} class="btn btn-secondary mt-6"
				>{$t('public.commerce.checkout.back_to', {
					values: { item: data.title || $t('public.commerce.checkout.item') }
				})}</a
			>
		{:else if data.status === 'cancelled'}
			<h1 id="return-heading" class="text-2xl font-semibold text-gray-900 dark:text-white">
				{$t('public.commerce.checkout.cancelled')}
			</h1>
			<p class="mt-3 text-gray-600 dark:text-gray-300">
				{$t('public.commerce.checkout.cancelled_help')}
			</p>
			<p class="mt-3 text-gray-600 dark:text-gray-300">
				{$t('public.commerce.checkout.cancelled_receipt')}
			</p>
			<a href={data.href} class="btn btn-primary mt-6"
				>{$t('public.commerce.checkout.back_to', {
					values: { item: data.title || $t('public.commerce.checkout.item') }
				})}</a
			>
		{:else if data.status === 'unknown'}
			<h1 id="return-heading" class="text-2xl font-semibold text-gray-900 dark:text-white">
				{$t('public.commerce.checkout.unknown')}
			</h1>
			<p class="mt-3 text-gray-600 dark:text-gray-300" role="alert">
				{$t('public.commerce.checkout.unknown_help')}
			</p>
			<div class="mt-6 flex flex-wrap gap-3">
				<a href="?type={data.type}&slug={data.slug}" class="btn btn-primary" data-sveltekit-reload
					>{$t('public.commerce.common.check_again')}</a
				>
				<a href={data.href} class="btn btn-secondary"
					>{$t('public.commerce.checkout.back_to', {
						values: { item: $t('public.commerce.checkout.item') }
					})}</a
				>
			</div>
		{:else}
			<h1 id="return-heading" class="text-2xl font-semibold text-gray-900 dark:text-white">
				{$t('public.commerce.checkout.incomplete')}
			</h1>
			<p class="mt-3 text-gray-600 dark:text-gray-300">
				{$t('public.commerce.checkout.incomplete_help')}
			</p>
			<a href="/" class="btn btn-secondary mt-6">{$t('public.commerce.common.homepage')}</a>
		{/if}
	</section>
</main>
