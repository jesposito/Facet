<script lang="ts">
	import { t } from 'svelte-i18n';
	import { siteFeatures } from '$lib/stores/features';
	import { buildSignupBody, readSignupOutcome } from '$lib/newsletter/signup';

	interface Props {
		slug: string;
		planFeatures?: string[];
		/** List id or slug to join. Leave out for the site's default list. */
		list?: string;
	}

	let { slug, planFeatures = [], list = '' }: Props = $props();

	// Shown when the plan includes the newsletter and the owner hasn't
	// switched it off in Settings → Features.
	let newsletterEnabled = $derived(
		$siteFeatures.newsletter && (planFeatures.length === 0 || planFeatures.includes('newsletter'))
	);

	let email = $state('');
	let name = $state('');
	let status = $state<'idle' | 'submitting' | 'success' | 'error'>('idle');
	let errorMessage = $state('');

	const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();

		const trimmedEmail = email.trim().toLowerCase();
		if (!trimmedEmail || !emailPattern.test(trimmedEmail)) {
			errorMessage = $t('public.sections.newsletter_invalid_email');
			status = 'error';
			return;
		}

		status = 'submitting';
		errorMessage = '';

		try {
			const response = await fetch(`/api/public/${slug}/subscribe`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(
					buildSignupBody({ email: trimmedEmail, name, source: 'newsletter', list })
				)
			});

			// The server answers the same way whether or not the address is
			// already subscribed, so only a real failure shows as one.
			const outcome = await readSignupOutcome(response);
			if (outcome === 'confirmation_required') {
				status = 'success';
				email = '';
				name = '';
			} else {
				errorMessage =
					outcome === 'list_unavailable'
						? $t('public.sections.newsletter_list_unavailable', {
								default: "This newsletter isn't taking signups right now."
							})
						: $t('public.sections.newsletter_error');
				status = 'error';
			}
		} catch (err) {
			console.error('Newsletter subscribe error:', err);
			errorMessage = $t('public.sections.newsletter_error');
			status = 'error';
		}
	}
</script>

{#if newsletterEnabled}
	<section class="newsletter-subscribe py-8">
		<div class="max-w-md mx-auto text-center">
			<h2 class="text-xl font-semibold mb-2">{$t('public.sections.newsletter_heading')}</h2>
			<p class="text-sm text-gray-600 dark:text-gray-400 mb-6">
				{$t('public.sections.newsletter_description')}
			</p>

			{#if status === 'success'}
				<div
					class="flex items-center justify-center gap-2 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg"
					role="status"
				>
					<svg
						class="w-5 h-5 text-green-600 dark:text-green-400 flex-shrink-0"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M5 13l4 4L19 7"
						/>
					</svg>
					<span class="text-green-800 dark:text-green-300 font-medium">
						{$t('public.sections.newsletter_success')}
					</span>
				</div>
			{:else}
				<form onsubmit={handleSubmit} class="space-y-3" novalidate>
					<!-- Honeypot field — hidden from real users, traps bots -->
					<div class="sr-only" aria-hidden="true">
						<label for="website_url_field">Website URL</label>
						<input
							type="text"
							id="website_url_field"
							name="website_url"
							tabindex="-1"
							autocomplete="off"
						/>
					</div>

					<div>
						<label for="newsletter-name" class="sr-only">
							{$t('public.sections.newsletter_name_placeholder')}
						</label>
						<input
							type="text"
							id="newsletter-name"
							bind:value={name}
							class="input w-full text-sm"
							placeholder={$t('public.sections.newsletter_name_placeholder')}
							autocomplete="name"
						/>
					</div>

					<div>
						<label for="newsletter-email" class="sr-only">
							{$t('public.sections.newsletter_email_placeholder')}
						</label>
						<input
							type="email"
							id="newsletter-email"
							bind:value={email}
							required
							class="input w-full text-sm"
							placeholder={$t('public.sections.newsletter_email_placeholder')}
							autocomplete="email"
						/>
					</div>

					{#if status === 'error' && errorMessage}
						<p class="text-sm text-red-600 dark:text-red-400" role="alert">{errorMessage}</p>
					{/if}

					<button type="submit" class="btn btn-primary w-full" disabled={status === 'submitting'}>
						{status === 'submitting'
							? $t('shared.status.loading')
							: $t('public.sections.newsletter_button')}
					</button>
				</form>
			{/if}
		</div>
	</section>
{/if}
