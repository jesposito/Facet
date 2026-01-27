<script lang="ts">
	/**
	 * ViewBasicInfo - Basic information form section
	 *
	 * Handles name, slug, description, visibility, and password fields.
	 */
	import { t } from 'svelte-i18n';

	// Props
	let {
		name = $bindable(),
		slug = $bindable(), 
		description = $bindable(),
		visibility = $bindable(),
		password = $bindable(),
		view
	}: {
		name: string;
		slug: string;
		description: string;
		visibility: 'public' | 'unlisted' | 'private' | 'password';
		password: string;
		view: any;
	} = $props();

	/**
	 * Generate a URL slug from a name
	 */
	function generateSlug(text: string): string {
		return text
			.toLowerCase()
			.replace(/[^a-z0-9\s-]/g, '')
			.trim()
			.replace(/\s+/g, '-')
			.replace(/-+/g, '-')
			.replace(/^-|-$/g, '');
	}

	/**
	 * Auto-generate slug when name changes (if no existing slug)
	 */
	function handleNameChange() {
		if (!view?.slug) {
			slug = generateSlug(name);
		}
	}
</script>

	<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.view_editor.basic_info.title')}</h2>

	<div>
		<label for="name" class="label">{$t('admin.view_editor.basic_info.name_label')} *</label>
		<input
			type="text"
			id="name"
			bind:value={name}
			oninput={handleNameChange}
			class="input"
			placeholder={$t('admin.view_editor.basic_info.name_placeholder')}
			required
		/>
		<p class="text-xs text-gray-500 mt-1">{$t('admin.view_editor.basic_info.name_help')}</p>
	</div>

	<div>
		<label for="slug" class="label">{$t('admin.view_editor.basic_info.slug_label')} *</label>
		<div class="flex items-center gap-2">
			<span class="text-gray-500 text-sm">/</span>
			<input
				type="text"
				id="slug"
				bind:value={slug}
				class="input flex-1"
				placeholder={$t('admin.view_editor.basic_info.slug_placeholder')}
				required
			/>
		</div>
		<p class="text-xs text-gray-500 mt-1">{$t('admin.view_editor.basic_info.slug_help', { values: { slug } })}</p>
	</div>

	<div>
		<label for="description" class="label">{$t('admin.view_editor.basic_info.description_label')}</label>
		<textarea
			id="description"
			bind:value={description}
			class="input min-h-[80px]"
			placeholder={$t('admin.view_editor.basic_info.description_placeholder')}
		></textarea>
		<p class="text-xs text-gray-500 mt-1">{$t('admin.view_editor.basic_info.description_help')}</p>
	</div>

	<div>
		<label for="visibility" class="label">{$t('admin.view_editor.basic_info.visibility_label')} *</label>
		<select id="visibility" bind:value={visibility} class="input">
			<option value="public">{$t('admin.view_editor.basic_info.visibility_public')}</option>
			<option value="unlisted">{$t('admin.view_editor.basic_info.visibility_unlisted')}</option>
			<option value="password">{$t('admin.view_editor.basic_info.visibility_password')}</option>
			<option value="private">{$t('admin.view_editor.basic_info.visibility_private')}</option>
		</select>
		<p class="text-xs text-gray-500 mt-1">{$t('admin.view_editor.basic_info.visibility_help')}</p>
	</div>

	{#if visibility === 'password'}
		<div>
			<label for="password" class="label">
				{password ? $t('admin.view_editor.basic_info.password_change') : $t('admin.view_editor.basic_info.password_set')} *
			</label>
			<input
				type="password"
				id="password"
				bind:value={password}
				class="input"
				placeholder={password ? $t('admin.view_editor.basic_info.password_placeholder_change') : $t('admin.view_editor.basic_info.password_placeholder_set')}
				autocomplete="new-password"
			/>
			<p class="text-xs text-gray-500 mt-1">
				{password ? $t('admin.view_editor.basic_info.password_help_keep') : $t('admin.view_editor.basic_info.password_help_visitors')}
			</p>
		</div>
	{/if}