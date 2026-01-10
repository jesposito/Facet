<script lang="ts">
	import type { Project } from '$lib/pocketbase';
	import { truncate, parseMarkdown } from '$lib/utils';

	interface Props {
		items: Project[];
		layout?: string;
		viewSlug?: string;
	}

	let { items, layout = 'grid-3', viewSlug = '' }: Props = $props();

const isLinkable = (project: Project) => {
	const visibility = (project as unknown as Record<string, string>).visibility;
	return visibility === 'public' && project.slug;
};

const projectHref = (project: Project) => {
	if (!isLinkable(project)) return null;
	return viewSlug ? `/projects/${project.slug}?from=${encodeURIComponent(viewSlug)}` : `/projects/${project.slug}`;
};

	function getLinkIcon(type: string) {
		switch (type.toLowerCase()) {
			case 'github':
				return `<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/></svg>`;
			case 'website':
			case 'demo':
				return `<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" /></svg>`;
			default:
				return `<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" /></svg>`;
		}
	}

	// Split items for featured layout
	let featuredItem = $derived(items[0]);
	let remainingItems = $derived(items.slice(1));

	const thumbSrc = (project: Project) =>
		(project as unknown as Record<string, string>).cover_image_thumb_url ||
		(project as unknown as Record<string, string>).cover_image_url ||
		'';

	const largeSrc = (project: Project) =>
		(project as unknown as Record<string, string>).cover_image_large_url ||
		(project as unknown as Record<string, string>).cover_image_url ||
		'';
</script>

<section id="projects" class="mb-16">
	<h2 class="section-title">Projects</h2>

	{#if layout === 'featured' && items.length > 0}
		<!-- Featured Layout: First item large, rest in grid -->
		<div class="space-y-8">
			<!-- Featured Project -->
			<article class="card overflow-hidden group animate-fade-in">
				<div class="grid md:grid-cols-2 gap-0">
					{#if thumbSrc(featuredItem)}
						<div class="aspect-video md:aspect-auto md:h-full overflow-hidden bg-gray-100 dark:bg-gray-700">
							<img
								src={largeSrc(featuredItem)}
								srcset={`${thumbSrc(featuredItem)} 640w, ${largeSrc(featuredItem)} 1280w`}
								sizes="(max-width: 768px) 100vw, 50vw"
								alt={featuredItem.title}
								class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
							/>
						</div>
					{:else}
						<div class="aspect-video md:aspect-auto md:h-full bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
							<span class="text-6xl font-bold text-white/50">
								{featuredItem.title.charAt(0)}
							</span>
						</div>
					{/if}

					<div class="p-6 flex flex-col justify-center">
						<div class="flex items-start justify-between gap-2">
								<h3 class="text-2xl font-semibold text-gray-900 dark:text-white">
								{#if isLinkable(featuredItem)}
									<a href={projectHref(featuredItem)} class="hover:text-primary-600 dark:hover:text-primary-400">
										{featuredItem.title}
									</a>
								{:else}
									{featuredItem.title}
								{/if}
							</h3>
							<span class="shrink-0 px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded">
								Featured
							</span>
						</div>

						{#if featuredItem.summary}
							<p class="mt-3 text-gray-600 dark:text-gray-400">
								{featuredItem.summary}
							</p>
						{/if}

						{#if featuredItem.tech_stack && featuredItem.tech_stack.length > 0}
							<div class="mt-4 flex flex-wrap gap-2">
								{#each featuredItem.tech_stack as tech}
									<span class="px-2.5 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
										{tech}
									</span>
								{/each}
							</div>
						{/if}

						{#if featuredItem.links && featuredItem.links.length > 0}
							<div class="mt-4 flex items-center gap-4">
								{#each featuredItem.links as link}
									<a
										href={link.url}
										target="_blank"
										rel="noopener noreferrer"
										class="flex items-center gap-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-primary-600 dark:hover:text-primary-400"
									>
										{@html getLinkIcon(link.type)}
										<span class="capitalize">{link.type}</span>
									</a>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			</article>

			<!-- Remaining Projects in Grid -->
			{#if remainingItems.length > 0}
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
					{#each remainingItems as project (project.id)}
						<article class="card overflow-hidden group animate-fade-in">
							{#if thumbSrc(project)}
								<div class="aspect-video overflow-hidden bg-gray-100 dark:bg-gray-700">
									<img
										src={largeSrc(project)}
										srcset={`${thumbSrc(project)} 640w, ${largeSrc(project)} 1280w`}
										sizes="(max-width: 768px) 100vw, (max-width: 1024px) 50vw, 33vw"
										alt={project.title}
										class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
									/>
								</div>
							{:else}
								<div class="aspect-video bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
									<span class="text-4xl font-bold text-white/50">
										{project.title.charAt(0)}
									</span>
								</div>
							{/if}

							<div class="p-5">
							<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
								{#if isLinkable(project)}
									<a href={projectHref(project)} class="hover:text-primary-600 dark:hover:text-primary-400">
										{project.title}
									</a>
								{:else}
									{project.title}
								{/if}
							</h3>

							{#if project.summary}
								<p class="mt-2 text-gray-600 dark:text-gray-400 text-sm">
									{truncate(project.summary, 100)}
								</p>
							{/if}

							{#if project.tech_stack && project.tech_stack.length > 0}
								<div class="mt-3 flex flex-wrap gap-1.5">
									{#each project.tech_stack.slice(0, 3) as tech}
										<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
											{tech}
										</span>
									{/each}
									{#if project.tech_stack.length > 3}
										<span class="px-2 py-0.5 text-xs text-gray-500">
											+{project.tech_stack.length - 3}
										</span>
									{/if}
								</div>
							{/if}
						</div>
					</article>
				{/each}
			</div>
		{/if}
	</div>

{:else if layout === 'list'}
		<!-- List Layout -->
		<div class="space-y-4">
			{#each items as project (project.id)}
				<article class="card overflow-hidden group animate-fade-in">
					<div class="flex flex-col sm:flex-row">
						{#if thumbSrc(project)}
							<div class="sm:w-48 sm:shrink-0 aspect-video sm:aspect-square overflow-hidden bg-gray-100 dark:bg-gray-700">
								<img
									src={largeSrc(project)}
									srcset={`${thumbSrc(project)} 480w, ${largeSrc(project)} 960w`}
									sizes="(max-width: 640px) 100vw, 200px"
									alt={project.title}
									class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
								/>
							</div>
						{/if}

						<div class="p-5 flex-1">
							<div class="flex items-start justify-between gap-2">
								<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
									{#if isLinkable(project)}
										<a href={projectHref(project)} class="hover:text-primary-600 dark:hover:text-primary-400">
											{project.title}
										</a>
									{:else}
										{project.title}
									{/if}
								</h3>
								{#if project.is_featured}
									<span class="shrink-0 px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded">
										Featured
									</span>
								{/if}
							</div>

							{#if project.summary}
								<p class="mt-2 text-gray-600 dark:text-gray-400 text-sm">
									{project.summary}
								</p>
							{/if}

							<div class="mt-3 flex flex-wrap items-center gap-4">
								{#if project.tech_stack && project.tech_stack.length > 0}
									<div class="flex flex-wrap gap-1.5">
										{#each project.tech_stack.slice(0, 5) as tech}
											<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
												{tech}
											</span>
										{/each}
										{#if project.tech_stack.length > 5}
											<span class="px-2 py-0.5 text-xs text-gray-500">
												+{project.tech_stack.length - 5}
											</span>
										{/if}
									</div>
								{/if}

								{#if project.links && project.links.length > 0}
									<div class="flex items-center gap-3 ml-auto">
										{#each project.links as link}
											<a
												href={link.url}
												target="_blank"
												rel="noopener noreferrer"
												class="flex items-center gap-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-primary-600 dark:hover:text-primary-400"
											>
												{@html getLinkIcon(link.type)}
												<span class="capitalize">{link.type}</span>
											</a>
										{/each}
									</div>
								{/if}
							</div>
						</div>
					</div>
				</article>
			{/each}
		</div>

	{:else if layout === 'grid-2'}
		<!-- 2-Column Grid Layout -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			{#each items as project (project.id)}
				<article class="card overflow-hidden group animate-fade-in">
					{#if thumbSrc(project)}
						<div class="aspect-video overflow-hidden bg-gray-100 dark:bg-gray-700">
							<img
								src={largeSrc(project)}
								srcset={`${thumbSrc(project)} 640w, ${largeSrc(project)} 1280w`}
								sizes="(max-width: 768px) 100vw, 50vw"
								alt={project.title}
								class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
							/>
						</div>
					{:else}
						<div class="aspect-video bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
							<span class="text-5xl font-bold text-white/50">
								{project.title.charAt(0)}
							</span>
						</div>
					{/if}

					<div class="p-6">
						<div class="flex items-start justify-between gap-2">
							<h3 class="text-xl font-semibold text-gray-900 dark:text-white">
								{#if isLinkable(project)}
									<a href={projectHref(project)} class="hover:text-primary-600 dark:hover:text-primary-400">
										{project.title}
									</a>
								{:else}
									{project.title}
								{/if}
							</h3>
							{#if project.is_featured}
								<span class="shrink-0 px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded">
									Featured
								</span>
							{/if}
						</div>

						{#if project.summary}
							<p class="mt-2 text-gray-600 dark:text-gray-400">
								{truncate(project.summary, 150)}
							</p>
						{/if}

						{#if project.tech_stack && project.tech_stack.length > 0}
							<div class="mt-4 flex flex-wrap gap-2">
								{#each project.tech_stack.slice(0, 5) as tech}
									<span class="px-2.5 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
										{tech}
									</span>
								{/each}
								{#if project.tech_stack.length > 5}
									<span class="px-2.5 py-1 text-sm text-gray-500">
										+{project.tech_stack.length - 5}
									</span>
								{/if}
							</div>
						{/if}

						{#if project.links && project.links.length > 0}
							<div class="mt-4 flex items-center gap-4">
								{#each project.links as link}
									<a
										href={link.url}
										target="_blank"
										rel="noopener noreferrer"
										class="flex items-center gap-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-primary-600 dark:hover:text-primary-400"
									>
										{@html getLinkIcon(link.type)}
										<span class="capitalize">{link.type}</span>
									</a>
								{/each}
							</div>
						{/if}
					</div>
				</article>
			{/each}
		</div>

	{:else}
		<!-- Default: 3-Column Grid Layout -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each items as project (project.id)}
				<article class="card overflow-hidden group animate-fade-in">
					{#if thumbSrc(project)}
						<div class="aspect-video overflow-hidden bg-gray-100 dark:bg-gray-700">
							<img
								src={largeSrc(project)}
								srcset={`${thumbSrc(project)} 640w, ${largeSrc(project)} 1280w`}
								sizes="(max-width: 768px) 100vw, (max-width: 1024px) 50vw, 33vw"
								alt={project.title}
								class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
							/>
						</div>
					{:else}
						<div class="aspect-video bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
							<span class="text-4xl font-bold text-white/50">
								{project.title.charAt(0)}
							</span>
						</div>
					{/if}

					<div class="p-5">
						<div class="flex items-start justify-between gap-2">
							<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
								{#if isLinkable(project)}
									<a href={projectHref(project)} class="hover:text-primary-600 dark:hover:text-primary-400">
										{project.title}
									</a>
								{:else}
									{project.title}
								{/if}
							</h3>
							{#if project.is_featured}
								<span class="shrink-0 px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded">
									Featured
								</span>
							{/if}
						</div>

						{#if project.summary}
							<p class="mt-2 text-gray-600 dark:text-gray-400 text-sm">
								{truncate(project.summary, 120)}
							</p>
						{/if}

						{#if project.tech_stack && project.tech_stack.length > 0}
							<div class="mt-3 flex flex-wrap gap-1.5">
								{#each project.tech_stack.slice(0, 4) as tech}
									<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
										{tech}
									</span>
								{/each}
								{#if project.tech_stack.length > 4}
									<span class="px-2 py-0.5 text-xs text-gray-500">
										+{project.tech_stack.length - 4}
									</span>
								{/if}
							</div>
						{/if}

						{#if project.links && project.links.length > 0}
							<div class="mt-4 flex items-center gap-3">
								{#each project.links as link}
									<a
										href={link.url}
										target="_blank"
										rel="noopener noreferrer"
										class="flex items-center gap-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-primary-600 dark:hover:text-primary-400"
									>
										{@html getLinkIcon(link.type)}
										<span class="capitalize">{link.type}</span>
									</a>
								{/each}
							</div>
						{/if}
					</div>
				</article>
			{/each}
		</div>
	{/if}
</section>
