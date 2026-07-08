<script lang="ts">
	import { preventDefault } from 'svelte/legacy';
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { toasts, confirm } from '$lib/stores';
	import { TAG_COLORS, TAG_COLOR_LIST, type TagColor } from '$lib/colors';
	import AdminTagBadge from '$components/admin/AdminTagBadge.svelte';

	interface AdminTag {
		id: string;
		name: string;
		color: string;
		sort_order: number;
	}

	let tags: AdminTag[] = $state([]);
	let loading = $state(true);
	let saving = $state(false);

	let showForm = $state(false);
	let editingTag: AdminTag | null = $state(null);
	let formName = $state('');
	let formColor: TagColor = $state('blue');

	onMount(loadTags);

	async function loadTags() {
		loading = true;
		try {
			const result = await pb.collection('admin_tags').getList<AdminTag>(1, 100, {
				sort: 'sort_order,name'
			});
			tags = result.items;
		} catch (err) {
			console.error('Failed to load tags:', err);
			toasts.add('error', $t('admin.tags_page.load_failed'));
		} finally {
			loading = false;
		}
	}

	function openNewForm() {
		editingTag = null;
		formName = '';
		formColor = 'blue';
		showForm = true;
	}

	function openEditForm(tag: AdminTag) {
		editingTag = tag;
		formName = tag.name;
		formColor = (tag.color as TagColor) || 'blue';
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		editingTag = null;
		formName = '';
		formColor = 'blue';
	}

	async function handleSubmit() {
		if (!formName.trim()) {
			toasts.add('error', $t('admin.tags_page.name_required'));
			return;
		}

		saving = true;
		try {
			const data = {
				name: formName.trim(),
				color: formColor,
				sort_order: editingTag?.sort_order ?? tags.length
			};

			if (editingTag) {
				await pb.collection('admin_tags').update(editingTag.id, data);
				toasts.add('success', $t('admin.tags_page.tag_updated'));
			} else {
				await pb.collection('admin_tags').create(data);
				toasts.add('success', $t('admin.tags_page.tag_created'));
			}

			closeForm();
			await loadTags();
		} catch (err: any) {
			console.error('Failed to save tag:', err);
			const message = err?.response?.data?.name?.message || $t('admin.tags_page.save_failed');
			toasts.add('error', message);
		} finally {
			saving = false;
		}
	}

	async function deleteTag(tag: AdminTag) {
		const confirmed = await confirm({
			title: $t('admin.tags_page.delete_confirm_title'),
			message: $t('admin.tags_page.delete_confirm_message', { values: { name: tag.name } }),
			confirmText: $t('admin.tags_page.delete_confirm_button'),
			danger: true
		});
		if (!confirmed) return;

		try {
			await pb.collection('admin_tags').delete(tag.id);
			toasts.add('success', $t('admin.tags_page.tag_deleted'));
			await loadTags();
		} catch (err) {
			console.error('Failed to delete tag:', err);
			toasts.add('error', $t('admin.tags_page.delete_failed'));
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.tags_page.title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 mb-6">
		<div>
			<div class="flex items-center gap-2 mb-1">
				<a href="/admin/settings" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" aria-label={$t('admin.tags_page.back_to_settings')}>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</a>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.tags_page.title')}</h1>
			</div>
			<p class="text-sm text-gray-600 dark:text-gray-400">
				{$t('admin.tags_page.description')}
			</p>
		</div>
		<button class="btn btn-primary w-full sm:w-auto" onclick={openNewForm}>
			{$t('admin.tags_page.new_tag')}
		</button>
	</div>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">{$t('admin.tags_page.loading')}</div>
		</div>
	{:else if showForm}
		<form onsubmit={preventDefault(handleSubmit)} class="card p-6 space-y-4">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
					{editingTag ? $t('admin.tags_page.form_title_edit') : $t('admin.tags_page.form_title_new')}
				</h2>
				<button type="button" class="text-gray-500 hover:text-gray-700" onclick={closeForm} aria-label={$t('shared.aria.close')}>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<div>
				<label for="tag_name" class="label">{$t('admin.tags_page.name_label')}</label>
				<input
					type="text"
					id="tag_name"
					bind:value={formName}
					class="input"
					placeholder={$t('admin.tags_page.name_placeholder')}
					required
					maxlength="50"
				/>
			</div>

			<div>
				<span id="color-picker-label" class="label">{$t('admin.tags_page.color_label')}</span>
				<div class="flex flex-wrap gap-2" role="radiogroup" aria-labelledby="color-picker-label">
					{#each TAG_COLOR_LIST as color}
						{@const info = TAG_COLORS[color]}
						<button
							type="button"
							class="flex items-center gap-2 px-3 py-2 rounded-lg border transition-all
								{formColor === color
									? 'ring-2 ring-primary-500 border-primary-300 dark:border-primary-700'
									: 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'}"
							onclick={() => formColor = color}
						>
							<span class="w-4 h-4 rounded-full {info.bg} {info.border} border"></span>
							<span class="text-sm text-gray-700 dark:text-gray-200">{info.label}</span>
						</button>
					{/each}
				</div>
			</div>

			<div class="pt-2">
				<p class="label">{$t('admin.tags_page.preview_label')}</p>
				<AdminTagBadge name={formName || 'Tag Name'} color={formColor} size="md" />
			</div>

			<div class="flex justify-end gap-3 pt-2">
				<button type="button" class="btn btn-secondary" onclick={closeForm}>{$t('admin.tags_page.cancel')}</button>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{editingTag ? $t('admin.tags_page.update_tag') : $t('admin.tags_page.create_tag')}
				</button>
			</div>
		</form>
	{:else if tags.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('admin.tags_page.empty_title')}</h3>
			<p class="text-gray-500 dark:text-gray-400 mb-4">
				{$t('admin.tags_page.empty_description')}
			</p>
			<button class="btn btn-primary" onclick={openNewForm}>
				{$t('admin.tags_page.empty_create_button')}
			</button>
		</div>
	{:else}
		<div class="card divide-y divide-gray-200 dark:divide-gray-700">
			{#each tags as tag (tag.id)}
				<div class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<AdminTagBadge name={tag.name} color={tag.color} size="md" />
					</div>
					<div class="flex items-center gap-2">
						<button
							class="p-2 text-gray-500 hover:text-blue-600 transition-colors"
							onclick={() => openEditForm(tag)}
							title={$t('admin.tags_page.edit_title')}
						>
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
							</svg>
						</button>
						<button
							class="p-2 text-gray-500 hover:text-red-600 transition-colors"
							onclick={() => deleteTag(tag)}
							title={$t('admin.tags_page.delete_title')}
						>
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
							</svg>
						</button>
					</div>
				</div>
			{/each}
		</div>

		<p class="mt-4 text-sm text-gray-500 dark:text-gray-400 text-center">
			{$t('admin.tags_page.footer_note')}
		</p>
	{/if}
</div>
