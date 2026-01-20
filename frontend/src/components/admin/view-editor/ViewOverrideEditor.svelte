<script lang="ts">
	/**
	 * ViewOverrideEditor - Modal for customizing item field overrides per view
	 * 
	 * Allows users to override specific fields (title, description, bullets, etc.)
	 * for items within a particular view, without modifying the original data.
	 */
	import { OVERRIDABLE_FIELDS } from '$lib/pocketbase';
	import { icon } from '$lib/icons';
	import type { OverrideEditorState } from '$lib/view-editor/types';

	// Props
	let {
		editingOverride,
		onSave,
		onClose
	}: {
		editingOverride: OverrideEditorState;
		onSave: (overrides: Record<string, string | string[]>) => void;
		onClose: () => void;
	} = $props();

	const overridableFields = $derived(OVERRIDABLE_FIELDS[editingOverride.sectionKey] || []);

	let localOverrides = $state<Record<string, string | string[]>>({});
	let lastItemKey = $state('');

	const currentItemKey = $derived(`${editingOverride.sectionKey}:${editingOverride.itemId}`);

	$effect(() => {
		if (currentItemKey !== lastItemKey) {
			localOverrides = { ...editingOverride.overrides };
			lastItemKey = currentItemKey;
		}
	});

	/**
	 * Format a field value for display
	 */
	function formatFieldValue(value: unknown): string {
		if (Array.isArray(value)) {
			return value.join('\n');
		}
		return String(value || '');
	}

	/**
	 * Parse input value based on field type
	 */
	function parseFieldValue(field: string, value: string): string | string[] {
		// bullets field should be an array
		if (field === 'bullets') {
			return value.split('\n').filter(line => line.trim());
		}
		return value;
	}

	/**
	 * Handle input change for an override field
	 */
	function handleInput(field: string, event: Event) {
		const target = event.target as HTMLInputElement | HTMLTextAreaElement;
		localOverrides[field] = parseFieldValue(field, target.value);
	}

	/**
	 * Clear an override, reverting to original value
	 */
	function clearOverride(field: string) {
		delete localOverrides[field];
		localOverrides = { ...localOverrides }; // Trigger reactivity
	}

	/**
	 * Check if a field has an override
	 */
	function hasOverride(field: string): boolean {
		return field in localOverrides;
	}

	/**
	 * Save overrides and close modal
	 */
	function handleSave() {
		// Clean up empty overrides before saving
		const cleanedOverrides: Record<string, string | string[]> = {};
		for (const [field, value] of Object.entries(localOverrides)) {
			if (value && (typeof value === 'string' ? value.trim() : value.length > 0)) {
				cleanedOverrides[field] = value;
			}
		}
		onSave(cleanedOverrides);
	}
</script>

<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
	<div class="card w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col">
		<!-- Header -->
		<div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
			<div>
				<h2 class="text-lg font-bold text-gray-900 dark:text-white">Customize for this View</h2>
				<p class="text-sm text-gray-500">{editingOverride.itemLabel}</p>
			</div>
			<button type="button" class="btn btn-ghost" onclick={onClose}>
				{@html icon('x')}
			</button>
		</div>

		<!-- Content -->
		<div class="p-4 space-y-6 overflow-y-auto flex-1">
			<p class="text-sm text-gray-600 dark:text-gray-400">
				Override fields below to customize how this item appears in this view. Leave fields empty to use the original value.
			</p>

			{#each overridableFields as field}
				{@const originalValue = editingOverride.originalData[field]}
				{@const fieldHasOverride = hasOverride(field)}
				{@const isArrayField = field === 'bullets'}

				<div class="space-y-2">
					<div class="flex items-center justify-between">
						<label for="override_{field}" class="label capitalize">{field.replace('_', ' ')}</label>
						{#if fieldHasOverride}
							<button
								type="button"
								class="text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
								onclick={() => clearOverride(field)}
							>
								Reset to original
							</button>
						{/if}
					</div>

					<!-- Original value (collapsed) -->
					<details class="text-sm">
						<summary class="text-gray-500 cursor-pointer hover:text-gray-700 dark:hover:text-gray-300">
							View original value
						</summary>
						<div class="mt-2 p-2 bg-gray-50 dark:bg-gray-800 rounded text-gray-600 dark:text-gray-400 whitespace-pre-wrap text-xs">
							{formatFieldValue(originalValue) || '(empty)'}
						</div>
					</details>

					<!-- Override input -->
					{#if isArrayField}
						<textarea
							id="override_{field}"
							class="input min-h-[100px] font-mono text-sm"
							placeholder="Enter one item per line..."
							value={fieldHasOverride ? formatFieldValue(localOverrides[field]) : ''}
							oninput={(e) => handleInput(field, e)}
						></textarea>
						<p class="text-xs text-gray-500">Enter one bullet point per line</p>
					{:else if field === 'description' || field === 'summary'}
						<textarea
							id="override_{field}"
							class="input min-h-[100px]"
							placeholder="Enter override value or leave empty for original..."
							value={fieldHasOverride ? String(localOverrides[field]) : ''}
							oninput={(e) => handleInput(field, e)}
						></textarea>
					{:else}
						<input
							type="text"
							id="override_{field}"
							class="input"
							placeholder="Enter override value or leave empty for original..."
							value={fieldHasOverride ? String(localOverrides[field]) : ''}
							oninput={(e) => handleInput(field, e)}
						/>
					{/if}
				</div>
			{/each}

			{#if overridableFields.length === 0}
				<p class="text-gray-500 text-center py-8">
					This section type does not support field overrides.
				</p>
			{/if}
		</div>

		<!-- Footer -->
		<div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
			<button type="button" class="btn btn-ghost" onclick={onClose}>
				Cancel
			</button>
			<button type="button" class="btn btn-primary" onclick={handleSave}>
				Save Overrides
			</button>
		</div>
	</div>
</div>
