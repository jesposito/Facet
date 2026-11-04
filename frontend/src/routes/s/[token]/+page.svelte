<script lang="ts">
	import { t } from 'svelte-i18n';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	// This counts down on the page only. Another check always waits for the
	// recipient's click, because a check can spend a use of the link.
	let waited = $state(0);
	const remaining = $derived(Math.max(0, (data.retryAfter ?? 0) - waited));

	$effect(() => {
		if (remaining === 0) return;
		const timer = setTimeout(() => (waited += 1), 1000);
		return () => clearTimeout(timer);
	});
</script>

<svelte:head>
	<title>{$t('public.share_entry.page_title')}</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
	<div class="text-center max-w-md mx-auto px-4">
		<div class="card p-8" data-code={data.code}>
			<div
				class="w-16 h-16 mx-auto mb-4 rounded-full bg-red-100 dark:bg-red-900 flex items-center justify-center"
			>
				<svg
					class="w-8 h-8 text-red-600 dark:text-red-400"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
					/>
				</svg>
			</div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">
				{$t(`public.share_entry.${data.code}.title`)}
			</h1>
			<p class="text-gray-600 dark:text-gray-400 mb-6">
				{$t(`public.share_entry.${data.code}.message`)}
			</p>
			<div class="flex flex-wrap justify-center gap-3">
				{#if data.code !== 'invalid'}
					<button
						type="button"
						class="btn btn-primary"
						disabled={remaining > 0}
						onclick={() => location.reload()}
					>
						{remaining > 0
							? $t('public.share_entry.retry_in', { values: { seconds: remaining } })
							: $t('public.share_entry.retry')}
					</button>
				{/if}
				<a href="/" class={data.code === 'invalid' ? 'btn btn-primary' : 'btn btn-secondary'}>
					{$t('public.share_entry.home')}
				</a>
			</div>
		</div>
	</div>
</div>
