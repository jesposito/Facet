<script lang="ts">
	import { t } from 'svelte-i18n';
	import { env } from '$env/dynamic/public';
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';

	interface Props {
		viewSlug: string;
	}

	let { viewSlug }: Props = $props();

	let email = $state('');
	let name = $state('');
	let status = $state<'idle' | 'loading' | 'success' | 'error'>('idle');
	let errorMessage = $state('');

	// Turnstile
	let turnstileContainer: HTMLDivElement | undefined = $state();
	let turnstileToken = $state('');
	let scriptLoaded = $state(false);
	const siteKey = env.PUBLIC_TURNSTILE_SITE_KEY || '';

	function loadTurnstileScript(): Promise<void> {
		return new Promise((resolve, reject) => {
			if (typeof window !== 'undefined' && (window as any).turnstile) {
				scriptLoaded = true;
				resolve();
				return;
			}

			const existing = document.querySelector('script[src*="challenges.cloudflare.com/turnstile"]');
			if (existing) {
				existing.addEventListener('load', () => { scriptLoaded = true; resolve(); });
				if ((window as any).turnstile) { scriptLoaded = true; resolve(); }
				return;
			}

			const script = document.createElement('script');
			script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
			script.async = true;
			script.onload = () => { scriptLoaded = true; resolve(); };
			script.onerror = () => reject(new Error('Failed to load Turnstile'));
			document.head.appendChild(script);
		});
	}

	onMount(() => {
		if (browser && siteKey) {
			loadTurnstileScript().then(() => {
				if (turnstileContainer && (window as any).turnstile) {
					(window as any).turnstile.render(turnstileContainer, {
						sitekey: siteKey,
						size: 'invisible',
						callback: (token: string) => { turnstileToken = token; },
						'expired-callback': () => { turnstileToken = ''; }
					});
				}
			}).catch(() => {});
		}
	});

	const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

	async function handleSubmit(e: Event) {
		e.preventDefault();

		const trimmedEmail = email.trim().toLowerCase();
		if (!trimmedEmail || !emailRegex.test(trimmedEmail)) {
			status = 'error';
			errorMessage = $t('public.sections.newsletter_invalid_email');
			return;
		}

		status = 'loading';
		errorMessage = '';

		try {
			const response = await fetch(`/api/public/${viewSlug}/subscribe`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					email: trimmedEmail,
					name: name.trim(),
					turnstile_token: turnstileToken || undefined
				})
			});

			if (response.ok) {
				status = 'success';
				email = '';
				name = '';
			} else if (response.status === 429) {
				status = 'error';
				errorMessage = $t('public.sections.newsletter_error');
			} else {
				const data = await response.json().catch(() => ({}));
				status = 'error';
				errorMessage = data.error || $t('public.sections.newsletter_error');
			}
		} catch {
			status = 'error';
			errorMessage = $t('public.sections.newsletter_error');
		}
	}
</script>

<section id="newsletter" class="mb-16">
	<!-- Full-width CTA banner with gradient -->
	<div class="relative overflow-hidden rounded-3xl bg-gradient-to-br from-primary-600 via-primary-700 to-primary-800 dark:from-primary-800 dark:via-primary-900 dark:to-stone-900 p-8 sm:p-10 md:p-14 shadow-editorial-lg animate-fade-in">
		<!-- Background decorative elements -->
		<div class="absolute inset-0 overflow-hidden pointer-events-none" aria-hidden="true">
			<!-- Soft radial glow -->
			<div class="absolute -top-24 -right-24 w-96 h-96 rounded-full bg-white/[0.07] blur-3xl"></div>
			<div class="absolute -bottom-16 -left-16 w-64 h-64 rounded-full bg-white/[0.05] blur-2xl"></div>
			<!-- Subtle grid texture -->
			<div class="absolute inset-0 opacity-[0.04]" style="background-image: url('data:image/svg+xml,%3Csvg width=&quot;40&quot; height=&quot;40&quot; xmlns=&quot;http://www.w3.org/2000/svg&quot;%3E%3Cpath d=&quot;M0 0h40v40H0z&quot; fill=&quot;none&quot; stroke=&quot;white&quot; stroke-width=&quot;0.5&quot;/%3E%3C/svg%3E'); background-size: 40px 40px;"></div>
		</div>

		<div class="relative z-10 max-w-2xl mx-auto text-center">
			<!-- Mail icon -->
			<div class="inline-flex items-center justify-center w-12 h-12 rounded-2xl bg-white/10 backdrop-blur-sm mb-5">
				<svg class="w-6 h-6 text-white/90" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
					<path stroke-linecap="round" stroke-linejoin="round" d="M21.75 6.75v10.5a2.25 2.25 0 01-2.25 2.25h-15a2.25 2.25 0 01-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25m19.5 0v.243a2.25 2.25 0 01-1.07 1.916l-7.5 4.615a2.25 2.25 0 01-2.36 0L3.32 8.91a2.25 2.25 0 01-1.07-1.916V6.75" />
				</svg>
			</div>

			<h2 class="text-2xl sm:text-3xl font-serif text-white mb-3">{$t('public.sections.newsletter_heading')}</h2>
			<p class="text-primary-100/80 dark:text-stone-300/70 mb-8 max-w-lg mx-auto leading-relaxed">{$t('public.sections.newsletter_description')}</p>

			{#if status === 'success'}
				<div class="animate-scale-in inline-flex items-center gap-2.5 px-6 py-3.5 bg-white/15 backdrop-blur-sm rounded-2xl border border-white/20" role="status">
					<div class="flex items-center justify-center w-6 h-6 rounded-full bg-emerald-400/20">
						<svg class="w-4 h-4 text-emerald-300" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
							<polyline points="20 6 9 17 4 12"></polyline>
						</svg>
					</div>
					<span class="text-white font-medium">{$t('public.sections.newsletter_success')}</span>
				</div>
			{:else}
				<form onsubmit={handleSubmit} class="max-w-md mx-auto">
					<!-- Integrated input group -->
					<div class="space-y-3 mb-4">
						<div class="relative">
							<input
								type="email"
								bind:value={email}
								placeholder={$t('public.sections.newsletter_email_placeholder')}
								required
								class="newsletter-input"
								disabled={status === 'loading'}
							/>
						</div>
						<div class="relative">
							<input
								type="text"
								bind:value={name}
								placeholder={$t('public.sections.newsletter_name_placeholder')}
								class="newsletter-input"
								disabled={status === 'loading'}
							/>
						</div>
					</div>

					<button
						type="submit"
						class="newsletter-button"
						disabled={status === 'loading'}
					>
						{#if status === 'loading'}
							<span class="spinner" aria-hidden="true"></span>
						{/if}
						{$t('public.sections.newsletter_button')}
					</button>

					{#if errorMessage}
						<p class="error-message mt-3" role="alert">{errorMessage}</p>
					{/if}

					<!-- Hidden Turnstile container -->
					{#if siteKey}
						<div bind:this={turnstileContainer} class="turnstile-container" aria-hidden="true"></div>
					{/if}
				</form>
			{/if}
		</div>
	</div>
</section>

<style>
	.newsletter-input {
		width: 100%;
		padding: 0.75rem 1rem;
		border: 1px solid rgba(255, 255, 255, 0.2);
		border-radius: 1rem;
		background: rgba(255, 255, 255, 0.12);
		-webkit-backdrop-filter: blur(8px);
		backdrop-filter: blur(8px);
		color: #ffffff;
		font-size: 0.9375rem;
		transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
	}

	.newsletter-input::placeholder {
		color: rgba(255, 255, 255, 0.5);
	}

	.newsletter-input:focus {
		outline: none;
		border-color: rgba(255, 255, 255, 0.4);
		background: rgba(255, 255, 255, 0.18);
		box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.1);
	}

	.newsletter-input:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.newsletter-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		width: 100%;
		padding: 0.75rem 2rem;
		background: rgba(255, 255, 255, 0.95);
		color: var(--color-primary-700, #0369a1);
		border: none;
		border-radius: 1rem;
		font-size: 0.9375rem;
		font-weight: 700;
		cursor: pointer;
		transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
	}

	.newsletter-button:hover:not(:disabled) {
		background: #ffffff;
		transform: translateY(-1px);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.18);
	}

	.newsletter-button:active:not(:disabled) {
		transform: translateY(0);
		box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
	}

	.newsletter-button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.spinner {
		display: inline-block;
		width: 1rem;
		height: 1rem;
		border: 2px solid currentColor;
		border-right-color: transparent;
		border-radius: 50%;
		animation: spin 0.6s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.error-message {
		font-size: 0.8125rem;
		color: #fecaca;
		text-align: center;
	}

	.turnstile-container {
		position: absolute;
		overflow: hidden;
		height: 0;
		width: 0;
	}

	@media (prefers-reduced-motion: reduce) {
		.spinner {
			animation: none;
		}
	}
</style>
