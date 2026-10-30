<script lang="ts">
	import { onMount } from 'svelte';
	import { parseMarkdown } from '$lib/utils';
	import { safeExperienceUrl, type ProjectExperience } from './types';
	import VideoChapters from './VideoChapters.svelte';
	import CodeWalkthrough from './CodeWalkthrough.svelte';
	import StoryActivity from './StoryActivity.svelte';
	import './experience.css';
	import './designs.css';
	let {
		experience,
		title,
		summary = '',
		backHref = '/',
		backLabel = 'Back to home',
		preview = false
	}: {
		experience: ProjectExperience;
		title: string;
		summary?: string;
		backHref?: string;
		backLabel?: string;
		preview?: boolean;
	} = $props();
	let article: HTMLElement;
	onMount(() => {
		const playing = (event: Event) => {
			article.querySelectorAll('video, audio').forEach((el) => {
				if (el !== event.target) (el as HTMLMediaElement).pause();
			});
		};
		article.addEventListener('play', playing, true);
		return () => article.removeEventListener('play', playing, true);
	});
	// The project owns the page heading. Authored prose starts below section headings.
	const prose = (text: string) => parseMarkdown(text).replace(/<(\/?)h[12](?=[\s>])/gi, '<$1h3');
</script>

<article
	bind:this={article}
	class="project-experience"
	class:experience-preview={preview}
	data-design-preset={experience.design?.preset}
	data-design-width={experience.design?.width || 'standard'}
>
	<header class="experience-hero experience-tone-{experience.hero.tone}">
		<div class="experience-width">
			{#if !preview}<a class="experience-back" href={backHref}>← {backLabel}</a>{/if}
			<div
				class="experience-hero-grid"
				class:image-left={experience.hero.imagePosition === 'left'}
				class:has-image={!!safeExperienceUrl(experience.hero.image?.src)}
			>
				<div>
					<h1>{title || 'Untitled project'}</h1>
					{#if summary}<p class="experience-summary">{summary}</p>{/if}
					{#if experience.hero.intro}<div class="experience-prose">
							{@html prose(experience.hero.intro)}
						</div>{/if}
					{#if experience.hero.action && safeExperienceUrl(experience.hero.action.href, true)}<a
							class="experience-primary"
							href={preview && experience.hero.action.href.startsWith('#')
								? '#preview-' + experience.hero.action.href.slice(1)
								: safeExperienceUrl(experience.hero.action.href, true)}
							>{experience.hero.action.label || 'Explore the project'}
							<span aria-hidden="true">↗</span></a
						>{/if}
				</div>
				{#if experience.hero.image && safeExperienceUrl(experience.hero.image.src)}<figure>
						<img
							src={safeExperienceUrl(experience.hero.image.src)}
							alt={experience.hero.image.decorative ? '' : experience.hero.image.alt}
							fetchpriority="high"
						/>
						{#if experience.hero.imageNote}<figcaption class="experience-image-note">
								{experience.hero.imageNote}
							</figcaption>{/if}
					</figure>{/if}
			</div>
		</div>
	</header>
	{#if experience.sections.length > 1}<nav class="experience-index" aria-label="In this project">
			<div class="experience-width">
				{#each experience.sections as section (section.id)}<a
						href={'#' + (preview ? 'preview-' : '') + section.id}>{section.title || 'Section'}</a
					>{/each}
			</div>
		</nav>{/if}
	{#each experience.sections as section (section.id)}
		<section
			id={(preview ? 'preview-' : '') + section.id}
			class="experience-section experience-tone-{section.tone}"
		>
			<div class="experience-width">
				<div
					class="experience-section-heading"
					class:experience-heading-split={section.headingLayout === 'split'}
				>
					<h2>{section.title || 'Explore the project'}</h2>
					{#if section.intro}<div class="experience-prose">{@html prose(section.intro)}</div>{/if}
				</div>
				<div class="experience-blocks" class:experience-split={section.layout === 'split'}>
					{#each section.blocks as block (block.id)}
						<div class="experience-block experience-block-{block.type}">
							{#if block.type === 'text'}<div class="experience-prose">
									{@html prose(block.body)}
								</div>
							{:else if block.type === 'image'}<figure>
									{#if safeExperienceUrl(block.image.src)}<img
											src={safeExperienceUrl(block.image.src)}
											alt={block.image.decorative ? '' : block.image.alt}
											loading="lazy"
										/>{/if}{#if block.caption}<figcaption>{block.caption}</figcaption>{/if}
								</figure>
							{:else if block.type === 'video'}<VideoChapters
									title={block.title}
									lessons={block.lessons}
									presentation={block.presentation}
								/>
							{:else if block.type === 'code'}<CodeWalkthrough
									title={block.title}
									steps={block.steps}
								/>
							{:else if block.type === 'disclosure'}<details>
									<summary>{block.title || 'Read more'}</summary>
									<div class="experience-prose">{@html prose(block.body)}</div>
								</details>
							{:else if block.type === 'story'}<StoryActivity {block} />{/if}
						</div>
					{/each}
				</div>
			</div>
		</section>
	{/each}
</article>
