<script lang="ts">
	import { t } from 'svelte-i18n';

	interface Props {
		/** Compact lists the examples in one line per group, for pages with work to get to */
		variant?: 'full' | 'compact';
		/** Adds a link to the create flow, for places without a create action nearby */
		showCreate?: boolean;
	}

	let { variant = 'full', showCreate = false }: Props = $props();

	const uid = $props.id();
	const headingId = `facet-use-cases-${uid}`;

	const groups = [
		{ key: 'standalone', examples: ['resume', 'proposal', 'event'] },
		{ key: 'site', examples: ['project', 'about', 'writing'] }
	];
</script>

<section class="card p-4 sm:p-6 mb-6" aria-labelledby={headingId}>
	<h2 id={headingId} class="text-lg font-semibold text-gray-900 dark:text-white">
		{$t('admin.views.use_cases.title')}
	</h2>
	<p class="mt-1 max-w-prose text-sm text-gray-600 dark:text-gray-300">
		{$t(
			variant === 'compact' ? 'admin.views.use_cases.compact_lead' : 'admin.views.use_cases.lead'
		)}
	</p>

	{#if variant === 'compact'}
		<details class="mt-3">
			<summary class="cursor-pointer text-sm font-medium text-primary-700 dark:text-primary-400">
				{$t('admin.views.use_cases.explore')}
			</summary>
			{@render examples()}
		</details>
	{:else}
		{@render examples()}
	{/if}
</section>

{#snippet examples()}
	<div class="use-case-groups mt-4">
		{#each groups as group (group.key)}
			<div class="min-w-0">
				<h3 class="text-sm font-semibold text-gray-900 dark:text-white">
					{$t(`admin.views.use_cases.${group.key}_title`)}
				</h3>
				<ul class="mt-1 divide-y divide-gray-100 dark:divide-gray-700">
					{#each group.examples as id (id)}
						<li class="py-2">
							<p class="text-sm font-medium text-gray-900 dark:text-white">
								{$t(`admin.views.use_cases.${id}_title`)}
							</p>
							<p class="text-sm text-gray-600 dark:text-gray-300">
								{$t(`admin.views.use_cases.${id}_text`)}
							</p>
						</li>
					{/each}
				</ul>
			</div>
		{/each}
	</div>

	<p class="mt-4 max-w-prose text-sm text-gray-600 dark:text-gray-300">
		{$t('admin.views.use_cases.access_note')}
	</p>
	<div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm">
		<a
			href="/admin/homepage#nav-section"
			class="text-primary-700 dark:text-primary-400 hover:underline"
		>
			{$t('admin.views.use_cases.nav_link')}
		</a>
		{#if showCreate}
			<a href="/admin/views/new" class="text-primary-700 dark:text-primary-400 hover:underline">
				{$t('admin.views.create_facet')}
			</a>
		{/if}
	</div>
{/snippet}

<style>
	/* Sized by available width, not the viewport: live preview halves the admin
	   column at desktop widths, where a viewport breakpoint would stay two-up. */
	.use-case-groups {
		display: grid;
		gap: 1rem 2rem;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 16rem), 1fr));
	}
</style>
