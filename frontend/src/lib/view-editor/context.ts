/**
 * View Editor Context
 * 
 * Provides context-based state sharing for the view editor component hierarchy.
 * Uses Svelte's context API to share the editor instance across components.
 * 
 * Note: ViewEditor type will be defined when createViewEditor.svelte.ts is implemented.
 * For now, we use a generic type placeholder.
 */

import { getContext, setContext } from 'svelte';

// Placeholder type until createViewEditor.svelte.ts is implemented
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type ViewEditor = any;

const VIEW_EDITOR_KEY = Symbol('view-editor');

/**
 * Set the view editor instance in context.
 * Should be called once in the parent component that owns the editor.
 */
export function setViewEditor(editor: ViewEditor): void {
	setContext(VIEW_EDITOR_KEY, editor);
}

/**
 * Get the view editor instance from context.
 * Should be called in child components that need access to the editor.
 * 
 * @throws Error if called outside of a component that has the editor in context
 */
export function useViewEditor(): ViewEditor {
	const editor = getContext<ViewEditor | undefined>(VIEW_EDITOR_KEY);
	if (!editor) {
		throw new Error(
			'useViewEditor() must be called within a component that has a ViewEditor in context. ' +
			'Make sure setViewEditor() was called in a parent component.'
		);
	}
	return editor;
}

/**
 * Try to get the view editor instance from context.
 * Returns undefined if not in context (useful for optional integration).
 */
export function tryUseViewEditor(): ViewEditor | undefined {
	return getContext<ViewEditor | undefined>(VIEW_EDITOR_KEY);
}
