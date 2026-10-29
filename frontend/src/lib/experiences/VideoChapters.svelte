<script lang="ts">
	import type { ExperienceLesson } from './types';
	import { safeExperienceUrl } from './types';
	let { title, lessons }: { title: string; lessons: ExperienceLesson[] } = $props();
	let selected = $state('');
	let active = $derived(lessons.find((l) => l.id === selected) || lessons[0]);
	let player: HTMLMediaElement | undefined = $state();
	let failed = $state(false);
	function choose(id: string) {
		player?.pause();
		selected = id;
		failed = false;
	}
	$effect(() => {
		const current = player;
		return () => current?.pause();
	});
</script>

<div class="experience-video">
	{#if title}<h3>{title}</h3>{/if}
	{#if active}
		<div class="experience-player">
			{#key active.id + active.src}
				{#if safeExperienceUrl(active.src)}
					{#if /\.(mp3|m4a|wav|ogg|opus)(?:[?#]|$)/i.test(active.src)}
						<audio
							bind:this={player}
							controls
							preload="metadata"
							src={safeExperienceUrl(active.src)}
							onerror={() => (failed = true)}
							aria-label={active.title}
						></audio>
					{:else}
						<video
							bind:this={player}
							controls
							playsinline
							preload="metadata"
							src={safeExperienceUrl(active.src)}
							poster={safeExperienceUrl(active.poster) || undefined}
							onerror={() => (failed = true)}
							aria-label={active.title}
						>
							<track
								kind="captions"
								src={safeExperienceUrl(active.captions?.src) || undefined}
								srclang={active.captions?.language || 'en'}
								label={active.captions?.label || 'Captions'}
								default
							/>
						</video>
					{/if}
				{:else}<p class="experience-empty">A video will appear here once it has been added.</p>{/if}
			{/key}
		</div>
		{#if failed}<p role="status">
				This recording couldn't load.{#if active.transcript}
					You can read the transcript below.{/if}
			</p>{/if}
		{#if lessons.length > 1}
			<div class="experience-chapters" aria-label={title || 'Video chapters'}>
				{#each lessons as lesson, i (lesson.id)}<button
						type="button"
						class:chosen={active.id === lesson.id}
						aria-pressed={active.id === lesson.id}
						onclick={() => choose(lesson.id)}
						><span>{String(i + 1).padStart(2, '0')}</span>{lesson.title ||
							'Untitled chapter'}</button
					>{/each}
			</div>
		{/if}
		<h4>{active.title}</h4>
		{#if active.description}<p>{active.description}</p>{/if}
		{#if active.credit}<p class="experience-credit">{active.credit}</p>{/if}
		{#if active.transcript}<details>
				<summary>Read the transcript</summary>
				<div class="experience-transcript">{active.transcript}</div>
			</details>{/if}
	{:else}<p class="experience-empty">No chapters have been added yet.</p>{/if}
</div>
