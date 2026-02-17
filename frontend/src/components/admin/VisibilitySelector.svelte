<script lang="ts">
    import { t } from 'svelte-i18n';

    interface Props {
        value: string;
        includePassword?: boolean;
        id?: string;
    }

    let { value = $bindable('public'), includePassword = false, id = 'visibility' }: Props = $props();

    // Reactive description based on current selection
    let description = $derived.by(() => {
        if (includePassword && value === 'password') {
            return $t('admin.content.common.visibility_password_desc');
        }
        return $t(`admin.content.common.visibility_${value}_desc`);
    });
</script>

<div>
    <label for={id} class="label">{$t('admin.content.common.visibility_label')}</label>
    <select {id} bind:value class="input">
        <option value="public">{$t('admin.content.common.visibility_public')}</option>
        <option value="unlisted">{$t('admin.content.common.visibility_unlisted')}</option>
        {#if includePassword}
            <option value="password">{$t('admin.content.common.visibility_password')}</option>
        {/if}
        <option value="private">{$t('admin.content.common.visibility_private')}</option>
    </select>
    <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{description}</p>
</div>
