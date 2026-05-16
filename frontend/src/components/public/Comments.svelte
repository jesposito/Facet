<script lang="ts">
	import { t } from 'svelte-i18n';
	import { browser } from '$app/environment';
	import { hasFeature } from '$lib/stores/plan';
	import CommentForm from './CommentForm.svelte';
	import CommentThread, { type Comment } from './CommentThread.svelte';
	import ReportModal from './ReportModal.svelte';

	interface Props {
		contentType: string;
		contentId: string;
	}

	let { contentType, contentId }: Props = $props();

	const hasDiscussions = hasFeature('discussions');

	let comments = $state<Comment[]>([]);
	let loading = $state(true);
	let totalComments = $state(0);
	let page = $state(1);
	let hasMore = $state(false);

	let replyingTo = $state<string | null>(null);
	let reportingCommentId = $state<string | null>(null);

	const PAGE_SIZE = 50;

	let pinnedComments = $derived(comments.filter((c) => !c.parent_id && c.is_pinned));
	let topLevelComments = $derived(comments.filter((c) => !c.parent_id && !c.is_pinned));
	let repliesByParent = $derived.by(() => {
		const map = new Map<string, Comment[]>();
		for (const c of comments) {
			if (c.parent_id) {
				const existing = map.get(c.parent_id) || [];
				existing.push(c);
				map.set(c.parent_id, existing);
			}
		}
		return map;
	});

	async function fetchComments(pageNum: number = 1, append: boolean = false) {
		if (!browser) return;
		loading = true;
		try {
			const resp = await fetch(`/api/comments/${contentType}/${contentId}?page=${pageNum}&per_page=${PAGE_SIZE}`);
			if (!resp.ok) throw new Error('Failed to load comments');
			const data = await resp.json();
			const fetched: Comment[] = data.comments || [];
			totalComments = data.total || fetched.length;
			if (append) {
				comments = [...comments, ...fetched];
			} else {
				comments = fetched;
			}
			hasMore = pageNum * PAGE_SIZE < totalComments;
		} catch (err) {
			console.error('Failed to fetch comments:', err);
		} finally {
			loading = false;
		}
	}

	function loadMore() {
		page += 1;
		fetchComments(page, true);
	}

	function handleCommentSubmitted() {
		page = 1;
		fetchComments(1);
	}

	function handleReplySubmitted() {
		replyingTo = null;
		page = 1;
		fetchComments(1);
	}

	function startReply(id: string) {
		replyingTo = replyingTo === id ? null : id;
	}

	function cancelReply() {
		replyingTo = null;
	}

	function startReport(id: string) {
		reportingCommentId = id;
	}

	function closeReport() {
		reportingCommentId = null;
	}

	$effect(() => {
		if ($hasDiscussions && browser) {
			fetchComments();
		}
	});
</script>

{#if $hasDiscussions}
	<section class="mt-12" aria-label={$t('comments.title')}>
		<h3 class="text-xl font-semibold text-stone-900 dark:text-white mb-6 flex items-center gap-2">
			<svg class="w-5 h-5 text-stone-500 dark:text-stone-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
			</svg>
			{$t('comments.title')}
			{#if totalComments > 0}
				<span class="text-base font-normal text-stone-500 dark:text-stone-400 ms-2">({totalComments})</span>
			{/if}
		</h3>

		<!-- New comment form -->
		<div class="mb-8">
			<CommentForm
				{contentType}
				{contentId}
				onSubmit={handleCommentSubmitted}
			/>
		</div>

		<!-- Loading state -->
		{#if loading && comments.length === 0}
			<div class="flex items-center justify-center py-8" role="status" aria-live="polite">
				<svg class="animate-spin h-6 w-6 text-stone-500" fill="none" viewBox="0 0 24 24" aria-hidden="true">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
				<span class="sr-only">{$t('shared.loading') ?? 'Loading'}</span>
			</div>
		{:else if comments.length === 0}
			<p class="text-center text-stone-500 dark:text-stone-400 py-8">
				{$t('comments.no_comments')}
			</p>
		{:else}
			<ul class="space-y-4" aria-label={$t('comments.title')}>
				{#each pinnedComments as comment (comment.id)}
					<CommentThread
						{contentType}
						{contentId}
						{comment}
						replies={repliesByParent.get(comment.id) || []}
						{replyingTo}
						onStartReply={startReply}
						onCancelReply={cancelReply}
						onReplySubmitted={handleReplySubmitted}
						onReport={startReport}
					/>
				{/each}
				{#each topLevelComments as comment (comment.id)}
					<CommentThread
						{contentType}
						{contentId}
						{comment}
						replies={repliesByParent.get(comment.id) || []}
						{replyingTo}
						onStartReply={startReply}
						onCancelReply={cancelReply}
						onReplySubmitted={handleReplySubmitted}
						onReport={startReport}
					/>
				{/each}
			</ul>

			{#if hasMore}
				<div class="mt-6 text-center">
					<button
						type="button"
						onclick={loadMore}
						disabled={loading}
						aria-busy={loading}
						class="px-6 py-2 text-sm font-medium text-stone-600 dark:text-stone-300 bg-stone-100 dark:bg-stone-800 hover:bg-stone-200 dark:hover:bg-stone-700 rounded-lg border border-stone-200 dark:border-stone-700 transition-colors disabled:opacity-50"
					>
						{#if loading}
							<svg class="animate-spin inline-block h-4 w-4 me-2" fill="none" viewBox="0 0 24 24" aria-hidden="true">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
						{/if}
						{$t('comments.load_more')}
					</button>
				</div>
			{/if}
		{/if}
	</section>

	<!-- Report modal -->
	{#if reportingCommentId}
		<ReportModal
			commentId={reportingCommentId}
			open={true}
			onClose={closeReport}
		/>
	{/if}
{/if}
