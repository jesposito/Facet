<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	interface Props {
		value: string;
		mode?: 'wysiwyg' | 'markdown';
		minHeight?: string;
		placeholder?: string;
		toolbar?: 'full' | 'compact';
		onchange?: (value: string) => void;
	}

	let {
		value = $bindable(''),
		mode: initialMode = 'wysiwyg',
		minHeight = '200px',
		placeholder = 'Start writing...',
		toolbar = 'full',
		onchange
	}: Props = $props();

	let currentMode = $state<'wysiwyg' | 'markdown'>('wysiwyg');
	let editorElement: HTMLDivElement | undefined = $state();
	let editor: any = $state(null);
	let isMounted = $state(false);
	let markdownText = $state('');
	let editorRevision = $state(0);
	let isUpdatingFromEditor = false;
	let isUpdatingFromProp = false;
	let destroyed = false;
	let editorExtensions: any[] | null = null;
	let EditorClass: any = null;

	// Restore preferred mode from localStorage or use prop
	onMount(async () => {
		try {
			const stored = localStorage.getItem('editor-preferred-mode');
			currentMode = (stored === 'wysiwyg' || stored === 'markdown') ? stored : initialMode;
			markdownText = value || '';

			const { Editor } = await import('@tiptap/core');
			const { default: StarterKit } = await import('@tiptap/starter-kit');
			const { Markdown } = await import('@tiptap/markdown');
			const { default: Link } = await import('@tiptap/extension-link');
			const { default: Image } = await import('@tiptap/extension-image');
			const { default: Placeholder } = await import('@tiptap/extension-placeholder');

			// Guard against unmount during async imports (CRIT #7: memory leak)
			if (destroyed) return;

			EditorClass = Editor;
			editorExtensions = [
				// Disable Link in StarterKit to avoid duplicate extension (v3 includes it by default)
				StarterKit.configure({ link: false }),
				Markdown,
				Link.configure({ openOnClick: false, HTMLAttributes: { class: 'text-primary-600 underline' } }),
				Image,
				Placeholder.configure({ placeholder })
			];

			if (toolbar === 'full') {
				const { Table } = await import('@tiptap/extension-table');
				const { default: TableRow } = await import('@tiptap/extension-table-row');
				const { default: TableHeader } = await import('@tiptap/extension-table-header');
				const { default: TableCell } = await import('@tiptap/extension-table-cell');
				if (destroyed) return;
				editorExtensions.push(
					Table.configure({ resizable: true }),
					TableRow,
					TableHeader,
					TableCell
				);
			}

			// Yield to let Svelte flush any pending $state changes to the DOM,
			// ensuring editorElement is bound after any {#if} re-renders.
			await new Promise<void>(resolve => queueMicrotask(() => resolve()));
			if (destroyed) return;

			// Only create editor if in WYSIWYG mode and element exists (CRIT #1 fix)
			if (currentMode === 'wysiwyg' && editorElement) {
				createEditor();
			}

			isMounted = true;
		} catch (err) {
			console.error('[MarkdownEditor] Failed to initialize TipTap editor:', err);
			// Still mark as mounted so the toolbar/fallback renders
			isMounted = true;
		}
	});

	function createEditor() {
		if (!EditorClass || !editorExtensions || !editorElement || destroyed) return;
		editor?.destroy();
		// Pass empty content on init — we set markdown content after creation
		// because TipTap's `content` option expects HTML, not markdown.
		editor = new EditorClass({
			element: editorElement,
			extensions: editorExtensions,
			content: '',
			onUpdate: ({ editor: ed }: { editor: any }) => {
				if (isUpdatingFromProp) return;
				isUpdatingFromEditor = true;
				const md = ed.getMarkdown();
				value = md;
				markdownText = md;
				onchange?.(md);
				isUpdatingFromEditor = false;
				editorRevision++;
			},
			onSelectionUpdate: () => {
				editorRevision++;
			}
		});
		// Set initial content via the Markdown extension's setContent
		const initialContent = markdownText || value || '';
		if (initialContent) {
			isUpdatingFromProp = true;
			editor.commands.setContent(initialContent, false, { preserveWhitespace: 'full' });
			isUpdatingFromProp = false;
		}
	}

	onDestroy(() => {
		destroyed = true;
		editor?.destroy();
	});

	// Sync external value changes into the editor
	$effect(() => {
		if (!editor || !isMounted || isUpdatingFromEditor) return;
		const currentValue = value;
		const editorMd = editor.getMarkdown();
		// Normalize both values to avoid false positives from whitespace differences
		if (currentValue?.trim() !== editorMd?.trim()) {
			isUpdatingFromProp = true;
			editor.commands.setContent(currentValue || '', false, { preserveWhitespace: 'full' });
			markdownText = currentValue || '';
			isUpdatingFromProp = false;
		}
	});

	function toggleMode() {
		if (currentMode === 'wysiwyg') {
			markdownText = editor?.getMarkdown() ?? value;
			currentMode = 'markdown';
		} else {
			value = markdownText;
			onchange?.(markdownText);
			currentMode = 'wysiwyg';
			// The {#if} block destroys the old editorElement div, so we must
			// always recreate the editor on the new DOM node after Svelte renders it.
			queueMicrotask(() => {
				if (editorElement) {
					createEditor();
				}
			});
		}
		localStorage.setItem('editor-preferred-mode', currentMode);
	}

	function handleMarkdownInput(e: Event) {
		const target = e.target as HTMLTextAreaElement;
		markdownText = target.value;
		value = markdownText;
		onchange?.(markdownText);
	}

	function handleKeydown(e: KeyboardEvent) {
		if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 'm' || e.key === 'M')) {
			e.preventDefault();
			toggleMode();
		}
	}

	// Toolbar action helpers
	function cmd(command: string, ...args: any[]) {
		if (!editor) return;
		const chain = editor.chain().focus();
		if (args.length > 0) {
			(chain as any)[command](...args).run();
		} else {
			(chain as any)[command]().run();
		}
	}

	function isActive(_revision: number, name: string, attrs?: Record<string, any>): boolean {
		return editor?.isActive(name, attrs) ?? false;
	}

	/** Validate URL against safe protocol allowlist */
	function isSafeUrl(url: string): boolean {
		const trimmed = url.trim().toLowerCase();
		return trimmed.startsWith('https://') || trimmed.startsWith('http://') || trimmed.startsWith('mailto:') || trimmed.startsWith('/');
	}

	function insertLink() {
		const url = prompt('Enter URL:');
		if (url && isSafeUrl(url)) {
			editor?.chain().focus().setLink({ href: url.trim() }).run();
		} else if (url) {
			alert('Only http://, https://, mailto:, and relative URLs are allowed.');
		}
	}

	function insertImage() {
		const url = prompt('Enter image URL:');
		if (url && isSafeUrl(url)) {
			editor?.chain().focus().setImage({ src: url.trim() }).run();
		} else if (url) {
			alert('Only http://, https://, and relative URLs are allowed.');
		}
	}

	function insertTable() {
		editor?.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run();
	}

	let toolbarActive = $derived({
		bold: isActive(editorRevision, 'bold'),
		italic: isActive(editorRevision, 'italic'),
		strike: isActive(editorRevision, 'strike'),
		h1: isActive(editorRevision, 'heading', { level: 1 }),
		h2: isActive(editorRevision, 'heading', { level: 2 }),
		h3: isActive(editorRevision, 'heading', { level: 3 }),
		bulletList: isActive(editorRevision, 'bulletList'),
		orderedList: isActive(editorRevision, 'orderedList'),
		blockquote: isActive(editorRevision, 'blockquote'),
		code: isActive(editorRevision, 'code'),
		codeBlock: isActive(editorRevision, 'codeBlock'),
		link: isActive(editorRevision, 'link')
	});
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="markdown-editor border border-gray-300 dark:border-gray-600 rounded-lg overflow-hidden bg-white dark:bg-gray-800">
	<!-- Toolbar -->
	{#if isMounted}
		<div role="toolbar" aria-label="Text formatting" class="flex flex-wrap items-center gap-0.5 px-2 py-1.5 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
			{#if currentMode === 'wysiwyg'}
				<!-- Formatting buttons -->
				<button
					type="button"
					class="toolbar-btn {toolbarActive.bold ? 'active' : ''}"
					onclick={() => cmd('toggleBold')}
					aria-label="Bold"
					title="Bold (Ctrl+B)"
				>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/><path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/></svg>
				</button>
				<button
					type="button"
					class="toolbar-btn {toolbarActive.italic ? 'active' : ''}"
					onclick={() => cmd('toggleItalic')}
					aria-label="Italic"
					title="Italic (Ctrl+I)"
				>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="4" x2="10" y2="4"/><line x1="14" y1="20" x2="5" y2="20"/><line x1="15" y1="4" x2="9" y2="20"/></svg>
				</button>

				{#if toolbar === 'full'}
					<button
						type="button"
						class="toolbar-btn {toolbarActive.strike ? 'active' : ''}"
						onclick={() => cmd('toggleStrike')}
						aria-label="Strikethrough"
						title="Strikethrough"
					>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="12" x2="20" y2="12"/><path d="M17.5 7.5c0-2-1.5-3.5-5.5-3.5S6.5 5 6.5 7.5c0 4 11 4 11 8 0 2.5-2 3.5-5.5 3.5S6 17.5 6 16"/></svg>
					</button>
				{/if}

				<span class="toolbar-sep"></span>

				{#if toolbar === 'full'}
					<button
						type="button"
						class="toolbar-btn {toolbarActive.h1 ? 'active' : ''}"
						onclick={() => cmd('toggleHeading', { level: 1 })}
						aria-label="Heading 1"
						title="Heading 1"
					>
						<span class="text-xs font-bold">H1</span>
					</button>
				{/if}
				<button
					type="button"
					class="toolbar-btn {toolbarActive.h2 ? 'active' : ''}"
					onclick={() => cmd('toggleHeading', { level: 2 })}
					aria-label="Heading 2"
					title="Heading 2"
				>
					<span class="text-xs font-bold">H2</span>
				</button>
				<button
					type="button"
					class="toolbar-btn {toolbarActive.h3 ? 'active' : ''}"
					onclick={() => cmd('toggleHeading', { level: 3 })}
					aria-label="Heading 3"
					title="Heading 3"
				>
					<span class="text-xs font-bold">H3</span>
				</button>

				<span class="toolbar-sep"></span>

				<button
					type="button"
					class="toolbar-btn {toolbarActive.bulletList ? 'active' : ''}"
					onclick={() => cmd('toggleBulletList')}
					aria-label="Bullet list"
					title="Bullet list"
				>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="9" y1="6" x2="20" y2="6"/><line x1="9" y1="12" x2="20" y2="12"/><line x1="9" y1="18" x2="20" y2="18"/><circle cx="4" cy="6" r="1" fill="currentColor"/><circle cx="4" cy="12" r="1" fill="currentColor"/><circle cx="4" cy="18" r="1" fill="currentColor"/></svg>
				</button>

				{#if toolbar === 'full'}
					<button
						type="button"
						class="toolbar-btn {toolbarActive.orderedList ? 'active' : ''}"
						onclick={() => cmd('toggleOrderedList')}
						aria-label="Ordered list"
						title="Ordered list"
					>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="10" y1="6" x2="21" y2="6"/><line x1="10" y1="12" x2="21" y2="12"/><line x1="10" y1="18" x2="21" y2="18"/><text x="2" y="8" font-size="8" fill="currentColor" stroke="none" font-family="sans-serif">1</text><text x="2" y="14" font-size="8" fill="currentColor" stroke="none" font-family="sans-serif">2</text><text x="2" y="20" font-size="8" fill="currentColor" stroke="none" font-family="sans-serif">3</text></svg>
					</button>

					<button
						type="button"
						class="toolbar-btn {toolbarActive.blockquote ? 'active' : ''}"
						onclick={() => cmd('toggleBlockquote')}
						aria-label="Blockquote"
						title="Blockquote"
					>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor"><path d="M4.583 17.321C3.553 16.227 3 15 3 13.011c0-3.5 2.457-6.637 6.03-8.188l.893 1.378c-3.335 1.804-3.987 4.145-4.247 5.621.537-.278 1.24-.375 1.929-.311 1.804.167 3.226 1.648 3.226 3.489a3.5 3.5 0 01-3.5 3.5 3.871 3.871 0 01-2.748-1.179zm10 0C13.553 16.227 13 15 13 13.011c0-3.5 2.457-6.637 6.03-8.188l.893 1.378c-3.335 1.804-3.987 4.145-4.247 5.621.537-.278 1.24-.375 1.929-.311 1.804.167 3.226 1.648 3.226 3.489a3.5 3.5 0 01-3.5 3.5 3.871 3.871 0 01-2.748-1.179z"/></svg>
					</button>
				{/if}

				<span class="toolbar-sep"></span>

				<button
					type="button"
					class="toolbar-btn {toolbarActive.code ? 'active' : ''}"
					onclick={() => cmd('toggleCode')}
					aria-label="Inline code"
					title="Inline code"
				>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
				</button>

				{#if toolbar === 'full'}
					<button
						type="button"
						class="toolbar-btn {toolbarActive.codeBlock ? 'active' : ''}"
						onclick={() => cmd('toggleCodeBlock')}
						aria-label="Code block"
						title="Code block"
					>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><polyline points="14 15 18 12 14 9"/><polyline points="10 9 6 12 10 15"/></svg>
					</button>
				{/if}

				<button
					type="button"
					class="toolbar-btn {toolbarActive.link ? 'active' : ''}"
					onclick={insertLink}
					aria-label="Insert link"
					title="Insert link"
				>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
				</button>

				{#if toolbar === 'full'}
					<button
						type="button"
						class="toolbar-btn"
						onclick={insertImage}
						aria-label="Insert image"
						title="Insert image"
					>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/></svg>
					</button>

					<button
						type="button"
						class="toolbar-btn"
						onclick={insertTable}
						aria-label="Insert table"
						title="Insert table"
					>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/><line x1="15" y1="3" x2="15" y2="21"/></svg>
					</button>

					<button
						type="button"
						class="toolbar-btn"
						onclick={() => cmd('setHorizontalRule')}
						aria-label="Horizontal rule"
						title="Horizontal rule"
					>
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="3" y1="12" x2="21" y2="12"/></svg>
					</button>
				{/if}

				<span class="toolbar-sep"></span>

				<button
					type="button"
					class="toolbar-btn"
					onclick={() => cmd('undo')}
					aria-label="Undo"
					title="Undo (Ctrl+Z)"
				>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7v6h6"/><path d="M21 17a9 9 0 00-9-9 9 9 0 00-6 2.3L3 13"/></svg>
				</button>
				<button
					type="button"
					class="toolbar-btn"
					onclick={() => cmd('redo')}
					aria-label="Redo"
					title="Redo (Ctrl+Shift+Z)"
				>
					<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 7v6h-6"/><path d="M3 17a9 9 0 019-9 9 9 0 016 2.3L21 13"/></svg>
				</button>
			{/if}

			<!-- Spacer -->
			<div class="flex-1"></div>

			<!-- Mode toggle -->
			<button
				type="button"
				class="toolbar-btn px-2 gap-1 text-xs font-medium"
				onclick={toggleMode}
				aria-label="Toggle editor mode (Ctrl+Shift+M)"
				title="Toggle editor mode (Ctrl+Shift+M)"
			>
				{#if currentMode === 'wysiwyg'}
					<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
					<span>Markdown</span>
				{:else}
					<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 013 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
					<span>WYSIWYG</span>
				{/if}
			</button>
		</div>
	{/if}

	<!-- Editor content area -->
	<div class="relative" style="min-height: {minHeight}">
		{#if currentMode === 'wysiwyg'}
			<div
				bind:this={editorElement}
				class="editor-content prose dark:prose-invert max-w-none p-4 focus-within:outline-none"
				style="min-height: {minHeight}"
				role="textbox"
				aria-multiline="true"
				aria-label="Rich text editor"
			></div>
		{:else}
			<textarea
				class="w-full p-4 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 font-mono text-sm resize-y focus:outline-none"
				style="min-height: {minHeight}"
				placeholder={placeholder}
				value={markdownText}
				oninput={handleMarkdownInput}
			></textarea>
		{/if}
	</div>
</div>

<style>
	.toolbar-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 1.75rem;
		height: 1.75rem;
		padding: 0 0.25rem;
		border-radius: 0.25rem;
		color: theme('colors.stone.600');
		transition: all 0.15s ease;
		cursor: pointer;
		border: none;
		background: transparent;
	}

	:global(.dark) .toolbar-btn {
		color: theme('colors.stone.400');
	}

	.toolbar-btn:hover {
		background-color: theme('colors.stone.200');
		color: theme('colors.stone.900');
	}

	:global(.dark) .toolbar-btn:hover {
		background-color: theme('colors.stone.700');
		color: theme('colors.stone.100');
	}

	.toolbar-btn.active {
		background-color: theme('colors.primary.100');
		color: theme('colors.primary.700');
	}

	:global(.dark) .toolbar-btn.active {
		background-color: theme('colors.primary.900');
		color: theme('colors.primary.300');
	}

	.toolbar-sep {
		display: inline-block;
		width: 1px;
		height: 1.25rem;
		margin: 0 0.25rem;
		background-color: theme('colors.stone.300');
	}

	:global(.dark) .toolbar-sep {
		background-color: theme('colors.stone.600');
	}

	/* TipTap editor styling */
	.editor-content :global(.tiptap) {
		outline: none;
		min-height: inherit;
	}

	.editor-content :global(.tiptap p.is-editor-empty:first-child::before) {
		content: attr(data-placeholder);
		float: left;
		color: theme('colors.stone.400');
		pointer-events: none;
		height: 0;
	}

	:global(.dark) .editor-content :global(.tiptap p.is-editor-empty:first-child::before) {
		color: theme('colors.stone.500');
	}

	/* Table styling */
	.editor-content :global(.tiptap table) {
		border-collapse: collapse;
		width: 100%;
		margin: 1rem 0;
	}

	.editor-content :global(.tiptap th),
	.editor-content :global(.tiptap td) {
		border: 1px solid theme('colors.stone.300');
		padding: 0.5rem;
		text-align: left;
	}

	:global(.dark) .editor-content :global(.tiptap th),
	:global(.dark) .editor-content :global(.tiptap td) {
		border-color: theme('colors.stone.600');
	}

	.editor-content :global(.tiptap th) {
		background-color: theme('colors.stone.100');
		font-weight: 600;
	}

	:global(.dark) .editor-content :global(.tiptap th) {
		background-color: theme('colors.stone.700');
	}
</style>
