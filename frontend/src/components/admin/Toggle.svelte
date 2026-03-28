<script lang="ts">
	interface Props {
		checked?: boolean;
		disabled?: boolean;
		label?: string;
		description?: string;
		id?: string;
	}

	let {
		checked = $bindable(false),
		disabled = false,
		label = '',
		description = '',
		id = ''
	}: Props = $props();
</script>

<label class="relative inline-flex items-center {disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}" for={id || undefined}>
	<input
		type="checkbox"
		class="sr-only peer"
		bind:checked
		{disabled}
		{id}
	/>
	<div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:after:border-gray-500 peer-checked:bg-primary-600"></div>
	{#if label}
		<div class="ms-3">
			<span class="text-sm text-gray-700 dark:text-gray-300">{label}</span>
			{#if description}
				<p class="text-xs text-gray-500 dark:text-gray-400">{description}</p>
			{/if}
		</div>
	{/if}
</label>
