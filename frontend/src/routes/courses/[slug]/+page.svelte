<script lang="ts">
	import { locale, t } from 'svelte-i18n';
	import type { PageData } from './$types';
	import { parseMarkdown, formatDate } from '$lib/utils';
	import { formatPrice, flattenLessons, isReadable, type LessonState } from '$lib/commerce';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import VisibilityBadge from '$components/shared/VisibilityBadge.svelte';
	import PurchaseGate from '$components/public/PurchaseGate.svelte';
	import EnrollmentForm from '$components/public/course/EnrollmentForm.svelte';
	import CertificatePanel from '$components/public/course/CertificatePanel.svelte';
	import { siteFeatures } from '$lib/stores/features';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();
	let course = $derived(data.course);

	let lessons = $derived(flattenLessons(course));
	let firstReadable = $derived(lessons.find((l) => isReadable(l.state)) ?? null);
	let isPaidLocked = $derived(course.access_tier === 'paid' && !course.has_access);
	let needsEnrollment = $derived(
		course.access_tier === 'free' && course.require_enrollment && !course.has_access
	);
	let hasPreview = $derived(lessons.some((l) => l.state === 'preview'));
	let resumeLesson = $derived(
		data.progress
			? (lessons.find((l) => isReadable(l.state) && !data.progress!.completedIds.includes(l.id)) ??
					null)
			: null
	);

	function learnUrl(lessonId: string) {
		return `/courses/${encodeURIComponent(course.slug)}/learn?lesson=${encodeURIComponent(lessonId)}`;
	}

	const stateLabel = $derived<Record<LessonState, string>>({
		available: $t('public.commerce.course.available'),
		preview: $t('public.commerce.course.preview'),
		locked: $t('public.commerce.course.locked'),
		drip_locked: $t('public.commerce.course.drip_locked'),
		sequential_locked: $t('public.commerce.course.sequential_locked')
	});

	let metaDescription = $derived(
		course.description
			.replace(/[#*_`>[\]()!]/g, '')
			.replace(/\s+/g, ' ')
			.trim()
			.slice(0, 160) || course.title
	);
</script>

<svelte:head>
	<title>{course.title} | {$t('public.commerce.course.courses')}</title>
	<meta name="description" content={metaDescription} />
	<link rel="canonical" href="/courses/{course.slug}" />
	<meta property="og:title" content={course.title} />
	<meta property="og:description" content={metaDescription} />
	<meta property="og:type" content="website" />
	{#if course.cover_image_url}
		<meta property="og:image" content={course.cover_image_url} />
	{/if}
	{#if course.visibility !== 'public' || course.is_draft}
		<meta name="robots" content="noindex" />
	{/if}
</svelte:head>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<div class="fixed top-4 right-4 z-40 flex items-center gap-2 print:hidden">
		<ThemeToggle />
	</div>

	<header class="relative bg-gradient-to-br from-gray-900 to-gray-800 text-white">
		{#if course.cover_image_url}
			<div class="absolute inset-0">
				<img src={course.cover_image_url} alt="" class="w-full h-full object-cover opacity-20" />
				<div
					class="absolute inset-0 bg-gradient-to-t from-gray-900 via-gray-900/80 to-transparent"
				></div>
			</div>
		{/if}
		<div class="relative max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16">
			<a
				href="/"
				class="inline-flex items-center gap-2 text-sm text-gray-300 hover:text-white mb-6 rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-white"
			>
				<svg
					class="w-4 h-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M15 19l-7-7 7-7"
					/>
				</svg>
				{$t('public.commerce.course.back_home')}
			</a>
			<div class="flex flex-wrap items-center gap-2 mb-3">
				{#if course.access_tier === 'free'}
					<span class="text-xs font-semibold px-2.5 py-1 rounded-full bg-emerald-500/80 text-white"
						>{$t('public.commerce.course.free')}</span
					>
				{:else}
					<span class="text-xs font-semibold px-2.5 py-1 rounded-full bg-white/15 text-white"
						>{formatPrice(course.price, $locale)}</span
					>
				{/if}
				{#if course.has_access && course.access_tier === 'paid'}
					<span class="text-xs font-semibold px-2.5 py-1 rounded-full bg-primary-500/80 text-white"
						>{$t('public.commerce.course.purchased')}</span
					>
				{/if}
				{#if course.visibility !== 'public' || course.is_draft}
					<VisibilityBadge visibility={course.visibility} isDraft={course.is_draft} />
				{/if}
			</div>
			<h1 class="text-3xl sm:text-4xl font-bold leading-tight">{course.title}</h1>
			<dl class="mt-4 flex flex-wrap gap-x-6 gap-y-2 text-sm text-gray-300">
				{#if course.instructor_name}
					<div>
						<dt class="sr-only">{$t('public.commerce.course.instructor')}</dt>
						<dd>
							{$t('public.commerce.course.with', { values: { name: course.instructor_name } })}
						</dd>
					</div>
				{/if}
				{#if course.difficulty}
					<div>
						<dt class="sr-only">{$t('public.commerce.course.level')}</dt>
						<dd class="capitalize">{course.difficulty}</dd>
					</div>
				{/if}
				{#if course.estimated_hours}
					<div>
						<dt class="sr-only">{$t('public.commerce.course.length')}</dt>
						<dd>
							{$t('public.commerce.course.hours', { values: { count: course.estimated_hours } })}
						</dd>
					</div>
				{/if}
				<div>
					<dt class="sr-only">{$t('public.commerce.course.lessons')}</dt>
					<dd>
						{$t('public.commerce.course.lesson_count', { values: { count: course.total_lessons } })}
					</dd>
				</div>
			</dl>
		</div>
	</header>

	<main id="main-content" class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12 space-y-10">
		{#if course.description}
			<section
				class="rich-content-safe prose prose-lg dark:prose-invert max-w-none prose-a:text-primary-600 dark:prose-a:text-primary-400"
			>
				{@html parseMarkdown(course.description)}
			</section>
		{/if}

		{#if !$siteFeatures.courses && (isPaidLocked || needsEnrollment)}
			<!-- Courses are switched off: no new purchases or enrolments.
			     People who already have access never reach this branch. -->
			<p
				class="rounded-lg border border-gray-200 bg-white p-4 text-gray-700 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200"
				data-testid="courses-closed"
			>
				{$t('public.commerce.course.closed')}
			</p>
		{:else if isPaidLocked}
			<PurchaseGate
				contentType="courses"
				contentId={course.id}
				slug={course.slug}
				title={course.title}
				price={course.price}
				summary={hasPreview
					? 'Buy the course to unlock every lesson. The free preview lessons below are open now.'
					: 'Buy the course to unlock every lesson.'}
			/>
		{:else if needsEnrollment}
			<EnrollmentForm
				slug={course.slug}
				title={course.title}
				intro={hasPreview
					? "It's free. We'll email you a link to confirm the address, and the course opens when you follow it. The free preview lessons below are open now."
					: ''}
			/>
		{:else if firstReadable}
			<div class="flex flex-wrap items-center gap-4">
				<a href={learnUrl(resumeLesson?.id ?? firstReadable.id)} class="btn btn-primary">
					{data.progress?.completedIds.length ? 'Continue learning' : 'Start learning'}
				</a>
				{#if data.progress && course.show_progress_bar}
					<div class="min-w-48 flex-1" data-testid="course-progress">
						<div class="flex justify-between text-sm text-gray-600 dark:text-gray-300">
							<span id="course-progress-label">{$t('public.commerce.course.progress')}</span>
							<span>{Math.round(data.progress.percent)}%</span>
						</div>
						<div
							class="mt-1 h-2 rounded-full bg-gray-200 dark:bg-gray-700"
							role="progressbar"
							aria-labelledby="course-progress-label"
							aria-valuemin="0"
							aria-valuemax="100"
							aria-valuenow={Math.round(data.progress.percent)}
						>
							<div
								class="h-2 rounded-full bg-primary-600"
								style="width: {Math.min(100, data.progress.percent)}%"
							></div>
						</div>
					</div>
				{/if}
			</div>
		{/if}

		{#if data.certificate}
			<CertificatePanel
				certificate={data.certificate}
				courseCompleted={data.progress?.completed ?? false}
			/>
		{/if}

		{#if course.learning_outcomes.length}
			<section aria-labelledby="outcomes-heading">
				<h2 id="outcomes-heading" class="text-xl font-semibold text-gray-900 dark:text-white mb-3">
					{$t('public.commerce.course.outcomes')}
				</h2>
				<ul class="list-disc pl-5 space-y-1 text-gray-700 dark:text-gray-300">
					{#each course.learning_outcomes as outcome, i (i)}
						<li>{outcome}</li>
					{/each}
				</ul>
			</section>
		{/if}

		<section aria-labelledby="curriculum-heading">
			<h2 id="curriculum-heading" class="text-xl font-semibold text-gray-900 dark:text-white mb-4">
				{$t('public.commerce.course.outline')}
			</h2>
			{#if course.modules.length === 0 || course.total_lessons === 0}
				<p class="text-gray-600 dark:text-gray-300">{$t('public.commerce.course.empty')}</p>
			{:else}
				<ol class="space-y-6" data-testid="curriculum">
					{#each course.modules as module, mi (module.id)}
						<li class="card p-5">
							<h3 class="font-semibold text-gray-900 dark:text-white">
								<span class="text-gray-500 dark:text-gray-400 font-normal">Module {mi + 1}.</span>
								{module.title}
							</h3>
							{#if module.description}
								<p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{module.description}</p>
							{/if}
							{#if module.drip_unlock_date}
								<p class="mt-1 text-sm text-amber-700 dark:text-amber-400">
									Opens {formatDate(module.drip_unlock_date, {
										month: 'long',
										day: 'numeric',
										year: 'numeric'
									})}
								</p>
							{/if}
							{#if module.lessons.length}
								<ol class="mt-3 divide-y divide-gray-200 dark:divide-gray-700">
									{#each module.lessons as lesson, li (lesson.id)}
										<li
											class="py-2 flex items-center justify-between gap-3"
											data-lesson-state={lesson.state}
										>
											{#if isReadable(lesson.state)}
												<a
													href={learnUrl(lesson.id)}
													class="text-gray-900 dark:text-gray-100 hover:text-primary-600 dark:hover:text-primary-400 underline-offset-2 hover:underline rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
												>
													{mi + 1}.{li + 1}
													{lesson.title}
												</a>
											{:else}
												<span class="text-gray-600 dark:text-gray-400"
													>{mi + 1}.{li + 1} {lesson.title}</span
												>
											{/if}
											<span class="shrink-0 text-xs text-gray-500 dark:text-gray-400">
												{#if lesson.estimated_minutes}{lesson.estimated_minutes} min ·
												{/if}{stateLabel[lesson.state]}
											</span>
										</li>
									{/each}
								</ol>
							{/if}
						</li>
					{/each}
				</ol>
			{/if}
		</section>

		{#if course.instructor_name && course.instructor_bio}
			<section aria-labelledby="instructor-heading">
				<h2
					id="instructor-heading"
					class="text-xl font-semibold text-gray-900 dark:text-white mb-2"
				>
					About {course.instructor_name}
				</h2>
				<div class="rich-content-safe prose dark:prose-invert max-w-none">
					{@html parseMarkdown(course.instructor_bio)}
				</div>
			</section>
		{/if}
	</main>
</div>
