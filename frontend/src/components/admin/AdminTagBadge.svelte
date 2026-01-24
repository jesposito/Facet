<script lang="ts">
	import { getTagColor, type TagColor } from '$lib/colors';

	interface Props {
		name: string;
		color?: string;
		size?: 'sm' | 'md';
		removable?: boolean;
		onRemove?: () => void;
	}

	let { name, color, size = 'sm', removable = false, onRemove }: Props = $props();

	let colorInfo = $derived(getTagColor(color));
</script>

<span
	class="inline-flex items-center gap-1 rounded border font-medium
		{size === 'sm' ? 'px-1.5 py-0.5 text-xs' : 'px-2 py-1 text-sm'}
		{colorInfo.bg} {colorInfo.bgDark} {colorInfo.text} {colorInfo.textDark} {colorInfo.border} {colorInfo.borderDark}"
>
	{name}
	{#if removable && onRemove}
		<button
			type="button"
			class="ml-0.5 hover:opacity-70 transition-opacity"
			onclick={onRemove}
			aria-label="Remove tag {name}"
		>
			<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>
	{/if}
</span>
