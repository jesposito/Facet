<script lang="ts">
	import type { Profile } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';

	interface Props {
		profile: Profile | null;
	}

	let { profile }: Props = $props();

	const year = new Date().getFullYear();

	// Pick a primary contact link for the Soft Premium CTA band.
	// Prefer an explicit email link, then fall back to the first contact link.
	// Only these schemes may be used as an href — blocks javascript:/data: etc.
	// from an operator-supplied (or imported) contact URL becoming an XSS vector.
	const SAFE_CTA_SCHEMES = ['http:', 'https:', 'mailto:', 'tel:'];
	const ctaLink = $derived.by(() => {
		const links = profile?.contact_links ?? [];
		if (links.length === 0) return null;
		const email = links.find((l) => l.type === 'email');
		const chosen = email ?? links[0];
		if (!chosen?.url) return null;
		const raw =
			chosen.type === 'email' && !chosen.url.startsWith('mailto:')
				? `mailto:${chosen.url}`
				: chosen.url;
		let parsed: URL;
		try {
			// base handles scheme-relative/relative inputs deterministically.
			parsed = new URL(raw, 'https://localhost');
		} catch {
			return null;
		}
		if (!SAFE_CTA_SCHEMES.includes(parsed.protocol)) return null;
		const external = parsed.protocol === 'http:' || parsed.protocol === 'https:';
		return { href: raw, external };
	});
</script>

<footer class="border-t border-stone-200/60 dark:border-stone-700/40 bg-stone-50/80 dark:bg-stone-900/50 backdrop-blur-sm">
	<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
		<!-- Soft Premium CTA band: warm editorial card above the brand row.
		     Gated to soft-premium via scoped CSS (hidden in classic so the
		     classic footer is byte-identical). Decorative-only; the CTA itself
		     is a real link with discernible text. -->
		{#if ctaLink}
			<section class="cta-band" aria-labelledby="footer-cta-headline">
				<span class="cta-band__grain grain" aria-hidden="true"></span>
				<div class="cta-band__inner">
					<h2 id="footer-cta-headline" class="cta-band__headline font-accent">
						{$t('public.footer.cta_headline')}
					</h2>
					<a
						class="cta-band__link"
						href={ctaLink.href}
						target={ctaLink.external ? '_blank' : undefined}
						rel={ctaLink.external ? 'noopener noreferrer' : undefined}
					>
						{$t('public.footer.cta_link')}
						<svg
							class="cta-band__arrow"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							aria-hidden="true"
						>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M13 6l6 6-6 6" />
						</svg>
					</a>
				</div>
			</section>
		{/if}

		<div class="flex flex-col sm:flex-row items-center justify-between gap-5">
			<!-- Brand text with serif treatment -->
			<div class="text-center sm:text-start">
				<p class="text-stone-800 dark:text-stone-200 text-sm font-medium">
					<span class="font-serif text-base">{profile?.name || 'Facet'}</span>
				</p>
				<p class="text-stone-500 dark:text-stone-400 text-xs mt-0.5">
					&copy; {year} {$t('public.footer.all_rights_reserved')}
				</p>
			</div>

			<!-- Social links as rounded hover buttons -->
			{#if profile?.contact_links}
				<div class="flex items-center gap-2">
					{#each profile.contact_links as link}
						<a
							href={link.url}
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center justify-center w-11 h-11 rounded-xl text-stone-500 dark:text-stone-400 hover:text-stone-700 dark:hover:text-stone-200 hover:bg-stone-200/60 dark:hover:bg-stone-700/40 transition-all duration-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-600 dark:focus-visible:ring-primary-400 focus-visible:ring-offset-2 focus-visible:ring-offset-stone-50 dark:focus-visible:ring-offset-stone-900"
							aria-label={link.type}
						>
							{#if link.type === 'github'}
								<svg class="w-[1.125rem] h-[1.125rem]" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
									<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
								</svg>
							{:else if link.type === 'linkedin'}
								<svg class="w-[1.125rem] h-[1.125rem]" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
									<path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433c-1.144 0-2.063-.926-2.063-2.065 0-1.138.92-2.063 2.063-2.063 1.14 0 2.064.925 2.064 2.063 0 1.139-.925 2.065-2.064 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/>
								</svg>
							{:else if link.type === 'email'}
								<svg class="w-[1.125rem] h-[1.125rem]" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
								</svg>
							{/if}
						</a>
					{/each}
				</div>
			{/if}
		</div>

		<div class="mt-6 text-center">
			<p class="text-xs text-stone-500 dark:text-stone-300">
				{@html $t('public.footer.powered_by', { values: { link: '<a href="https://github.com/jesposito/Facet" class="hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-600 dark:focus-visible:ring-primary-400 rounded-sm">Facet</a>' } })}
			</p>
		</div>
	</div>
</footer>

<style>
	/* Classic mode: the CTA band renders no visible chrome (byte-identical
	   classic footer). It only appears under the opt-in Soft Premium design. */
	.cta-band {
		display: none;
	}

	:global([data-design='soft-premium']) .cta-band {
		position: relative;
		display: block;
		overflow: hidden;
		margin-bottom: 2rem;
		padding: 1.75rem 1.5rem;
		border-radius: var(--r-5, 1.5rem);
		/* Dark editorial card built from the warm ink token. */
		background:
			radial-gradient(
				120% 140% at 100% 0%,
				rgb(var(--color-primary-900-rgb) / 0.55),
				transparent 60%
			),
			var(--ink, #2a2522);
		color: var(--surface, #ffffff);
	}

	:global([data-design='soft-premium']) .cta-band__grain {
		position: absolute;
		inset: 0;
		border-radius: inherit;
		opacity: 0.5;
		pointer-events: none;
	}

	:global([data-design='soft-premium']) .cta-band__inner {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 1rem;
	}

	@media (min-width: 640px) {
		:global([data-design='soft-premium']) .cta-band__inner {
			flex-direction: row;
			align-items: center;
			justify-content: space-between;
			gap: 1.5rem;
		}
	}

	:global([data-design='soft-premium']) .cta-band__headline {
		margin: 0;
		font-size: clamp(1.375rem, 2.5vw, 1.875rem);
		font-weight: 600;
		line-height: 1.15;
		letter-spacing: -0.01em;
		color: var(--surface, #ffffff);
	}

	:global([data-design='soft-premium']) .cta-band__link {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		flex-shrink: 0;
		padding: 0.6875rem 1.25rem;
		border-radius: var(--r-full, 9999px);
		background: var(--surface, #ffffff);
		color: var(--ink, #2a2522);
		font-size: 0.9375rem;
		font-weight: 600;
		text-decoration: none;
		transition:
			transform 0.18s ease,
			box-shadow 0.18s ease,
			opacity 0.18s ease;
	}

	:global([data-design='soft-premium']) .cta-band__link:hover {
		transform: translateY(-1px);
		box-shadow: 0 8px 24px rgb(0 0 0 / 0.25);
		opacity: 0.96;
	}

	:global([data-design='soft-premium']) .cta-band__link:focus-visible {
		outline: 2px solid var(--surface, #ffffff);
		outline-offset: 2px;
	}

	:global([data-design='soft-premium']) .cta-band__arrow {
		width: 1.125rem;
		height: 1.125rem;
		transition: transform 0.18s ease;
	}

	:global([data-design='soft-premium']) .cta-band__link:hover .cta-band__arrow {
		transform: translateX(2px);
	}

	@media (prefers-reduced-motion: reduce) {
		:global([data-design='soft-premium']) .cta-band__link,
		:global([data-design='soft-premium']) .cta-band__arrow {
			transition: none;
		}
	}
</style>
