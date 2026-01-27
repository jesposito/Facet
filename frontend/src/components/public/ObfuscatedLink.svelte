<script lang="ts">
	/**
	 * ObfuscatedLink - Anti-scraping contact link component
	 *
	 * Uses CSS-based obfuscation to protect contact information from bots:
	 * - Decoy characters hidden with display:none
	 * - Reversed text corrected with CSS direction
	 * - Accessibility maintained with proper ARIA attributes
	 *
	 * Protection level: Medium (blocks simple scrapers, readable by screen readers)
	 */

	interface Props {
		type?: 'email' | 'phone' | 'url' | 'copy';
		value: string; // The actual contact value (e.g., "hello@example.com")
		href?: string | null; // Pre-computed href (if provided, overrides type-based calculation)
		label?: string; // Display text (defaults to value if not provided)
		icon?: string; // Optional icon HTML
	}

	let {
		type = 'email',
		value,
		href: providedHref = null,
		label = '',
		icon = ''
	}: Props = $props();

	// Generate obfuscated version with decoy characters
	function obfuscate(text: string): string {
		if (!text) return '';

		// Insert invisible decoy characters between real characters
		// Format: realChar + <span class="decoy">DECOY</span>
		const chars = text.split('');
		const decoys = ['X', 'Y', 'Z', '9', '8', '7'];

		return chars
			.map((char, i) => {
				const decoy = decoys[i % decoys.length];
				return `${char}<span class="decoy" aria-hidden="true">${decoy}</span>`;
			})
			.join('');
	}

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

	let displayLabel = $derived(label || value);
	let obfuscatedLabel = $derived(obfuscate(displayLabel));

	// Copy to clipboard for non-linkable types
	let copying = $state(false);

	async function copyToClipboard() {
		if (typeof navigator === 'undefined' || !navigator.clipboard) return;

		copying = true;
		try {
			await navigator.clipboard.writeText(value);
			setTimeout(() => {
				copying = false;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
			copying = false;
		}
	}
</script>

{#if computedHref}
	<a
		href={computedHref}
		class="obfuscated-link"
		data-type={type}
		target={type === 'url' ? '_blank' : undefined}
		rel={type === 'url' ? 'noopener noreferrer' : undefined}
		aria-label={`${type}: ${value}`}
	>
		{#if icon}
			<span class="icon" aria-hidden="true">{@html icon}</span>
		{/if}
		<span class="label" aria-hidden="true">{@html obfuscatedLabel}</span>
		<span class="sr-only">{displayLabel}</span>
	</a>
{:else}
	<!-- Non-linkable type (Discord, Slack) - copy only -->
	<div class="obfuscated-link copy-only" data-type={type}>
		{#if icon}
			<span class="icon" aria-hidden="true">{@html icon}</span>
		{/if}
		<span class="label" aria-hidden="true">{@html obfuscatedLabel}</span>
		<span class="sr-only">{displayLabel}</span>
		<button
			type="button"
			class="copy-button"
			onclick={copyToClipboard}
			aria-label="Copy to clipboard"
			disabled={copying}
		>
			{#if copying}
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
			{:else}
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
				</svg>
			{/if}
		</button>
	</div>
{/if}

<style>
	.obfuscated-link {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--accent-color, #3b82f6);
		text-decoration: none;
		transition: opacity 0.2s;
	}

	.obfuscated-link:hover {
		opacity: 0.8;
		text-decoration: underline;
	}

	.icon {
		display: inline-flex;
		width: 1.25rem;
		height: 1.25rem;
	}

	/* Hide decoy characters from visual rendering */
	.label :global(.decoy) {
		display: none;
		position: absolute;
		left: -9999px;
		width: 0;
		height: 0;
		overflow: hidden;
	}

	/* Screen reader only text */
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

	/* Accessibility: ensure screen readers can access the real value */
	.label {
		user-select: all; /* Allow selecting the real text */
	}

	/* Copy-only styles */
	.copy-only {
		cursor: default;
	}

	.copy-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.75rem;
		height: 1.75rem;
		padding: 0.25rem;
		margin-left: 0.5rem;
		border: 1px solid var(--border-color, #e5e7eb);
		border-radius: 0.25rem;
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

	/* Dark mode styles */
	:global(.dark) .obfuscated-link {
		color: #38bdf8;
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
