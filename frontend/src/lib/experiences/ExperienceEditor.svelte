<script lang="ts">
	import SingleMediaPicker from '$lib/components/admin/SingleMediaPicker.svelte';
	import BlockEditor from './BlockEditor.svelte';
	import ExperienceRenderer from './ExperienceRenderer.svelte';
	import { newExperience, newBlock, experienceId } from './templates';
	import {
		experienceTones,
		readExperience,
		type ProjectExperience,
		type ExperienceBlock
	} from './types';
	let {
		value = $bindable(null),
		title,
		summary = '',
		onchange
	}: {
		value?: ProjectExperience | null;
		title: string;
		summary?: string;
		onchange?: () => void;
	} = $props();
	let preview = $state(false);
	let mobile = $state(false);
	let darkPreview = $state(false);
	const blockTypes: ExperienceBlock['type'][] = [
		'text',
		'image',
		'video',
		'code',
		'disclosure',
		'story'
	];
	let previewDoc = $derived(value ? readExperience({ ...value, enabled: true }) : null);
	function move<T>(items: T[], index: number, delta: number) {
		const next = index + delta;
		if (next < 0 || next >= items.length) return;
		[items[index], items[next]] = [items[next], items[index]];
		onchange?.();
	}
	function start(kind: 'video' | 'code' | 'text') {
		value = newExperience(kind);
		onchange?.();
	}
</script>

<div class="experience-editor card p-6 space-y-4" oninput={onchange} {onchange}>
	<h2 class="text-lg font-semibold">Project experience</h2>
	<p>
		Build a project people can explore, watch or try. Your description stays available when the
		experience is switched off.
	</p>
	{#if !value}
		<div class="experience-edit-row">
			<button type="button" onclick={() => start('text')}>Start a case study</button><button
				type="button"
				onclick={() => start('video')}>Start a video course</button
			><button type="button" onclick={() => start('code')}>Start a code walkthrough</button>
		</div>
	{:else if value.version !== 1}
		<p role="alert">
			This experience uses a newer format. Update Facet before editing it. Your saved content will
			be preserved.
		</p>
	{:else}
		<label class="experience-toggle"
			><input type="checkbox" bind:checked={value.enabled} /> Show this experience on the published project</label
		>
		<div class="experience-edit-row">
			<button type="button" aria-pressed={preview} onclick={() => (preview = !preview)}
				>{preview ? 'Return to editor' : 'Preview experience'}</button
			>
		</div>
		{#if preview}
			<div class="experience-edit-row">
				<label class="experience-toggle"
					><input type="checkbox" bind:checked={mobile} /> Mobile width</label
				><label class="experience-toggle"
					><input type="checkbox" bind:checked={darkPreview} /> Dark preview</label
				>
			</div>
			{#if previewDoc}<div
					class="experience-preview-frame"
					class:mobile
					class:dark={darkPreview}
					data-experience-theme={darkPreview ? 'dark' : 'light'}
				>
					<ExperienceRenderer experience={previewDoc} {title} {summary} preview />
				</div>
			{:else}<p role="alert">
					Some fields need attention before previewing. Check unique field keys, URL formats and the
					document limits.
				</p>{/if}
		{:else}
			<details class="experience-edit-item" open>
				<summary>Opening section</summary>
				<div class="experience-edit-fields">
					<label
						>Introduction<textarea rows="4" maxlength="50000" bind:value={value.hero.intro}
						></textarea></label
					>
					<div class="experience-edit-row">
						<label
							>Colour<select bind:value={value.hero.tone}
								>{#each experienceTones as tone}<option value={tone}>{tone}</option>{/each}</select
							></label
						><label
							>Image placement<select bind:value={value.hero.imagePosition}
								><option value="right">Right</option><option value="left">Left</option></select
							></label
						>
					</div>
					{#if value.hero.image}<SingleMediaPicker
							label="Opening image"
							bind:value={value.hero.image.src}
							{onchange}
						/><label
							>Image description<input maxlength="500" bind:value={value.hero.image.alt} /></label
						><button
							type="button"
							onclick={() => {
								if (value) delete value.hero.image;
								onchange?.();
							}}>Remove opening image</button
						>
					{:else}<button
							type="button"
							onclick={() => {
								if (value) value.hero.image = { src: '', alt: '' };
								onchange?.();
							}}>Add opening image</button
						>{/if}
					{#if value.hero.action}<div class="experience-edit-row">
							<label
								>Button label<input maxlength="200" bind:value={value.hero.action.label} /></label
							><label
								>Button link<input
									maxlength="2048"
									bind:value={value.hero.action.href}
									placeholder="#first-section or https://…"
								/></label
							>
						</div>
						<button
							type="button"
							onclick={() => {
								if (value) delete value.hero.action;
								onchange?.();
							}}>Remove button</button
						>
					{:else}<button
							type="button"
							onclick={() => {
								if (value) value.hero.action = { label: '', href: '' };
								onchange?.();
							}}>Add button</button
						>{/if}
				</div>
			</details>
			{#each value.sections as section, si (section.id)}
				<details class="experience-edit-item" open>
					<summary>Section {si + 1}: {section.title || 'Untitled'}</summary>
					<div class="experience-edit-fields">
						<label>Heading<input maxlength="200" bind:value={section.title} /></label>
						<label
							>Introduction<textarea rows="3" maxlength="50000" bind:value={section.intro}
							></textarea></label
						>
						<p class="hint">Link directly to this section with #{section.id}</p>
						<div class="experience-edit-row">
							<label
								>Colour<select bind:value={section.tone}
									>{#each experienceTones as tone}<option value={tone}>{tone}</option
										>{/each}</select
								></label
							><label
								>Layout<select bind:value={section.layout}
									><option value="stack">One column</option><option value="split"
										>Two columns</option
									></select
								></label
							>
						</div>
						<div class="experience-edit-row">
							<button
								type="button"
								disabled={si === 0}
								onclick={() => value && move(value.sections, si, -1)}>Move section up</button
							><button
								type="button"
								disabled={si === value.sections.length - 1}
								onclick={() => value && move(value.sections, si, 1)}>Move section down</button
							><button
								type="button"
								onclick={() => {
									value?.sections.splice(si, 1);
									onchange?.();
								}}>Remove section</button
							>
						</div>
						{#each section.blocks as block, bi (block.id)}
							<details class="experience-edit-item" open>
								<summary>{block.type.charAt(0).toUpperCase() + block.type.slice(1)} block</summary>
								<div class="experience-edit-fields">
									<BlockEditor bind:block={section.blocks[bi]} {onchange} />
									<div class="experience-edit-row">
										<button
											type="button"
											disabled={bi === 0}
											onclick={() => move(section.blocks, bi, -1)}>Move block up</button
										><button
											type="button"
											disabled={bi === section.blocks.length - 1}
											onclick={() => move(section.blocks, bi, 1)}>Move block down</button
										><button
											type="button"
											onclick={() => {
												section.blocks.splice(bi, 1);
												onchange?.();
											}}>Remove block</button
										>
									</div>
								</div>
							</details>
						{/each}
						<div class="experience-edit-row" aria-label="Add a block">
							{#each blockTypes as type}<button
									type="button"
									disabled={section.blocks.length >= 20}
									onclick={() => {
										section.blocks.push(newBlock(type));
										onchange?.();
									}}>Add {type}</button
								>{/each}
						</div>
					</div>
				</details>
			{/each}
			<button
				type="button"
				disabled={value.sections.length >= 30}
				onclick={() => {
					value?.sections.push({
						id: experienceId(),
						title: '',
						intro: '',
						tone: 'paper',
						layout: 'stack',
						blocks: []
					});
					onchange?.();
				}}>Add section</button
			>
		{/if}
	{/if}
</div>

<style>
	.experience-editor :global(.experience-edit-fields) {
		display: grid;
		gap: 1rem;
		padding: 1rem 0;
	}
	.experience-editor :global(.experience-edit-row) {
		display: flex;
		align-items: end;
		flex-wrap: wrap;
		gap: 0.7rem;
	}
	.experience-editor :global(.experience-edit-row > label) {
		flex: 1;
		min-width: 150px;
	}
	.experience-editor :global(label:not(.project-experience *)) {
		display: grid;
		gap: 0.4rem;
		font-size: 0.875rem;
		font-weight: 600;
	}
	.experience-editor :global(input:not([type='checkbox']):not(.project-experience *)),
	.experience-editor :global(textarea:not(.project-experience *)),
	.experience-editor :global(select:not(.project-experience *)) {
		width: 100%;
		background: transparent;
		border: 1px solid #879298;
		border-radius: 5px;
		padding: 0.6rem;
		font: inherit;
		font-weight: 400;
		min-width: 0;
	}
	.experience-editor :global(select option) {
		color: #203532;
		background: #fff;
	}
	.experience-editor :global(button:not(.project-experience *)) {
		border: 1px solid #879298;
		border-radius: 5px;
		padding: 0.55rem 0.75rem;
		font-size: 0.85rem;
	}
	.experience-editor :global(button:disabled:not(.project-experience *)) {
		opacity: 0.45;
	}
	.experience-editor :global(.hint) {
		font-size: 0.8rem;
		font-weight: 400;
		opacity: 0.8;
	}
	.experience-editor :global(.code-input) {
		font-family: ui-monospace, monospace;
	}
	.experience-editor :global(.experience-edit-item) {
		border: 1px solid #87929866;
		border-radius: 6px;
		padding: 1rem;
	}
	.experience-editor :global(summary:not(.project-experience *)) {
		cursor: pointer;
		font-weight: 600;
	}
	.experience-toggle {
		display: flex !important;
		align-items: center;
		gap: 0.65rem !important;
	}
	.experience-preview-frame {
		width: 100%;
		overflow: auto;
		border: 1px solid #879298;
		border-radius: 6px;
	}
	.experience-preview-frame.mobile {
		max-width: 390px;
		margin-inline: auto;
	}
	.experience-preview-frame :global(.project-experience label) {
		font-size: inherit;
	}
</style>
