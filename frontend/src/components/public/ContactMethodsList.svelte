<script lang="ts">
	/**
	 * ContactMethodsList - Displays contact methods in public views
	 *
	 * Features:
	 * - Renders contact methods with appropriate protection level
	 * - Filters by view visibility
	 * - Highlights primary contacts
	 * - Supports all protection levels: none, obfuscation, click_to_reveal, captcha
	 */

	import { type ContactMethod } from '$lib/pocketbase';
	import ObfuscatedLink from './ObfuscatedLink.svelte';
	import ClickToReveal from './ClickToReveal.svelte';

	export let contacts: ContactMethod[] = [];
	export let viewId: string = '';
	export let layout: 'vertical' | 'horizontal' | 'grid' = 'vertical';

	// Filter contacts for this view
	$: visibleContacts = contacts.filter((contact) => {
		// If view_visibility is empty or null, show on all views
		if (!contact.view_visibility || Object.keys(contact.view_visibility).length === 0) {
			return true;
		}
		// Otherwise, check if this view is explicitly enabled
		return contact.view_visibility[viewId] === true;
	});

	// Sort by sort_order (descending), then by is_primary
	$: sortedContacts = [...visibleContacts].sort((a, b) => {
		if (a.is_primary !== b.is_primary) return a.is_primary ? -1 : 1;
		return b.sort_order - a.sort_order;
	});

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

	// Determine if contact type should be a URL link
	function getLinkType(type: string): 'email' | 'phone' | 'url' {
		if (type === 'email') return 'email';
		if (type === 'phone' || type === 'whatsapp') return 'phone';
		return 'url';
	}
</script>

{#if sortedContacts.length > 0}
	<div class="contact-methods-list layout-{layout}" role="list">
		{#each sortedContacts as contact}
			<div
				class="contact-method {contact.is_primary ? 'primary' : ''}"
				role="listitem"
			>
				{#if contact.protection_level === 'none'}
					<!-- No protection - direct link -->
					<a
						href={getLinkType(contact.type) === 'email'
							? `mailto:${contact.value}`
							: getLinkType(contact.type) === 'phone'
							? `tel:${contact.value.replace(/\s/g, '')}`
							: contact.value}
						class="contact-link"
						target={getLinkType(contact.type) === 'url' ? '_blank' : undefined}
						rel={getLinkType(contact.type) === 'url' ? 'noopener noreferrer' : undefined}
					>
						<span class="icon" aria-hidden="true">{getIcon(contact)}</span>
						<span class="label">{getLabel(contact)}</span>
						<span class="value">{contact.value}</span>
						{#if contact.is_primary}
							<span class="primary-badge">Primary</span>
						{/if}
					</a>

				{:else if contact.protection_level === 'obfuscation'}
					<!-- CSS Obfuscation -->
					<div class="contact-wrapper">
						<ObfuscatedLink
							type={getLinkType(contact.type)}
							value={contact.value}
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
							type={getLinkType(contact.type)}
							value={contact.value}
							label={getLabel(contact)}
							icon={getIcon(contact)}
							contactId={contact.id}
						/>
						{#if contact.is_primary}
							<span class="primary-badge">Primary</span>
						{/if}
					</div>

				{:else if contact.protection_level === 'captcha'}
					<!-- CAPTCHA Protection (coming soon) -->
					<div class="contact-wrapper">
						<button type="button" class="captcha-button" disabled>
							<span class="icon" aria-hidden="true">{getIcon(contact)}</span>
							<span>{getLabel(contact)}</span>
							<span class="badge">CAPTCHA Required (Coming Soon)</span>
						</button>
						{#if contact.is_primary}
							<span class="primary-badge">Primary</span>
						{/if}
					</div>
				{/if}
			</div>
		{/each}
	</div>
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
		/* Highlight primary contacts */
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
		font-size: 1.5rem;
		line-height: 1;
	}

	.label {
		font-weight: 500;
		color: var(--text-secondary, #6b7280);
	}

	.value {
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		color: var(--accent-color, #3b82f6);
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

	/* Dark mode support */
	@media (prefers-color-scheme: dark) {
		.contact-link {
			border-color: #374151;
			background: #1f2937;
			color: #f9fafb;
		}

		.contact-link:hover {
			border-color: var(--accent-color, #3b82f6);
			background: #111827;
		}

		.contact-wrapper {
			border-color: #374151;
			background: #1f2937;
		}

		.label {
			color: #9ca3af;
		}

		.captcha-button {
			border-color: #374151;
			background: #1f2937;
			color: #9ca3af;
		}

		.badge {
			background: #374151;
		}
	}
</style>
