<script lang="ts">
	import { t } from 'svelte-i18n';
	import { withFacet } from '$lib/discovery';

	interface Props {
		contentType: 'posts' | 'projects';
		contentId: string;
		fromView?: string | null;
	}

	type DownloadFile = { filename: string; display_name: string; url: string };
	type FileRead = DownloadFile[] | 'refused' | 'failed';

	const READ_TIMEOUT_MS = 10_000;
	const SIGNED_LINK = /^\/api\/downloads\/[\w-]+$/;

	let { contentType, contentId, fromView = null }: Props = $props();

	let filesPath = $derived(
		withFacet(`/api/content/${contentType}/${contentId}/download-urls`, fromView)
	);

	let files = $state<DownloadFile[]>([]);
	let loading = $state(false);
	let failed = $state(false);
	let pending = $state<string | null>(null);
	let notice = $state<string | null>(null);

	// The read in flight. Starting another read, or leaving, abandons it.
	let request: AbortController | null = null;

	function isText(value: unknown): boolean {
		return typeof value === 'string' && value !== '';
	}

	// One bad entry fails the whole reply: only same-site signed links get rendered.
	function parseFiles(data: unknown): DownloadFile[] | null {
		const list = (data as { files?: unknown } | null)?.files;
		if (!Array.isArray(list)) return null;
		const valid = list.every(
			(file) =>
				isText(file?.filename) &&
				isText(file?.display_name) &&
				isText(file?.url) &&
				SIGNED_LINK.test(file.url)
		);
		return valid ? list : null;
	}

	// Resolves to null when the read was abandoned, so its answer must go unused.
	async function read(): Promise<FileRead | null> {
		request?.abort();
		const controller = new AbortController();
		request = controller;
		const timeout = setTimeout(() => controller.abort(), READ_TIMEOUT_MS);
		let result: FileRead = 'failed';
		try {
			const resp = await fetch(filesPath, { signal: controller.signal, cache: 'no-store' });
			if ([401, 403, 404].includes(resp.status)) result = 'refused';
			else if (resp.ok) result = parseFiles(await resp.json()) ?? 'failed';
		} catch {
			// A network failure, timeout or unreadable body stays 'failed'.
		} finally {
			clearTimeout(timeout);
		}
		return request === controller ? result : null;
	}

	// Someone who may not have the files gets a refusal and sees no list. A
	// server or network problem says so instead, because the files may be there.
	async function load() {
		loading = true;
		const result = await read();
		if (result === null) return;
		loading = false;
		failed = result === 'failed';
		files = Array.isArray(result) ? result : [];
	}

	// Signed links expire after 15 minutes and a page can stay open longer, so
	// each click asks for a new one, under the visitor's access at that moment.
	async function download(event: MouseEvent, file: DownloadFile) {
		if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey)
			return;
		event.preventDefault();
		if (pending) return;
		pending = file.filename;
		notice = null;
		const result = await read();
		if (result === null) return;
		pending = null;
		if (result === 'failed') {
			notice = 'downloads.link_failed';
			return;
		}
		files = result === 'refused' ? [] : result;
		const latest = files.find((candidate) => candidate.filename === file.filename);
		if (latest) window.location.assign(latest.url);
		else notice = 'downloads.unavailable';
	}

	$effect(() => {
		files = [];
		failed = false;
		pending = null;
		notice = null;
		void load();
		return () => {
			request?.abort();
			request = null;
		};
	});
</script>

{#if files.length > 0 || failed || notice}
	<section class="mb-10" aria-label={$t('downloads.title')}>
		<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
			{$t('downloads.title')}
		</h2>
		{#if failed}
			<p role="alert" class="mb-3 text-sm text-gray-700 dark:text-gray-300">
				{$t('downloads.load_failed')}
			</p>
			<button
				type="button"
				onclick={load}
				disabled={loading}
				aria-busy={loading}
				class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-lg border border-gray-200 dark:border-gray-700 transition-colors disabled:opacity-50"
			>
				{$t('downloads.retry')}
			</button>
		{/if}
		{#if files.length > 0}
			<ul class="space-y-2">
				{#each files as file (file.filename)}
					<li>
						<a
							href={file.url}
							aria-busy={pending === file.filename}
							onclick={(event) => download(event, file)}
							class="text-primary-600 dark:text-primary-400 hover:underline"
						>
							{file.display_name}
						</a>
					</li>
				{/each}
			</ul>
		{/if}
		{#if notice}
			<p role="alert" class="mt-3 text-sm text-gray-700 dark:text-gray-300">{$t(notice)}</p>
		{/if}
	</section>
{/if}
