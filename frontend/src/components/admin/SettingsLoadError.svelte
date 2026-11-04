<script lang="ts">
	import { t } from 'svelte-i18n';

	let {
		message,
		pending,
		onretry
	}: {
		message: string;
		pending: boolean;
		onretry: () => void;
	} = $props();
</script>

<div
	class="mb-4 card p-4 border-l-4 border-red-500 bg-red-50 dark:bg-red-900/20"
	role="alert"
	aria-busy={pending}
>
	<div class="flex flex-wrap items-center gap-3">
		<p class="flex-1 min-w-[12rem] text-sm text-red-700 dark:text-red-300">{message}</p>
		<button
			type="button"
			class="btn btn-sm btn-secondary"
			aria-disabled={pending}
			onclick={onretry}
		>
			{pending
				? $t('admin.settings_page.recoverycopy.retrying')
				: $t('admin.settings_page.recoverycopy.retry')}
		</button>
	</div>
</div>
