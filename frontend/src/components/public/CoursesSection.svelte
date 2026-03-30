<script lang="ts">
	import { t } from 'svelte-i18n';
	import { parseMarkdown } from '$lib/utils';

	interface Props {
		items: any[];
		layout?: 'grid-3' | 'grid-2' | 'list' | 'featured';
		viewSlug?: string;
		showHeader?: boolean;
	}

	let { items = [], layout = 'grid-3', viewSlug = '', showHeader = true }: Props = $props();

	// Derived values for featured layout
	let featuredCourse = $derived(items[0]);
	let remainingCourses = $derived(items.slice(1));

	// Build the course URL with optional from parameter for back navigation
	function getCourseUrl(slug: string, isResume = false, course: any = null): string {
		let baseUrl = viewSlug ? `/courses/${slug}?from=${encodeURIComponent(viewSlug)}` : `/courses/${slug}`;

		// If this is a resume action and we have progress, go directly to learn page
		if (isResume && course?.is_purchased) {
			return `/courses/${slug}/learn`;
		}

		return baseUrl;
	}

	const coverImageUrl = (course: any) =>
		course.cover_image_url || '';

	function formatDifficulty(difficulty: string): string {
		if (!difficulty) return '';
		return difficulty.charAt(0).toUpperCase() + difficulty.slice(1);
	}

	function formatDuration(hours: number): string {
		if (!hours) return '';
		return $t('public.courses.hours', { values: { count: hours } });
	}

	// Get button text based on course state
	function getButtonText(course: any): string {
		if (course.is_purchased) {
			if (course.enable_certificates !== false && course.progress_percent >= 100) {
				return $t('public.courses.view_certificate');
			} else if (course.enable_progress !== false && course.progress_percent > 0) {
				return $t('public.courses.resume');
			}
			return $t('public.courses.start_course');
		} else if (course.access_tier === 'free') {
			return $t('public.courses.start_free');
		} else if (course.price) {
			return `${$t('public.courses.start')} · $${(course.price / 100).toFixed(2)}`;
		}
		return $t('public.courses.start');
	}

	// Difficulty badge color
	function difficultyColor(difficulty: string): string {
		switch (difficulty) {
			case 'beginner': return 'text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-900/20';
			case 'intermediate': return 'text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20';
			case 'advanced': return 'text-rose-700 dark:text-rose-400 bg-rose-50 dark:bg-rose-900/20';
			default: return 'text-stone-600 dark:text-stone-400 bg-stone-50 dark:bg-stone-800';
		}
	}
</script>

<section id="courses" class="mb-16">
	{#if showHeader}
		<h2 class="section-title">{$t('public.sections.courses')}</h2>
	{/if}

	{#if layout === 'featured' && items.length > 0}
		<!-- Featured layout: first course large, rest in grid -->
		<div class="space-y-8">
			<!-- Featured course (large) -->
			<article class="card p-0 overflow-hidden group animate-fade-in opacity-0" style="animation-fill-mode: forwards;">
				<div class="grid grid-cols-1 md:grid-cols-2">
					<!-- Image section -->
					<div class="relative">
						{#if coverImageUrl(featuredCourse)}
							<a href={getCourseUrl(featuredCourse.slug)} class="block aspect-video md:aspect-auto md:h-full overflow-hidden bg-stone-100 dark:bg-stone-700 relative">
								<img
									src={coverImageUrl(featuredCourse)}
									alt={featuredCourse.title}
									class="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-500 ease-out"
								/>
								<!-- Subtle bottom gradient overlay -->
								<div class="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-black/20 to-transparent pointer-events-none"></div>

								<!-- Gated content badge -->
								{#if featuredCourse.is_gated && !featuredCourse.is_purchased && featuredCourse.price}
									<div class="absolute top-3 right-3 backdrop-blur-md bg-white/70 dark:bg-stone-900/70 rounded-full px-2 py-1 shadow-lg border border-white/30 dark:border-stone-700/30 flex items-center gap-1.5">
										<svg class="w-3 h-3 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
										</svg>
										<span class="text-xs font-semibold text-stone-700 dark:text-stone-200">${(featuredCourse.price / 100).toFixed(2)}</span>
									</div>
								{:else if featuredCourse.access_tier === 'free'}
									<div class="absolute top-3 right-3 backdrop-blur-md bg-emerald-500/80 dark:bg-emerald-600/80 rounded-full px-2.5 py-1 shadow-lg border border-emerald-400/30">
										<span class="text-xs font-semibold text-white">{$t('public.courses.free')}</span>
									</div>
								{/if}

								<!-- Progress bar -->
								{#if featuredCourse.show_progress_bar !== false && featuredCourse.enable_progress !== false && featuredCourse.is_purchased && featuredCourse.progress_percent > 0 && featuredCourse.progress_percent < 100}
									<div class="absolute bottom-0 left-0 right-0 h-1 bg-black/20">
										<div
											class="h-full bg-primary-500 transition-all duration-500"
											style="width: {featuredCourse.progress_percent}%"
										></div>
									</div>
									{#if featuredCourse.lessons_completed && featuredCourse.total_lessons}
										<div class="absolute bottom-2.5 left-3 backdrop-blur-md bg-black/50 text-white text-xs rounded-full px-2.5 py-0.5 font-medium">
											{$t('public.courses.progress_count', { values: { completed: featuredCourse.lessons_completed, total: featuredCourse.total_lessons } })}
										</div>
									{/if}
								{/if}
							</a>
						{:else}
							<!-- Placeholder gradient header -->
							<div class="aspect-video md:aspect-auto md:h-full bg-gradient-to-br from-primary-500 via-primary-600 to-primary-800 dark:from-primary-700 dark:via-primary-800 dark:to-primary-950 flex items-center justify-center relative">
								<svg class="w-14 h-14 text-white/25" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
								</svg>
								{#if featuredCourse.is_gated && !featuredCourse.is_purchased && featuredCourse.price}
									<div class="absolute top-3 right-3 backdrop-blur-md bg-white/70 dark:bg-stone-900/70 rounded-full px-2 py-1 shadow-lg border border-white/30 dark:border-stone-700/30 flex items-center gap-1.5">
										<svg class="w-3 h-3 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
										</svg>
										<span class="text-xs font-semibold text-stone-700 dark:text-stone-200">${(featuredCourse.price / 100).toFixed(2)}</span>
									</div>
								{:else if featuredCourse.access_tier === 'free'}
									<div class="absolute top-3 right-3 backdrop-blur-md bg-emerald-500/80 dark:bg-emerald-600/80 rounded-full px-2.5 py-1 shadow-lg border border-emerald-400/30">
										<span class="text-xs font-semibold text-white">{$t('public.courses.free')}</span>
									</div>
								{/if}
							</div>
						{/if}
					</div>

					<!-- Content section -->
					<div class="p-6 flex flex-col">
						<h3 class="text-2xl font-semibold text-stone-900 dark:text-white line-clamp-2 leading-tight">
							<a href={getCourseUrl(featuredCourse.slug)} class="hover:text-primary-600 dark:hover:text-primary-400 transition-colors duration-200">
								{featuredCourse.title}
							</a>
						</h3>

						{#if featuredCourse.description}
							<div class="mt-3 text-base text-stone-600 dark:text-stone-300 line-clamp-3 leading-relaxed prose-custom">
								{@html parseMarkdown(featuredCourse.description)}
							</div>
						{/if}

						<!-- Metadata chips -->
						<div class="mt-4 flex flex-wrap items-center gap-2">
							{#if featuredCourse.difficulty}
								<span class="text-sm font-medium px-2.5 py-1 rounded-full {difficultyColor(featuredCourse.difficulty)}">
									{formatDifficulty(featuredCourse.difficulty)}
								</span>
							{/if}
							{#if featuredCourse.estimated_hours}
								<span class="text-sm text-stone-500 dark:text-stone-500 flex items-center gap-1">
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
									{formatDuration(featuredCourse.estimated_hours)}
								</span>
							{/if}
							{#if featuredCourse.total_lessons}
								<span class="text-sm text-stone-500 dark:text-stone-500 flex items-center gap-1">
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
									</svg>
									{$t('public.courses.lessons_count', { values: { count: featuredCourse.total_lessons } })}
								</span>
							{/if}
						</div>

						<!-- CTA button -->
						<div class="mt-auto pt-6">
							<a
								href={getCourseUrl(featuredCourse.slug, featuredCourse.is_purchased && featuredCourse.progress_percent > 0, featuredCourse)}
								class="inline-flex items-center gap-2 px-5 py-2.5 text-base font-medium rounded-lg
									bg-primary-600 text-white hover:bg-primary-700 active:bg-primary-800
									shadow-sm hover:shadow-md
									transition-all duration-200 ease-out
									hover:scale-[1.02] active:scale-[0.98]
									focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2"
							>
								{getButtonText(featuredCourse)}
								<svg class="w-4 h-4 transition-transform duration-200 group-hover:translate-x-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
								</svg>
							</a>
						</div>
					</div>
				</div>
			</article>

			<!-- Remaining courses in grid -->
			{#if remainingCourses.length > 0}
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
					{#each remainingCourses as course, index (course.id)}
						<article
							class="card p-0 overflow-hidden group animate-fade-in opacity-0 flex flex-col"
							style="animation-delay: {(index + 1) * 75}ms; animation-fill-mode: forwards;"
						>
							<!-- Cover image / gradient header -->
							{#if coverImageUrl(course)}
								<a href={getCourseUrl(course.slug)} class="block aspect-video overflow-hidden bg-stone-100 dark:bg-stone-700 relative">
									<img
										src={coverImageUrl(course)}
										alt={course.title}
										class="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-500 ease-out"
									/>
									<div class="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-black/20 to-transparent pointer-events-none"></div>

									{#if course.is_gated && !course.is_purchased && course.price}
										<div class="absolute top-3 right-3 backdrop-blur-md bg-white/70 dark:bg-stone-900/70 rounded-full px-2 py-1 shadow-lg border border-white/30 dark:border-stone-700/30 flex items-center gap-1.5">
											<svg class="w-3 h-3 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
											</svg>
											<span class="text-xs font-semibold text-stone-700 dark:text-stone-200">${(course.price / 100).toFixed(2)}</span>
										</div>
									{:else if course.access_tier === 'free'}
										<div class="absolute top-3 right-3 backdrop-blur-md bg-emerald-500/80 dark:bg-emerald-600/80 rounded-full px-2.5 py-1 shadow-lg border border-emerald-400/30">
											<span class="text-xs font-semibold text-white">{$t('public.courses.free')}</span>
										</div>
									{/if}

									{#if course.show_progress_bar !== false && course.enable_progress !== false && course.is_purchased && course.progress_percent > 0 && course.progress_percent < 100}
										<div class="absolute bottom-0 left-0 right-0 h-1 bg-black/20">
											<div
												class="h-full bg-primary-500 transition-all duration-500"
												style="width: {course.progress_percent}%"
											></div>
										</div>
										{#if course.lessons_completed && course.total_lessons}
											<div class="absolute bottom-2.5 left-3 backdrop-blur-md bg-black/50 text-white text-xs rounded-full px-2.5 py-0.5 font-medium">
												{$t('public.courses.progress_count', { values: { completed: course.lessons_completed, total: course.total_lessons } })}
											</div>
										{/if}
									{/if}
								</a>
							{:else}
								<div class="aspect-video bg-gradient-to-br from-primary-500 via-primary-600 to-primary-800 dark:from-primary-700 dark:via-primary-800 dark:to-primary-950 flex items-center justify-center relative">
									<svg class="w-14 h-14 text-white/25" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
									</svg>
									{#if course.is_gated && !course.is_purchased && course.price}
										<div class="absolute top-3 right-3 backdrop-blur-md bg-white/70 dark:bg-stone-900/70 rounded-full px-2 py-1 shadow-lg border border-white/30 dark:border-stone-700/30 flex items-center gap-1.5">
											<svg class="w-3 h-3 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
											</svg>
											<span class="text-xs font-semibold text-stone-700 dark:text-stone-200">${(course.price / 100).toFixed(2)}</span>
										</div>
									{:else if course.access_tier === 'free'}
										<div class="absolute top-3 right-3 backdrop-blur-md bg-emerald-500/80 dark:bg-emerald-600/80 rounded-full px-2.5 py-1 shadow-lg border border-emerald-400/30">
											<span class="text-xs font-semibold text-white">{$t('public.courses.free')}</span>
										</div>
									{/if}
								</div>
							{/if}

							<!-- Card content -->
							<div class="p-5 flex flex-col flex-1">
								<h3 class="text-lg text-stone-900 dark:text-white line-clamp-2 leading-snug">
									<a href={getCourseUrl(course.slug)} class="hover:text-primary-600 dark:hover:text-primary-400 transition-colors duration-200">
										{course.title}
									</a>
								</h3>

								{#if course.description}
									<div class="mt-2.5 text-sm text-stone-500 dark:text-stone-400 line-clamp-2 leading-relaxed prose-custom">
										{@html parseMarkdown(course.description)}
									</div>
								{/if}

								<!-- Metadata chips -->
								<div class="mt-3 flex flex-wrap items-center gap-2">
									{#if course.difficulty}
										<span class="text-xs font-medium px-2 py-0.5 rounded-full {difficultyColor(course.difficulty)}">
											{formatDifficulty(course.difficulty)}
										</span>
									{/if}
									{#if course.estimated_hours}
										<span class="text-xs text-stone-500 dark:text-stone-500 flex items-center gap-1">
											<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
											</svg>
											{formatDuration(course.estimated_hours)}
										</span>
									{/if}
									{#if course.total_lessons}
										<span class="text-xs text-stone-500 dark:text-stone-500 flex items-center gap-1">
											<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
											</svg>
											{$t('public.courses.lessons_count', { values: { count: course.total_lessons } })}
										</span>
									{/if}
								</div>

								<!-- CTA button -->
								<div class="mt-auto pt-5">
									<a
										href={getCourseUrl(course.slug, course.is_purchased && course.progress_percent > 0, course)}
										class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg
											bg-primary-600 text-white hover:bg-primary-700 active:bg-primary-800
											shadow-sm hover:shadow-md
											transition-all duration-200 ease-out
											hover:scale-[1.02] active:scale-[0.98]
											focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2"
									>
										{getButtonText(course)}
										<svg class="w-4 h-4 transition-transform duration-200 group-hover:translate-x-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
										</svg>
									</a>
								</div>
							</div>
						</article>
					{/each}
				</div>
			{/if}
		</div>
	{:else if layout === 'list'}
		<!-- List layout: single column, horizontal cards -->
		<div class="space-y-6">
			{#each items as course, index (course.id)}
				<article
					class="card p-0 overflow-hidden group animate-fade-in opacity-0 flex flex-col sm:flex-row"
					style="animation-delay: {index * 75}ms; animation-fill-mode: forwards;"
				>
					<!-- Image section -->
					{#if coverImageUrl(course)}
						<a href={getCourseUrl(course.slug)} class="block sm:w-48 shrink-0 aspect-video sm:aspect-auto overflow-hidden bg-stone-100 dark:bg-stone-700 relative">
							<img
								src={coverImageUrl(course)}
								alt={course.title}
								class="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-500 ease-out"
							/>
							<div class="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-black/20 to-transparent pointer-events-none"></div>

							{#if course.is_gated && !course.is_purchased && course.price}
								<div class="absolute top-3 right-3 backdrop-blur-md bg-white/70 dark:bg-stone-900/70 rounded-full px-2 py-1 shadow-lg border border-white/30 dark:border-stone-700/30 flex items-center gap-1.5">
									<svg class="w-3 h-3 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
									</svg>
									<span class="text-xs font-semibold text-stone-700 dark:text-stone-200">${(course.price / 100).toFixed(2)}</span>
								</div>
							{:else if course.access_tier === 'free'}
								<div class="absolute top-3 right-3 backdrop-blur-md bg-emerald-500/80 dark:bg-emerald-600/80 rounded-full px-2.5 py-1 shadow-lg border border-emerald-400/30">
									<span class="text-xs font-semibold text-white">{$t('public.courses.free')}</span>
								</div>
							{/if}

							{#if course.show_progress_bar !== false && course.enable_progress !== false && course.is_purchased && course.progress_percent > 0 && course.progress_percent < 100}
								<div class="absolute bottom-0 left-0 right-0 h-1 bg-black/20">
									<div
										class="h-full bg-primary-500 transition-all duration-500"
										style="width: {course.progress_percent}%"
									></div>
								</div>
								{#if course.lessons_completed && course.total_lessons}
									<div class="absolute bottom-2.5 left-3 backdrop-blur-md bg-black/50 text-white text-xs rounded-full px-2.5 py-0.5 font-medium">
										{$t('public.courses.progress_count', { values: { completed: course.lessons_completed, total: course.total_lessons } })}
									</div>
								{/if}
							{/if}
						</a>
					{:else}
						<div class="sm:w-48 shrink-0 aspect-video sm:aspect-auto bg-gradient-to-br from-primary-500 via-primary-600 to-primary-800 dark:from-primary-700 dark:via-primary-800 dark:to-primary-950 flex items-center justify-center relative">
							<svg class="w-14 h-14 text-white/25" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
							</svg>
							{#if course.is_gated && !course.is_purchased && course.price}
								<div class="absolute top-3 right-3 backdrop-blur-md bg-white/70 dark:bg-stone-900/70 rounded-full px-2 py-1 shadow-lg border border-white/30 dark:border-stone-700/30 flex items-center gap-1.5">
									<svg class="w-3 h-3 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
									</svg>
									<span class="text-xs font-semibold text-stone-700 dark:text-stone-200">${(course.price / 100).toFixed(2)}</span>
								</div>
							{:else if course.access_tier === 'free'}
								<div class="absolute top-3 right-3 backdrop-blur-md bg-emerald-500/80 dark:bg-emerald-600/80 rounded-full px-2.5 py-1 shadow-lg border border-emerald-400/30">
									<span class="text-xs font-semibold text-white">{$t('public.courses.free')}</span>
								</div>
							{/if}
						</div>
					{/if}

					<!-- Content section -->
					<div class="p-5 flex flex-col flex-1">
						<h3 class="text-lg text-stone-900 dark:text-white line-clamp-2 leading-snug">
							<a href={getCourseUrl(course.slug)} class="hover:text-primary-600 dark:hover:text-primary-400 transition-colors duration-200">
								{course.title}
							</a>
						</h3>

						{#if course.description}
							<div class="mt-2.5 text-sm text-stone-500 dark:text-stone-400 line-clamp-2 leading-relaxed prose-custom">
								{@html parseMarkdown(course.description)}
							</div>
						{/if}

						<!-- Metadata chips -->
						<div class="mt-3 flex flex-wrap items-center gap-2">
							{#if course.difficulty}
								<span class="text-xs font-medium px-2 py-0.5 rounded-full {difficultyColor(course.difficulty)}">
									{formatDifficulty(course.difficulty)}
								</span>
							{/if}
							{#if course.estimated_hours}
								<span class="text-xs text-stone-500 dark:text-stone-500 flex items-center gap-1">
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
									{formatDuration(course.estimated_hours)}
								</span>
							{/if}
							{#if course.total_lessons}
								<span class="text-xs text-stone-500 dark:text-stone-500 flex items-center gap-1">
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
									</svg>
									{$t('public.courses.lessons_count', { values: { count: course.total_lessons } })}
								</span>
							{/if}
						</div>

						<!-- CTA button -->
						<div class="mt-auto pt-5">
							<a
								href={getCourseUrl(course.slug, course.is_purchased && course.progress_percent > 0, course)}
								class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg
									bg-primary-600 text-white hover:bg-primary-700 active:bg-primary-800
									shadow-sm hover:shadow-md
									transition-all duration-200 ease-out
									hover:scale-[1.02] active:scale-[0.98]
									focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2"
							>
								{getButtonText(course)}
								<svg class="w-4 h-4 transition-transform duration-200 group-hover:translate-x-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
								</svg>
							</a>
						</div>
					</div>
				</article>
			{/each}
		</div>
	{:else}
		<!-- Grid layouts (grid-3, grid-2) -->
		<div class="grid grid-cols-1 {layout === 'grid-2' ? 'md:grid-cols-2' : 'md:grid-cols-2 lg:grid-cols-3'} gap-6">
			{#each items as course, index (course.id)}
				<article
					class="card p-0 overflow-hidden group animate-fade-in opacity-0 flex flex-col"
					style="animation-delay: {index * 75}ms; animation-fill-mode: forwards;"
				>
					<!-- Cover image / gradient header -->
					{#if coverImageUrl(course)}
						<a href={getCourseUrl(course.slug)} class="block aspect-video overflow-hidden bg-stone-100 dark:bg-stone-700 relative">
							<img
								src={coverImageUrl(course)}
								alt={course.title}
								class="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-500 ease-out"
							/>
							<!-- Subtle bottom gradient overlay for text readability -->
							<div class="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-black/20 to-transparent pointer-events-none"></div>

							<!-- Gated content badge - frosted glass -->
							{#if course.is_gated && !course.is_purchased && course.price}
								<div class="absolute top-3 right-3 backdrop-blur-md bg-white/70 dark:bg-stone-900/70 rounded-full px-2 py-1 shadow-lg border border-white/30 dark:border-stone-700/30 flex items-center gap-1.5">
									<svg class="w-3 h-3 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
									</svg>
									<span class="text-xs font-semibold text-stone-700 dark:text-stone-200">${(course.price / 100).toFixed(2)}</span>
								</div>
							{:else if course.access_tier === 'free'}
								<div class="absolute top-3 right-3 backdrop-blur-md bg-emerald-500/80 dark:bg-emerald-600/80 rounded-full px-2.5 py-1 shadow-lg border border-emerald-400/30">
									<span class="text-xs font-semibold text-white">{$t('public.courses.free')}</span>
								</div>
							{/if}

							<!-- Progress bar for enrolled students -->
							{#if course.show_progress_bar !== false && course.enable_progress !== false && course.is_purchased && course.progress_percent > 0 && course.progress_percent < 100}
								<div class="absolute bottom-0 left-0 right-0 h-1 bg-black/20">
									<div
										class="h-full bg-primary-500 transition-all duration-500"
										style="width: {course.progress_percent}%"
									></div>
								</div>
								{#if course.lessons_completed && course.total_lessons}
									<div class="absolute bottom-2.5 left-3 backdrop-blur-md bg-black/50 text-white text-xs rounded-full px-2.5 py-0.5 font-medium">
										{$t('public.courses.progress_count', { values: { completed: course.lessons_completed, total: course.total_lessons } })}
									</div>
								{/if}
							{/if}
						</a>
					{:else}
						<!-- Placeholder gradient header -->
						<div class="aspect-video bg-gradient-to-br from-primary-500 via-primary-600 to-primary-800 dark:from-primary-700 dark:via-primary-800 dark:to-primary-950 flex items-center justify-center relative">
							<svg class="w-14 h-14 text-white/25" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
							</svg>
							{#if course.is_gated && !course.is_purchased && course.price}
								<div class="absolute top-3 right-3 backdrop-blur-md bg-white/70 dark:bg-stone-900/70 rounded-full px-2 py-1 shadow-lg border border-white/30 dark:border-stone-700/30 flex items-center gap-1.5">
									<svg class="w-3 h-3 text-stone-600 dark:text-stone-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
									</svg>
									<span class="text-xs font-semibold text-stone-700 dark:text-stone-200">${(course.price / 100).toFixed(2)}</span>
								</div>
							{:else if course.access_tier === 'free'}
								<div class="absolute top-3 right-3 backdrop-blur-md bg-emerald-500/80 dark:bg-emerald-600/80 rounded-full px-2.5 py-1 shadow-lg border border-emerald-400/30">
									<span class="text-xs font-semibold text-white">{$t('public.courses.free')}</span>
								</div>
							{/if}
						</div>
					{/if}

					<!-- Card content -->
					<div class="p-5 flex flex-col flex-1">
						<h3 class="text-lg text-stone-900 dark:text-white line-clamp-2 leading-snug">
							<a href={getCourseUrl(course.slug)} class="hover:text-primary-600 dark:hover:text-primary-400 transition-colors duration-200">
								{course.title}
							</a>
						</h3>

						{#if course.description}
							<div class="mt-2.5 text-sm text-stone-500 dark:text-stone-400 line-clamp-2 leading-relaxed prose-custom">
								{@html parseMarkdown(course.description)}
							</div>
						{/if}

						<!-- Metadata chips -->
						<div class="mt-3 flex flex-wrap items-center gap-2">
							{#if course.difficulty}
								<span class="text-xs font-medium px-2 py-0.5 rounded-full {difficultyColor(course.difficulty)}">
									{formatDifficulty(course.difficulty)}
								</span>
							{/if}
							{#if course.estimated_hours}
								<span class="text-xs text-stone-500 dark:text-stone-500 flex items-center gap-1">
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
									{formatDuration(course.estimated_hours)}
								</span>
							{/if}
							{#if course.total_lessons}
								<span class="text-xs text-stone-500 dark:text-stone-500 flex items-center gap-1">
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
									</svg>
									{$t('public.courses.lessons_count', { values: { count: course.total_lessons } })}
								</span>
							{/if}
						</div>

						<!-- CTA button - pushed to bottom with flex -->
						<div class="mt-auto pt-5">
							<a
								href={getCourseUrl(course.slug, course.is_purchased && course.progress_percent > 0, course)}
								class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg
									bg-primary-600 text-white hover:bg-primary-700 active:bg-primary-800
									shadow-sm hover:shadow-md
									transition-all duration-200 ease-out
									hover:scale-[1.02] active:scale-[0.98]
									focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2"
							>
								{getButtonText(course)}
								<svg class="w-4 h-4 transition-transform duration-200 group-hover:translate-x-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
								</svg>
							</a>
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</section>
