<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { icon } from '$lib/icons';
	import { t } from 'svelte-i18n';

	interface Props {
		accept?: string;
		multiple?: boolean;
		disabled?: boolean;
		class?: string;
	}

	let {
		accept = '*/*',
		multiple = true,
		disabled = false,
		class: className = ''
	}: Props = $props();

	const dispatch = createEventDispatcher<{
		drop: { files: File[] };
	}>();

	let isDragging = $state(false);
	let dragCounter = 0;

	function handleDragEnter(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		if (disabled) return;
		dragCounter++;
		if (e.dataTransfer?.types.includes('Files')) {
			isDragging = true;
		}
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		dragCounter--;
		if (dragCounter === 0) {
			isDragging = false;
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		if (disabled) return;
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'copy';
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		e.stopPropagation();
		isDragging = false;
		dragCounter = 0;
		if (disabled) return;

		const files = Array.from(e.dataTransfer?.files || []);
		if (files.length === 0) return;

		// Filter by accept types if specified
		const filteredFiles = filterFilesByAccept(files, accept);
		if (filteredFiles.length === 0) return;

		// If not multiple, only take the first file
		const finalFiles = multiple ? filteredFiles : [filteredFiles[0]];
		dispatch('drop', { files: finalFiles });
	}

	function filterFilesByAccept(files: File[], acceptPattern: string): File[] {
		if (acceptPattern === '*/*' || !acceptPattern) return files;

		const patterns = acceptPattern.split(',').map(p => p.trim().toLowerCase());

		return files.filter(file => {
			const fileType = file.type.toLowerCase();
			const fileName = file.name.toLowerCase();

			return patterns.some(pattern => {
				// Match by MIME type (e.g., 'image/*', 'image/png')
				if (pattern.includes('/')) {
					if (pattern.endsWith('/*')) {
						const baseType = pattern.slice(0, -2);
						return fileType.startsWith(baseType);
					}
					return fileType === pattern;
				}
				// Match by extension (e.g., '.jpg', '.png')
				if (pattern.startsWith('.')) {
					return fileName.endsWith(pattern);
				}
				return false;
			});
		});
	}

	function handleClick() {
		if (disabled) return;
		const input = document.createElement('input');
		input.type = 'file';
		input.accept = accept;
		input.multiple = multiple;
		input.onchange = () => {
			const files = Array.from(input.files || []);
			if (files.length > 0) {
				dispatch('drop', { files: multiple ? files : [files[0]] });
			}
		};
		input.click();
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			handleClick();
		}
	}
</script>

<div
	class="drop-zone {className} {isDragging ? 'dragging' : ''} {disabled ? 'disabled' : ''}"
	ondragenter={handleDragEnter}
	ondragleave={handleDragLeave}
	ondragover={handleDragOver}
	ondrop={handleDrop}
	onclick={handleClick}
	onkeydown={handleKeyDown}
	role="button"
	tabindex={disabled ? -1 : 0}
	aria-label={$t('admin.media.dropzone_label')}
	aria-disabled={disabled}
>
	<div class="drop-zone-content">
		<div class="drop-zone-icon">
			{@html icon('download')}
		</div>
		<div class="drop-zone-text">
			{#if isDragging}
				<span class="drop-zone-primary">{$t('admin.media.dropzone_drop')}</span>
			{:else}
				<span class="drop-zone-primary">{$t('admin.media.dropzone_drag')}</span>
				<span class="drop-zone-secondary">{$t('admin.media.dropzone_or_click')}</span>
			{/if}
		</div>
	</div>
</div>

<style>
	.drop-zone {
		border: 2px dashed var(--color-gray-300);
		border-radius: 0.5rem;
		padding: 2rem;
		text-align: center;
		cursor: pointer;
		transition: all 0.2s ease;
		background-color: var(--color-gray-50);
	}

	:global(.dark) .drop-zone {
		border-color: var(--color-gray-600);
		background-color: var(--color-gray-800);
	}

	.drop-zone:hover:not(.disabled),
	.drop-zone:focus:not(.disabled) {
		border-color: var(--color-primary-500);
		background-color: var(--color-primary-50);
	}

	:global(.dark) .drop-zone:hover:not(.disabled),
	:global(.dark) .drop-zone:focus:not(.disabled) {
		border-color: var(--color-primary-400);
		background-color: var(--color-primary-900);
	}

	.drop-zone.dragging {
		border-color: var(--color-primary-500);
		background-color: var(--color-primary-100);
		border-style: solid;
	}

	:global(.dark) .drop-zone.dragging {
		border-color: var(--color-primary-400);
		background-color: var(--color-primary-800);
	}

	.drop-zone.disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.drop-zone-content {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
	}

	.drop-zone-icon {
		color: var(--color-gray-400);
		transform: scale(1.5);
	}

	:global(.dark) .drop-zone-icon {
		color: var(--color-gray-500);
	}

	.drop-zone.dragging .drop-zone-icon,
	.drop-zone:hover:not(.disabled) .drop-zone-icon {
		color: var(--color-primary-500);
	}

	:global(.dark) .drop-zone.dragging .drop-zone-icon,
	:global(.dark) .drop-zone:hover:not(.disabled) .drop-zone-icon {
		color: var(--color-primary-400);
	}

	.drop-zone-text {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.drop-zone-primary {
		font-weight: 500;
		color: var(--color-gray-700);
	}

	:global(.dark) .drop-zone-primary {
		color: var(--color-gray-200);
	}

	.drop-zone-secondary {
		font-size: 0.875rem;
		color: var(--color-gray-500);
	}

	:global(.dark) .drop-zone-secondary {
		color: var(--color-gray-400);
	}
</style>
