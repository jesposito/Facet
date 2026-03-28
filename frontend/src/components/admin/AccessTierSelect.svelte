<script lang="ts">
	import { t } from 'svelte-i18n';

	interface Props {
		value: string;
		id?: string;
	}

	let { value = $bindable('free'), id = 'access_tier' }: Props = $props();

	let description = $derived.by(() => {
		if (value === 'paid') {
			return $t('admin.courses.access_paid_description');
		}
		return $t('admin.courses.access_free_description');
	});
</script>

<div>
	<label for={id} class="label">{$t('admin.courses.access_tier')}</label>
	<select {id} bind:value class="input">
		<option value="free">{$t('admin.courses.access_free')}</option>
		<option value="paid">{$t('admin.courses.access_paid')}</option>
	</select>
	<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{description}</p>
</div>
