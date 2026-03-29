<script lang="ts">
	import { pb, type Experience } from '$lib/pocketbase';
	import { groupExperiencesByCompany, type CompanyGroup, type GroupedExperienceItem } from '$lib/experienceGrouping';
	import { formatDateRange, parseMarkdown } from '$lib/utils';
	import { t } from 'svelte-i18n';

	interface Props {
		items: Experience[];
		layout?: string;
	}

	let { items, layout = 'default' }: Props = $props();

	// Get translated "Present" text for date ranges
	let presentText = $derived($t('shared.time.present'));

	// Group experiences by company
	let groupedItems = $derived(groupExperiencesByCompany(items));

	function stripBulletPrefix(text: string): string {
		return text.replace(/^\s*[•\-\*–—·●◦▪▸►]\s*/, '').replace(/^\s*\d+[.)]\s+/, '');
	}

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

	function isGroup(entry: GroupedExperienceItem): entry is CompanyGroup {
		return 'type' in entry && entry.type === 'group';
	}
</script>

<section id="experience" class="mb-16" itemscope itemtype="https://schema.org/Person">
	<h2 class="section-title">{$t('public.sections.experience')}</h2>

	{#if layout === 'timeline'}
		<!-- Timeline Layout -->
		<div class="relative">
			<!-- Timeline line -->
			<div class="absolute left-4 top-0 bottom-0 w-0.5 bg-primary-200 dark:bg-primary-800"></div>

			<div class="space-y-8 pl-12">
				{#each groupedItems as entry (isGroup(entry) ? entry.groupId : entry.id)}
					{#if isGroup(entry)}
						<!-- Company group: header on main timeline + positions on main timeline -->
						<div class="relative animate-fade-in" itemscope itemtype="https://schema.org/Organization">
							<!-- Company header: large dot on main timeline -->
							<div class="absolute -left-12 w-8 h-8 bg-primary-700 rounded-full ring-4 ring-white dark:ring-gray-900 z-10 flex items-center justify-center">
								{#if entry.companyLogoUrl || entry.companyLogo}
									<img
										src={entry.companyLogoUrl}
										alt="{entry.company} logo"
										class="w-5 h-5 object-contain"
									/>
								{/if}
							</div>

							<!-- Company header content -->
							<div class="flex gap-4 items-start">
								{#if entry.companyLogoUrl || entry.companyLogo}
									<div class="shrink-0 w-10 h-10 rounded-lg overflow-hidden flex items-center justify-center {getLogoBgClass(entry.logoBackground) || 'bg-gray-100 dark:bg-gray-800'}">
										<img
											src={entry.companyLogoUrl}
											alt="{entry.company} logo"
											class="w-8 h-8 object-contain"
										/>
									</div>
								{/if}
								<div>
									<span class="text-base font-bold text-gray-900 dark:text-white" itemprop="name">
										{entry.company}
									</span>
									<div class="text-sm text-gray-500 dark:text-gray-400">
										{formatDateRange(entry.overallStartDate, entry.overallEndDate, presentText)}
										&nbsp;· {$t('public.experience.positions_count', { values: { count: entry.positions.length } })}
									</div>
								</div>
							</div>

							<!-- Positions: each with a small dot on the MAIN timeline -->
							<div class="mt-5 space-y-5">
								{#each entry.positions as pos (pos.id)}
									<article class="relative" itemprop="hasOccupation" itemscope itemtype="https://schema.org/Occupation">
										<!-- Small dot centered on the main timeline line -->
										<div class="absolute -left-[2.375rem] top-1 w-3 h-3 bg-primary-400 dark:bg-primary-600 rounded-full ring-2 ring-white dark:ring-gray-900 z-10"></div>

										<div>
											<h3 class="experience-group-title text-base font-semibold text-gray-900 dark:text-white" itemprop="name">
												{pos.title}
											</h3>
											<div class="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-sm text-gray-500 dark:text-gray-400 mt-0.5">
												<span class="font-medium">{formatDateRange(pos.start_date, pos.end_date, presentText)}</span>
												{#if pos.location}
													<span>• {pos.location}</span>
												{/if}
											</div>

											{#if pos.description}
												<div class="mt-2 prose-custom text-gray-600 dark:text-gray-300 text-sm" itemprop="description">
													{@html parseMarkdown(pos.description)}
												</div>
											{/if}

											{#if pos.bullets && pos.bullets.length > 0}
												<ul class="mt-1.5 space-y-1">
													{#each pos.bullets as bullet}
														<li class="flex items-start gap-2 text-gray-600 dark:text-gray-300 text-sm">
															<span class="text-primary-500 mt-0.5">•</span>
															<span>{stripBulletPrefix(bullet)}</span>
														</li>
													{/each}
												</ul>
											{/if}

											{#if pos.skills && pos.skills.length > 0}
												<div class="mt-2 flex flex-wrap gap-1.5">
													{#each pos.skills as skill}
														<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-full">
															{skill}
														</span>
													{/each}
												</div>
											{/if}
										</div>
									</article>
								{/each}
							</div>
						</div>

					{:else}
						<!-- Ungrouped item: render exactly as before -->
						<article class="relative animate-fade-in">
							<!-- Timeline node -->
							<div class="absolute -left-12 w-8 h-8 bg-primary-600 rounded-full ring-4 ring-white dark:ring-gray-900 z-10"></div>

							<!-- Content with optional logo as visual anchor -->
							<div class="flex gap-4">
								{#if entry.company_logo_url || entry.company_logo}
									<div class="shrink-0 w-10 h-10 rounded-lg overflow-hidden flex items-center justify-center {getLogoBgClass(entry.logo_background) || 'bg-gray-100 dark:bg-gray-800'}">
										<img
											src={entry.company_logo_url || pb.files.getUrl(entry, entry.company_logo!, { thumb: '40x40' })}
											alt="{entry.company} logo"
											class="w-8 h-8 object-contain"
										/>
									</div>
								{/if}
								<div class="flex-1 min-w-0">
									<div class="pb-2">
										<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
											{entry.title}
										</h3>
										<span class="text-primary-600 dark:text-primary-400 font-medium">
											{entry.company}
										</span>
										<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-500 dark:text-gray-400 mt-1">
											<span class="font-medium">{formatDateRange(entry.start_date, entry.end_date, presentText)}</span>
											{#if entry.location}
												<span>• {entry.location}</span>
											{/if}
										</div>
									</div>

									{#if entry.description}
										<div class="prose-custom text-gray-600 dark:text-gray-300 text-sm">
											{@html parseMarkdown(entry.description)}
										</div>
									{/if}

									{#if entry.bullets && entry.bullets.length > 0}
										<ul class="mt-2 space-y-1">
											{#each entry.bullets as bullet}
												<li class="flex items-start gap-2 text-gray-600 dark:text-gray-300 text-sm">
													<span class="text-primary-500 mt-0.5">•</span>
													<span>{stripBulletPrefix(bullet)}</span>
												</li>
											{/each}
										</ul>
									{/if}

									{#if entry.skills && entry.skills.length > 0}
										<div class="mt-3 flex flex-wrap gap-1.5">
											{#each entry.skills as skill}
												<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-full">
													{skill}
												</span>
											{/each}
										</div>
									{/if}
								</div>
							</div>
						</article>
					{/if}
				{/each}
			</div>
		</div>

	{:else if layout === 'compact'}
		<!-- Compact Layout -->
		<div class="divide-y divide-gray-200 dark:divide-gray-700">
			{#each groupedItems as entry (isGroup(entry) ? entry.groupId : entry.id)}
				{#if isGroup(entry)}
					<!-- Company group header + indented position list -->
					<div class="py-4 first:pt-0 animate-fade-in" itemscope itemtype="https://schema.org/Organization">
						<!-- Company header row -->
						<div class="flex flex-col sm:flex-row sm:items-baseline gap-1 sm:gap-4">
							<div class="flex-1 flex flex-wrap items-center gap-x-2">
								{#if entry.companyLogoUrl || entry.companyLogo}
									<img
										src={entry.companyLogoUrl}
										alt="{entry.company} logo"
										class="w-4 h-4 object-contain rounded {getLogoBgClass(entry.logoBackground)}"
									/>
								{/if}
								<span class="font-bold text-gray-900 dark:text-white" itemprop="name">
									{entry.company}
								</span>
								<span class="text-xs text-gray-400 dark:text-gray-500">· {$t('public.experience.positions_count', { values: { count: entry.positions.length } })}</span>
							</div>
							<span class="text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap">
								{formatDateRange(entry.overallStartDate, entry.overallEndDate, presentText)}
							</span>
						</div>

						<!-- Indented position list with accent border -->
						<div class="mt-2 ml-4 space-y-3 border-l-2 border-primary-200 dark:border-primary-800 pl-4">
							{#each entry.positions as pos (pos.id)}
								<article class="relative" itemprop="hasOccupation" itemscope itemtype="https://schema.org/Occupation">
									<!-- Dot on the left border -->
									<div class="absolute -left-[1.3125rem] top-1.5 w-2 h-2 bg-primary-400 dark:bg-primary-600 rounded-full"></div>
									<div class="flex flex-col sm:flex-row sm:items-baseline gap-0.5 sm:gap-3">
										<div class="flex-1">
											<h3 class="font-semibold text-gray-900 dark:text-white text-sm" itemprop="name">
												{pos.title}
											</h3>
										</div>
										<span class="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">
											{formatDateRange(pos.start_date, pos.end_date, presentText)}
										</span>
									</div>
									{#if pos.location}
										<p class="text-xs text-gray-500 dark:text-gray-400">{pos.location}</p>
									{/if}
									{#if pos.description}
										<div class="mt-1 prose-custom text-gray-600 dark:text-gray-300 text-sm" itemprop="description">
											{@html parseMarkdown(pos.description)}
										</div>
									{/if}
								</article>
							{/each}
						</div>
					</div>

				{:else}
					<!-- Ungrouped item: render exactly as before -->
					<article class="py-4 first:pt-0 animate-fade-in">
						<div class="flex flex-col sm:flex-row sm:items-baseline gap-1 sm:gap-4">
							<div class="flex-1 flex flex-wrap items-baseline gap-x-2">
								<div class="flex items-center gap-2">
									{#if entry.company_logo_url || entry.company_logo}
										<img
											src={entry.company_logo_url || pb.files.getUrl(entry, entry.company_logo!, { thumb: '20x20' })}
											alt="{entry.company} logo"
											class="w-4 h-4 object-contain rounded {getLogoBgClass(entry.logo_background)}"
										/>
									{/if}
									<h3 class="font-semibold text-gray-900 dark:text-white">
										{entry.title}
									</h3>
								</div>
								<span class="text-gray-500 dark:text-gray-400">{$t('public.experience.at')}</span>
								<span class="text-primary-600 dark:text-primary-400 font-medium">
									{entry.company}
								</span>
							</div>
							<span class="text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap">
								{formatDateRange(entry.start_date, entry.end_date, presentText)}
							</span>
						</div>

						{#if entry.location}
							<p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">{entry.location}</p>
						{/if}

						{#if entry.description}
							<div class="mt-2 prose-custom text-gray-600 dark:text-gray-300 text-sm">
								{@html parseMarkdown(entry.description)}
							</div>
						{/if}
					</article>
				{/if}
			{/each}
		</div>

	{:else}
		<!-- Default Layout (Cards) -->
		<div class="space-y-8">
			{#each groupedItems as entry (isGroup(entry) ? entry.groupId : entry.id)}
				{#if isGroup(entry)}
					<!-- Company group card: header + position sub-entries -->
					<article class="experience-group-card card p-6 animate-fade-in" itemscope itemtype="https://schema.org/Organization">
						<!-- Company header -->
						<div class="experience-group-header flex flex-col sm:flex-row sm:items-start gap-4">
							{#if entry.companyLogoUrl || entry.companyLogo}
								<div class="experience-group-logo flex-shrink-0">
									<img
										src={entry.companyLogoUrl}
										alt="{entry.company} logo"
										class="w-12 h-12 object-contain rounded {getLogoBgClass(entry.logoBackground)}"
									/>
								</div>
							{/if}
							<div class="flex-1">
								<h3 class="experience-group-company text-xl font-bold text-gray-900 dark:text-white" itemprop="name">
									{entry.company}
								</h3>
								<div class="experience-group-meta flex flex-wrap items-center gap-x-4 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
									<span>{formatDateRange(entry.overallStartDate, entry.overallEndDate, presentText)}</span>
									<span>· {$t('public.experience.positions_count', { values: { count: entry.positions.length } })}</span>
								</div>
							</div>
						</div>

						<!-- Positions with accent border and dot indicators -->
						<div class="experience-group-timeline mt-4 ml-[1.35rem] border-l-[3px] border-primary-200 dark:border-primary-800">
							{#each entry.positions as pos, posIdx (pos.id)}
								{#if posIdx > 0}
									<div class="border-t border-gray-100 dark:border-gray-800 ml-4"></div>
								{/if}
								<div class="experience-group-position relative py-4 pl-5" itemprop="hasOccupation" itemscope itemtype="https://schema.org/Occupation">
									<!-- Position dot on the accent border -->
									<div class="experience-group-dot absolute -left-[0.55rem] top-[1.375rem] w-4 h-4 bg-primary-400 dark:bg-primary-600 rounded-full"></div>
									<h4 class="experience-group-title text-base font-semibold text-gray-900 dark:text-white" itemprop="name">
										{pos.title}
									</h4>
									<div class="experience-group-date flex flex-wrap items-center gap-x-4 gap-y-1 mt-0.5 text-sm text-gray-500 dark:text-gray-400">
										<span>
											<meta itemprop="startDate" content={pos.start_date || ''} />
											<meta itemprop="endDate" content={pos.end_date || ''} />
											{formatDateRange(pos.start_date, pos.end_date, presentText)}
										</span>
										{#if pos.location}
											<span class="flex items-center gap-1">
												<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
												</svg>
												{pos.location}
											</span>
										{/if}
									</div>

									{#if pos.description}
										<div class="mt-3 prose-custom text-gray-600 dark:text-gray-300" itemprop="description">
											{@html parseMarkdown(pos.description)}
										</div>
									{/if}

									{#if pos.bullets && pos.bullets.length > 0}
										<ul class="experience-group-bullets mt-2 space-y-0" itemprop="responsibilities">
											{#each pos.bullets as bullet}
												<li class="flex items-start gap-2 text-gray-600 dark:text-gray-300">
													<span class="text-primary-500 mt-1">•</span>
													<span class="mt-[0.26rem]">{stripBulletPrefix(bullet)}</span>
												</li>
											{/each}
										</ul>
									{/if}

									{#if pos.skills && pos.skills.length > 0}
										<div class="experience-group-skills mt-3 flex flex-wrap gap-2">
											{#each pos.skills as skill}
												<span class="experience-group-skill px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-full" itemprop="skills">
													{skill}
												</span>
											{/each}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					</article>

				{:else}
					<!-- Ungrouped item: render exactly as before -->
					<article class="card p-6 animate-fade-in" itemprop="hasOccupation" itemscope itemtype="https://schema.org/Occupation">
						<div class="experience-group-header flex flex-col sm:flex-row sm:items-start gap-4">
							{#if entry.company_logo_url || entry.company_logo}
								<div class="experience-group-logo flex-shrink-0">
									<img
										src={entry.company_logo_url || pb.files.getUrl(entry, entry.company_logo!, { thumb: '48x48' })}
										alt="{entry.company} logo"
										class="w-12 h-12 object-contain rounded {getLogoBgClass(entry.logo_background)}"
									/>
								</div>
							{/if}
							<div class="flex-1">
								<h3 class="text-xl font-semibold text-gray-900 dark:text-white" itemprop="name">
									{entry.title}
								</h3>
								<p class="text-lg text-primary-600 dark:text-primary-400 font-medium" itemprop="occupationLocation" itemscope itemtype="https://schema.org/Organization">
									<span itemprop="name">{entry.company}</span>
								</p>
								<div class="experience-group-meta flex flex-wrap items-center gap-x-4 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
									<span>
										<meta itemprop="startDate" content={entry.start_date || ''} />
										<meta itemprop="endDate" content={entry.end_date || ''} />
										{formatDateRange(entry.start_date, entry.end_date, presentText)}
									</span>
									{#if entry.location}
										<span class="flex items-center gap-1">
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
											</svg>
											{entry.location}
										</span>
									{/if}
								</div>
							</div>
						</div>

						{#if entry.description}
							<div class="mt-4 prose-custom text-gray-600 dark:text-gray-300" itemprop="description">
								{@html parseMarkdown(entry.description)}
							</div>
						{/if}

						{#if entry.bullets && entry.bullets.length > 0}
							<ul class="mt-4 space-y-2" itemprop="responsibilities">
								{#each entry.bullets as bullet}
									<li class="flex items-start gap-2 text-gray-600 dark:text-gray-300">
										<span class="text-primary-500 mt-1">•</span>
										<span class="mt-[0.26rem]">{stripBulletPrefix(bullet)}</span>
									</li>
								{/each}
							</ul>
						{/if}

						{#if entry.skills && entry.skills.length > 0}
							<div class="mt-4 flex flex-wrap gap-2">
								{#each entry.skills as skill}
									<span class="experience-group-skill px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-full" itemprop="skills">
										{skill}
									</span>
								{/each}
							</div>
						{/if}
					</article>
				{/if}
			{/each}
		</div>
	{/if}
</section>
