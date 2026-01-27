<script lang="ts">
	import { t } from 'svelte-i18n';

	interface Props {
		visibility: string;
		isDraft: boolean;
	}

	let { visibility, isDraft }: Props = $props();
</script>

{#if isDraft}
	<span class="px-2 py-0.5 text-xs font-medium bg-amber-500 text-amber-900 rounded">
		{$t('shared.visibility.draft')}
	</span>
{:else if visibility === 'private'}
	<span class="px-2 py-0.5 text-xs font-medium bg-red-500 text-white rounded">
		{$t('shared.visibility.private')}
	</span>
{:else if visibility === 'unlisted'}
	<span class="px-2 py-0.5 text-xs font-medium bg-gray-500 text-white rounded">
		{$t('shared.visibility.unlisted')}
	</span>
{/if}
