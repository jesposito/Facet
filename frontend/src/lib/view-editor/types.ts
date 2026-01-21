/**
 * View Editor Types
 * 
 * Centralized type definitions for the view editor component hierarchy.
 * These types are shared across the editor store, context, and extracted components.
 */

import type { 
	View, 
	ViewSection, 
	ItemConfig, 
	Profile, 
	SectionWidth, 
	ShareToken 
} from '$lib/pocketbase';
import type { AccentColor } from '$lib/colors';

// ============================================================================
// Section Configuration Types
// ============================================================================

/**
 * Section definition - describes a content section type
 */
export interface SectionDef {
	label: string;
	collection: string;
}

/**
 * Section configuration state - runtime state for each section in the editor
 */
export interface SectionConfig {
	enabled: boolean;
	items: string[];
	expanded: boolean;
	layout: string;
	width: SectionWidth;
	itemConfig: Record<string, ItemConfig>;
}

/**
 * Section item - represents an item within a section
 * The `id` field is critical for svelte-dnd-action identity
 */
export interface SectionItem {
	id: string;
	label: string;
	visibility: string;
	is_draft?: boolean;
	data: Record<string, unknown>;
}

/**
 * Section order item - used for drag-and-drop reordering
 * Must have stable `id` for svelte-dnd-action
 */
export interface SectionOrderItem {
	id: string;
	key: string;
}

// ============================================================================
// Override Editor Types
// ============================================================================

/**
 * Override editor state - tracks the current item being edited
 */
export interface OverrideEditorState {
	sectionKey: string;
	itemId: string;
	itemLabel: string;
	originalData: Record<string, unknown>;
	overrides: Record<string, string | string[]>;
}

// ============================================================================
// AI Resume Generator Types
// ============================================================================

/**
 * AI Print status - returned from /api/ai-print/status
 */
export interface AIPrintStatus {
	available: boolean;
	pandoc_installed: boolean;
	ai_configured: boolean;
}

/**
 * Resume generation configuration
 */
export interface ResumeGenerationConfig {
	format: 'pdf' | 'docx';
	target_role: string;
	style: 'chronological' | 'functional' | 'hybrid';
	length: 'one-page' | 'two-page' | 'full';
	emphasis: string[];
}

/**
 * Export record - represents a generated resume export
 */
export interface ExportRecord {
	id: string;
	format: string;
	status: string;
	generated_at: string;
	download_url?: string;
	error_message?: string;
}

// ============================================================================
// Share Token Types (re-export for convenience)
// ============================================================================

export type { ShareToken };

/**
 * Token form state
 */
export interface TokenFormState {
	name: string;
	expires: string;
	maxUses: number;
}

// ============================================================================
// Editor State Types
// ============================================================================

/**
 * Loading states for various async operations
 */
export interface EditorLoadingState {
	loading: boolean;
	saving: boolean;
	deleting: boolean;
	generating: boolean;
	generatingToken: boolean;
}

/**
 * Form field state - all the form fields for the view
 */
export interface ViewFormState {
	name: string;
	slug: string;
	description: string;
	visibility: 'public' | 'unlisted' | 'private' | 'password';
	password: string;
	heroHeadline: string;
	heroSummary: string;
	heroLocation: string;
	ctaText: string;
	ctaUrl: string;
	isActive: boolean;
	accentColor: AccentColor | null;
	heroImageUrl: string | null;
}

/**
 * Preview state
 */
export interface PreviewState {
	showPreview: boolean;
	previewMode: 'desktop' | 'mobile';
}

// ============================================================================
// Re-exports from pocketbase for convenience
// ============================================================================

export type { View, ViewSection, ItemConfig, Profile, SectionWidth };
