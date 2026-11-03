<script lang="ts">
	import { enhance } from '$app/forms';
	import { tick } from 'svelte';
	import { t } from 'svelte-i18n';
	import type { ActionData, PageData } from './$types';

	let { data, form }: { data: PageData; form: ActionData } = $props();

	let submitting = $state(false);
	let heading: HTMLHeadingElement | undefined = $state();

	const check = $derived(data.check);
	const maskedEmail = $derived('maskedEmail' in check ? check.maskedEmail : '');

	// One state drives the whole page. A form result wins over the initial
	// token check because it is newer.
	const view = $derived.by(() => {
		if (form?.action === 'confirm') {
			if (form.result === 'unsubscribed') return 'done';
			if (form.result === 'already_unsubscribed') return 'already';
			if (form.result === 'unavailable') return 'unavailable';
			return 'invalid';
		}
		if (!data.hasToken) return 'no-token';
		return check.state === 'ready' ? 'confirm' : check.state;
	});

	const titles = $derived({
		confirm: $t('public.commerce.unsubscribe.title'),
		done: $t('public.commerce.unsubscribe.done_title'),
		already: $t('public.commerce.unsubscribe.already_title'),
		invalid: $t('public.commerce.unsubscribe.invalid_title'),
		unavailable: $t('public.commerce.unsubscribe.unavailable_title'),
		'no-token': $t('public.commerce.unsubscribe.title')
	});

	function enhanceSubmit() {
		submitting = true;
		return async ({ update }: { update: (opts?: { reset?: boolean }) => Promise<void> }) => {
			await update({ reset: false });
			submitting = false;
			// Move focus to the new heading so screen readers announce the result.
			await tick();
			heading?.focus();
		};
	}
</script>

<svelte:head>
	<title>{titles[view]}</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
</svelte:head>

<main class="unsubscribe" data-testid="unsubscribe-page" data-state={view}>
	<h1 bind:this={heading} tabindex="-1">{titles[view]}</h1>

	<div role="status" aria-live="polite">
		{#if view === 'confirm'}
			<p>
				{#if maskedEmail}{$t('public.commerce.unsubscribe.confirm_before')}<strong
						>{maskedEmail}</strong
					>{$t('public.commerce.unsubscribe.confirm_after')}{:else}{$t(
						'public.commerce.unsubscribe.confirm'
					)}{/if}
				{$t('public.commerce.unsubscribe.confirm_help')}
			</p>
		{:else if view === 'done'}
			<p>{$t('public.commerce.unsubscribe.done')}</p>
		{:else if view === 'already'}
			<p>
				{#if maskedEmail}{$t('public.commerce.unsubscribe.already_before')}<strong
						>{maskedEmail}</strong
					>{$t('public.commerce.unsubscribe.already_after')}{:else}{$t(
						'public.commerce.unsubscribe.already'
					)}{/if}
			</p>
		{:else if view === 'invalid'}
			<p>{$t('public.commerce.unsubscribe.invalid')}</p>
		{:else if view === 'unavailable'}
			<p>{$t('public.commerce.unsubscribe.unavailable')}</p>
		{:else}
			<p>{$t('public.commerce.unsubscribe.intro')}</p>
		{/if}
	</div>

	{#if view === 'confirm' || view === 'unavailable'}
		{#if data.hasToken}
			<form method="POST" action="?/confirm" use:enhance={enhanceSubmit}>
				<button type="submit" class="btn btn-primary" disabled={submitting}>
					{submitting
						? $t('public.commerce.unsubscribe.submitting')
						: view === 'unavailable'
							? $t('public.commerce.common.try_again')
							: $t('public.commerce.unsubscribe.action')}
				</button>
			</form>
		{/if}
	{/if}

	{#if view === 'invalid' || view === 'no-token' || view === 'unavailable'}
		<section aria-labelledby="signed-link-heading" data-testid="unsubscribe-signed-link-help">
			<h2 id="signed-link-heading">{$t('public.commerce.unsubscribe.signed_link_heading')}</h2>
			<p>{$t('public.commerce.unsubscribe.signed_link_required')}</p>
		</section>
	{/if}

	<p class="home"><a href="/">{$t('public.commerce.unsubscribe.back')}</a></p>
</main>

<style>
	.unsubscribe {
		max-width: 36rem;
		margin: 4rem auto;
		padding: 0 1.25rem;
		display: grid;
		gap: 1.25rem;
	}

	h1 {
		font-size: 1.75rem;
		font-weight: 600;
	}

	h1:focus {
		outline: none;
	}

	h2 {
		font-size: 1.125rem;
		font-weight: 600;
		margin-bottom: 0.5rem;
	}

	.home a {
		text-decoration: underline;
	}
</style>
