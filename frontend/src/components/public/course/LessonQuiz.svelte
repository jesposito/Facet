<script lang="ts">
	/**
	 * Quiz for one lesson.
	 *
	 * Questions come from the learner endpoint, which sends option labels
	 * only. Whether an answer was right, and any explanation, is shown only
	 * from the server's reply to a real submission. Passing marks the lesson
	 * complete on the server; `oncompleted` lets the page reload progress.
	 */
	import { onMount, tick, type Snippet } from 'svelte';
	import { t } from 'svelte-i18n';
	import {
		attemptsLeft,
		buildQuizSubmission,
		quizError,
		toLearnerQuiz,
		toQuizResult,
		type LearnerQuiz,
		type QuizErrorKind,
		type QuizResult
	} from '$lib/commerce';

	interface Props {
		slug: string;
		lessonId: string;
		/** Free courses can offer sign-up when the quiz needs a learner cookie. */
		accessHint?: Snippet;
		/** Whether the course records progress, so a pass completes the lesson. */
		tracksProgress?: boolean;
		oncompleted?: () => void | Promise<void>;
	}

	let { slug, lessonId, accessHint, tracksProgress = false, oncompleted }: Props = $props();

	const uid = $props.id();
	let phase = $state<'loading' | 'ready' | 'error'>('loading');
	let loadError = $state<{ kind: QuizErrorKind; message: string } | null>(null);
	let quiz = $state<LearnerQuiz | null>(null);
	let selected = $state<Record<string, number | undefined>>({});
	let submitting = $state(false);
	let submitError = $state('');
	let result = $state<QuizResult | null>(null);
	let resultHeading = $state<HTMLElement | null>(null);

	let endpoint = $derived(
		`/api/courses/${encodeURIComponent(slug)}/lessons/${encodeURIComponent(lessonId)}/quiz`
	);
	let remaining = $derived(quiz ? attemptsLeft(quiz) : null);
	let outOfAttempts = $derived(remaining === 0 && !result);
	let resultById = $derived(new Map((result?.results ?? []).map((r) => [r.question_id, r])));

	async function load() {
		phase = 'loading';
		loadError = null;
		try {
			const res = await fetch(endpoint, { credentials: 'same-origin' });
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				loadError = quizError(res.status, String(body.error ?? ''));
				phase = 'error';
				return;
			}
			quiz = toLearnerQuiz(body);
			selected = {};
			result = null;
			submitError = '';
			phase = 'ready';
		} catch {
			loadError = { kind: 'other', message: $t('public.commerce.errors.network') };
			phase = 'error';
		}
	}

	onMount(load);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!quiz || submitting || result) return;
		submitError = '';

		const unanswered = quiz.questions.find((q) => !Number.isInteger(selected[q.id]));
		if (unanswered) {
			submitError = $t('public.commerce.quiz.answer_all');
			document.getElementById(`${uid}-q-${unanswered.id}-0`)?.focus();
			return;
		}

		submitting = true;
		try {
			const res = await fetch(`${endpoint}/submit`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'same-origin',
				body: JSON.stringify(buildQuizSubmission(quiz.questions, selected))
			});
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				const failure = quizError(res.status, String(body.error ?? ''));
				if (failure.kind === 'attempts')
					quiz.attempts_used = Math.max(quiz.attempts_used, quiz.max_attempts);
				submitError = failure.message;
				return;
			}
			result = toQuizResult(body);
			quiz.attempts_used += 1;
			await tick();
			resultHeading?.focus();
			if (result.passed && tracksProgress) await oncompleted?.();
		} catch {
			submitError = $t('public.commerce.quiz.network_submit');
		} finally {
			submitting = false;
		}
	}

	function retry() {
		selected = {};
		result = null;
		submitError = '';
		// Reload so the attempt count and any shuffled order come from the server.
		load();
	}
</script>

<section class="mt-6" aria-labelledby="{uid}-heading" data-testid="lesson-quiz" data-phase={phase}>
	{#if phase === 'loading'}
		<p role="status" class="text-gray-600 dark:text-gray-300">
			{$t('public.commerce.quiz.loading')}
		</p>
	{:else if phase === 'error' && loadError}
		<div class="card p-5">
			<h2 id="{uid}-heading" class="text-lg font-semibold text-gray-900 dark:text-white">
				{$t('public.commerce.quiz.title')}
			</h2>
			<p role="alert" class="mt-2 text-sm text-gray-700 dark:text-gray-200">{loadError.message}</p>
			{#if loadError.kind === 'access' && accessHint}
				<div class="mt-4">{@render accessHint()}</div>
			{:else if loadError.kind === 'other'}
				<button type="button" class="btn btn-secondary mt-4" onclick={load}
					>{$t('public.commerce.common.try_again')}</button
				>
			{/if}
		</div>
	{:else if quiz}
		<h2 id="{uid}-heading" class="text-xl font-semibold text-gray-900 dark:text-white">
			{quiz.title || $t('public.commerce.quiz.title')}
		</h2>
		<p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
			{$t('public.commerce.quiz.pass_mark', { values: { score: quiz.passing_score } })}
			{#if remaining !== null}{$t('public.commerce.quiz.attempts_left', {
					values: { count: remaining }
				})}{/if}
		</p>
		{#if quiz.previous_attempt && !result}
			<p class="mt-1 text-sm text-gray-600 dark:text-gray-300" data-testid="quiz-previous">
				{$t(
					quiz.previous_attempt.passed
						? 'public.commerce.quiz.previous_passed'
						: 'public.commerce.quiz.previous',
					{ values: { score: Math.round(quiz.previous_attempt.score) } }
				)}
			</p>
		{/if}

		{#if quiz.questions.length === 0}
			<p class="mt-4 text-gray-600 dark:text-gray-300">{$t('public.commerce.quiz.empty')}</p>
		{:else if outOfAttempts}
			<p class="mt-4 text-gray-700 dark:text-gray-200" data-testid="quiz-no-attempts">
				{$t('public.commerce.quiz.no_attempts')}
			</p>
		{:else}
			<form
				class="mt-6 space-y-6"
				onsubmit={submit}
				novalidate
				aria-describedby={submitError ? `${uid}-error` : undefined}
			>
				{#each quiz.questions as question, qi (question.id)}
					{@const outcome = resultById.get(question.id)}
					<fieldset class="card p-5" data-testid="quiz-question">
						<legend class="px-1 font-medium text-gray-900 dark:text-white">
							<span class="text-gray-500 dark:text-gray-400 font-normal">{qi + 1}.</span>
							{question.text}
						</legend>
						<div class="mt-3 space-y-2">
							{#each question.options as option, oi (oi)}
								<label
									class="flex items-start gap-2 text-gray-800 dark:text-gray-200"
									for="{uid}-q-{question.id}-{oi}"
								>
									<input
										id="{uid}-q-{question.id}-{oi}"
										type="radio"
										name="{uid}-q-{question.id}"
										value={oi}
										checked={selected[question.id] === oi}
										onchange={() => (selected[question.id] = oi)}
										disabled={submitting || !!result}
										class="mt-1"
									/>
									<span>{option}</span>
								</label>
							{/each}
						</div>
						{#if outcome}
							<p
								class="mt-3 text-sm {outcome.correct
									? 'text-emerald-700 dark:text-emerald-400'
									: 'text-red-700 dark:text-red-400'}"
								data-testid="quiz-outcome"
							>
								{outcome.correct
									? $t('public.commerce.quiz.correct')
									: $t('public.commerce.quiz.incorrect')}
								{#if outcome.explanation}<span class="text-gray-700 dark:text-gray-300">
										{outcome.explanation}</span
									>{/if}
							</p>
						{/if}
					</fieldset>
				{/each}

				{#if submitError}
					<p id="{uid}-error" role="alert" class="text-sm text-red-700 dark:text-red-400">
						{submitError}
					</p>
				{/if}

				{#if !result}
					<button
						type="submit"
						class="btn btn-primary"
						disabled={submitting}
						aria-busy={submitting}
					>
						{submitting ? $t('public.commerce.quiz.submitting') : $t('public.commerce.quiz.submit')}
					</button>
				{/if}
			</form>
		{/if}

		{#if result}
			<div class="mt-6 card p-5" data-testid="quiz-result" data-passed={result.passed}>
				<h3
					bind:this={resultHeading}
					tabindex="-1"
					class="text-lg font-semibold text-gray-900 dark:text-white focus:outline-none"
				>
					{$t(result.passed ? 'public.commerce.quiz.passed' : 'public.commerce.quiz.not_passed', {
						values: { score: Math.round(result.score) }
					})}
				</h3>
				<p role="status" class="mt-1 text-sm text-gray-600 dark:text-gray-300">
					{$t('public.commerce.quiz.points', {
						values: { earned: result.earned_points, total: result.total_points }
					})}
					{#if result.passed}{#if tracksProgress}{$t(
								'public.commerce.quiz.lesson_complete'
							)}{/if}{:else}{$t('public.commerce.quiz.need_score', {
							values: { score: quiz.passing_score }
						})}{/if}
				</p>
				{#if remaining !== 0}
					<button type="button" class="btn btn-secondary mt-4" onclick={retry}
						>{$t('public.commerce.quiz.retry')}</button
					>
				{/if}
			</div>
		{/if}
	{/if}
</section>
