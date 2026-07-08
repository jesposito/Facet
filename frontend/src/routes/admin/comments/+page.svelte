<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { pb } from '$lib/pocketbase';
	import { toasts, confirm } from '$lib/stores';
	import { hasFeature } from '$lib/stores/plan';

	const hasDiscussions = hasFeature('discussions');

	type Comment = {
		id: string;
		author_name: string;
		author_email: string;
		body: string;
		status: 'pending' | 'approved' | 'spam' | 'rejected';
		content_type: string;
		content_id: string;
		content_title: string;
		report_count: number;
		is_pinned: boolean;
		created: string;
	};

	type TabKey = 'pending' | 'approved' | 'spam' | 'reported';

	let comments: Comment[] = $state([]);
	let loading = $state(true);
	let activeTab: TabKey = $state('pending');
	let currentPage = $state(1);
	let totalPages = $state(1);
	let totalItems = $state(0);
	let pendingCount = $state(0);
	let selectedIds: Set<string> = $state(new Set());
	let selectAll = $state(false);

	const perPage = 50;

	let abortController: AbortController | null = null;

	onMount(async () => {
		if (!$hasDiscussions) {
			goto('/admin');
			return;
		}
		await loadComments();
		await loadPendingCount();
	});

	onDestroy(() => {
		if (abortController) abortController.abort();
	});

	async function loadPendingCount() {
		try {
			const response = await fetch('/api/admin/comments?status=pending&page=1&per_page=1', {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (response.ok) {
				const data = await response.json();
				pendingCount = data.totalItems || 0;
			}
		} catch {
			// Best-effort
		}
	}

	async function loadComments() {
		if (abortController) abortController.abort();
		abortController = new AbortController();

		loading = true;
		try {
			let url: string;
			if (activeTab === 'reported') {
				url = `/api/admin/comment-reports?status=open&page=${currentPage}&per_page=${perPage}`;
			} else {
				url = `/api/admin/comments?status=${activeTab}&page=${currentPage}&per_page=${perPage}`;
			}

			const response = await fetch(url, {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {},
				signal: abortController.signal
			});
			if (!response.ok) throw new Error('Failed to load comments');
			const data = await response.json();
			comments = data.data || [];
			totalPages = data.totalPages || 1;
			totalItems = data.totalItems || 0;
		} catch (err) {
			if (err instanceof DOMException && err.name === 'AbortError') return;
			console.error('Failed to load comments:', err);
			toasts.add('error', 'Failed to load comments');
		} finally {
			loading = false;
		}
	}

	function setTab(tab: TabKey) {
		activeTab = tab;
		currentPage = 1;
		selectedIds = new Set();
		selectAll = false;
		loadComments();
	}

	function goToPage(page: number) {
		currentPage = page;
		selectedIds = new Set();
		selectAll = false;
		loadComments();
	}

	function toggleSelectAll() {
		if (selectAll) {
			selectedIds = new Set();
			selectAll = false;
		} else {
			selectedIds = new Set(comments.map((c) => c.id));
			selectAll = true;
		}
	}

	function toggleSelect(id: string) {
		const next = new Set(selectedIds);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		selectedIds = next;
		selectAll = next.size === comments.length && comments.length > 0;
	}

	async function updateStatus(id: string, status: string) {
		try {
			const response = await fetch(`/api/admin/comments/${id}`, {
				method: 'PATCH',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify({ status })
			});
			if (!response.ok) throw new Error('Failed to update comment');
			await loadComments();
			await loadPendingCount();
		} catch (err) {
			console.error('Failed to update comment:', err);
			toasts.add('error', 'Failed to update comment');
		}
	}

	async function togglePin(comment: Comment) {
		try {
			const response = await fetch(`/api/admin/comments/${comment.id}`, {
				method: 'PATCH',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify({ is_pinned: !comment.is_pinned })
			});
			if (!response.ok) throw new Error('Failed to toggle pin');
			await loadComments();
		} catch (err) {
			console.error('Failed to toggle pin:', err);
			toasts.add('error', 'Failed to update comment');
		}
	}

	async function deleteComment(comment: Comment) {
		const confirmed = await confirm({
			title: $t('admin.comments.delete'),
			message: 'Delete this comment permanently?',
			confirmText: $t('admin.comments.delete'),
			danger: true
		});
		if (!confirmed) return;

		try {
			const response = await fetch(`/api/admin/comments/${comment.id}`, {
				method: 'DELETE',
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (!response.ok) throw new Error('Failed to delete comment');
			await loadComments();
			await loadPendingCount();
		} catch (err) {
			console.error('Failed to delete comment:', err);
			toasts.add('error', 'Failed to delete comment');
		}
	}

	async function bulkAction(action: string) {
		if (selectedIds.size === 0) return;

		if (action === 'delete') {
			const confirmed = await confirm({
				title: $t('admin.comments.bulk_delete'),
				message: `Delete ${selectedIds.size} comment(s) permanently?`,
				confirmText: $t('admin.comments.delete'),
				danger: true
			});
			if (!confirmed) return;
		}

		try {
			const response = await fetch('/api/admin/comments/bulk-action', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify({ ids: [...selectedIds], action })
			});
			if (!response.ok) throw new Error('Failed to perform bulk action');
			selectedIds = new Set();
			selectAll = false;
			await loadComments();
			await loadPendingCount();
		} catch (err) {
			console.error('Failed to perform bulk action:', err);
			toasts.add('error', 'Failed to perform bulk action');
		}
	}

	function getStatusClass(status: string): string {
		switch (status) {
			case 'pending':
				return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300';
			case 'approved':
				return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300';
			case 'spam':
				return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300';
			case 'rejected':
				return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400';
			default:
				return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400';
		}
	}

	function getEmptyMessage(): string {
		switch (activeTab) {
			case 'pending':
				return $t('admin.comments.no_pending');
			case 'spam':
				return $t('admin.comments.no_spam');
			case 'reported':
				return $t('admin.comments.no_reported');
			default:
				return $t('admin.comments.no_comments');
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.comments.title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
	<!-- Header -->
	<div>
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">
			{$t('admin.comments.title')}
		</h1>
		<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
			Moderate and manage comments across your content.
		</p>
	</div>

	<!-- Filter Tabs -->
	<div class="flex items-center gap-1" role="tablist" aria-label="Comment filters">
		<button
			type="button"
			role="tab"
			class="px-3 py-1.5 text-xs rounded-full transition-colors {activeTab === 'pending'
				? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'
				: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'}"
			aria-selected={activeTab === 'pending'}
			onclick={() => setTab('pending')}
		>
			{$t('admin.comments.pending')}
			{#if pendingCount > 0}
				<span class="ml-1 inline-flex items-center justify-center px-1.5 py-0.5 text-xs font-medium rounded-full bg-yellow-200 text-yellow-800 dark:bg-yellow-800 dark:text-yellow-200">
					{pendingCount}
				</span>
			{/if}
		</button>
		<button
			type="button"
			role="tab"
			class="px-3 py-1.5 text-xs rounded-full transition-colors {activeTab === 'approved'
				? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
				: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'}"
			aria-selected={activeTab === 'approved'}
			onclick={() => setTab('approved')}
		>
			{$t('admin.comments.approved')}
		</button>
		<button
			type="button"
			role="tab"
			class="px-3 py-1.5 text-xs rounded-full transition-colors {activeTab === 'spam'
				? 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
				: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'}"
			aria-selected={activeTab === 'spam'}
			onclick={() => setTab('spam')}
		>
			{$t('admin.comments.spam')}
		</button>
		<button
			type="button"
			role="tab"
			class="px-3 py-1.5 text-xs rounded-full transition-colors {activeTab === 'reported'
				? 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300'
				: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'}"
			aria-selected={activeTab === 'reported'}
			onclick={() => setTab('reported')}
		>
			{$t('admin.comments.reported')}
		</button>
	</div>

	<!-- Bulk Actions -->
	{#if selectedIds.size > 0}
		<div
			class="flex items-center gap-3 p-3 bg-primary-50 dark:bg-primary-900/20 rounded-lg border border-primary-200 dark:border-primary-800"
			role="status"
		>
			<input
				type="checkbox"
				checked={selectAll}
				onchange={toggleSelectAll}
				aria-label="Select all"
				class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
			/>
			<span class="text-sm text-primary-700 dark:text-primary-300">
				{$t('admin.comments.selected_count', { values: { count: selectedIds.size } })}
			</span>
			<div class="flex items-center gap-2 ml-auto">
				<button type="button" class="btn btn-sm bg-green-700 hover:bg-green-800 text-white" onclick={() => bulkAction('approve')}>
					{$t('admin.comments.bulk_approve')}
				</button>
				<button type="button" class="btn btn-sm bg-gray-600 hover:bg-gray-700 text-white" onclick={() => bulkAction('reject')}>
					{$t('admin.comments.bulk_reject')}
				</button>
				<button type="button" class="btn btn-sm bg-orange-700 hover:bg-orange-800 text-white" onclick={() => bulkAction('spam')}>
					{$t('admin.comments.bulk_spam')}
				</button>
				<button type="button" class="btn btn-danger btn-sm" onclick={() => bulkAction('delete')}>
					{$t('admin.comments.bulk_delete')}
				</button>
			</div>
		</div>
	{/if}

	<!-- Comment List -->
	<div class="card">
		{#if loading}
			<div class="flex items-center justify-center py-12" role="status" aria-label="Loading comments">
				<svg class="animate-spin h-6 w-6 text-primary-600" fill="none" viewBox="0 0 24 24" aria-hidden="true">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
			</div>
		{:else if comments.length === 0}
			<div class="text-center py-12 px-6">
				<svg class="w-12 h-12 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
				</svg>
				<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
					{getEmptyMessage()}
				</h3>
			</div>
		{:else}
			<!-- Select all header -->
			<div class="flex items-center gap-3 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
				<input
					type="checkbox"
					checked={selectAll}
					onchange={toggleSelectAll}
					aria-label="Select all comments"
					class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
				/>
				<span class="text-xs text-gray-500 dark:text-gray-400">
					{totalItems} comment{totalItems === 1 ? '' : 's'}
				</span>
			</div>

			<!-- Comment cards -->
			{#each comments as comment (comment.id)}
				<div class="flex gap-3 px-4 py-4 border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
					<!-- Checkbox -->
					<div class="pt-0.5">
						<input
							type="checkbox"
							checked={selectedIds.has(comment.id)}
							onchange={() => toggleSelect(comment.id)}
							aria-label="Select comment by {comment.author_name}"
							class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
						/>
					</div>

					<!-- Comment content -->
					<div class="flex-1 min-w-0">
						<!-- Author + meta row -->
						<div class="flex items-center gap-2 mb-1 flex-wrap">
							<span class="font-medium text-sm text-gray-900 dark:text-white">{comment.author_name}</span>
							<span class="text-xs text-gray-400 dark:text-gray-500">{comment.author_email}</span>
							<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium {getStatusClass(comment.status)}">
								{$t(`admin.comments.${comment.status}`)}
							</span>
							{#if comment.is_pinned}
								<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
									{$t('admin.comments.pin')}
								</span>
							{/if}
							{#if comment.report_count > 0}
								<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300">
									{$t('admin.comments.reports')}: {comment.report_count}
								</span>
							{/if}
						</div>

						<!-- Content reference -->
						<div class="text-xs text-gray-500 dark:text-gray-400 mb-1">
							<span class="capitalize">{comment.content_type}</span>
							{#if comment.content_title}
								&mdash;
								<a href="/admin/{comment.content_type}s/{comment.content_id}" class="text-primary-600 dark:text-primary-400 hover:underline">
									{comment.content_title}
								</a>
							{/if}
						</div>

						<!-- Comment body -->
						<p class="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap break-words">
							{comment.body}
						</p>

						<!-- Timestamp -->
						<div class="text-xs text-gray-400 dark:text-gray-500 mt-1">
							{new Date(comment.created).toLocaleString()}
						</div>
					</div>

					<!-- Action buttons -->
					<div class="flex items-start gap-1 shrink-0">
						{#if comment.status !== 'approved'}
							<button
								type="button"
								class="p-1.5 text-gray-400 hover:text-green-600 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
								onclick={() => updateStatus(comment.id, 'approved')}
								aria-label={$t('admin.comments.approve')}
								title={$t('admin.comments.approve')}
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
							</button>
						{/if}
						{#if comment.status !== 'rejected'}
							<button
								type="button"
								class="p-1.5 text-gray-400 hover:text-red-500 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
								onclick={() => updateStatus(comment.id, 'rejected')}
								aria-label={$t('admin.comments.reject')}
								title={$t('admin.comments.reject')}
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
								</svg>
							</button>
						{/if}
						{#if comment.status !== 'spam'}
							<button
								type="button"
								class="p-1.5 text-gray-400 hover:text-orange-500 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
								onclick={() => updateStatus(comment.id, 'spam')}
								aria-label={$t('admin.comments.mark_spam')}
								title={$t('admin.comments.mark_spam')}
							>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 21v-4m0 0V5a2 2 0 012-2h6.5l1 1H21l-3 6 3 6h-8.5l-1-1H5a2 2 0 00-2 2zm9-13.5V9" />
								</svg>
							</button>
						{/if}
						<button
							type="button"
							class="p-1.5 text-gray-400 hover:text-blue-500 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
							onclick={() => togglePin(comment)}
							aria-label={comment.is_pinned ? $t('admin.comments.unpin') : $t('admin.comments.pin')}
							title={comment.is_pinned ? $t('admin.comments.unpin') : $t('admin.comments.pin')}
						>
							<svg class="w-4 h-4" fill={comment.is_pinned ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
							</svg>
						</button>
						<button
							type="button"
							class="p-1.5 text-gray-400 hover:text-red-500 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
							onclick={() => deleteComment(comment)}
							aria-label={$t('admin.comments.delete')}
							title={$t('admin.comments.delete')}
						>
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
							</svg>
						</button>
					</div>
				</div>
			{/each}

			<!-- Pagination -->
			{#if totalPages > 1}
				<nav class="flex items-center justify-between px-4 py-3 border-t border-gray-200 dark:border-gray-700" aria-label="Comment pagination">
					<p class="text-sm text-gray-500 dark:text-gray-400">
						Showing {(currentPage - 1) * perPage + 1} to {Math.min(currentPage * perPage, totalItems)} of {totalItems}
					</p>
					<div class="flex gap-1">
						<button type="button" class="btn btn-ghost btn-sm" disabled={currentPage <= 1} onclick={() => goToPage(currentPage - 1)} aria-label="Previous page">
							&laquo;
						</button>
						{#each Array.from({ length: Math.min(totalPages, 5) }, (_, i) => {
							const start = Math.max(1, Math.min(currentPage - 2, totalPages - 4));
							return start + i;
						}) as p (p)}
							<button
								type="button"
								class="btn btn-sm {p === currentPage ? 'btn-primary' : 'btn-ghost'}"
								onclick={() => goToPage(p)}
								aria-current={p === currentPage ? 'page' : undefined}
							>
								{p}
							</button>
						{/each}
						<button type="button" class="btn btn-ghost btn-sm" disabled={currentPage >= totalPages} onclick={() => goToPage(currentPage + 1)} aria-label="Next page">
							&raquo;
						</button>
					</div>
				</nav>
			{/if}
		{/if}
	</div>
</div>
