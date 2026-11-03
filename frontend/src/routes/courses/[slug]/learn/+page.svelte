<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { t } from 'svelte-i18n';
	import type { PageData } from './$types';
	import { parseMarkdown } from '$lib/utils';
	import { flattenLessons, isReadable } from '$lib/commerce';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import LessonQuiz from '$components/public/course/LessonQuiz.svelte';
	import EnrollmentForm from '$components/public/course/EnrollmentForm.svelte';
	import CertificatePanel from '$components/public/course/CertificatePanel.svelte';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();
	let course = $derived(data.course);
	let lessons = $derived(flattenLessons(course));
	let index = $derived(lessons.findIndex((l) => l.id === data.lesson.id));
	let prev = $derived(
		lessons
			.slice(0, index)
			.reverse()
			.find((l) => isReadable(l.state)) ?? null
	);
	let next = $derived(lessons.slice(index + 1).find((l) => isReadable(l.state)) ?? null);
	let completed = $derived(new Set(data.completedIds));
	let isQuiz = $derived(data.lesson.content_type === 'quiz');

	let saving = $state(false);
	let saveError = $state('');

	function learnUrl(lessonId: string) {
		return `/courses/${encodeURIComponent(course.slug)}/learn?lesson=${encodeURIComponent(lessonId)}`;
	}

	async function markComplete() {
		saving = true;
		saveError = '';
		try {
			const res = await fetch(
				`/api/courses/${encodeURIComponent(course.slug)}/lessons/${encodeURIComponent(data.lesson.id)}/complete`,
				{ method: 'POST', credentials: 'same-origin' }
			);
			if (res.ok) {
				await invalidateAll();
			} else {
				const body = await res.json().catch(() => ({}));
				saveError =
					res.status === 401
						? $t('public.commerce.learn.error_expired')
						: body.error === 'complete previous lessons first'
							? $t('public.commerce.learn.error_sequential')
							: $t('public.commerce.learn.error_save');
			}
		} catch {
			saveError = $t('public.commerce.learn.error_network');
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>{data.lesson.title} | {course.title}</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<div class="fixed top-4 right-4 z-40 flex items-center gap-2 print:hidden">
		<ThemeToggle />
	</div>

	<div
		class="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-10 lg:grid lg:grid-cols-[16rem_1fr] lg:gap-10"
	>
		<nav aria-label={$t('public.commerce.learn.course_lessons')} class="mb-8 lg:mb-0">
			<a
				href="/courses/{course.slug}"
				class="text-sm text-primary-600 dark:text-primary-400 hover:underline rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
			>
				← {course.title}
			</a>
			<ol class="mt-4 space-y-4">
				{#each course.modules as module (module.id)}
					<li>
						<p
							class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400"
						>
							{module.title}
						</p>
						<ol class="mt-2 space-y-1">
							{#each module.lessons as lesson (lesson.id)}
								<li>
									{#if isReadable(lesson.state)}
										<a
											href={learnUrl(lesson.id)}
											aria-current={lesson.id === data.lesson.id ? 'page' : undefined}
											class="block rounded px-2 py-1 text-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500
												{lesson.id === data.lesson.id
												? 'bg-primary-50 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300 font-medium'
												: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'}"
										>
											{lesson.title}{#if completed.has(lesson.id)}<span class="sr-only">
													{$t('public.commerce.learn.completed')}</span
												><span aria-hidden="true"> ✓</span>{/if}
										</a>
									{:else}
										<span class="block px-2 py-1 text-sm text-gray-400 dark:text-gray-500">
											{lesson.title}<span class="sr-only">
												{$t('public.commerce.learn.locked')}</span
											>
										</span>
									{/if}
								</li>
							{/each}
						</ol>
					</li>
				{/each}
			</ol>
		</nav>

		<main id="main-content">
			<h1 class="text-3xl font-bold text-gray-900 dark:text-white">{data.lesson.title}</h1>
			{#if data.lesson.state === 'preview' && !course.has_access}
				<p class="mt-2 text-sm text-gray-600 dark:text-gray-300">
					{$t('public.commerce.learn.preview')}
					<a href="/courses/{course.slug}" class="text-primary-600 dark:text-primary-400 underline"
						>{$t('public.commerce.learn.full_course')}</a
					>
				</p>
			{/if}

			{#if data.content.video_url}
				<p class="mt-6">
					<a
						href={data.content.video_url}
						target="_blank"
						rel="noopener noreferrer"
						class="btn btn-secondary">{$t('public.commerce.learn.watch_video')}</a
					>
				</p>
			{/if}

			{#if data.content.content}
				<article
					class="mt-6 rich-content-safe prose prose-lg dark:prose-invert max-w-none prose-a:text-primary-600 dark:prose-a:text-primary-400"
				>
					{@html parseMarkdown(data.content.content)}
				</article>
			{:else if !data.content.video_url && !isQuiz}
				<p class="mt-6 text-gray-600 dark:text-gray-300">
					{$t('public.commerce.learn.no_content')}
				</p>
			{/if}

			{#if isQuiz}
				{#key data.lesson.id}
					<LessonQuiz
						slug={course.slug}
						lessonId={data.lesson.id}
						tracksProgress={data.progressAvailable}
						oncompleted={invalidateAll}
					>
						{#snippet accessHint()}
							{#if course.access_tier === 'free'}
								<EnrollmentForm
									slug={course.slug}
									title={course.title}
									heading={$t('public.commerce.learn.quiz_signup')}
									intro={$t('public.commerce.learn.quiz_intro')}
								/>
							{:else}
								<a href="/courses/{course.slug}" class="btn btn-primary"
									>{$t('public.commerce.learn.full_course')}</a
								>
							{/if}
						{/snippet}
					</LessonQuiz>
				{/key}
			{/if}

			{#if data.content.attachments.length}
				<section class="mt-8" aria-labelledby="attachments-heading">
					<h2 id="attachments-heading" class="text-lg font-semibold text-gray-900 dark:text-white">
						{$t('public.commerce.learn.downloads')}
					</h2>
					<ul class="mt-2 space-y-1">
						{#each data.content.attachments as file (file.url)}
							<li>
								<a href={file.url} class="text-primary-600 dark:text-primary-400 underline"
									>{file.filename}</a
								>
							</li>
						{/each}
					</ul>
				</section>
			{/if}

			{#if data.progressAvailable}
				<div class="mt-10">
					{#if completed.has(data.lesson.id)}
						<p role="status" class="text-sm text-emerald-700 dark:text-emerald-400">
							{$t('public.commerce.learn.completed_status')}
						</p>
					{:else if isQuiz}
						<p class="text-sm text-gray-600 dark:text-gray-300">
							{$t('public.commerce.learn.pass_to_complete')}
						</p>
					{:else}
						<button
							type="button"
							class="btn btn-secondary"
							onclick={markComplete}
							disabled={saving}
							aria-busy={saving}
						>
							{saving
								? $t('public.commerce.common.saving')
								: $t('public.commerce.learn.mark_complete')}
						</button>
					{/if}
					{#if saveError}
						<p role="alert" class="mt-2 text-sm text-red-700 dark:text-red-400">{saveError}</p>
					{/if}
				</div>
			{/if}

			{#if data.certificate}
				<div class="mt-10">
					<CertificatePanel certificate={data.certificate} courseCompleted={data.courseCompleted} />
				</div>
			{/if}

			<nav
				aria-label={$t('public.commerce.learn.lesson_navigation')}
				class="mt-12 pt-6 border-t border-gray-200 dark:border-gray-700 flex justify-between gap-4"
			>
				{#if prev}
					<a href={learnUrl(prev.id)} class="btn btn-ghost">← {prev.title}</a>
				{:else}
					<span></span>
				{/if}
				{#if next}
					<a href={learnUrl(next.id)} class="btn btn-ghost">{next.title} →</a>
				{/if}
			</nav>
		</main>
	</div>
</div>
