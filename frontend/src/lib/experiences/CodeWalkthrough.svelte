<script lang="ts">
	import type { ExperienceStep } from './types';
	import { safeExperienceUrl } from './types';
	import { parseMarkdown } from '$lib/utils';
	let { title, steps }: { title: string; steps: ExperienceStep[] } = $props();
	let selected = $state('');
	let active = $derived(steps.find((s) => s.id === selected) || steps[0]);
	let copyStatus = $state('');
	async function copy() {
		try {
			await navigator.clipboard.writeText(active?.code || '');
			copyStatus = 'Copied';
		} catch {
			copyStatus = 'Select the code below to copy it.';
		}
	}
</script>

<div class="experience-walkthrough">
	{#if title}<h3>{title}</h3>{/if}
	{#if active}
		<div class="experience-chapters" aria-label={title || 'Walkthrough steps'}>
			{#each steps as step, i (step.id)}<button
					type="button"
					class:chosen={active.id === step.id}
					aria-pressed={active.id === step.id}
					onclick={() => {
						selected = step.id;
						copyStatus = '';
					}}><span>{i + 1}</span>{step.title || 'Untitled step'}</button
				>{/each}
		</div>
		<h4>{active.title}</h4>
		<div class="experience-prose">
			{@html parseMarkdown(active.body).replace(/<(\/?)h[12](?=[\s>])/gi, '<$1h3')}
		</div>
		{#if active.code}
			<div class="experience-code-label">
				<span>{active.language || 'Code'}</span><button type="button" onclick={copy}
					>Copy code</button
				><span role="status">{copyStatus}</span>
			</div>
			<!-- svelte-ignore a11y_no_noninteractive_tabindex (Scrollable code must be reachable with a keyboard.) -->
			<pre role="region" tabindex="0" aria-label={active.title + ' code'}><code>{active.code}</code
				></pre>
		{/if}
		{#if active.image && safeExperienceUrl(active.image.src)}<img
				src={safeExperienceUrl(active.image.src)}
				alt={active.image.decorative ? '' : active.image.alt}
				loading="lazy"
			/>{/if}
	{:else}<p class="experience-empty">No steps have been added yet.</p>{/if}
</div>
