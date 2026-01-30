<script lang="ts">
	import { pb, type Education } from '$lib/pocketbase';
	import { formatDateRange, parseMarkdown } from '$lib/utils';
	import { t } from 'svelte-i18n';

	interface Props {
		items: Education[];
		layout?: string;
	}

	let { items, layout = 'default' }: Props = $props();

	// Get translated "Present" text for date ranges
	let presentText = $derived($t('shared.time.present'));

	// Get background class for logo based on logo_background field
	function getLogoBgClass(bg: string | undefined): string {
		switch (bg) {
			case 'white': return 'bg-white';
			case 'light-gray': return 'bg-gray-100';
			case 'dark-gray': return 'bg-gray-700';
			case 'black': return 'bg-black';
			case 'accent': return 'bg-primary-500';
			default: return '';
		}
	}
</script>

<section id="education" class="mb-16">
	<h2 class="section-title">{$t('public.sections.education')}</h2>

	{#if layout === 'timeline'}
		<!-- Timeline Layout -->
		<div class="relative">
			<div class="absolute left-4 top-0 bottom-0 w-0.5 bg-primary-200 dark:bg-primary-800"></div>

			<div class="space-y-6 pl-12">
				{#each items as item (item.id)}
					<article class="relative animate-fade-in">
						<!-- Timeline node (small accent dot) -->
						<div class="absolute -left-12 w-8 h-8 flex items-center justify-center z-10">
							<div class="w-3 h-3 bg-primary-600 rounded-full ring-4 ring-white dark:ring-gray-900"></div>
						</div>

						<!-- Content with optional logo as visual anchor -->
						<div class="flex gap-4">
							{#if item.institution_logo_url || item.institution_logo}
								<div class="shrink-0 w-10 h-10 rounded-lg overflow-hidden flex items-center justify-center {getLogoBgClass(item.logo_background) || 'bg-gray-100 dark:bg-gray-800'}">
									<img
										src={item.institution_logo_url || pb.files.getUrl(item, item.institution_logo!, { thumb: '40x40' })}
										alt="{item.institution} logo"
										class="w-8 h-8 object-contain"
									/>
								</div>
							{/if}
							<div class="flex-1 min-w-0">
								<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
									{item.institution}
								</h3>
								{#if item.degree || item.field}
									<p class="text-primary-600 dark:text-primary-400">
										{[item.degree, item.field].filter(Boolean).join(' in ')}
									</p>
								{/if}
								<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
									{formatDateRange(item.start_date, item.end_date, presentText)}
								</p>

								{#if item.description}
									<div class="mt-2 prose-custom text-gray-600 dark:text-gray-300 text-sm">
										{@html parseMarkdown(item.description)}
									</div>
								{/if}
							</div>
						</div>
					</article>
				{/each}
			</div>
		</div>

	{:else}
		<!-- Default Layout (Cards with icons) -->
		<div class="space-y-6">
			{#each items as item (item.id)}
				<article class="card p-6 animate-fade-in">
					<div class="flex items-start gap-4">
						{#if item.institution_logo_url || item.institution_logo}
							<div class="shrink-0">
								<img
									src={item.institution_logo_url || pb.files.getUrl(item, item.institution_logo!, { thumb: '48x48' })}
									alt="{item.institution} logo"
									class="w-12 h-12 object-contain rounded {getLogoBgClass(item.logo_background)}"
								/>
							</div>
						{:else}
							<div class="shrink-0 w-12 h-12 rounded-lg bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
								<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path d="M12 14l9-5-9-5-9 5 9 5z" />
									<path d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5zm0 0l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14zm-4 6v-7.5l4-2.222" />
								</svg>
							</div>
						{/if}

						<div class="flex-1">
							<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
								{item.institution}
							</h3>
							{#if item.degree || item.field}
								<p class="text-primary-600 dark:text-primary-400">
									{[item.degree, item.field].filter(Boolean).join(' in ')}
								</p>
							{/if}
							<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
								{formatDateRange(item.start_date, item.end_date, presentText)}
							</p>

							{#if item.description}
								<div class="mt-3 prose-custom text-gray-600 dark:text-gray-300 text-sm">
									{@html parseMarkdown(item.description)}
								</div>
							{/if}
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</section>
