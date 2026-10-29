<script lang="ts">
	import SingleMediaPicker from '$lib/components/admin/SingleMediaPicker.svelte';
	import { newLesson, newStep } from './templates';
	import type { ExperienceBlock } from './types';
	let { block = $bindable(), onchange }: { block: ExperienceBlock; onchange?: () => void } =
		$props();
	function move<T>(items: T[], index: number, delta: number) {
		const next = index + delta;
		if (next < 0 || next >= items.length) return;
		[items[index], items[next]] = [items[next], items[index]];
		onchange?.();
	}
</script>

{#if block.type === 'text' || block.type === 'disclosure'}
	{#if block.type === 'disclosure'}<label
			>Heading<input maxlength="200" bind:value={block.title} /></label
		>{/if}
	<label
		>Text <span class="hint">Markdown is supported.</span><textarea
			rows="7"
			maxlength="50000"
			bind:value={block.body}></textarea></label
	>
{:else if block.type === 'image'}
	<SingleMediaPicker label="Image" bind:value={block.image.src} {onchange} />
	<label>Image description<input maxlength="500" bind:value={block.image.alt} /></label>
	<label>Caption<input maxlength="500" bind:value={block.caption} /></label>
{:else if block.type === 'video'}
	<label>Player heading<input maxlength="200" bind:value={block.title} /></label>
	{#each block.lessons as lesson, index (lesson.id)}
		<details class="experience-edit-item" open={index === 0}>
			<summary>Chapter {index + 1}: {lesson.title || 'Untitled'}</summary>
			<div class="experience-edit-fields">
				<label>Title<input maxlength="200" bind:value={lesson.title} /></label>
				<SingleMediaPicker
					label="Video or audio"
					bind:value={lesson.src}
					imagesOnly={false}
					accept="video/*,audio/*"
					{onchange}
				/>
				<SingleMediaPicker label="Poster image" bind:value={lesson.poster} {onchange} />
				<label
					>Description<textarea rows="3" maxlength="500" bind:value={lesson.description}
					></textarea></label
				>
				<label>Credit<input maxlength="500" bind:value={lesson.credit} /></label>
				{#if lesson.captions}
					<SingleMediaPicker
						label="Captions (WebVTT)"
						bind:value={lesson.captions.src}
						imagesOnly={false}
						accept=".vtt,text/vtt"
						{onchange}
					/>
					<div class="experience-edit-row">
						<label
							>Language code<input
								maxlength="500"
								bind:value={lesson.captions.language}
								placeholder="en"
							/></label
						><label>Caption label<input maxlength="200" bind:value={lesson.captions.label} /></label
						>
					</div>
					<button
						type="button"
						onclick={() => {
							delete lesson.captions;
							onchange?.();
						}}>Remove captions</button
					>
				{:else}<button
						type="button"
						onclick={() => {
							lesson.captions = { src: '', language: 'en', label: 'English' };
							onchange?.();
						}}>Add captions</button
					>{/if}
				<label
					>Transcript<textarea rows="5" maxlength="50000" bind:value={lesson.transcript}
					></textarea></label
				>
				<div class="experience-edit-row">
					<button
						type="button"
						disabled={index === 0}
						onclick={() => block.type === 'video' && move(block.lessons, index, -1)}
						>Move chapter up</button
					>
					<button
						type="button"
						disabled={index === block.lessons.length - 1}
						onclick={() => block.type === 'video' && move(block.lessons, index, 1)}
						>Move chapter down</button
					>
					<button
						type="button"
						onclick={() => {
							if (block.type === 'video') block.lessons.splice(index, 1);
							onchange?.();
						}}>Remove chapter</button
					>
				</div>
			</div>
		</details>
	{/each}
	<button
		type="button"
		disabled={block.lessons.length >= 30}
		onclick={() => {
			if (block.type === 'video') block.lessons.push(newLesson());
			onchange?.();
		}}>Add chapter</button
	>
{:else if block.type === 'code'}
	<label>Walkthrough heading<input maxlength="200" bind:value={block.title} /></label>
	{#each block.steps as step, index (step.id)}
		<details class="experience-edit-item" open={index === 0}>
			<summary>Step {index + 1}: {step.title || 'Untitled'}</summary>
			<div class="experience-edit-fields">
				<label>Title<input maxlength="200" bind:value={step.title} /></label>
				<label
					>Explanation<textarea rows="4" maxlength="50000" bind:value={step.body}></textarea></label
				>
				<label
					>Code<textarea
						class="code-input"
						rows="8"
						spellcheck="false"
						maxlength="50000"
						bind:value={step.code}></textarea></label
				>
				<label
					>Language<input
						maxlength="500"
						bind:value={step.language}
						placeholder="JavaScript"
					/></label
				>
				{#if step.image}<SingleMediaPicker
						label="Screenshot"
						bind:value={step.image.src}
						{onchange}
					/><label>Image description<input maxlength="500" bind:value={step.image.alt} /></label
					><button
						type="button"
						onclick={() => {
							delete step.image;
							onchange?.();
						}}>Remove screenshot</button
					>
				{:else}<button
						type="button"
						onclick={() => {
							step.image = { src: '', alt: '' };
							onchange?.();
						}}>Add screenshot</button
					>{/if}
				<div class="experience-edit-row">
					<button
						type="button"
						disabled={index === 0}
						onclick={() => block.type === 'code' && move(block.steps, index, -1)}
						>Move step up</button
					>
					<button
						type="button"
						disabled={index === block.steps.length - 1}
						onclick={() => block.type === 'code' && move(block.steps, index, 1)}
						>Move step down</button
					>
					<button
						type="button"
						onclick={() => {
							if (block.type === 'code') block.steps.splice(index, 1);
							onchange?.();
						}}>Remove step</button
					>
				</div>
			</div>
		</details>
	{/each}
	<button
		type="button"
		disabled={block.steps.length >= 30}
		onclick={() => {
			if (block.type === 'code') block.steps.push(newStep());
			onchange?.();
		}}>Add step</button
	>
{:else if block.type === 'story'}
	<label>Heading<input maxlength="200" bind:value={block.title} /></label>
	<label
		>Story template<textarea rows="8" maxlength="50000" bind:value={block.template}
		></textarea></label
	>
	<p class="hint">
		Use a field key in braces, such as &#123;character&#125;. Add [[pause]] where the reader should
		pause.
	</p>
	{#each block.fields as field, index}
		<div class="experience-edit-item experience-edit-fields">
			<div class="experience-edit-row">
				<label
					>Key<input
						pattern={'[a-zA-Z][a-zA-Z0-9_-]{0,31}'}
						maxlength="32"
						bind:value={field.key}
					/></label
				><label>Label<input maxlength="200" bind:value={field.label} /></label>
			</div>
			<label>Starting value<input maxlength="500" bind:value={field.initial} /></label>
			<button
				type="button"
				onclick={() => {
					if (block.type === 'story') block.fields.splice(index, 1);
					onchange?.();
				}}>Remove field</button
			>
		</div>
	{/each}
	<button
		type="button"
		disabled={block.fields.length >= 12}
		onclick={() => {
			if (block.type === 'story')
				block.fields.push({ key: 'field' + (block.fields.length + 1), label: '', initial: '' });
			onchange?.();
		}}>Add field</button
	>
	<label
		>Pause length (milliseconds)<input
			type="number"
			min="0"
			max="5000"
			step="100"
			bind:value={block.pauseMs}
		/></label
	>
	<label>Credit<input maxlength="500" bind:value={block.credit} /></label>
{/if}
