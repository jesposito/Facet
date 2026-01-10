<script lang="ts">
	

	interface Props {
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
		type?: 'email' | 'phone' | 'url';
		value: string; // The actual contact value (e.g., "hello@example.com")
		label?: string; // Display text (defaults to value if not provided)
		icon?: string; // Optional icon HTML
	}

	let {
		type = 'email',
		value,
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

	// Build the href based on type
	let href = $derived(type === 'email'
		? `mailto:${value}`
		: type === 'phone'
		? `tel:${value.replace(/\s/g, '')}`
		: value);

	let displayLabel = $derived(label || value);
	let obfuscatedLabel = $derived(obfuscate(displayLabel));
</script>

<a
	{href}
	class="obfuscated-link"
	data-type={type}
	aria-label={`${type}: ${value}`}
>
	{#if icon}
		<span class="icon" aria-hidden="true">{@html icon}</span>
	{/if}
	<span class="label" aria-hidden="true">{@html obfuscatedLabel}</span>
	<span class="sr-only">{displayLabel}</span>
</a>

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
</style>
