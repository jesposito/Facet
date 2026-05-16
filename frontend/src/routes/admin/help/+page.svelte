<script lang="ts">
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';

	// Each section has 3–5 FAQ-style Q&A pairs, rendered as native <details>/<summary>
	// for built-in keyboard accessibility + progressive enhancement. The page only
	// surfaces features that actually exist on self-hosted Facet — no SaaS content
	// (no billing, no custom domains, no managed AI credits, no member portal, no
	// API keys, no newsletter drip sequences, no coupons).
	interface QA {
		qKey: string;
		aKey: string;
	}
	interface HelpSection {
		slug: string;
		titleKey: string;
		introKey?: string;
		adminPath?: string;
		adminLabelKey?: string;
		items: QA[];
	}

	const sections: HelpSection[] = [
		{
			slug: 'getting-started',
			titleKey: 'admin.help.getting_started.title',
			introKey: 'admin.help.getting_started.intro',
			adminPath: '/admin',
			adminLabelKey: 'admin.help.getting_started.open_dashboard',
			items: [
				{ qKey: 'admin.help.getting_started.q_tour', aKey: 'admin.help.getting_started.a_tour' },
				{ qKey: 'admin.help.getting_started.q_profile', aKey: 'admin.help.getting_started.a_profile' },
				{ qKey: 'admin.help.getting_started.q_preview', aKey: 'admin.help.getting_started.a_preview' },
				{ qKey: 'admin.help.getting_started.q_encryption', aKey: 'admin.help.getting_started.a_encryption' }
			]
		},
		{
			slug: 'homepage',
			titleKey: 'admin.help.homepage.title',
			introKey: 'admin.help.homepage.intro',
			adminPath: '/admin/homepage',
			adminLabelKey: 'admin.help.homepage.open_editor',
			items: [
				{ qKey: 'admin.help.homepage.q_brand', aKey: 'admin.help.homepage.a_brand' },
				{ qKey: 'admin.help.homepage.q_hero', aKey: 'admin.help.homepage.a_hero' },
				{ qKey: 'admin.help.homepage.q_nav', aKey: 'admin.help.homepage.a_nav' },
				{ qKey: 'admin.help.homepage.q_sections', aKey: 'admin.help.homepage.a_sections' },
				{ qKey: 'admin.help.homepage.q_visibility', aKey: 'admin.help.homepage.a_visibility' }
			]
		},
		{
			slug: 'facets',
			titleKey: 'admin.help.facets.title',
			introKey: 'admin.help.facets.intro',
			adminPath: '/admin/views',
			adminLabelKey: 'admin.help.facets.open_list',
			items: [
				{ qKey: 'admin.help.facets.q_what', aKey: 'admin.help.facets.a_what' },
				{ qKey: 'admin.help.facets.q_create', aKey: 'admin.help.facets.a_create' },
				{ qKey: 'admin.help.facets.q_search', aKey: 'admin.help.facets.a_search' },
				{ qKey: 'admin.help.facets.q_overrides', aKey: 'admin.help.facets.a_overrides' },
				{ qKey: 'admin.help.facets.q_visibility', aKey: 'admin.help.facets.a_visibility' }
			]
		},
		{
			slug: 'content',
			titleKey: 'admin.help.content.title',
			introKey: 'admin.help.content.intro',
			adminPath: '/admin/posts',
			adminLabelKey: 'admin.help.content.open_posts',
			items: [
				{ qKey: 'admin.help.content.q_editor', aKey: 'admin.help.content.a_editor' },
				{ qKey: 'admin.help.content.q_shortcodes', aKey: 'admin.help.content.a_shortcodes' },
				{ qKey: 'admin.help.content.q_media', aKey: 'admin.help.content.a_media' },
				{ qKey: 'admin.help.content.q_drafts', aKey: 'admin.help.content.a_drafts' },
				{ qKey: 'admin.help.content.q_tags', aKey: 'admin.help.content.a_tags' }
			]
		},
		{
			slug: 'ai',
			titleKey: 'admin.help.ai.title',
			introKey: 'admin.help.ai.intro',
			adminPath: '/admin/settings/integrations',
			adminLabelKey: 'admin.help.ai.open_integrations',
			items: [
				{ qKey: 'admin.help.ai.q_byo', aKey: 'admin.help.ai.a_byo' },
				{ qKey: 'admin.help.ai.q_providers', aKey: 'admin.help.ai.a_providers' },
				{ qKey: 'admin.help.ai.q_usage', aKey: 'admin.help.ai.a_usage' },
				{ qKey: 'admin.help.ai.q_local', aKey: 'admin.help.ai.a_local' }
			]
		},
		{
			slug: 'settings',
			titleKey: 'admin.help.settings.title',
			introKey: 'admin.help.settings.intro',
			adminPath: '/admin/settings/site',
			adminLabelKey: 'admin.help.settings.open_site',
			items: [
				{ qKey: 'admin.help.settings.q_appearance', aKey: 'admin.help.settings.a_appearance' },
				{ qKey: 'admin.help.settings.q_language', aKey: 'admin.help.settings.a_language' },
				{ qKey: 'admin.help.settings.q_smtp', aKey: 'admin.help.settings.a_smtp' },
				{ qKey: 'admin.help.settings.q_backups', aKey: 'admin.help.settings.a_backups' },
				{ qKey: 'admin.help.settings.q_audit', aKey: 'admin.help.settings.a_audit' }
			]
		}
	];
</script>

<svelte:head>
	<title>{$t('admin.help.page_title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<header class="mb-8">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">
			{$t('admin.help.page_title')}
		</h1>
		<p class="text-sm text-gray-600 dark:text-gray-400">
			{$t('admin.help.page_intro')}
		</p>
	</header>

	{#each sections as section (section.slug)}
		<section id={section.slug} class="mb-10 scroll-mt-20" aria-labelledby="{section.slug}-heading">
			<div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-2 mb-3">
				<h2
					id="{section.slug}-heading"
					class="text-lg font-semibold text-gray-900 dark:text-white"
				>
					{$t(section.titleKey)}
				</h2>
				{#if section.adminPath && section.adminLabelKey}
					<a
						href={section.adminPath}
						class="inline-flex items-center gap-1 text-sm text-primary-600 dark:text-primary-400 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-500 rounded"
					>
						<span>{$t(section.adminLabelKey)}</span>
						<svg
							class="w-4 h-4 shrink-0"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-hidden="true"
						>
							<line x1="7" y1="17" x2="17" y2="7"></line>
							<polyline points="7 7 17 7 17 17"></polyline>
						</svg>
					</a>
				{/if}
			</div>

			{#if section.introKey}
				<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
					{$t(section.introKey)}
				</p>
			{/if}

			<div class="space-y-2">
				{#each section.items as item (item.qKey)}
					<details
						class="card overflow-hidden group [&_summary::-webkit-details-marker]:hidden"
					>
						<summary
							class="flex items-center justify-between gap-3 px-4 py-3 cursor-pointer list-none hover:bg-gray-50 dark:hover:bg-gray-700/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-inset"
						>
							<strong class="font-medium text-sm text-gray-900 dark:text-white">
								{$t(item.qKey)}
							</strong>
							<svg
								class="w-4 h-4 text-gray-400 shrink-0 transition-transform duration-200 group-open:rotate-180"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
								aria-hidden="true"
							>
								<polyline points="6 9 12 15 18 9"></polyline>
							</svg>
						</summary>
						<div class="px-4 pb-4 pt-1 text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
							{$t(item.aKey)}
						</div>
					</details>
				{/each}
			</div>
		</section>
	{/each}

	<aside
		class="mt-10 mb-4 rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 p-5"
		aria-labelledby="help-more-heading"
	>
		<h2 id="help-more-heading" class="text-sm font-semibold text-gray-900 dark:text-white mb-1">
			{$t('admin.help.more.title')}
		</h2>
		<p class="text-sm text-gray-600 dark:text-gray-400">
			{$t('admin.help.more.intro')}
		</p>
		<ul class="mt-3 space-y-1.5 text-sm">
			<li>
				<a
					href="/admin/settings/about"
					class="text-primary-600 dark:text-primary-400 hover:underline"
				>
					{$t('admin.help.more.open_about')}
				</a>
			</li>
			<li>
				<a
					href="https://github.com/jesposito/Facet"
					target="_blank"
					rel="noopener noreferrer"
					class="text-primary-600 dark:text-primary-400 hover:underline"
				>
					{$t('admin.help.more.open_repo')}
					<span class="sr-only">{$t('shared.aria.opens_in_new_tab')}</span>
				</a>
			</li>
		</ul>
	</aside>
</div>
