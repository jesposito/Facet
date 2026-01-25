<script lang="ts">
	import type { Custom } from '$lib/pocketbase';
	import { marked } from 'marked';
	import DOMPurify from 'dompurify';

	interface Props {
		item: Custom;
		layout?: string;
	}

	let { item, layout = 'default' }: Props = $props();

	function renderMarkdown(text: string): string {
		if (!text) return '';
		const html = marked.parse(text, { async: false }) as string;
		return DOMPurify.sanitize(html);
	}
</script>

<section id="custom-{item.id}" class="mb-16" data-layout={layout}>
	<h2 class="section-title">{item.title}</h2>

	{#if item.content}
		<div class="card p-6 prose prose-gray dark:prose-invert max-w-none animate-fade-in">
			{@html renderMarkdown(item.content)}
		</div>
	{/if}
</section>
