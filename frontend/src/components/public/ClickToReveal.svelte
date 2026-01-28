<script lang="ts">
	/**
	 * ClickToReveal - Click-to-reveal contact component
	 *
	 * Hides contact information until user clicks to reveal it.
	 * Provides strong anti-scraping protection while maintaining UX.
	 *
	 * Protection level: High (prevents automated scraping, requires user interaction)
	 */
	import { t } from 'svelte-i18n';

	interface Props {
		type?: 'email' | 'phone' | 'url' | 'copy';
		value: string; // The actual contact value
		href?: string | null; // Pre-computed href (if provided, overrides type-based calculation)
		label?: string; // Display text for the button
		icon?: string; // Optional icon HTML
		contactId?: string; // Optional: for tracking reveals via API
	}

	let {
		type = 'email',
		value,
		href: providedHref = null,
		label = '',
		icon = '',
		contactId = ''
	}: Props = $props();

	let revealed = $state(false);
	let copying = $state(false);

	// Use provided href or fall back to type-based calculation
	let computedHref = $derived(
		providedHref !== null
			? providedHref
			: type === 'email'
				? `mailto:${value}`
				: type === 'phone'
					? `tel:${value.replace(/\s/g, '')}`
					: type === 'copy'
						? null
						: value
	);

	let buttonLabel = $derived(label || $t(`public.contacts.show_${type === 'copy' ? 'contact' : type}`));

	function reveal() {
		revealed = true;

		// Optional: Track reveal event via API for analytics
		if (contactId && typeof window !== 'undefined') {
			fetch('/api/contact/reveal', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ contact_id: contactId })
			}).catch(() => {
				// Silent fail - analytics shouldn't block UX
			});
		}
	}

	async function copyToClipboard() {
		if (typeof navigator === 'undefined' || !navigator.clipboard) return;

		copying = true;
		try {
			await navigator.clipboard.writeText(value);
			// Show brief success state
			setTimeout(() => {
				copying = false;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
			copying = false;
		}
	}
</script>

{#if !revealed}
	<button
		type="button"
		class="reveal-button"
		onclick={reveal}
		aria-label={`Reveal ${type}`}
	>
		{#if icon}
			<span class="icon" aria-hidden="true">{@html icon}</span>
		{/if}
		<span>{buttonLabel}</span>
		<svg
			class="arrow"
			xmlns="http://www.w3.org/2000/svg"
			width="16"
			height="16"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
		>
			<polyline points="9 18 15 12 9 6"></polyline>
		</svg>
	</button>
{:else}
	<div class="revealed-content">
		{#if computedHref}
			<a
				href={computedHref}
				class="contact-link"
				target={type === 'url' ? '_blank' : undefined}
				rel={type === 'url' ? 'noopener noreferrer' : undefined}
				aria-label={`${type}: ${value}`}
			>
				{#if icon}
					<span class="icon" aria-hidden="true">{@html icon}</span>
				{/if}
				<span class="value">{value}</span>
			</a>
		{:else}
			<!-- Non-linkable type (Discord, Slack) - display only -->
			<span class="contact-display">
				{#if icon}
					<span class="icon" aria-hidden="true">{@html icon}</span>
				{/if}
				<span class="value">{value}</span>
			</span>
		{/if}

		<button
			type="button"
			class="copy-button"
			class:copied={copying}
			onclick={copyToClipboard}
			aria-label="Copy to clipboard"
			disabled={copying}
		>
			{#if copying}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="16"
					height="16"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					aria-hidden="true"
				>
					<polyline points="20 6 9 17 4 12"></polyline>
				</svg>
				<span class="sr-only">{$t('public.contacts.copied')}</span>
			{:else}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="16"
					height="16"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					aria-hidden="true"
				>
					<rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
					<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
				</svg>
				<span class="sr-only">{$t('public.contacts.copy')}</span>
			{/if}
		</button>
	</div>
{/if}

<style>
	.reveal-button {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		border: 1px solid var(--border-color, #e5e7eb);
		border-radius: 0.375rem;
		background: var(--bg-secondary, #f9fafb);
		color: var(--text-primary, #111827);
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
	}

	.reveal-button:hover {
		background: var(--bg-hover, #f3f4f6);
		border-color: var(--accent-color, #3b82f6);
	}

	.reveal-button:focus-visible {
		outline: 2px solid var(--accent-color, #3b82f6);
		outline-offset: 2px;
	}

	.revealed-content {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
	}

	.contact-link {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--accent-color, #3b82f6);
		text-decoration: none;
		transition: opacity 0.2s;
	}

	.contact-link:hover {
		opacity: 0.8;
		text-decoration: underline;
	}

	.copy-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		padding: 0.25rem;
		border: 1px solid var(--border-color, #e5e7eb);
		border-radius: 0.375rem;
		background: var(--bg-secondary, #f9fafb);
		color: var(--text-secondary, #6b7280);
		cursor: pointer;
		transition: all 0.2s;
	}

	.copy-button:hover:not(:disabled) {
		background: var(--bg-hover, #f3f4f6);
		color: var(--accent-color, #3b82f6);
	}

	.copy-button:disabled {
		color: var(--success-color, #10b981);
		cursor: default;
	}

	/* Success animation for copy button */
	.copy-button.copied {
		animation: copySuccess 0.3s ease-out;
	}

	@keyframes copySuccess {
		0% {
			transform: scale(1);
		}
		50% {
			transform: scale(1.15);
		}
		100% {
			transform: scale(1);
		}
	}

	/* Respect reduced motion preference */
	@media (prefers-reduced-motion: reduce) {
		.copy-button.copied {
			animation: none;
		}
		.arrow {
			transition: none;
		}
	}

	.copy-button:focus-visible {
		outline: 2px solid var(--accent-color, #3b82f6);
		outline-offset: 2px;
	}

	.icon {
		display: inline-flex;
		width: 1.25rem;
		height: 1.25rem;
	}

	.arrow {
		transition: transform 0.2s;
	}

	.reveal-button:hover .arrow {
		transform: translateX(2px);
	}

	.value {
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	}

	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border-width: 0;
	}

	.contact-display {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--text-primary, #111827);
	}

	/* Dark mode styles */
	:global(.dark) .reveal-button {
		border-color: #374151;
		background: #1f2937;
		color: #f9fafb;
	}

	:global(.dark) .reveal-button:hover {
		background: #111827;
		border-color: #38bdf8;
	}

	:global(.dark) .reveal-button:focus-visible {
		outline-color: #38bdf8;
	}

	:global(.dark) .contact-link {
		color: #38bdf8;
	}

	:global(.dark) .contact-display {
		color: #f9fafb;
	}

	:global(.dark) .copy-button {
		border-color: #374151;
		background: #1f2937;
		color: #9ca3af;
	}

	:global(.dark) .copy-button:hover:not(:disabled) {
		background: #111827;
		color: #38bdf8;
	}

	:global(.dark) .copy-button:disabled {
		color: #10b981;
	}
</style>
