<script lang="ts">
	import { t } from 'svelte-i18n';
	import CommentForm from './CommentForm.svelte';

	export interface Comment {
		id: string;
		parent_id: string;
		author_name: string;
		avatar_url?: string;
		body: string;
		is_pinned: boolean;
		is_admin_reply: boolean;
		created: string;
	}

	interface Props {
		contentType: string;
		contentId: string;
		comment: Comment;
		replies: Comment[];
		replyingTo: string | null;
		onStartReply: (id: string) => void;
		onCancelReply: () => void;
		onReplySubmitted: () => void;
		onReport: (id: string) => void;
	}

	let {
		contentType,
		contentId,
		comment,
		replies,
		replyingTo,
		onStartReply,
		onCancelReply,
		onReplySubmitted,
		onReport
	}: Props = $props();

	const defaultAvatar = 'https://www.gravatar.com/avatar/?d=mp&s=40';

	function relativeTime(dateStr: string): string {
		try {
			const now = Date.now();
			const then = new Date(dateStr).getTime();
			const diffMs = now - then;
			const seconds = Math.floor(diffMs / 1000);
			const minutes = Math.floor(seconds / 60);
			const hours = Math.floor(minutes / 60);
			const days = Math.floor(hours / 24);
			const months = Math.floor(days / 30);
			const years = Math.floor(days / 365);

			if (years > 0) return `${years}y`;
			if (months > 0) return `${months}mo`;
			if (days > 0) return `${days}d`;
			if (hours > 0) return `${hours}h`;
			if (minutes > 0) return `${minutes}m`;
			return $t('comments.just_now');
		} catch {
			return dateStr;
		}
	}
</script>

<li
	class="rounded-lg border border-stone-200 dark:border-stone-700 bg-white dark:bg-stone-800"
	aria-level="1"
>
	<article class="p-4">
		<div class="flex items-start gap-3">
			<img
				src={comment.avatar_url || defaultAvatar}
				alt=""
				class="w-10 h-10 rounded-full flex-shrink-0"
				loading="lazy"
			/>
			<div class="flex-1 min-w-0">
				<header class="flex flex-wrap items-center gap-2 mb-1">
					<span class="font-medium text-stone-900 dark:text-white text-sm">{comment.author_name}</span>
					{#if comment.is_pinned}
						<span
							class="inline-flex items-center gap-1 px-1.5 py-0.5 text-xs font-medium bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 rounded"
							aria-label={$t('comments.pinned')}
						>
							<svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
								<path d="M5 5a2 2 0 012-2h6a2 2 0 012 2v2H5V5zm0 4h10v6a2 2 0 01-2 2H7a2 2 0 01-2-2V9z" />
							</svg>
							<span>{$t('comments.pinned')}</span>
						</span>
					{/if}
					{#if comment.is_admin_reply}
						<span
							class="px-1.5 py-0.5 text-xs font-medium bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300 rounded"
							aria-label={$t('comments.author_badge')}
						>
							{$t('comments.author_badge')}
						</span>
					{/if}
					<time
						class="text-xs text-stone-600 dark:text-stone-300"
						datetime={comment.created}
					>
						{relativeTime(comment.created)}
					</time>
				</header>
				<p class="text-stone-700 dark:text-stone-300 text-sm whitespace-pre-wrap break-words">{comment.body}</p>
				<div class="mt-2 flex items-center gap-3">
					<button
						type="button"
						onclick={() => onStartReply(comment.id)}
						class="text-xs text-stone-600 dark:text-stone-300 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
					>
						{$t('comments.reply')}
					</button>
					<button
						type="button"
						onclick={() => onReport(comment.id)}
						class="text-xs text-stone-600 dark:text-stone-300 hover:text-red-500 dark:hover:text-red-400 transition-colors inline-flex items-center gap-1"
						aria-label={$t('comments.report')}
					>
						<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 21v-4m0 0V5a2 2 0 012-2h6.5l1 1H21l-3 6 3 6h-8.5l-1-1H5a2 2 0 00-2 2zm9-13.5V9" />
						</svg>
						<span class="sr-only">{$t('comments.report')}</span>
					</button>
				</div>
			</div>
		</div>

		{#if replyingTo === comment.id}
			<div class="mt-3 ms-13">
				<CommentForm
					{contentType}
					{contentId}
					parentId={comment.id}
					onSubmit={onReplySubmitted}
					onCancel={onCancelReply}
				/>
			</div>
		{/if}
	</article>

	{#if replies.length > 0}
		<div class="border-t border-stone-100 dark:border-stone-700">
			<ul class="divide-y divide-stone-100 dark:divide-stone-700" aria-label={$t('comments.replies')}>
				{#each replies as reply (reply.id)}
					<li
						class="ps-8 pe-4 py-3 {reply.is_admin_reply ? 'bg-primary-50/50 dark:bg-primary-900/10' : ''}"
						aria-level="2"
					>
						<article class="flex items-start gap-3 border-s-2 border-stone-200 dark:border-stone-600 ps-4">
							<img
								src={reply.avatar_url || defaultAvatar}
								alt=""
								class="w-8 h-8 rounded-full flex-shrink-0"
								loading="lazy"
							/>
							<div class="flex-1 min-w-0">
								<header class="flex flex-wrap items-center gap-2 mb-1">
									<span class="font-medium text-stone-900 dark:text-white text-sm">{reply.author_name}</span>
									{#if reply.is_admin_reply}
										<span
											class="px-1.5 py-0.5 text-xs font-medium bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300 rounded"
											aria-label={$t('comments.author_badge')}
										>
											{$t('comments.author_badge')}
										</span>
									{/if}
									<time
										class="text-xs text-stone-600 dark:text-stone-300"
										datetime={reply.created}
									>
										{relativeTime(reply.created)}
									</time>
								</header>
								<p class="text-stone-700 dark:text-stone-300 text-sm whitespace-pre-wrap break-words">{reply.body}</p>
								<div class="mt-2">
									<button
										type="button"
										onclick={() => onReport(reply.id)}
										class="text-xs text-stone-600 dark:text-stone-300 hover:text-red-500 dark:hover:text-red-400 transition-colors inline-flex items-center gap-1"
										aria-label={$t('comments.report')}
									>
										<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 21v-4m0 0V5a2 2 0 012-2h6.5l1 1H21l-3 6 3 6h-8.5l-1-1H5a2 2 0 00-2 2zm9-13.5V9" />
										</svg>
										<span class="sr-only">{$t('comments.report')}</span>
									</button>
								</div>
							</div>
						</article>
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</li>
