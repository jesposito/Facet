// AI resume generation controller — rune-friendly state container that
// callers (the homepage and the facet view route) share so we don't
// hand-copy the same ~150 LOC of modal state + fetch logic into both.
//
// Two pages used to declare `aiPrintStatus`, `generating`, `generationConfig`,
// `generatedUrl`, and the matching `checkAIPrintStatus` + `generateResume`
// functions inline. Diverging fixes in one but not the other was a constant
// review hazard. This module is the single source of truth for that
// behavior; the pages just hold a controller handle, render a button, and
// mount the AIResumeModal.

import { pb } from '$lib/pocketbase';

export type AIPrintFormat = 'pdf' | 'docx';
export type AIPrintStyle = 'chronological' | 'functional' | 'hybrid';
export type AIPrintLength = 'one-page' | 'two-page' | 'full';

export interface AIPrintStatus {
	available: boolean;
	ai_configured: boolean;
	pandoc_installed: boolean;
}

export interface AIPrintConfig {
	format: AIPrintFormat;
	target_role: string;
	style: AIPrintStyle;
	length: AIPrintLength;
}

/**
 * Controller handle. Inputs are read via getters so callers can wire
 * reactive page data without re-creating the controller when route props
 * change. The controller owns `open`, `generating`, `status`, `config`, and
 * `downloadUrl` as `$state`; expose only the getters needed by templates.
 */
export interface AIResumeController {
	/** Whether the generate-resume modal is open. Read by the modal. */
	readonly open: boolean;
	/** True while the POST is in flight. */
	readonly generating: boolean;
	/** Status of the AI-print backend (whether the Generate button should appear). */
	readonly status: AIPrintStatus;
	/** Current form config — also bound directly via `bind:value` from the modal. */
	readonly config: AIPrintConfig;
	/** Set when generation completes; used to swap modal body to download CTA. */
	readonly downloadUrl: string | null;

	/**
	 * Open the modal. Pass the triggering element so focus can be restored to
	 * it on close — relying on `document.activeElement` at open-time races
	 * any synchronous menu-close that fires alongside the call, dropping focus
	 * to <body>. See WCAG 2.4.3.
	 */
	show(trigger?: HTMLElement | null): void;
	/** Element that should receive focus when the modal closes. */
	readonly trigger: HTMLElement | null;
	/** Close the modal and reset the download URL. */
	dismiss(): void;
	/** Fetch /api/ai-print/status and populate `status`. Safe to call once on mount. */
	checkStatus(): Promise<void>;
	/** POST to /api/view/{slug}/generate, hand back the URL, auto-trigger the browser download. */
	generate(): Promise<void>;
}

export function createAIResumeController(opts: {
	/**
	 * Returns the current view slug for the resume-generation endpoint, or
	 * undefined when we're on a page that doesn't have a view (e.g. the
	 * homepage with no default view). The Generate button hides when undefined.
	 */
	getSlug: () => string | undefined;
	/**
	 * Returns the target-role string sent with the generate request — typically
	 * `view.hero_headline || profile.headline`. Empty string is acceptable.
	 */
	getTargetRole: () => string;
	/**
	 * Optional alert hook for the user-visible error path. Defaults to
	 * `globalThis.alert`. Override in tests / SSR.
	 */
	alertError?: (message: string) => void;
}): AIResumeController {
	let open = $state(false);
	let generating = $state(false);
	let status = $state<AIPrintStatus>({
		available: false,
		ai_configured: false,
		pandoc_installed: false
	});
	let config = $state<AIPrintConfig>({
		format: 'pdf',
		target_role: '',
		style: 'chronological',
		length: 'two-page'
	});
	let downloadUrl = $state<string | null>(null);
	let trigger = $state<HTMLElement | null>(null);

	const reportError =
		opts.alertError ??
		((m: string) => {
			if (typeof window !== 'undefined') window.alert(m);
		});

	async function checkStatus(): Promise<void> {
		try {
			const response = await fetch('/api/ai-print/status', {
				headers: { Authorization: pb.authStore.token || '' }
			});
			if (!response.ok) return;
			const result = await response.json();
			status = {
				available: !!result.available,
				ai_configured: !!result.ai_configured,
				pandoc_installed: !!result.pandoc_installed
			};
		} catch (err) {
			// Status checks are best-effort — failure means the Generate button
			// stays hidden, which is the same as "AI not configured."
			console.error('[AI-PRINT] status check failed:', err);
		}
	}

	async function generate(): Promise<void> {
		const slug = opts.getSlug();
		if (!slug) return;

		generating = true;
		downloadUrl = null;

		try {
			const body: AIPrintConfig = {
				...config,
				target_role: opts.getTargetRole()
			};
			const response = await fetch(`/api/view/${slug}/generate`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token || ''
				},
				body: JSON.stringify(body)
			});
			const result = await response.json();
			if (!response.ok) {
				throw new Error(result?.error || 'Generation failed');
			}

			downloadUrl = result.download_url;

			// Auto-download by synthesizing a link click. Browsers ignore Save-As
			// prompts triggered without a user gesture, but this is fired inside
			// the click that initiated `generate`, so the gesture chain survives.
			if (downloadUrl) {
				const a = document.createElement('a');
				a.href = downloadUrl;
				a.download = `resume.${config.format}`;
				document.body.appendChild(a);
				a.click();
				document.body.removeChild(a);
			}
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to generate resume';
			reportError(message);
		} finally {
			generating = false;
		}
	}

	return {
		get open() {
			return open;
		},
		get generating() {
			return generating;
		},
		get status() {
			return status;
		},
		get config() {
			return config;
		},
		get downloadUrl() {
			return downloadUrl;
		},
		get trigger() {
			return trigger;
		},
		show(triggerEl: HTMLElement | null = null) {
			// Capture the trigger BEFORE the modal opens. Doing this synchronously
			// here (rather than reading document.activeElement when the modal
			// mounts) avoids losing focus when the trigger is unmounted in the
			// same tick — e.g. a menuitem inside a popover whose close runs
			// alongside this call.
			trigger = triggerEl;
			open = true;
		},
		dismiss() {
			open = false;
			downloadUrl = null;
		},
		checkStatus,
		generate
	};
}
