<script lang="ts">
	import { t } from 'svelte-i18n';
	import { hasFeature } from '$lib/stores/plan';

	interface Props {
		contentType: string;
		contentId: string;
	}

	let { contentType, contentId }: Props = $props();

	const hasDiscussions = hasFeature('discussions');

	type Comment = {
		id: string;
		author_name: string;
		body: string;
		is_pinned: boolean;
		is_admin_reply: boolean;
		parent_id: string;
		avatar_url?: string;
		created: string;
		replies?: Comment[];
	};

	let comments: Comment[] = $state([]);
	let loading = $state(true);
	let totalComments = $state(0);
	let page = $state(1);
	let hasMore = $state(false);
	const perPage = 50;

	// Submit form state
	let authorName = $state('');
	let authorEmail = $state('');
	let body = $state('');
	let replyToId = $state('');
	let submitting = $state(false);
	let submitted = $state(false);
	let submittedPending = $state(false);
	let submitError = $state('');

	// Report form state
	let reportingCommentId = $state('');
	let reportReason = $state('');
	let reportDetail = $state('');
	let reportSubmitting = $state(false);
	let reportSubmitted = $state(false);

	const maxBodyLength = 5000;

	$effect(() => {
		if ($hasDiscussions) {
			loadComments();
		}
	});

	async function loadComments(append = false) {
		loading = true;
		try {
			const url = `/api/comments/${contentType}/${contentId}?page=${page}&per_page=${perPage}`;
			const response = await fetch(url);
			if (!response.ok) throw new Error('Failed to load comments');
			const data = await response.json();

			const newComments: Comment[] = data.comments || [];
			totalComments = data.total || 0;
			hasMore = page * perPage < totalComments;

			// Thread by parent_id: collect top-level, attach replies
			const topLevel: Comment[] = [];
			const replyMap: Record<string, Comment[]> = {};

			for (const c of newComments) {
				if (c.parent_id) {
					if (!replyMap[c.parent_id]) replyMap[c.parent_id] = [];
					replyMap[c.parent_id].push(c);
				} else {
					topLevel.push(c);
				}
			}

			for (const c of topLevel) {
				c.replies = replyMap[c.id] || [];
			}

			if (append) {
				comments = [...comments, ...topLevel];
			} else {
				comments = topLevel;
			}
		} catch (err) {
			console.error('Failed to load comments:', err);
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		page += 1;
		await loadComments(true);
	}

	function startReply(commentId: string) {
		replyToId = commentId;
		submitted = false;
		submitError = '';
	}

	function cancelReply() {
		replyToId = '';
	}

	async function submitComment(parentId = '') {
		submitError = '';
		if (!authorName.trim()) {
			submitError = 'Name is required';
			return;
		}
		if (!body.trim()) {
			submitError = 'Comment text is required';
			return;
		}

		submitting = true;
		try {
			const payload: Record<string, string> = {
				author_name: authorName.trim(),
				body: body.trim(),
				parent_id: parentId
			};
			if (authorEmail.trim()) {
				payload.author_email = authorEmail.trim();
			}

			const response = await fetch(`/api/comments/${contentType}/${contentId}`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});

			if (!response.ok) {
				const err = await response.json().catch(() => ({}));
				submitError = err.error || 'Failed to submit comment';
				return;
			}

			const result = await response.json();
			submittedPending = result.status === 'pending';
			submitted = true;
			replyToId = '';
			authorName = '';
			authorEmail = '';
			body = '';

			if (!submittedPending) {
				// Reload to show new comment
				page = 1;
				await loadComments();
			}
		} catch {
			submitError = 'Failed to submit comment';
		} finally {
			submitting = false;
		}
	}

	function startReport(commentId: string) {
		reportingCommentId = commentId;
		reportReason = '';
		reportDetail = '';
		reportSubmitted = false;
	}

	function cancelReport() {
		reportingCommentId = '';
	}

	async function submitReport() {
		if (!reportReason) return;
		reportSubmitting = true;
		try {
			await fetch(`/api/comments/${reportingCommentId}/report`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ reason: reportReason, detail: reportDetail })
			});
			reportSubmitted = true;
			setTimeout(() => {
				reportingCommentId = '';
				reportSubmitted = false;
			}, 2000);
		} catch {
			// Silently fail
		} finally {
			reportSubmitting = false;
		}
	}

	function formatDate(dateStr: string): string {
		try {
			return new Date(dateStr).toLocaleDateString(undefined, {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		} catch {
			return dateStr;
		}
	}
</script>

{#if $hasDiscussions}
	<section class="mt-12" aria-label={$t('comments.title')}>
		<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-6">
			{$t('comments.title')}
			{#if totalComments > 0}
				<span class="text-base font-normal text-gray-500 dark:text-gray-400 ml-2">({totalComments})</span>
			{/if}
		</h2>

		<!-- Comment list -->
		{#if loading && comments.length === 0}
			<div class="flex items-center justify-center py-8" role="status">
				<svg class="animate-spin h-5 w-5 text-primary-600" fill="none" viewBox="0 0 24 24" aria-hidden="true">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
			</div>
		{:else if comments.length === 0}
			<p class="text-gray-500 dark:text-gray-400 text-sm mb-8">{$t('comments.no_comments')}</p>
		{:else}
			<div class="space-y-6 mb-8">
				{#each comments as comment (comment.id)}
					<article class="flex gap-3">
						<!-- Avatar -->
						<div class="shrink-0">
							{#if comment.avatar_url}
								<img
									src={comment.avatar_url}
									alt={comment.author_name}
									class="w-9 h-9 rounded-full bg-gray-200 dark:bg-gray-700"
									loading="lazy"
								/>
							{:else}
								<div class="w-9 h-9 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center text-sm font-medium text-gray-600 dark:text-gray-400" aria-hidden="true">
									{comment.author_name[0]?.toUpperCase() ?? '?'}
								</div>
							{/if}
						</div>

						<!-- Comment body -->
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 mb-1 flex-wrap">
								<span class="font-medium text-sm text-gray-900 dark:text-white">{comment.author_name}</span>
								{#if comment.is_admin_reply}
									<span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-primary-100 text-primary-800 dark:bg-primary-900/30 dark:text-primary-300">
										{$t('comments.author_badge')}
									</span>
								{/if}
								{#if comment.is_pinned}
									<span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300">
										{$t('comments.pinned')}
									</span>
								{/if}
								<span class="text-xs text-gray-400 dark:text-gray-500">{formatDate(comment.created)}</span>
							</div>

							<p class="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap break-words">{comment.body}</p>

							<!-- Actions -->
							<div class="flex items-center gap-3 mt-2">
								<button
									type="button"
									class="text-xs text-gray-400 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
									onclick={() => startReply(comment.id)}
								>
									{$t('comments.reply')}
								</button>
								<button
									type="button"
									class="text-xs text-gray-400 hover:text-red-500 transition-colors"
									onclick={() => startReport(comment.id)}
								>
									{$t('comments.report')}
								</button>
							</div>

							<!-- Report form -->
							{#if reportingCommentId === comment.id}
								<div class="mt-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
									{#if reportSubmitted}
										<p class="text-sm text-green-600 dark:text-green-400">{$t('comments.report_submitted')}</p>
									{:else}
										<p class="text-sm font-medium text-gray-900 dark:text-white mb-2">{$t('comments.report_title')}</p>
										<div class="space-y-2">
											<select
												bind:value={reportReason}
												class="w-full text-sm rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white px-3 py-1.5"
											>
												<option value="">{$t('comments.report_reason')}</option>
												<option value="spam">{$t('comments.report_spam')}</option>
												<option value="harassment">{$t('comments.report_harassment')}</option>
												<option value="off_topic">{$t('comments.report_off_topic')}</option>
												<option value="other">{$t('comments.report_other')}</option>
											</select>
											<textarea
												bind:value={reportDetail}
												placeholder={$t('comments.report_detail')}
												rows="2"
												class="w-full text-sm rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white px-3 py-1.5 resize-none"
											></textarea>
											<div class="flex gap-2">
												<button
													type="button"
													class="text-xs px-3 py-1 rounded bg-red-600 hover:bg-red-700 text-white disabled:opacity-50"
													disabled={!reportReason || reportSubmitting}
													onclick={submitReport}
												>
													{$t('comments.submit')}
												</button>
												<button
													type="button"
													class="text-xs px-3 py-1 rounded border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
													onclick={cancelReport}
												>
													{$t('comments.cancel')}
												</button>
											</div>
										</div>
									{/if}
								</div>
							{/if}

							<!-- Reply form -->
							{#if replyToId === comment.id}
								<div class="mt-3">
									{@render ReplyForm({
										onSubmit: () => submitComment(comment.id),
										onCancel: cancelReply,
										submitting,
										submitError,
										maxBodyLength
									})}
								</div>
							{/if}

							<!-- Replies -->
							{#if comment.replies && comment.replies.length > 0}
								<div class="mt-4 space-y-4 pl-4 border-l-2 border-gray-100 dark:border-gray-800">
									{#each comment.replies as reply (reply.id)}
										<article class="flex gap-3">
											<div class="shrink-0">
												{#if reply.avatar_url}
													<img
														src={reply.avatar_url}
														alt={reply.author_name}
														class="w-7 h-7 rounded-full bg-gray-200 dark:bg-gray-700"
														loading="lazy"
													/>
												{:else}
													<div class="w-7 h-7 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center text-xs font-medium text-gray-600 dark:text-gray-400" aria-hidden="true">
														{reply.author_name[0]?.toUpperCase() ?? '?'}
													</div>
												{/if}
											</div>
											<div class="flex-1 min-w-0">
												<div class="flex items-center gap-2 mb-1 flex-wrap">
													<span class="font-medium text-sm text-gray-900 dark:text-white">{reply.author_name}</span>
													{#if reply.is_admin_reply}
														<span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-primary-100 text-primary-800 dark:bg-primary-900/30 dark:text-primary-300">
															{$t('comments.author_badge')}
														</span>
													{/if}
													<span class="text-xs text-gray-400 dark:text-gray-500">{formatDate(reply.created)}</span>
												</div>
												<p class="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap break-words">{reply.body}</p>
												<div class="mt-1">
													<button
														type="button"
														class="text-xs text-gray-400 hover:text-red-500 transition-colors"
														onclick={() => startReport(reply.id)}
													>
														{$t('comments.report')}
													</button>
												</div>
											</div>
										</article>
									{/each}
								</div>
							{/if}
						</div>
					</article>
				{/each}
			</div>

			<!-- Load more -->
			{#if hasMore}
				<div class="text-center mb-8">
					<button
						type="button"
						class="text-sm text-primary-600 dark:text-primary-400 hover:underline"
						disabled={loading}
						onclick={loadMore}
					>
						{$t('comments.load_more')}
					</button>
				</div>
			{/if}
		{/if}

		<!-- Submitted confirmation -->
		{#if submitted && !replyToId}
			<div class="mb-6 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
				<p class="text-sm text-green-700 dark:text-green-300">
					{submittedPending ? $t('comments.awaiting') : $t('comments.submitted')}
				</p>
			</div>
		{/if}

		<!-- New comment form (top-level, only show if not showing a submitted confirmation without reply) -->
		{#if !submitted || replyToId}
			<div class="mt-2">
				<h3 class="text-base font-medium text-gray-900 dark:text-white mb-4">{$t('comments.write')}</h3>
				<form
					onsubmit={(e) => { e.preventDefault(); submitComment(''); }}
					class="space-y-4"
					novalidate
				>
					<!-- Honeypot field - hidden from real users -->
					<div style="position: absolute; left: -9999px; top: -9999px;" aria-hidden="true">
						<label for="website_url_hp">Website</label>
						<input type="text" id="website_url_hp" name="website_url" tabindex="-1" autocomplete="off" />
					</div>

					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="comment-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
								{$t('comments.name')} <span class="text-red-500">*</span>
							</label>
							<input
								id="comment-name"
								type="text"
								bind:value={authorName}
								required
								maxlength="100"
								class="w-full text-sm rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
							/>
						</div>
						<div>
							<label for="comment-email" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
								{$t('comments.email')}
							</label>
							<input
								id="comment-email"
								type="email"
								bind:value={authorEmail}
								class="w-full text-sm rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
							/>
							<p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{$t('comments.email_hint')}</p>
						</div>
					</div>

					<div>
						<label for="comment-body" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							{$t('comments.body')} <span class="text-red-500">*</span>
						</label>
						<textarea
							id="comment-body"
							bind:value={body}
							required
							rows="4"
							maxlength={maxBodyLength}
							class="w-full text-sm rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white px-3 py-2 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-y"
						></textarea>
						<p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
							{$t('comments.chars_remaining', { values: { count: maxBodyLength - body.length } })}
						</p>
					</div>

					{#if submitError}
						<p class="text-sm text-red-600 dark:text-red-400">{submitError}</p>
					{/if}

					<div>
						<button
							type="submit"
							disabled={submitting}
							class="px-4 py-2 text-sm font-medium rounded-md bg-primary-600 hover:bg-primary-700 text-white disabled:opacity-50 transition-colors"
						>
							{submitting ? '...' : $t('comments.submit')}
						</button>
					</div>
				</form>
			</div>
		{/if}
	</section>
{/if}

<!-- Inline reply form snippet (used for replies) -->
{#snippet ReplyForm({ onSubmit, onCancel, submitting, submitError, maxBodyLength }: { onSubmit: () => void; onCancel: () => void; submitting: boolean; submitError: string; maxBodyLength: number })}
	<form onsubmit={(e) => { e.preventDefault(); onSubmit(); }} class="space-y-3" novalidate>
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
			<input
				type="text"
				placeholder={$t('comments.name')}
				bind:value={authorName}
				required
				maxlength="100"
				class="text-sm rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white px-3 py-1.5"
			/>
			<input
				type="email"
				placeholder={$t('comments.email')}
				bind:value={authorEmail}
				class="text-sm rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white px-3 py-1.5"
			/>
		</div>
		<textarea
			placeholder={$t('comments.body')}
			bind:value={body}
			required
			rows="3"
			maxlength={maxBodyLength}
			class="w-full text-sm rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white px-3 py-1.5 resize-none"
		></textarea>
		{#if submitError}
			<p class="text-xs text-red-600 dark:text-red-400">{submitError}</p>
		{/if}
		<div class="flex gap-2">
			<button
				type="submit"
				disabled={submitting}
				class="text-sm px-3 py-1.5 rounded bg-primary-600 hover:bg-primary-700 text-white disabled:opacity-50 transition-colors"
			>
				{submitting ? '...' : $t('comments.submit')}
			</button>
			<button
				type="button"
				class="text-sm px-3 py-1.5 rounded border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
				onclick={onCancel}
			>
				{$t('comments.cancel')}
			</button>
		</div>
	</form>
{/snippet}
