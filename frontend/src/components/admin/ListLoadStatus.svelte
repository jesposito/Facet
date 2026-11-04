<script lang="ts">
	import { t } from 'svelte-i18n';
	import type { ListLoadState } from '$lib/admin/list-load';

	interface Props {
		status: ListLoadState;
		label: string;
		onretry: () => void;
	}

	let { status, label, onretry }: Props = $props();
</script>

{#if status.failed}
	<div role="alert" class="card p-4 mb-4 border-red-300 dark:border-red-700">
		<p class="text-red-700 dark:text-red-300">
			{$t(status.loaded ? 'admin.list_status.refresh_failed' : 'admin.list_status.load_failed', {
				values: { type: label }
			})}
		</p>
		<button
			type="button"
			class="btn btn-secondary mt-3"
			onclick={onretry}
			disabled={status.loading}
		>
			{$t('admin.dashboard.retry')}
		</button>
	</div>
{/if}

<div role="status">
	{#if status.mutating}
		<p class="mb-4 text-sm text-gray-700 dark:text-gray-300">{$t('admin.list_status.saving')}</p>
	{/if}
	{#if status.ancillaryFailed}
		<p class="mb-4 text-sm text-gray-700 dark:text-gray-300">
			{$t('admin.list_status.ancillary_failed')}
		</p>
	{/if}
</div>
