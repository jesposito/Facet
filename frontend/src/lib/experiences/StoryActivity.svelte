<script lang="ts">
	import { onMount } from 'svelte';
	import type { ExperienceBlock } from './types';
	let { block }: { block: Extract<ExperienceBlock, { type: 'story' }> } = $props();
	let values: Record<string, string> = $state({});
	let supported = $state(false);
	let speaking = $state(false);
	let status = $state('');
	let delivery = $state('calm');
	let usePause = $state(true);
	let timer: ReturnType<typeof setTimeout> | undefined;
	let sequence = 0;
	let story = $derived(
		block.template.replace(
			/\{([a-zA-Z][\w-]*)\}/g,
			(token, key) => values[key] ?? block.fields.find((f) => f.key === key)?.initial ?? token
		)
	);
	let visibleStory = $derived(story.replace(/\s*\[\[pause\]\]\s*/g, '\n\n'));
	function stop() {
		sequence++;
		clearTimeout(timer);
		if (speaking && supported) window.speechSynthesis.cancel();
		speaking = false;
	}
	function read() {
		stop();
		if (!supported) {
			status = 'Your browser cannot read aloud. The full story is below.';
			return;
		}
		const parts = story.split('[[pause]]');
		const run = sequence;
		speaking = true;
		status = 'Reading your story';
		function speak(index: number) {
			if (run !== sequence) return;
			if (index >= parts.length) {
				speaking = false;
				status = 'Finished reading';
				return;
			}
			const utterance = new SpeechSynthesisUtterance(parts[index]);
			utterance.rate = delivery === 'dramatic' ? 0.85 : 1;
			utterance.pitch = delivery === 'dramatic' ? 0.9 : 1;
			utterance.onend = () => {
				if (run === sequence)
					timer = setTimeout(() => speak(index + 1), usePause ? block.pauseMs : 0);
			};
			utterance.onerror = () => {
				if (run === sequence) {
					speaking = false;
					status = 'Read aloud is unavailable. You can read the story below.';
				}
			};
			window.speechSynthesis.speak(utterance);
		}
		speak(0);
	}
	function reset() {
		stop();
		values = {};
		delivery = 'calm';
		usePause = true;
		status = 'Story reset';
	}
	onMount(() => {
		supported = 'speechSynthesis' in window && 'SpeechSynthesisUtterance' in window;
		window.addEventListener('pagehide', stop);
		return () => {
			stop();
			window.removeEventListener('pagehide', stop);
		};
	});
</script>

<div class="experience-story">
	<h3>{block.title || 'Make it your own'}</h3>
	<div class="experience-story-fields">
		{#each block.fields as field (field.key)}
			<label
				>{field.label}<input
					maxlength="500"
					value={values[field.key] ?? field.initial}
					oninput={(e) => {
						stop();
						values[field.key] = e.currentTarget.value;
					}}
				/></label
			>
		{/each}
	</div>
	<div class="experience-story-options">
		<label
			>Delivery <select bind:value={delivery} onchange={stop}
				><option value="calm">Calm</option><option value="dramatic">Dramatic</option></select
			></label
		>
		{#if block.pauseMs > 0 && block.template.includes('[[pause]]')}<label
				><input type="checkbox" bind:checked={usePause} onchange={stop} /> Pause for effect</label
			>{/if}
	</div>
	<div class="experience-story-output" aria-label="Your story">{visibleStory}</div>
	<div class="experience-actions">
		<button type="button" onclick={read} disabled={speaking}>Read aloud</button>
		<button
			type="button"
			onclick={() => {
				stop();
				status = 'Reading stopped';
			}}
			disabled={!speaking}>Stop</button
		>
		<button type="button" onclick={reset}>Reset</button>
		<button type="button" onclick={() => window.print()}>Print</button>
	</div>
	<p role="status" class="experience-credit">
		{status ||
			(supported
				? 'Change a few details, then hear your story.'
				: 'You can read and edit the story here. Read aloud depends on your browser.')}
	</p>
	{#if block.credit}<p class="experience-credit">{block.credit}</p>{/if}
</div>
