<script lang="ts">
	/**
	 * Purchase panel for paid posts, projects, talks and courses.
	 *
	 * Render it only when the backend projection says the item is locked
	 * (`is_gated === true` for posts/projects/talks, `is_purchased === false`
	 * for courses). It never unlocks anything itself: the button starts a real
	 * checkout session and the backend decides access afterwards.
	 */
	import { locale, t } from 'svelte-i18n';
	import { formatPrice, startCheckout, type PurchasableType } from '$lib/commerce';

	interface Props {
		contentType: PurchasableType;
		contentId: string;
		slug: string;
		title: string;
		/** Price in minor units (cents), as stored by the backend. */
		price: number;
		currency?: string;
		/** Short line shown above the button, e.g. what the purchase includes. */
		summary?: string;
		allowCoupon?: boolean;
	}

	let {
		contentType,
		contentId,
		slug,
		title,
		price,
		currency = 'usd',
		summary = '',
		allowCoupon = true
	}: Props = $props();

	const uid = $props.id();
	let busy = $state(false);
	let errorMessage = $state('');
	let showCoupon = $state(false);
	let couponCode = $state('');

	let formattedPrice = $derived(formatPrice(price, $locale, currency));

	async function buy(event: SubmitEvent) {
		event.preventDefault();
		if (busy) return;
		busy = true;
		errorMessage = '';
		const result = await startCheckout(
			{ contentType, contentId, slug, couponCode: showCoupon ? couponCode : '' },
			window.location.origin
		);
		if (result.ok) {
			window.location.assign(result.url);
			return;
		}
		errorMessage = result.message;
		busy = false;
	}
</script>

<section
	class="card p-6 border border-primary-200 dark:border-primary-800"
	aria-labelledby="{uid}-heading"
	data-testid="purchase-gate"
>
	<div class="flex items-start gap-3">
		<svg
			class="w-6 h-6 mt-0.5 shrink-0 text-primary-600 dark:text-primary-400"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
			aria-hidden="true"
		>
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
			/>
		</svg>
		<div class="min-w-0 flex-1">
			<h2 id="{uid}-heading" class="text-lg font-semibold text-gray-900 dark:text-white">
				{$t(`public.commerce.purchase.heading_${contentType}`)}
			</h2>
			<p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
				{summary || $t('public.commerce.purchase.summary', { values: { title } })}
			</p>
		</div>
	</div>

	<form
		class="mt-5 space-y-3"
		onsubmit={buy}
		aria-describedby={errorMessage ? `${uid}-error` : undefined}
	>
		{#if allowCoupon}
			{#if showCoupon}
				<div>
					<label
						for="{uid}-coupon"
						class="block text-sm font-medium text-gray-700 dark:text-gray-200"
						>{$t('public.commerce.purchase.coupon')}</label
					>
					<input
						id="{uid}-coupon"
						type="text"
						bind:value={couponCode}
						autocomplete="off"
						autocapitalize="characters"
						maxlength="64"
						class="input mt-1 w-full sm:w-64"
						disabled={busy}
					/>
				</div>
			{:else}
				<button
					type="button"
					class="text-sm text-primary-600 dark:text-primary-400 underline underline-offset-2 hover:no-underline focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 rounded"
					onclick={() => (showCoupon = true)}
				>
					{$t('public.commerce.purchase.have_coupon')}
				</button>
			{/if}
		{/if}

		<div class="flex flex-wrap items-center gap-3">
			<button type="submit" class="btn btn-primary" disabled={busy} aria-busy={busy}>
				{busy
					? $t('public.commerce.purchase.starting')
					: $t('public.commerce.purchase.buy_for', { values: { price: formattedPrice } })}
			</button>
			<span class="text-xs text-gray-500 dark:text-gray-400"
				>{$t('public.commerce.purchase.stripe')}</span
			>
		</div>

		{#if errorMessage}
			<p id="{uid}-error" role="alert" class="text-sm text-red-700 dark:text-red-400">
				{errorMessage}
			</p>
		{/if}
	</form>
</section>
