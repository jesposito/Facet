<script lang="ts">
	/**
	 * ContactMethodsList - Displays contact methods in public views
	 *
	 * Features:
	 * - Renders contact methods with appropriate protection level
	 * - Filters by view visibility
	 * - Highlights primary contacts
	 * - Supports all protection levels: none, obfuscation, click_to_reveal, captcha
	 * - Non-linkable types (Discord, Slack) render as copy-only
	 */

	import { type ContactMethod } from '$lib/pocketbase';
	import { buildContactHref, isLinkableContactType } from '$lib/utils';
	import ObfuscatedLink from './ObfuscatedLink.svelte';
	import ClickToReveal from './ClickToReveal.svelte';

	interface Props {
		contacts?: ContactMethod[];
		viewId?: string;
		layout?: 'vertical' | 'horizontal' | 'grid';
	}

	let { contacts = [], viewId = '', layout = 'vertical' }: Props = $props();

	// Filter contacts for this view
	let visibleContacts = $derived(contacts.filter((contact) => {
		// If view_visibility is empty or null, show on all views
		if (!contact.view_visibility || Object.keys(contact.view_visibility).length === 0) {
			return true;
		}
		// Otherwise, check if this view is explicitly enabled
		return contact.view_visibility[viewId] === true;
	}));

	// Sort by sort_order (descending), then by is_primary
	let sortedContacts = $derived([...visibleContacts].sort((a, b) => {
		if (a.is_primary !== b.is_primary) return a.is_primary ? -1 : 1;
		return b.sort_order - a.sort_order;
	}));

	// Get icon for contact type
	function getIcon(contact: ContactMethod): string {
		// Use custom icon if set, otherwise use default for type
		if (contact.icon) return contact.icon;

		const icons: Record<string, string> = {
			email: '📧',
			phone: '📱',
			linkedin: '💼',
			github: '🐙',
			twitter: '🐦',
			facebook: '👥',
			instagram: '📷',
			website: '🌐',
			whatsapp: '💬',
			telegram: '✈️',
			discord: '🎮',
			slack: '💼',
			other: '🔗'
		};

		return icons[contact.type] || '🔗';
	}

	// Get label for contact
	function getLabel(contact: ContactMethod): string {
		if (contact.label) return contact.label;

		const labels: Record<string, string> = {
			email: 'Email',
			phone: 'Phone',
			linkedin: 'LinkedIn',
			github: 'GitHub',
			twitter: 'Twitter',
			facebook: 'Facebook',
			instagram: 'Instagram',
			website: 'Website',
			whatsapp: 'WhatsApp',
			telegram: 'Telegram',
			discord: 'Discord',
			slack: 'Slack',
			other: 'Contact'
		};

		return labels[contact.type] || 'Contact';
	}

	// Determine link type for child components (ObfuscatedLink, ClickToReveal)
	function getLinkType(type: string): 'email' | 'phone' | 'url' | 'copy' {
		if (!isLinkableContactType(type)) return 'copy';
		if (type === 'email') return 'email';
		if (type === 'phone') return 'phone';
		// WhatsApp now uses wa.me links (url type), not tel:
		return 'url';
	}

	// Copy to clipboard functionality for non-linkable contacts
	let copyingId: string | null = $state(null);

	async function copyToClipboard(value: string, contactId: string) {
		if (typeof navigator === 'undefined' || !navigator.clipboard) return;

		copyingId = contactId;
		try {
			await navigator.clipboard.writeText(value);
			setTimeout(() => {
				copyingId = null;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
			copyingId = null;
		}
	}
</script>

{#if sortedContacts.length > 0}
	<section id="contacts" class="mb-16">
		<h2 class="section-title">Contact</h2>
		<div class="contact-methods-list layout-{layout}" role="list">
			{#each sortedContacts as contact}
				{@const href = buildContactHref(contact.type, contact.value)}
				{@const linkType = getLinkType(contact.type)}
				<div
					class="contact-method {contact.is_primary ? 'primary' : ''}"
					role="listitem"
				>
					{#if contact.protection_level === 'none'}
						<!-- No protection - direct link or copy-only for non-linkable types -->
						{#if href}
							<a
								{href}
								class="contact-link"
								target={linkType === 'url' ? '_blank' : undefined}
								rel={linkType === 'url' ? 'noopener noreferrer' : undefined}
							>
								<span class="icon" aria-hidden="true">{getIcon(contact)}</span>
								<span class="label">{getLabel(contact)}</span>
								<span class="value">{contact.value}</span>
								{#if contact.is_primary}
									<span class="primary-badge">Primary</span>
								{/if}
							</a>
						{:else}
							<!-- Non-linkable type (Discord, Slack) - copy only -->
							<div class="contact-link copy-only">
								<span class="icon" aria-hidden="true">{getIcon(contact)}</span>
								<span class="label">{getLabel(contact)}</span>
								<span class="value">{contact.value}</span>
								<button
									type="button"
									class="copy-button"
									onclick={() => copyToClipboard(contact.value, contact.id)}
									aria-label="Copy {getLabel(contact)} to clipboard"
									disabled={copyingId === contact.id}
								>
									{#if copyingId === contact.id}
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
										</svg>
										<span class="sr-only">Copied!</span>
									{:else}
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
										</svg>
										<span class="sr-only">Copy</span>
									{/if}
								</button>
								{#if contact.is_primary}
									<span class="primary-badge">Primary</span>
								{/if}
							</div>
						{/if}

				{:else if contact.protection_level === 'obfuscation'}
					<!-- CSS Obfuscation -->
					<div class="contact-wrapper">
						<ObfuscatedLink
							type={linkType}
							value={contact.value}
							href={href}
							label={getLabel(contact)}
							icon={getIcon(contact)}
						/>
						{#if contact.is_primary}
							<span class="primary-badge">Primary</span>
						{/if}
					</div>

				{:else if contact.protection_level === 'click_to_reveal'}
					<!-- Click to Reveal -->
					<div class="contact-wrapper">
						<ClickToReveal
							type={linkType}
							value={contact.value}
							href={href}
							label={getLabel(contact)}
							icon={getIcon(contact)}
							contactId={contact.id}
						/>
						{#if contact.is_primary}
							<span class="primary-badge">Primary</span>
						{/if}
					</div>

				{:else if contact.protection_level === 'captcha'}
					<!-- CAPTCHA Protection -->
					<div class="contact-wrapper">
						<button type="button" class="captcha-button" disabled>
							<span class="icon" aria-hidden="true">{getIcon(contact)}</span>
							<span>{getLabel(contact)}</span>
							<span class="badge">CAPTCHA Required</span>
						</button>
						{#if contact.is_primary}
							<span class="primary-badge">Primary</span>
						{/if}
					</div>
				{/if}
			</div>
		{/each}
		</div>
	</section>
{/if}

<style>
	.contact-methods-list {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.contact-methods-list.layout-horizontal {
		flex-direction: row;
		flex-wrap: wrap;
	}

	.contact-methods-list.layout-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
		gap: 1rem;
	}

	.contact-method {
		position: relative;
	}

	.contact-method.primary {
		font-weight: 500;
	}

	.contact-link {
		display: inline-flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		border: 1px solid var(--border-color, #e5e7eb);
		border-radius: 0.5rem;
		background: var(--bg-secondary, #ffffff);
		color: var(--text-primary, #111827);
		text-decoration: none;
		transition: all 0.2s;
		width: 100%;
		min-width: 0; /* Allow flexbox children to shrink */
		overflow: hidden;
	}

	.contact-link:hover {
		border-color: var(--accent-color, #3b82f6);
		background: var(--bg-hover, #f9fafb);
		transform: translateY(-1px);
		box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
	}

	.contact-link:focus-visible {
		outline: 2px solid var(--accent-color, #3b82f6);
		outline-offset: 2px;
	}

	.contact-wrapper {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.75rem 1rem;
		border: 1px solid var(--border-color, #e5e7eb);
		border-radius: 0.5rem;
		background: var(--bg-secondary, #ffffff);
	}

	.icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-size: 1.25rem;
		width: 1.5rem;
		height: 1.5rem;
		flex-shrink: 0;
	}

	.label {
		font-weight: 500;
		color: var(--text-secondary, #6b7280);
	}

	.value {
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		color: var(--accent-color, #3b82f6);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		min-width: 0;
		flex: 1;
	}

	.primary-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.125rem 0.5rem;
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--accent-color, #3b82f6);
		background: color-mix(in srgb, var(--accent-color, #3b82f6) 10%, transparent);
		border: 1px solid var(--accent-color, #3b82f6);
		border-radius: 9999px;
		margin-left: auto;
	}

	.captcha-button {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		border: 1px solid var(--border-color, #e5e7eb);
		border-radius: 0.375rem;
		background: var(--bg-secondary, #f9fafb);
		color: var(--text-secondary, #6b7280);
		cursor: not-allowed;
		opacity: 0.6;
	}

	.badge {
		font-size: 0.75rem;
		padding: 0.125rem 0.5rem;
		background: var(--bg-tertiary, #e5e7eb);
		border-radius: 0.25rem;
	}

	/* Responsive adjustments */
	@media (max-width: 640px) {
		.contact-methods-list.layout-horizontal {
			flex-direction: column;
		}

		.contact-methods-list.layout-grid {
			grid-template-columns: 1fr;
		}

		.contact-link,
		.contact-wrapper {
			flex-wrap: wrap;
		}

		.value {
			font-size: 0.875rem;
			word-break: break-all;
		}
	}

	:global(.dark) .contact-link {
		border-color: #374151;
		background: #1f2937;
		color: #f9fafb;
	}

	:global(.dark) .contact-link:hover {
		border-color: var(--accent-color, #3b82f6);
		background: #111827;
	}

	:global(.dark) .contact-wrapper {
		border-color: #374151;
		background: #1f2937;
	}

	:global(.dark) .label {
		color: #9ca3af;
	}

	:global(.dark) .captcha-button {
		border-color: #374151;
		background: #1f2937;
		color: #9ca3af;
	}

	:global(.dark) .badge {
		background: #374151;
	}

	/* Copy-only styles for non-linkable contact types */
	.copy-only {
		cursor: default;
	}

	.copy-only:hover {
		transform: none;
		box-shadow: none;
	}

	.copy-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		padding: 0.25rem;
		margin-left: auto;
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
</style>
