<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { icon } from '$lib/icons';
	import { t } from 'svelte-i18n';
	import { formatDate } from '$lib/utils';

	interface MediaItem {
		collection: string;
		collection_id: string;
		record_id: string;
		field: string;
		filename: string;
		url: string;
		size: number;
		mime: string;
		uploaded_at: string;
		relative_path?: string;
		orphan?: boolean;
		display_name?: string;
		thumbnail_url?: string;
		external?: boolean;
		provider?: string;
		embed_url?: string;
	}

	interface Props {
		item: MediaItem | null;
	}

	let { item }: Props = $props();

	const dispatch = createEventDispatcher<{
		close: void;
	}>();

	function close() {
		dispatch('close');
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			close();
		}
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			close();
		}
	}

	function humanSize(bytes: number) {
		if (!bytes) return '0 B';
		const units = ['B', 'KB', 'MB', 'GB'];
		let i = 0;
		let size = bytes;
		while (size >= 1024 && i < units.length - 1) {
			size /= 1024;
			i++;
		}
		return `${size.toFixed(size >= 10 ? 0 : 1)} ${units[i]}`;
	}

	function isImage(item: MediaItem): boolean {
		return item.mime?.startsWith('image/') || false;
	}

	function isVideo(item: MediaItem): boolean {
		if (item.mime?.startsWith('video/')) return true;
		const provider = (item.provider || '').toLowerCase();
		return ['youtube', 'vimeo', 'loom'].includes(provider);
	}

	function isAudio(item: MediaItem): boolean {
		if (item.mime?.startsWith('audio/')) return true;
		const provider = (item.provider || '').toLowerCase();
		return ['spotify', 'soundcloud'].includes(provider);
	}

	function getEmbedUrl(item: MediaItem): string {
		if (item.embed_url) return item.embed_url;
		return item.url;
	}

	function getDisplayUrl(item: MediaItem): string {
		if (item.thumbnail_url) return item.thumbnail_url;
		return item.url;
	}
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if item}
	<div
		class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4"
		onclick={handleBackdropClick}
		onkeydown={handleKeyDown}
		role="dialog"
		aria-modal="true"
		aria-labelledby="preview-title"
		tabindex="-1"
	>
		<div class="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] flex flex-col overflow-hidden">
			<!-- Header -->
			<div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
				<h2 id="preview-title" class="text-lg font-semibold text-gray-900 dark:text-white truncate">
					{item.display_name || item.filename}
				</h2>
				<button
					class="btn btn-ghost btn-sm"
					onclick={close}
					aria-label={$t('admin.media.preview_close')}
				>
					{@html icon('x')}
				</button>
			</div>

			<!-- Preview area -->
			<div class="flex-1 overflow-auto p-4 bg-gray-100 dark:bg-gray-800 flex items-center justify-center min-h-[300px]">
				{#if isImage(item)}
					<img
						src={getDisplayUrl(item)}
						alt={item.display_name || item.filename}
						class="max-w-full max-h-[60vh] object-contain rounded"
					/>
				{:else if isVideo(item) && item.embed_url}
					<div class="w-full max-w-2xl aspect-video">
						<iframe
							src={getEmbedUrl(item)}
							title={item.display_name || item.filename}
							class="w-full h-full rounded"
							allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
							allowfullscreen
						></iframe>
					</div>
				{:else if isVideo(item)}
					<video
						src={item.url}
						controls
						class="max-w-full max-h-[60vh] rounded"
					>
						<track kind="captions" />
					</video>
				{:else if isAudio(item) && item.embed_url}
					<div class="w-full max-w-xl">
						<iframe
							src={getEmbedUrl(item)}
							title={item.display_name || item.filename}
							class="w-full h-20 rounded"
							allow="autoplay; encrypted-media"
						></iframe>
					</div>
				{:else if isAudio(item)}
					<audio src={item.url} controls class="w-full max-w-md">
						Your browser does not support the audio element.
					</audio>
				{:else if item.external}
					<div class="text-center p-8">
						<div class="text-4xl text-gray-400 dark:text-gray-500 mb-4">
							{@html icon('link')}
						</div>
						<p class="text-gray-600 dark:text-gray-400 mb-4">{$t('admin.media.preview_external')}</p>
						<a
							href={item.url}
							target="_blank"
							rel="noopener noreferrer"
							class="btn btn-primary"
						>
							{$t('admin.media.preview_open_link')}
						</a>
					</div>
				{:else}
					<div class="text-center p-8">
						<div class="text-4xl text-gray-400 dark:text-gray-500 mb-4">
							{@html icon('file')}
						</div>
						<p class="text-gray-600 dark:text-gray-400 mb-4">{$t('admin.media.preview_no_preview')}</p>
						<a
							href={item.url}
							target="_blank"
							rel="noopener noreferrer"
							class="btn btn-primary"
						>
							{$t('admin.media.preview_download')}
						</a>
					</div>
				{/if}
			</div>

			<!-- Metadata footer -->
			<div class="p-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
				<div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
					{#if item.filename}
						<div>
							<span class="text-gray-500 dark:text-gray-400">{$t('admin.media.preview_filename')}</span>
							<p class="text-gray-900 dark:text-white truncate" title={item.filename}>{item.filename}</p>
						</div>
					{/if}
					{#if item.mime}
						<div>
							<span class="text-gray-500 dark:text-gray-400">{$t('admin.media.preview_type')}</span>
							<p class="text-gray-900 dark:text-white">{item.mime}</p>
						</div>
					{/if}
					{#if item.size}
						<div>
							<span class="text-gray-500 dark:text-gray-400">{$t('admin.media.preview_size')}</span>
							<p class="text-gray-900 dark:text-white">{humanSize(item.size)}</p>
						</div>
					{/if}
					{#if item.uploaded_at}
						<div>
							<span class="text-gray-500 dark:text-gray-400">{$t('admin.media.preview_uploaded')}</span>
							<p class="text-gray-900 dark:text-white">{formatDate(item.uploaded_at, { month: 'short', day: 'numeric', year: 'numeric' })}</p>
						</div>
					{/if}
					{#if item.provider}
						<div>
							<span class="text-gray-500 dark:text-gray-400">{$t('admin.media.preview_provider')}</span>
							<p class="text-gray-900 dark:text-white capitalize">{item.provider}</p>
						</div>
					{/if}
					{#if item.collection}
						<div>
							<span class="text-gray-500 dark:text-gray-400">{$t('admin.media.preview_collection')}</span>
							<p class="text-gray-900 dark:text-white">{item.collection}</p>
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}
