import { writable, derived } from 'svelte/store';

export interface AutosaveData {
	[formKey: string]: {
		data: Record<string, any>;
		savedAt: number;
		isEditing?: boolean; // true if editing existing item, false if new
		editingId?: string; // ID of item being edited
	};
}

export interface AutosaveStatus {
	formKey: string;
	status: 'idle' | 'saving' | 'saved' | 'error';
	lastSaved: number | null;
	hasChanges: boolean;
}

const autosaveData = writable<AutosaveData>({});
const autosaveStatus = writable<Record<string, AutosaveStatus>>({});

// Debounce utility for localStorage saves
function debounce<T extends (...args: any[]) => any>(func: T, delay: number): T {
	let timeoutId: ReturnType<typeof setTimeout>;
	return ((...args: any[]) => {
		clearTimeout(timeoutId);
		timeoutId = setTimeout(() => func(...args), delay);
	}) as T;
}

/**
 * Creates an autosave manager for a specific form
 */
export function createAutosave(formKey: string, options: {
	saveDelay?: number; // ms to wait before saving after last change (default: 2000)
	excludeFields?: string[]; // fields to exclude from autosave
	maxAge?: number; // max age of saved data in ms (default: 7 days)
} = {}) {
	const {
		saveDelay = 2000,
		excludeFields = [],
		maxAge = 7 * 24 * 60 * 60 * 1000 // 7 days
	} = options;

	autosaveStatus.update(statuses => ({
		...statuses,
		[formKey]: {
			formKey,
			status: 'idle',
			lastSaved: null,
			hasChanges: false
		}
	}));
	const loadSavedData = (): any => {
		if (typeof window === 'undefined') return null;
		
		try {
			const saved = localStorage.getItem(`autosave-${formKey}`);
			if (!saved) return null;

			const parsed = JSON.parse(saved);
			
			// Check if data is too old
			if (parsed.savedAt && Date.now() - parsed.savedAt > maxAge) {
				localStorage.removeItem(`autosave-${formKey}`);
				return null;
			}
			
			return parsed;
		} catch (error) {
			console.warn('Failed to load autosave data:', error);
			localStorage.removeItem(`autosave-${formKey}`);
			return null;
		}
	};

	// Save data to localStorage
	const saveToStorage = (data: Record<string, any>, isEditing = false, editingId?: string) => {
		if (typeof window === 'undefined') return;

		const filteredData = Object.fromEntries(
			Object.entries(data).filter(([key]) => !excludeFields.includes(key))
		);

		const autosaveEntry = {
			data: filteredData,
			savedAt: Date.now(),
			isEditing,
			editingId
		};

		try {
			localStorage.setItem(`autosave-${formKey}`, JSON.stringify(autosaveEntry));
			
			autosaveStatus.update(statuses => ({
				...statuses,
				[formKey]: {
					...statuses[formKey],
					status: 'saved',
					lastSaved: Date.now(),
					hasChanges: false
				}
			}));
			setTimeout(() => {
				autosaveStatus.update(statuses => ({
					...statuses,
					[formKey]: {
						...statuses[formKey],
						status: 'idle'
					}
				}));
			}, 2000);

		} catch (error) {
			console.error('Failed to save autosave data:', error);
			autosaveStatus.update(statuses => ({
				...statuses,
				[formKey]: {
					...statuses[formKey],
					status: 'error'
				}
			}));
		}
	};

	const debouncedSave = debounce(saveToStorage, saveDelay);
	const hasUnsavedChanges = (currentData: Record<string, any>): boolean => {
		const saved = loadSavedData();
		if (!saved) return Object.keys(currentData).some(key => {
			const value = currentData[key];
			return value !== '' && value !== null && value !== undefined && value !== false && 
				   (Array.isArray(value) ? value.length > 0 : true);
		});

		const savedData = saved.data;
		for (const [key, value] of Object.entries(currentData)) {
			if (excludeFields.includes(key)) continue;
			if (JSON.stringify(value) !== JSON.stringify(savedData[key])) {
				return true;
			}
		}
		return false;
	};

	return {
		loadDraft: () => loadSavedData(),

		save: (data: Record<string, any>, isEditing = false, editingId?: string) => {
			autosaveStatus.update(statuses => ({
				...statuses,
				[formKey]: {
					...statuses[formKey],
					status: 'saving',
					hasChanges: true
				}
			}));

			debouncedSave(data, isEditing, editingId);
		},

		saveNow: (data: Record<string, any>, isEditing = false, editingId?: string) => {
			autosaveStatus.update(statuses => ({
				...statuses,
				[formKey]: {
					...statuses[formKey],
					status: 'saving'
				}
			}));
			
			saveToStorage(data, isEditing, editingId);
		},

		clearDraft: () => {
			if (typeof window === 'undefined') return;
			localStorage.removeItem(`autosave-${formKey}`);
			autosaveStatus.update(statuses => ({
				...statuses,
				[formKey]: {
					...statuses[formKey],
					status: 'idle',
					lastSaved: null,
					hasChanges: false
				}
			}));
		},

		hasUnsavedChanges,

		getStatus: () => derived(autosaveStatus, $statuses => $statuses[formKey]),
		markChanged: () => {
			autosaveStatus.update(statuses => ({
				...statuses,
				[formKey]: {
					...statuses[formKey],
					hasChanges: true
				}
			}));
		}
	};
}

/**
 * Global navigation guard - checks all forms for unsaved changes
 */
export function hasAnyUnsavedChanges(): boolean {
	if (typeof window === 'undefined') return false;
	
	for (let i = 0; i < localStorage.length; i++) {
		const key = localStorage.key(i);
		if (key?.startsWith('autosave-')) {
			return true;
		}
	}
	
	return false;
}

/**
 * Clear all autosave data (useful for cleanup)
 */
export function clearAllAutosaveData(): void {
	if (typeof window === 'undefined') return;

	const keysToRemove: string[] = [];
	for (let i = 0; i < localStorage.length; i++) {
		const key = localStorage.key(i);
		if (key?.startsWith('autosave-')) {
			keysToRemove.push(key);
		}
	}

	keysToRemove.forEach(key => localStorage.removeItem(key));
	
	// Clear status store
	autosaveStatus.set({});
}

/**
 * Get all forms that have autosave data
 */
export function getFormsWithDrafts(): string[] {
	if (typeof window === 'undefined') return [];

	const forms: string[] = [];
	for (let i = 0; i < localStorage.length; i++) {
		const key = localStorage.key(i);
		if (key?.startsWith('autosave-')) {
			forms.push(key.replace('autosave-', ''));
		}
	}

	return forms;
}

export { autosaveData, autosaveStatus };