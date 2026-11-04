<script lang="ts">
	import { preventDefault } from 'svelte/legacy';
	import { onMount } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { goto } from '$app/navigation';
	import { t } from 'svelte-i18n';
	import DOMPurify from 'isomorphic-dompurify';
	import { toasts, confirm } from '$lib/stores';
	import { brandName, hasFeature } from '$lib/stores/plan';
	import { createAutosave } from '$lib/stores/autosave';
	import { createFilterState } from '$lib/admin/filterState.svelte';
	import { pruneSelection, toggleSelection } from '$lib/admin/selection';
	import { createBulkActions } from '$lib/admin/bulk-mutation';
	import { createListLoader } from '$lib/admin/list-load';
	import {
		DraftReadError,
		findDraftRecord,
		isRecordId,
		mayHaveLanded,
		newRecordId
	} from '$lib/admin/create-record';
	import ListLoadStatus from '$components/admin/ListLoadStatus.svelte';
	import SingleMediaPicker from '$lib/components/admin/SingleMediaPicker.svelte';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';
	import AdminFilters from '$components/admin/AdminFilters.svelte';
	import Toggle from '$components/admin/Toggle.svelte';
	import VisibilitySelector from '$components/admin/VisibilitySelector.svelte';
	import ReorderAnnouncer from '$lib/components/admin/ReorderAnnouncer.svelte';
	import { pb } from '$lib/pocketbase';
	import { readAllRecords } from '$lib/admin/read-all-records';

	const hasCourses = hasFeature('courses');

	// Shared screen-reader announcer for keyboard curriculum reorder (modules + lessons).
	let reorderAnnouncer = $state<ReorderAnnouncer | undefined>(undefined);

	type Course = {
		id: string;
		title: string;
		slug: string;
		description: string;
		cover_image?: string;
		cover_image_library_url?: string;
		instructor_name: string;
		instructor_bio: string;
		access_tier: 'free' | 'paid';
		price: number;
		difficulty: 'beginner' | 'intermediate' | 'advanced';
		estimated_hours: number;
		tags: string[];
		learning_outcomes: string[];
		visibility: 'public' | 'unlisted' | 'private';
		is_draft: boolean;
		sequential_access: boolean;
		enable_progress: boolean;
		show_progress_bar: boolean;
		require_enrollment: boolean;
		enable_certificates: boolean;
		sort_order: number;
	};

	type Module = {
		id: string;
		course: string;
		title: string;
		description: string;
		sort_order: number;
		drip_days_after_enrollment?: number;
		drip_unlock_date?: string;
	};

	type Lesson = {
		id: string;
		module: string;
		title: string;
		content_type: 'video' | 'text' | 'mixed' | 'quiz';
		video_url: string;
		content: string;
		estimated_minutes: number;
		is_preview: boolean;
		sort_order: number;
	};

	interface QuizQuestion {
		id?: string;
		question_text: string;
		question_type: 'multiple_choice' | 'true_false';
		options: { text: string; is_correct: boolean }[];
		explanation: string;
		points: number;
		sort_order: number;
	}

	type LoadedQuestion = Omit<QuizQuestion, 'options'> & { options: QuizQuestion['options'] | null };
	type LoadedQuiz = {
		id: string;
		title: string;
		passing_score: number;
		max_attempts: number;
		randomize_questions: boolean;
		questions: LoadedQuestion[];
	};

	const WRITE_TIMEOUT_MS = 30_000;
	// Attachments can run to 50 MB each, so a Save carrying files gets longer than a plain write.
	const UPLOAD_TIMEOUT_MS = 600_000;
	// The GET's other 404s mean a wrong course or lesson, so only this one reads as "no quiz yet".
	const NO_QUIZ_ERROR = 'no quiz found for this lesson';

	function isRecord(value: unknown): value is Record<string, unknown> {
		return typeof value === 'object' && value !== null && !Array.isArray(value);
	}

	function isQuizOption(value: unknown) {
		return (
			isRecord(value) && typeof value.text === 'string' && typeof value.is_correct === 'boolean'
		);
	}

	function isLoadedQuestion(value: unknown) {
		return (
			isRecord(value) &&
			typeof value.id === 'string' &&
			typeof value.question_text === 'string' &&
			(value.question_type === 'multiple_choice' || value.question_type === 'true_false') &&
			// The stored options field is optional, and an empty one comes back as null.
			(value.options === null ||
				(Array.isArray(value.options) && value.options.every(isQuizOption))) &&
			typeof value.explanation === 'string' &&
			typeof value.points === 'number' &&
			typeof value.sort_order === 'number'
		);
	}

	function isLoadedQuiz(value: unknown, lessonId: string): value is LoadedQuiz {
		return (
			isRecord(value) &&
			typeof value.id === 'string' &&
			value.id !== '' &&
			value.lesson === lessonId &&
			typeof value.title === 'string' &&
			typeof value.passing_score === 'number' &&
			typeof value.max_attempts === 'number' &&
			typeof value.randomize_questions === 'boolean' &&
			Array.isArray(value.questions) &&
			value.questions.every(isLoadedQuestion)
		);
	}

	let courses: Course[] = $state([]);
	let allModules: Module[] = $state([]);
	let allLessons: Lesson[] = $state([]);
	const listLoad = createListLoader('courses');
	const bulk = createBulkActions(listLoad, {
		records: () => pb.collection('courses'),
		items: () => courses,
		reload: loadCourses,
		finish: (remaining) => {
			selectedIds = remaining;
			selectMode = remaining.size > 0;
		}
	});
	let loading = $derived($listLoad.loading && !$listLoad.loaded);
	let listLocked = $derived($listLoad.mutating || $listLoad.failed || $listLoad.loading);
	let showForm = $state(false);
	let activeTab: 'content' | 'curriculum' | 'settings' | 'details' | 'students' = $state('content');
	let editingCourse: Course | null = $state(null);

	// Select mode
	let selectMode = $state(false);
	let selectedIds: Set<string> = $state(new Set());

	// Filters
	const filterStore = createFilterState({
		enableSearch: true,
		enableVisibilityFilter: true,
		enableDraftFilter: true,
		enableTagFilter: false,
		searchPlaceholder: 'Search courses...'
	});
	let showAdvancedFilters = $state(false);

	let filteredCourses = $derived(
		filterStore.filterItems(
			courses,
			(course) =>
				`${course.title} ${course.description || ''} ${(course.tags || []).join(' ')} ${course.instructor_name || ''}`,
			(course) => course.visibility,
			(course) => course.is_draft,
			() => []
		)
	);

	// Autosave
	const autosave = createAutosave('admin-courses', { saveDelay: 1500 });
	let showRecoveryBanner = $state(false);
	let recoveryData: { savedAt: number; isEditing: boolean } | null = $state(null);

	// Form fields
	let title = $state('');
	let slug = $state('');
	let description = $state('');
	let coverImageFile: FileList | null = $state(null);
	let coverImageLibraryUrl = $state('');
	let clearCoverImage = $state(false);
	let instructorName = $state('');
	let instructorBio = $state('');
	// Read-only: shown so an imported paid course is flagged. Never sent on save.
	let accessTier: 'free' | 'paid' = $state('free');
	let difficulty: 'beginner' | 'intermediate' | 'advanced' = $state('beginner');
	let estimatedHours = $state(0);
	let tags: string[] = $state([]);
	let tagInput = $state('');
	let learningOutcomes: string[] = $state([]);
	let outcomeInput = $state('');
	let visibility: 'public' | 'unlisted' | 'private' = $state('public');
	let isDraft = $state(true);
	let sequentialAccess = $state(false);
	let enableProgress = $state(true);
	let showProgressBar = $state(true);
	let requireEnrollment = $state(false);
	let enableCertificates = $state(true);
	let saving = $state(false);
	let courseFormRun = 0;
	let courseFormLocked = $state(false);
	let courseSavingId: string | null = $state(null);
	let courseDraftId = newRecordId();
	let courseCreateSent = false;
	let courseCreateIssue: '' | 'unknown' | 'differs' = $state('');

	// Curriculum state
	let courseModules: Module[] = $state([]);
	let courseLessons: Record<string, Lesson[]> = $state({});
	let expandedModules: Set<string> = $state(new Set());
	let expandedLessons: Set<string> = $state(new Set());
	let loadingCurriculum = $state(false);
	// Set when a read confirms this course's rows. Writes wait for a confirmed, current read.
	let curriculumCourseId: string | null = $state(null);
	let curriculumFailed = $state(false);
	let curriculumReady = $derived.by(
		() =>
			!!editingCourse &&
			curriculumCourseId === editingCourse.id &&
			!curriculumFailed &&
			!loadingCurriculum
	);

	// Inline module form
	let showModuleForm = $state(false);
	let editingModule: Module | null = $state(null);
	let moduleTitle = $state('');
	let moduleDescription = $state('');
	let moduleDripDaysAfterEnrollment = $state('');
	let moduleDripUnlockDate = $state('');
	let moduleFormRun = 0;
	let moduleFormLocked = $state(false);
	let moduleSavingId: string | null = $state(null);
	let moduleDraftId = newRecordId();
	let moduleCreateSent = false;
	let moduleCreateIssue: '' | 'unknown' | 'differs' = $state('');

	// Inline lesson form
	let showLessonForm: string | null = $state(null);
	let editingLesson: { lesson: Lesson; moduleId: string } | null = $state(null);
	let lessonTitle = $state('');
	let lessonContentType: 'video' | 'text' | 'mixed' | 'quiz' = $state('mixed');
	let lessonVideoUrl = $state('');
	let lessonContent = $state('');
	let lessonEstimatedMinutes = $state(0);
	let lessonIsPreview = $state(false);
	let lessonAttachmentFiles: FileList | null = $state(null);
	let lessonFileInput = $state<HTMLInputElement | undefined>(undefined);
	let lessonPreviewMode = $state(false);
	// Set while this form's Save is in flight, so nothing changes under it. Leaving clears it.
	let lessonFormLocked = $state(false);
	// The lesson a Save is writing, until its reloads finish. Leaving the form doesn't clear this,
	// so the lesson can't reopen on old fields. One page write runs at a time, so one ID covers it.
	let lessonSavingId: string | null = $state(null);

	// Quiz builder state
	let quizTitle = $state('');
	let quizPassingScore = $state(70);
	let quizMaxAttempts = $state(0);
	let quizRandomize = $state(false);
	let quizQuestions: QuizQuestion[] = $state([]);
	let quizId: string | null = $state(null);
	let loadingQuiz = $state(false);
	// The lesson a read confirmed. A new lesson can't have a quiz yet, so it needs no read.
	let quizLessonId: string | null = $state(null);
	let quizFailed = $state(false);
	// Set when a save starts. Reads then confirm what's stored without replacing the draft.
	let quizDraftHeld = $state(false);
	let quizLanded = false;
	let quizSaveIssue: '' | 'partial' | 'conflict' = $state('');
	// Moves on whenever the lesson form opens or closes, so a slow save can tell it was left.
	let lessonFormRun = 0;
	// Names the record this form will create, so a lost response can be looked up, not repeated.
	let lessonDraftId = newRecordId();
	// The files the last create carried. Once set, a new save looks the record up before creating.
	let lessonCreateSent: File[] | null = null;
	let lessonCreateIssue: '' | 'unknown' | 'differs' = $state('');
	let quizReady = $derived.by(
		() =>
			!quizFailed && !loadingQuiz && (!editingLesson || quizLessonId === editingLesson.lesson.id)
	);
	let quizEditable = $derived(quizReady || quizDraftHeld);

	// Enrollment dashboard
	let showStudents = $state(true);
	let loadingStudents = $state(false);
	let studentsCourseId: string | null = $state(null);
	let studentsFailed = $state(false);
	let students: Array<{
		email: string;
		progress_percent: number;
		lessons_completed: number;
		total_lessons: number;
		started_at: string;
		completed_at: string;
		last_accessed: string;
		has_certificate: boolean;
	}> = $state([]);
	let enrollmentStats: {
		total_enrolled: number;
		completed: number;
		average_progress: number;
		completion_rate: number;
	} | null = $state(null);

	function isEnrollment(value: unknown): value is {
		students: typeof students;
		stats: NonNullable<typeof enrollmentStats>;
	} {
		if (!isRecord(value) || !Array.isArray(value.students) || !isRecord(value.stats)) return false;
		const stats = value.stats;
		return (
			['total_enrolled', 'completed', 'average_progress', 'completion_rate'].every(
				(key) => typeof stats[key] === 'number'
			) &&
			value.students.every(
				(student) =>
					isRecord(student) &&
					typeof student.email === 'string' &&
					typeof student.progress_percent === 'number' &&
					typeof student.lessons_completed === 'number' &&
					typeof student.total_lessons === 'number'
			)
		);
	}

	function getFormData() {
		return {
			title,
			slug,
			description,
			instructorName,
			instructorBio,
			difficulty,
			estimatedHours,
			tags,
			learningOutcomes,
			visibility,
			isDraft,
			sequentialAccess,
			enableProgress,
			showProgressBar,
			requireEnrollment,
			enableCertificates,
			coverImageLibraryUrl,
			draftId: editingCourse ? undefined : courseDraftId
		};
	}

	function restoreFromDraft(data: Record<string, any>) {
		courseFormRun += 1;
		courseDraftId = isRecordId(data.draftId) ? data.draftId : newRecordId();
		// A restored draft may already have reached the server before a lost response.
		courseCreateSent = isRecordId(data.draftId);
		courseCreateIssue = '';
		title = data.title || '';
		slug = data.slug || '';
		description = data.description || '';
		instructorName = data.instructorName || '';
		instructorBio = data.instructorBio || '';
		difficulty = data.difficulty || 'beginner';
		estimatedHours = data.estimatedHours || 0;
		tags = data.tags || [];
		learningOutcomes = data.learningOutcomes || [];
		visibility = data.visibility || 'public';
		isDraft = data.isDraft !== false;
		sequentialAccess = data.sequentialAccess || false;
		enableProgress = data.enableProgress !== false;
		showProgressBar = data.showProgressBar !== false;
		requireEnrollment = data.requireEnrollment || false;
		enableCertificates = data.enableCertificates !== false;
		coverImageLibraryUrl = data.coverImageLibraryUrl || '';
	}

	function handleFormChange() {
		if (showForm) {
			autosave.save(getFormData(), !!editingCourse, editingCourse?.id);
		}
	}

	onMount(() => {
		if (!$hasCourses) {
			goto('/admin');
		} else {
			loadCourses();
			checkRecoveryDraft();
		}
	});

	afterNavigate(({ shallow, type }) => {
		if (shallow && type === 'goto') return;

		courseFormRun += 1;
		courseFormLocked = false;
		showForm = false;
		editingCourse = null;
		selectMode = false;
		selectedIds = new Set();
		checkRecoveryDraft();
	});

	// The admin layout may mount this page after the initial navigation callback.
	function checkRecoveryDraft() {
		const draft = autosave.loadDraft();
		if (
			draft?.data &&
			Object.values(draft.data).some(
				(v) => v !== '' && v !== false && v !== 0 && !(Array.isArray(v) && v.length === 0)
			)
		) {
			recoveryData = { savedAt: draft.savedAt, isEditing: draft.isEditing || false };
			showRecoveryBanner = true;
		} else {
			showRecoveryBanner = false;
			recoveryData = null;
		}
	}

	function loadCourses() {
		return listLoad.load(
			(_requestKey, signal) =>
				Promise.all([
					readAllRecords<Course>('courses', { sort: 'sort_order', perPage: '500' }, signal),
					readAllRecords<Module>('modules', { sort: 'sort_order', perPage: '500' }, signal),
					readAllRecords<Lesson>('lessons', { sort: 'sort_order', perPage: '500' }, signal)
				]),
			([courseRecords, moduleRecords, lessonRecords]) => {
				courses = courseRecords;
				allModules = moduleRecords;
				allLessons = lessonRecords;
			}
		);
	}

	function loadCurriculum(courseId: string) {
		loadingCurriculum = true;
		return listLoad.read(
			'curriculum',
			(_requestKey, signal) =>
				Promise.all([
					readAllRecords<Module>(
						'modules',
						{ filter: `course="${courseId}"`, sort: 'sort_order', perPage: '500' },
						signal
					),
					readAllRecords<Lesson>(
						'lessons',
						{
							filter: `module.course="${courseId}"`,
							sort: 'module.sort_order,sort_order',
							perPage: '500'
						},
						signal
					)
				]),
			([modulesResult, lessonsResult]) => {
				courseModules = modulesResult;

				const lessonsByModule: Record<string, Lesson[]> = {};
				for (const module of courseModules) {
					lessonsByModule[module.id] = [];
				}
				for (const lesson of lessonsResult) {
					if (lessonsByModule[lesson.module]) {
						lessonsByModule[lesson.module].push(lesson);
					}
				}

				courseLessons = lessonsByModule;
				expandedModules = new Set(courseModules.map((m) => m.id));
				curriculumCourseId = courseId;
				curriculumFailed = false;
				loadingCurriculum = false;
			},
			(err) => {
				console.error('Failed to load curriculum:', err);
				toasts.add('error', $t('admin.courses.error_load_curriculum'));
				curriculumFailed = true;
				loadingCurriculum = false;
			}
		);
	}

	// Drops the last course's rows, open drafts and any read still in flight.
	function clearCurriculum() {
		listLoad.cancelRead('curriculum');
		curriculumCourseId = null;
		curriculumFailed = false;
		loadingCurriculum = false;
		courseModules = [];
		courseLessons = {};
		expandedModules = new Set();
		expandedLessons = new Set();
		closeModuleForm();
		closeLessonForm();
	}

	async function retryCurriculum() {
		if (!editingCourse) return;
		const opener = document.activeElement;
		await loadCurriculum(editingCourse.id);
		await listLoad.restoreFocus(opener);
	}

	// Curriculum writes share the page's write guard and also need a confirmed read of this course.
	function curriculumWrite<A extends unknown[]>(run: (...args: A) => Promise<void>) {
		return listLoad.exclusive(async (...args: A) => {
			if (curriculumReady) await run(...args);
			else toasts.add('error', $t('admin.courses.curriculum_failed'));
		});
	}

	function resetForm() {
		courseFormRun += 1;
		courseFormLocked = false;
		courseDraftId = newRecordId();
		courseCreateSent = false;
		courseCreateIssue = '';
		title = '';
		slug = '';
		description = '';
		coverImageFile = null;
		coverImageLibraryUrl = '';
		clearCoverImage = false;
		instructorName = '';
		instructorBio = '';
		accessTier = 'free';
		difficulty = 'beginner';
		estimatedHours = 0;
		tags = [];
		tagInput = '';
		learningOutcomes = [];
		outcomeInput = '';
		visibility = 'public';
		isDraft = true;
		sequentialAccess = false;
		enableProgress = true;
		showProgressBar = true;
		requireEnrollment = false;
		enableCertificates = true;
		editingCourse = null;
		activeTab = 'content';
		clearCurriculum();
		clearStudents();
	}

	function openNewForm() {
		resetForm();
		showForm = true;
		showRecoveryBanner = false;
	}

	function handleRestoreDraft() {
		if (!listLoad.canMutate()) return;
		const draft = autosave.loadDraft();
		if (draft?.data) {
			restoreFromDraft(draft.data);
			if (draft.isEditing && draft.editingId) {
				const course = courses.find((c) => c.id === draft.editingId);
				if (course) {
					editingCourse = course;
					clearCurriculum();
					clearStudents();
					loadCurriculum(course.id);
				}
			}
			showForm = true;
		}
		showRecoveryBanner = false;
	}

	function handleDismissDraft() {
		autosave.clearDraft();
		showRecoveryBanner = false;
		recoveryData = null;
	}

	async function openEditForm(course: Course) {
		if (course.id === courseSavingId) return;
		courseFormRun += 1;
		courseFormLocked = false;
		courseCreateIssue = '';
		clearCurriculum();
		editingCourse = course;
		clearStudents();
		title = course.title;
		slug = course.slug || '';
		description = course.description || '';
		coverImageFile = null;
		coverImageLibraryUrl = course.cover_image_library_url || '';
		clearCoverImage = false;
		instructorName = course.instructor_name || '';
		instructorBio = course.instructor_bio || '';
		accessTier = course.access_tier || 'free';
		difficulty = course.difficulty || 'beginner';
		estimatedHours = course.estimated_hours || 0;
		tags = course.tags || [];
		learningOutcomes = course.learning_outcomes || [];
		visibility = course.visibility || 'public';
		isDraft = course.is_draft !== false;
		sequentialAccess = course.sequential_access || false;
		enableProgress = course.enable_progress !== false;
		showProgressBar = course.show_progress_bar !== false;
		requireEnrollment = course.require_enrollment || false;
		enableCertificates = course.enable_certificates !== false;
		showForm = true;

		await loadCurriculum(course.id);
	}

	function closeForm() {
		showForm = false;
		resetForm();
		autosave.clearDraft();
	}

	function generateSlug() {
		slug = title
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-|-$/g, '');
	}

	function addTag() {
		const tag = tagInput.trim();
		if (tag && !tags.includes(tag)) {
			tags = [...tags, tag];
		}
		tagInput = '';
	}

	function removeTag(tag: string) {
		tags = tags.filter((t) => t !== tag);
	}

	function handleTagKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			addTag();
		}
	}

	function addOutcome() {
		const outcome = outcomeInput.trim();
		if (outcome && !learningOutcomes.includes(outcome)) {
			learningOutcomes = [...learningOutcomes, outcome];
		}
		outcomeInput = '';
	}

	function removeOutcome(outcome: string) {
		learningOutcomes = learningOutcomes.filter((o) => o !== outcome);
	}

	function handleOutcomeKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			addOutcome();
		}
	}

	function courseWriteOptions(files: File[]) {
		return {
			requestKey: null,
			signal: AbortSignal.timeout(files.length ? UPLOAD_TIMEOUT_MS : WRITE_TIMEOUT_MS)
		};
	}

	async function createDraftCourse(formData: FormData, files: File[], isCurrent: () => boolean) {
		const id = courseDraftId;
		const records = pb.collection('courses');
		formData.append('id', id);
		if (courseCreateSent) {
			const course = await findDraftRecord<Course>(records, id);
			if (course) return { course, readBack: true };
			if (!isCurrent()) return null;
		}

		courseCreateSent = true;
		try {
			const course = await records.create<Course>(formData, courseWriteOptions(files));
			return { course, readBack: false };
		} catch (err) {
			if (!mayHaveLanded(err)) {
				if (isCurrent()) courseCreateSent = false;
				throw err;
			}
			const course = await findDraftRecord<Course>(records, id);
			if (!course) throw err;
			return { course, readBack: true };
		}
	}

	function courseMatches(stored: Course, intent: ReturnType<typeof getFormData>, files: File[]) {
		return (
			files.length === 0 &&
			stored.title === intent.title.trim() &&
			stored.slug === intent.slug.trim() &&
			(stored.description || '') === intent.description.trim() &&
			(stored.instructor_name || '') === intent.instructorName.trim() &&
			(stored.instructor_bio || '') === intent.instructorBio.trim() &&
			stored.difficulty === intent.difficulty &&
			(stored.estimated_hours || 0) === intent.estimatedHours &&
			JSON.stringify(stored.tags || []) === JSON.stringify(intent.tags) &&
			JSON.stringify(stored.learning_outcomes || []) === JSON.stringify(intent.learningOutcomes) &&
			stored.visibility === intent.visibility &&
			stored.is_draft === intent.isDraft &&
			stored.sequential_access === intent.sequentialAccess &&
			stored.enable_progress === intent.enableProgress &&
			stored.show_progress_bar === intent.showProgressBar &&
			stored.require_enrollment === intent.requireEnrollment &&
			stored.enable_certificates === intent.enableCertificates &&
			(stored.cover_image_library_url || '') === intent.coverImageLibraryUrl
		);
	}

	const handleSubmit = listLoad.exclusive(async () => {
		if (!title.trim()) {
			toasts.add('error', $t('admin.courses.error_title_required'));
			return;
		}
		if (!slug.trim()) {
			toasts.add('error', $t('admin.courses.error_slug_required'));
			return;
		}

		const existingCourse = editingCourse;
		const formRun = courseFormRun;
		const isCurrent = () => formRun === courseFormRun;
		const intent = getFormData();
		const files = coverImageFile?.length ? [coverImageFile[0]] : [];
		let issue: typeof courseCreateIssue = '';
		saving = true;
		courseFormLocked = true;
		courseSavingId = existingCourse?.id ?? courseDraftId;
		try {
			const formData = new FormData();
			formData.append('title', intent.title.trim());
			formData.append('slug', intent.slug.trim());
			formData.append('description', intent.description.trim());
			formData.append('instructor_name', intent.instructorName.trim());
			formData.append('instructor_bio', intent.instructorBio.trim());
			// Course sales are Facet Cloud-only. New courses are free; an existing
			// course's tier and price are left exactly as stored, so an imported
			// paid course is never silently made free.
			if (!existingCourse) {
				formData.append('access_tier', 'free');
				formData.append('price', '0');
			}

			formData.append('difficulty', intent.difficulty);
			formData.append('estimated_hours', String(intent.estimatedHours));
			formData.append('tags', JSON.stringify(intent.tags));
			formData.append('learning_outcomes', JSON.stringify(intent.learningOutcomes));
			formData.append('visibility', intent.visibility);
			formData.append('is_draft', String(intent.isDraft));
			formData.append('sequential_access', String(intent.sequentialAccess));
			formData.append('enable_progress', String(intent.enableProgress));
			formData.append('show_progress_bar', String(intent.showProgressBar));
			formData.append('require_enrollment', String(intent.requireEnrollment));
			formData.append('enable_certificates', String(intent.enableCertificates));

			if (files.length) {
				formData.append('cover_image', files[0]);
				formData.append('cover_image_library_url', '');
			} else if (clearCoverImage && existingCourse?.cover_image) {
				formData.append('cover_image', '');
				formData.append('cover_image_library_url', intent.coverImageLibraryUrl || '');
			} else {
				formData.append('cover_image_library_url', intent.coverImageLibraryUrl || '');
			}

			let savedCourse = existingCourse;
			if (existingCourse) {
				await pb
					.collection('courses')
					.update(existingCourse.id, formData, courseWriteOptions(files));
			} else {
				const created = await createDraftCourse(formData, files, isCurrent);
				if (!created) return;
				savedCourse = created.course;
				if (isCurrent()) {
					editingCourse = created.course;
					if (created.readBack && !courseMatches(created.course, intent, files)) issue = 'differs';
				}
			}

			toasts.add('success', $t('admin.courses.saved'));
			if (isCurrent() && savedCourse && !issue) autosave.clearDraft();
			if (savedCourse && editingCourse?.id === savedCourse.id) await loadCurriculum(savedCourse.id);
			await loadCourses();
		} catch (err: unknown) {
			console.error('Failed to save course:', err);
			if (err instanceof DraftReadError && isCurrent()) issue = 'unknown';
			else {
				const error = err as { data?: { data?: { slug?: { message?: string } } } };
				if (error.data?.data?.slug?.message) {
					toasts.add('error', $t('admin.courses.error_slug_exists'));
				} else {
					toasts.add('error', $t('admin.courses.error_save'));
				}
			}
		} finally {
			if (isCurrent()) courseCreateIssue = issue;
			saving = false;
			courseFormLocked = false;
			courseSavingId = null;
		}
	});

	const deleteCourse = listLoad.exclusive(async (course: Course) => {
		const confirmed = await confirm({
			title: $t('admin.courses.delete_course'),
			message: $t('admin.courses.delete_course_confirm'),
			confirmText: $t('admin.courses.confirm_delete'),
			danger: true
		});
		if (!confirmed) return;

		try {
			await pb.collection('courses').delete(course.id);
			toasts.add('success', $t('admin.courses.deleted'));
			if (editingCourse?.id === course.id) {
				closeForm();
			}
			await loadCourses();
		} catch (err) {
			console.error('Failed to delete course:', err);
			toasts.add('error', $t('admin.courses.error_delete'));
		}
	});

	const togglePublish = listLoad.exclusive(async (course: Course) => {
		try {
			const newDraftState = !course.is_draft;
			await pb.collection('courses').update(course.id, { is_draft: newDraftState });
			toasts.add(
				'success',
				newDraftState ? $t('admin.courses.unpublished') : $t('admin.courses.published')
			);
			await loadCourses();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', $t('admin.courses.error_save'));
		}
	});

	function getCourseModuleCount(courseId: string): number {
		return allModules.filter((m) => m.course === courseId).length;
	}

	function getCourseLessonCount(courseId: string): number {
		return allLessons.filter((l) => {
			const mod = allModules.find((m) => m.id === l.module);
			return mod?.course === courseId;
		}).length;
	}

	// Select mode
	$effect(() => {
		const visibleSelection = pruneSelection(
			selectedIds,
			filteredCourses.map((course) => course.id)
		);
		if (visibleSelection !== selectedIds) selectedIds = visibleSelection;
	});

	function toggleSelectMode() {
		selectMode = !selectMode;
		if (!selectMode) selectedIds = new Set();
	}

	function toggleSelect(id: string) {
		selectedIds = toggleSelection(selectedIds, id);
	}

	function selectAll() {
		selectedIds = new Set(filteredCourses.map((course) => course.id));
	}

	function clearSelection() {
		selectedIds = new Set();
	}

	function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		return bulk.setVisibility(Array.from(selectedIds), visibility);
	}

	function bulkDelete() {
		const ids = Array.from(selectedIds);
		return bulk.delete(ids, () =>
			confirm({
				title: $t('admin.courses.delete_course'),
				message: $t('admin.courses.bulk_delete_confirm', { values: { count: ids.length } }),
				confirmText: $t('admin.courses.confirm_delete'),
				danger: true
			})
		);
	}

	// Module expand/collapse
	function toggleModuleExpand(moduleId: string) {
		if (expandedModules.has(moduleId)) {
			expandedModules.delete(moduleId);
		} else {
			expandedModules.add(moduleId);
		}
		expandedModules = expandedModules;
	}

	// Lesson expand/collapse
	function toggleLessonExpand(lessonId: string) {
		if (expandedLessons.has(lessonId)) {
			expandedLessons.delete(lessonId);
		} else {
			expandedLessons.add(lessonId);
		}
		expandedLessons = expandedLessons;
	}

	// Module management
	function openNewModuleForm() {
		moduleFormRun += 1;
		moduleFormLocked = false;
		moduleDraftId = newRecordId();
		moduleCreateSent = false;
		moduleCreateIssue = '';
		editingModule = null;
		moduleTitle = '';
		moduleDescription = '';
		moduleDripDaysAfterEnrollment = '';
		moduleDripUnlockDate = '';
		showModuleForm = true;
	}

	function openEditModuleForm(module: Module) {
		if (module.id === moduleSavingId) return;
		moduleFormRun += 1;
		moduleFormLocked = false;
		moduleCreateIssue = '';
		editingModule = module;
		moduleTitle = module.title;
		moduleDescription = module.description || '';
		moduleDripDaysAfterEnrollment = module.drip_days_after_enrollment
			? String(module.drip_days_after_enrollment)
			: '';
		moduleDripUnlockDate = module.drip_unlock_date || '';
		showModuleForm = true;
	}

	function closeModuleForm() {
		moduleFormRun += 1;
		moduleFormLocked = false;
		moduleDraftId = newRecordId();
		moduleCreateSent = false;
		moduleCreateIssue = '';
		showModuleForm = false;
		editingModule = null;
		moduleTitle = '';
		moduleDescription = '';
		moduleDripDaysAfterEnrollment = '';
		moduleDripUnlockDate = '';
	}

	function moduleWriteOptions() {
		return { requestKey: null, signal: AbortSignal.timeout(WRITE_TIMEOUT_MS) };
	}

	async function createDraftModule(
		data: Record<string, any>,
		isCurrent: () => boolean
	): Promise<{ module: Module; readBack: boolean } | null> {
		const id = moduleDraftId;
		const records = pb.collection('modules');
		data.id = id;
		if (moduleCreateSent) {
			const module = await findDraftRecord<Module>(records, id);
			if (module) return { module, readBack: true };
			if (!isCurrent()) return null;
		}

		moduleCreateSent = true;
		try {
			const module = await records.create<Module>(data, moduleWriteOptions());
			return { module, readBack: false };
		} catch (err) {
			if (!mayHaveLanded(err)) {
				if (isCurrent()) moduleCreateSent = false;
				throw err;
			}
			const module = await findDraftRecord<Module>(records, id);
			if (!module) throw err;
			return { module, readBack: true };
		}
	}

	function moduleMatches(stored: Module, intent: Record<string, any>) {
		return (
			stored.title === intent.title &&
			(stored.description || '') === intent.description &&
			stored.course === intent.course &&
			(stored.drip_days_after_enrollment || 0) === intent.drip_days_after_enrollment &&
			(stored.drip_unlock_date || '') === intent.drip_unlock_date
		);
	}

	const saveModule = curriculumWrite(async () => {
		if (!moduleTitle.trim() || !editingCourse) return;

		const courseId = editingCourse.id;
		const existingModule = editingModule;
		const formRun = moduleFormRun;
		const isCurrent = () => formRun === moduleFormRun;
		const data: Record<string, any> = {
			title: moduleTitle.trim(),
			description: moduleDescription.trim(),
			sort_order: existingModule ? existingModule.sort_order : courseModules.length + 1,
			course: courseId,
			drip_days_after_enrollment: moduleDripDaysAfterEnrollment
				? parseInt(moduleDripDaysAfterEnrollment)
				: 0,
			drip_unlock_date: moduleDripUnlockDate ? new Date(moduleDripUnlockDate).toISOString() : ''
		};
		let issue: typeof moduleCreateIssue = '';
		saving = true;
		moduleFormLocked = true;
		moduleSavingId = existingModule?.id ?? moduleDraftId;
		try {
			let savedModule = existingModule;
			if (existingModule) {
				await pb.collection('modules').update(existingModule.id, data, moduleWriteOptions());
			} else {
				const created = await createDraftModule(data, isCurrent);
				if (!created) return;
				savedModule = created.module;
				if (isCurrent()) {
					editingModule = created.module;
					if (created.readBack && !moduleMatches(created.module, data)) issue = 'differs';
				}
			}

			toasts.add(
				'success',
				$t(existingModule ? 'admin.courses.module_updated' : 'admin.courses.module_created')
			);
			if (editingCourse?.id === courseId) await loadCurriculum(courseId);
			if (
				isCurrent() &&
				!issue &&
				curriculumCourseId === courseId &&
				!curriculumFailed &&
				savedModule
			) {
				closeModuleForm();
			}
			await loadCourses();
		} catch (err) {
			console.error('Failed to save module:', err);
			if (err instanceof DraftReadError && isCurrent()) issue = 'unknown';
			else toasts.add('error', $t('admin.courses.error_save_module'));
		} finally {
			if (isCurrent()) moduleCreateIssue = issue;
			saving = false;
			moduleFormLocked = false;
			moduleSavingId = null;
		}
	});

	// Deletes through the route that removes a module or lesson and its records in one transaction.
	// A refusal changes nothing, so the list and any open draft stay. Only a delete that succeeded,
	// or an answer that leaves the outcome open, reads the list again.
	async function deleteCurriculumItem(
		courseId: string,
		path: string,
		messages: { deleted: string; failed: string; refused: string },
		onDeleted: () => void
	) {
		// Stays 0 when no reply can be trusted, and then the delete may still have landed.
		let status = 0;
		try {
			const response = await fetch(`/api/admin/courses/${courseId}/${path}`, {
				method: 'DELETE',
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {},
				signal: AbortSignal.timeout(WRITE_TIMEOUT_MS)
			});
			const data: unknown = await response.json();
			if (!response.ok || (isRecord(data) && data.status === 'deleted')) status = response.status;
		} catch (err) {
			console.error('Failed to delete curriculum item:', err);
		}

		const deleted = status === 200;
		if (deleted) {
			toasts.add('success', $t(messages.deleted));
			onDeleted();
		} else {
			toasts.add('error', $t(status === 409 ? messages.refused : messages.failed));
		}
		if (!deleted && status !== 0 && status !== 404) return;
		if (editingCourse?.id === courseId) await loadCurriculum(courseId);
		await loadCourses();
	}

	const deleteModule = curriculumWrite(async (module: Module) => {
		if (!editingCourse) return;
		// The dialog and the delete both wait, and the open course can change during either.
		const courseId = editingCourse.id;
		const path = `modules/${module.id}`;

		const confirmed = await confirm({
			title: $t('admin.courses.delete_module'),
			message: $t('admin.courses.delete_module_confirm'),
			confirmText: $t('admin.courses.confirm_delete'),
			danger: true
		});
		if (!confirmed) return;

		await deleteCurriculumItem(
			courseId,
			path,
			{
				deleted: 'admin.courses.module_deleted',
				failed: 'admin.courses.error_delete_module',
				refused: 'admin.courses.error_delete_module_attempts'
			},
			() => {
				if (editingModule?.id === module.id) closeModuleForm();
				if (showLessonForm === module.id) closeLessonForm();
			}
		);
	});

	function moveRow<T>(rows: T[], from: number, to: number) {
		const next = [...rows];
		next.splice(to, 0, ...next.splice(from, 1));
		return next;
	}

	// Renumbers the whole list from what's on screen, so moving again also clears tied positions.
	// Rows already in place aren't rewritten, and the reload decides what actually saved.
	async function saveCurriculumOrder(
		collection: 'modules' | 'lessons',
		ordered: { id: string; sort_order: number }[],
		courseId: string,
		stored: () => { id: string; sort_order: number }[]
	) {
		const pending = ordered
			.map((row, index) => ({ id: row.id, sort_order: index + 1, previous: row.sort_order }))
			.filter((row) => row.sort_order !== row.previous);
		const results = await Promise.allSettled(
			pending.map(({ id, sort_order }) =>
				pb
					.collection(collection)
					.update(id, { sort_order }, { signal: AbortSignal.timeout(WRITE_TIMEOUT_MS) })
			)
		);
		const reloaded = await loadCurriculum(courseId);
		const saved = reloaded
			? pending.filter((row) =>
					stored().some(({ id, sort_order }) => id === row.id && sort_order === row.sort_order)
				).length
			: results.filter((result) => result.status === 'fulfilled').length;
		if (saved === pending.length) return true;
		toasts.add(
			'error',
			$t(
				reloaded
					? 'admin.courses.curriculum_order_partial'
					: 'admin.courses.curriculum_order_unconfirmed',
				{ values: { saved, total: pending.length } }
			)
		);
		return false;
	}

	const moveModule = curriculumWrite(async (module: Module, direction: 'up' | 'down') => {
		if (!editingCourse) return;

		const currentIndex = courseModules.findIndex((m) => m.id === module.id);
		if (currentIndex === -1) return;
		if (direction === 'up' && currentIndex === 0) return;
		if (direction === 'down' && currentIndex === courseModules.length - 1) return;

		const targetIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1;
		const saved = await saveCurriculumOrder(
			'modules',
			moveRow(courseModules, currentIndex, targetIndex),
			editingCourse.id,
			() => courseModules
		);
		if (!saved) return;
		// Announce the new position (1-based) for screen-reader users.
		reorderAnnouncer?.announce(
			$t('admin.courses.module_moved_announce', {
				values: { title: module.title, position: targetIndex + 1, total: courseModules.length }
			})
		);
	});

	// Lesson management
	function openNewLessonForm(moduleId: string) {
		editingLesson = null;
		lessonTitle = '';
		lessonContentType = 'mixed';
		lessonVideoUrl = '';
		lessonContent = '';
		lessonEstimatedMinutes = 0;
		lessonIsPreview = false;
		showLessonForm = moduleId;
		resetQuizState();
	}

	function openEditLessonForm(lesson: Lesson, moduleId: string) {
		if (!curriculumReady || lesson.id === lessonSavingId) return;
		editingLesson = { lesson, moduleId };
		lessonTitle = lesson.title;
		lessonContentType = lesson.content_type;
		lessonVideoUrl = lesson.video_url || '';
		lessonContent = lesson.content || '';
		lessonEstimatedMinutes = lesson.estimated_minutes;
		lessonIsPreview = lesson.is_preview;
		showLessonForm = moduleId;
		expandedLessons.add(lesson.id);
		expandedLessons = expandedLessons;
		resetQuizState();
	}

	function closeLessonForm() {
		showLessonForm = null;
		editingLesson = null;
		lessonTitle = '';
		lessonContentType = 'mixed';
		lessonVideoUrl = '';
		lessonContent = '';
		lessonEstimatedMinutes = 0;
		lessonIsPreview = false;
		lessonPreviewMode = false;
		resetQuizState();
	}

	function clearStudents() {
		listLoad.cancelRead('students');
		studentsCourseId = null;
		studentsFailed = false;
		loadingStudents = false;
		students = [];
		enrollmentStats = null;
	}

	function loadStudents(courseId: string) {
		loadingStudents = true;
		studentsFailed = false;
		return listLoad.read(
			'students',
			async (_requestKey, signal) => {
				const response = await fetch(`/api/admin/courses/${courseId}/students`, {
					headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {},
					signal
				});
				if (!response.ok) throw new Error(`Enrollment read failed (${response.status})`);
				const data: unknown = await response.json();
				if (!isEnrollment(data)) throw new Error('Invalid enrollment response');
				return data;
			},
			(data) => {
				students = data.students;
				enrollmentStats = data.stats;
				studentsCourseId = courseId;
				loadingStudents = false;
			},
			(err) => {
				console.error('Failed to load students:', err);
				studentsFailed = true;
				loadingStudents = false;
			}
		);
	}

	// Auto-load students when switching to students tab
	$effect(() => {
		if (
			activeTab === 'students' &&
			editingCourse &&
			!loadingStudents &&
			!studentsFailed &&
			studentsCourseId !== editingCourse.id
		) {
			loadStudents(editingCourse.id);
		}
	});

	// Clears the input too, so it can't show files that a later Save won't send.
	function clearLessonFiles() {
		lessonAttachmentFiles = null;
		if (lessonFileInput) lessonFileInput.value = '';
	}

	function resetQuizState() {
		listLoad.cancelRead('quiz');
		lessonFormRun += 1;
		lessonFormLocked = false;
		lessonDraftId = newRecordId();
		lessonCreateSent = null;
		clearLessonFiles();
		lessonCreateIssue = '';
		quizTitle = '';
		quizPassingScore = 70;
		quizMaxAttempts = 0;
		quizRandomize = false;
		quizQuestions = [];
		quizId = null;
		loadingQuiz = false;
		quizLessonId = null;
		quizFailed = false;
		quizDraftHeld = false;
		quizSaveIssue = '';
	}

	// Reads a lesson's quiz when it's needed, whether it opened as a quiz or was just switched to one.
	$effect(() => {
		if (
			lessonContentType === 'quiz' &&
			editingCourse &&
			editingLesson &&
			quizLessonId !== editingLesson.lesson.id &&
			!loadingQuiz &&
			!quizFailed &&
			!quizDraftHeld
		) {
			loadQuizData(editingCourse.id, editingLesson.lesson.id);
		}
	});

	async function fetchQuiz(courseId: string, lessonId: string, signal: AbortSignal) {
		const response = await fetch(`/api/admin/courses/${courseId}/lessons/${lessonId}/quiz`, {
			headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {},
			signal
		});
		const data: unknown = await response.json();
		if (response.status === 404 && isRecord(data) && data.error === NO_QUIZ_ERROR) return null;
		if (!response.ok || !isLoadedQuiz(data, lessonId)) {
			throw new Error(`Quiz read failed (${response.status})`);
		}
		return data;
	}

	function applyQuiz(quiz: LoadedQuiz) {
		quizId = quiz.id;
		quizTitle = quiz.title;
		quizPassingScore = quiz.passing_score;
		quizMaxAttempts = quiz.max_attempts;
		quizRandomize = quiz.randomize_questions;
		quizQuestions = quiz.questions.map((question) => ({
			...question,
			options: question.options ?? [
				{ text: '', is_correct: true },
				{ text: '', is_correct: false }
			]
		}));
	}

	// A deep copy of the quiz on screen. Nothing typed later can reach a request built from it.
	function quizDraft() {
		return {
			title: quizTitle || lessonTitle,
			passing_score: quizPassingScore,
			max_attempts: quizMaxAttempts,
			randomize_questions: quizRandomize,
			// Starts at 1: the stored field is required, and PocketBase counts 0 as blank.
			questions: $state.snapshot(quizQuestions).map((q, i) => ({ ...q, sort_order: i + 1 }))
		};
	}
	type QuizDraft = ReturnType<typeof quizDraft>;

	// After a save, learns what's stored and leaves the draft on screen alone.
	function confirmStoredQuiz(quiz: LoadedQuiz | null, expected: QuizDraft) {
		quizId = quiz?.id ?? null;
		quizLanded =
			!!quiz &&
			quiz.title === expected.title &&
			quiz.passing_score === expected.passing_score &&
			quiz.max_attempts === expected.max_attempts &&
			quiz.randomize_questions === expected.randomize_questions &&
			quiz.questions.length === expected.questions.length &&
			quiz.questions.every((question, index) => {
				const draft = expected.questions[index];
				return (
					question.question_text === draft.question_text &&
					question.question_type === draft.question_type &&
					question.explanation === draft.explanation &&
					question.points === draft.points &&
					question.sort_order === index + 1 &&
					question.options !== null &&
					question.options.length === draft.options.length &&
					question.options.every(
						(option, optionIndex) =>
							option.text === draft.options[optionIndex].text &&
							option.is_correct === draft.options[optionIndex].is_correct
					)
				);
			});
		quizSaveIssue = quiz && !quizLanded ? 'conflict' : 'partial';
	}

	function loadQuizData(courseId: string, lessonId: string, expected?: QuizDraft) {
		loadingQuiz = true;
		quizFailed = false;
		return listLoad.read(
			'quiz',
			(_requestKey, signal) => fetchQuiz(courseId, lessonId, signal),
			(quiz) => {
				quizLessonId = lessonId;
				loadingQuiz = false;
				if (quizDraftHeld) confirmStoredQuiz(quiz, expected ?? quizDraft());
				else if (quiz) applyQuiz(quiz);
			},
			(err) => {
				console.error('Failed to load quiz data:', err);
				quizFailed = true;
				loadingQuiz = false;
			}
		);
	}

	async function retryQuiz() {
		if (!editingCourse || !editingLesson) return;
		const opener = document.activeElement;
		await loadQuizData(editingCourse.id, editingLesson.lesson.id);
		await listLoad.restoreFocus(opener);
	}

	// True only once the server confirms this quiz is stored.
	// Takes what the Save press captured, so the form changing can't change what's written.
	async function saveQuizData(
		courseId: string,
		lessonId: string,
		quizPayload: QuizDraft,
		savedQuizId: string | null,
		isCurrent: () => boolean
	) {
		const creating = !savedQuizId;
		let status: number;
		try {
			const response = await fetch(`/api/admin/courses/${courseId}/lessons/${lessonId}/quiz`, {
				method: creating ? 'POST' : 'PATCH',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify(quizPayload),
				signal: AbortSignal.timeout(WRITE_TIMEOUT_MS)
			});
			status = response.status;
			if (response.ok) {
				const data = await response.json();
				if (typeof data?.id !== 'string') throw new Error('Invalid quiz save response');
				if (isCurrent()) quizId = data.id;
				return true;
			}
		} catch (err) {
			console.error('Failed to save quiz data:', err);
			status = 0;
		}

		// A PATCH replaces the whole quiz, so pressing Save again is safe. A POST that may have
		// landed, or a 409, must be read back first so a retry can't start a duplicate.
		const unknown = status === 0 || status === 408 || status === 409 || status >= 500;
		if (!creating || !unknown || !isCurrent()) return false;
		quizLanded = false;
		await loadQuizData(courseId, lessonId, quizPayload);
		return quizLanded;
	}

	function addQuizQuestion() {
		quizQuestions = [
			...quizQuestions,
			{
				question_text: '',
				question_type: 'multiple_choice',
				options: [
					{ text: '', is_correct: true },
					{ text: '', is_correct: false }
				],
				explanation: '',
				points: 10,
				sort_order: quizQuestions.length
			}
		];
	}

	function removeQuizQuestion(index: number) {
		quizQuestions = quizQuestions.filter((_, i) => i !== index);
	}

	function addQuizOption(questionIndex: number) {
		const q = quizQuestions[questionIndex];
		if (q.options.length >= 6) return;
		q.options = [...q.options, { text: '', is_correct: false }];
		quizQuestions = [...quizQuestions];
	}

	function removeQuizOption(questionIndex: number, optionIndex: number) {
		const q = quizQuestions[questionIndex];
		if (q.options.length <= 2) return;
		q.options = q.options.filter((_, i) => i !== optionIndex);
		if (!q.options.some((o) => o.is_correct)) {
			q.options[0].is_correct = true;
		}
		quizQuestions = [...quizQuestions];
	}

	function setCorrectOption(questionIndex: number, optionIndex: number) {
		const q = quizQuestions[questionIndex];
		q.options = q.options.map((o, i) => ({ ...o, is_correct: i === optionIndex }));
		quizQuestions = [...quizQuestions];
	}

	function setQuestionType(questionIndex: number, type: 'multiple_choice' | 'true_false') {
		const q = quizQuestions[questionIndex];
		q.question_type = type;
		if (type === 'true_false') {
			q.options = [
				{ text: 'True', is_correct: true },
				{ text: 'False', is_correct: false }
			];
		} else if (q.options.length < 2) {
			q.options = [
				{ text: '', is_correct: true },
				{ text: '', is_correct: false }
			];
		}
		quizQuestions = [...quizQuestions];
	}

	function moveQuizQuestion(index: number, direction: -1 | 1) {
		const newIndex = index + direction;
		if (newIndex < 0 || newIndex >= quizQuestions.length) return;
		const items = [...quizQuestions];
		[items[index], items[newIndex]] = [items[newIndex], items[index]];
		quizQuestions = items;
	}

	// The lesson fields on screen, in the form they're stored.
	function lessonDraft() {
		return {
			title: lessonTitle.trim(),
			content_type: lessonContentType,
			video_url: lessonVideoUrl.trim(),
			content: lessonContent.trim(),
			estimated_minutes: lessonEstimatedMinutes,
			is_preview: lessonIsPreview
		};
	}

	function lessonMatches(stored: Lesson, intent: ReturnType<typeof lessonDraft>) {
		return (
			stored.title === intent.title &&
			stored.content_type === intent.content_type &&
			stored.video_url === intent.video_url &&
			stored.content === intent.content &&
			stored.estimated_minutes === intent.estimated_minutes &&
			stored.is_preview === intent.is_preview
		);
	}

	// requestKey must be null: with one, PocketBase swaps the caller's signal for its own.
	function lessonWriteOptions(files: File[]) {
		return {
			requestKey: null,
			signal: AbortSignal.timeout(files.length ? UPLOAD_TIMEOUT_MS : WRITE_TIMEOUT_MS)
		};
	}

	// Creates this form's lesson once. A create that may have landed is found by its ID, so
	// pressing Save again can't add a second lesson or send its attachments twice.
	async function createDraftLesson(formData: FormData, files: File[], isCurrent: () => boolean) {
		const id = lessonDraftId;
		const records = pb.collection('lessons');
		formData.append('id', id);
		if (lessonCreateSent) {
			const lesson = await findDraftRecord<Lesson>(records, id);
			if (lesson) return { lesson, readBack: true };
			if (!isCurrent()) return null;
		}

		lessonCreateSent = files;
		try {
			const lesson = await records.create<Lesson>(formData, lessonWriteOptions(files));
			return { lesson, readBack: false };
		} catch (err) {
			if (!mayHaveLanded(err)) {
				if (isCurrent()) lessonCreateSent = null;
				throw err;
			}
			const lesson = await findDraftRecord<Lesson>(records, id);
			if (!lesson) throw err;
			return { lesson, readBack: true };
		}
	}

	const saveLesson = curriculumWrite(async () => {
		if (!lessonTitle.trim() || !showLessonForm || !editingCourse) return;
		const isQuiz = lessonContentType === 'quiz';
		if (isQuiz && !quizReady) {
			toasts.add('error', $t('admin.courses.quiz_read_failed'));
			return;
		}
		if (
			isQuiz &&
			(quizQuestions.length === 0 || quizQuestions.some((q) => !q.question_text.trim()))
		) {
			toasts.add('error', $t('admin.courses.quiz_needs_question'));
			return;
		}

		const courseId = editingCourse.id;
		const moduleId = showLessonForm;
		const existingLesson = editingLesson?.lesson;
		const formRun = lessonFormRun;
		const isCurrent = () => formRun === lessonFormRun;
		const files = Array.from(lessonAttachmentFiles ?? []);
		// This press writes only what's captured here, before anything awaits.
		const lessonIntent = lessonDraft();
		const quizIntent = isQuiz ? quizDraft() : null;
		const quizIdAtPress = quizId;
		// An earlier create that may have landed carried these files, not this press's.
		const sent = lessonCreateSent ?? files;
		let issue: typeof lessonCreateIssue = '';

		saving = true;
		lessonFormLocked = true;
		lessonSavingId = existingLesson?.id ?? lessonDraftId;
		quizSaveIssue = '';
		if (isQuiz) quizDraftHeld = true;
		try {
			const formData = new FormData();
			formData.append('title', lessonIntent.title);
			formData.append('content_type', lessonIntent.content_type);
			formData.append('video_url', lessonIntent.video_url);
			formData.append('content', lessonIntent.content);
			formData.append('estimated_minutes', String(lessonIntent.estimated_minutes));
			formData.append('is_preview', String(lessonIntent.is_preview));
			formData.append(
				'sort_order',
				String(
					editingLesson
						? editingLesson.lesson.sort_order
						: (courseLessons[showLessonForm]?.length || 0) + 1
				)
			);
			formData.append('module', showLessonForm);

			for (const file of files) formData.append('attachments', file);

			let savedLessonId: string;
			if (existingLesson) {
				await pb
					.collection('lessons')
					.update(existingLesson.id, formData, lessonWriteOptions(files));
				savedLessonId = existingLesson.id;
			} else {
				const created = await createDraftLesson(formData, files, isCurrent);
				if (!created) return;
				savedLessonId = created.lesson.id;
				if (isCurrent()) {
					// The next save must update this lesson, and a new lesson can't have a quiz yet.
					editingLesson = { lesson: created.lesson, moduleId };
					quizLessonId = savedLessonId;
				}
				// That came from an earlier press, so only the owner's next Save may replace it.
				const sameFiles =
					files.length === sent.length && files.every((file, i) => file === sent[i]);
				if (created.readBack && !(sameFiles && lessonMatches(created.lesson, lessonIntent))) {
					if (!isCurrent()) toasts.add('error', $t('admin.courses.error_save_lesson'));
					else {
						if (sameFiles) clearLessonFiles();
						issue = 'differs';
					}
					if (editingCourse?.id === courseId) await loadCurriculum(courseId);
					await loadCourses();
					return;
				}
			}

			if (
				quizIntent &&
				!(await saveQuizData(courseId, savedLessonId, quizIntent, quizIdAtPress, isCurrent))
			) {
				if (isCurrent()) {
					// These files are stored now, so a retry shouldn't send them again.
					clearLessonFiles();
					if (!quizSaveIssue) quizSaveIssue = 'partial';
				} else {
					toasts.add('error', $t('admin.courses.error_save_lesson'));
				}
				if (editingCourse?.id === courseId) await loadCurriculum(courseId);
				await loadCourses();
				return;
			}

			toasts.add(
				'success',
				$t(existingLesson ? 'admin.courses.lesson_updated' : 'admin.courses.lesson_created')
			);
			if (isCurrent()) closeLessonForm();
			if (editingCourse?.id === courseId) await loadCurriculum(courseId);
			await loadCourses();
		} catch (err) {
			console.error('Failed to save lesson:', err);
			if (err instanceof DraftReadError && isCurrent()) issue = 'unknown';
			else toasts.add('error', $t('admin.courses.error_save_lesson'));
			// An update that timed out may still have landed, and the rows should show it.
			if (existingLesson && mayHaveLanded(err) && editingCourse?.id === courseId) {
				await loadCurriculum(courseId);
			}
		} finally {
			if (isCurrent()) lessonCreateIssue = issue;
			saving = false;
			lessonFormLocked = false;
			lessonSavingId = null;
		}
	});

	const deleteLesson = curriculumWrite(async (lesson: Lesson) => {
		if (!editingCourse) return;
		// The dialog and the delete both wait, and the open course can change during either.
		const courseId = editingCourse.id;
		const path = `modules/${lesson.module}/lessons/${lesson.id}`;

		const confirmed = await confirm({
			title: $t('admin.courses.delete_lesson'),
			message: $t('admin.courses.delete_lesson_confirm'),
			confirmText: $t('admin.courses.confirm_delete'),
			danger: true
		});
		if (!confirmed) return;

		await deleteCurriculumItem(
			courseId,
			path,
			{
				deleted: 'admin.courses.lesson_deleted',
				failed: 'admin.courses.error_delete_lesson',
				refused: 'admin.courses.error_delete_lesson_attempts'
			},
			() => {
				if (editingLesson?.lesson.id === lesson.id) closeLessonForm();
			}
		);
	});

	const moveLesson = curriculumWrite(
		async (lesson: Lesson, moduleId: string, direction: 'up' | 'down') => {
			if (!editingCourse) return;

			const moduleLessons = courseLessons[moduleId] || [];
			const currentIndex = moduleLessons.findIndex((l) => l.id === lesson.id);
			if (currentIndex === -1) return;
			if (direction === 'up' && currentIndex === 0) return;
			if (direction === 'down' && currentIndex === moduleLessons.length - 1) return;

			const targetIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1;
			const saved = await saveCurriculumOrder(
				'lessons',
				moveRow(moduleLessons, currentIndex, targetIndex),
				editingCourse.id,
				() => courseLessons[moduleId] || []
			);
			if (!saved) return;
			// Announce the new position (1-based) for screen-reader users.
			reorderAnnouncer?.announce(
				$t('admin.courses.lesson_moved_announce', {
					values: { title: lesson.title, position: targetIndex + 1, total: moduleLessons.length }
				})
			);
		}
	);

	// Inline lesson save
	async function saveInlineLesson(lesson: Lesson, fields: Partial<Lesson>) {
		if (!editingCourse) return;
		try {
			await pb.collection('lessons').update(lesson.id, fields);
			await loadCurriculum(editingCourse.id);
		} catch (err) {
			console.error('Failed to save lesson:', err);
			toasts.add('error', $t('admin.courses.error_save_lesson'));
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.courses.page_title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-6xl mx-auto">
	{#if showRecoveryBanner && recoveryData && $listLoad.loaded && !$listLoad.loading && !$listLoad.failed}
		<AutosaveRecoveryBanner
			savedAt={recoveryData.savedAt}
			isEditing={recoveryData.isEditing}
			visible={true}
			on:restore={handleRestoreDraft}
			on:dismiss={handleDismissDraft}
		/>
	{/if}

	{#if selectMode}
		<fieldset class="contents" disabled={listLocked}>
			<BulkActionBar
				selectedCount={selectedIds.size}
				totalCount={filteredCourses.length}
				on:selectAll={selectAll}
				on:clearSelection={clearSelection}
				on:setVisibility={(e) => bulkSetVisibility(e.detail)}
				on:delete={bulkDelete}
				on:cancel={toggleSelectMode}
			/>
		</fieldset>
	{/if}

	<div class="flex items-center justify-between mb-8">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
				{$t('admin.courses.list_title')}
			</h1>
			<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
				{$t('admin.courses.course_count', { values: { count: courses.length } })}
			</p>
		</div>
		<div class="flex items-center gap-2">
			{#if courses.length > 0 && !showForm}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					disabled={$listLoad.mutating}
					onclick={toggleSelectMode}
				>
					{selectMode ? $t('admin.courses.cancel') : $t('admin.courses.select')}
				</button>
			{/if}
			{#if !showForm}
				<button class="btn btn-primary" disabled={$listLoad.failed} onclick={openNewForm}>
					{$t('admin.courses.new_course')}
				</button>
			{/if}
		</div>
	</div>

	{#if !showForm}
		<AdminFilters bind:showAdvanced={showAdvancedFilters} {filterStore} availableTags={[]} />
	{/if}

	<ListLoadStatus
		status={$listLoad}
		label={$t('admin.courses.list_title').toLowerCase()}
		onretry={loadCourses}
	/>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">{$t('admin.courses.loading')}</div>
		</div>
	{:else if showForm}
		<!-- Course Editor - tabbed layout -->
		<form onsubmit={preventDefault(handleSubmit)} oninput={handleFormChange} class="space-y-0">
			<!-- Editor header + tab navigation -->
			<div class="card rounded-b-none border-b-0">
				<div class="flex items-center justify-between px-6 pt-5 pb-4">
					<div>
						<h2 class="text-xl font-bold text-gray-900 dark:text-white">
							{editingCourse ? $t('admin.courses.edit_course') : $t('admin.courses.new_course')}
						</h2>
						{#if editingCourse}
							<p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">/{editingCourse.slug}</p>
						{/if}
					</div>

					<button
						type="button"
						class="p-2 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-800 transition-colors"
						onclick={closeForm}
						aria-label={$t('admin.courses.cancel')}
					>
						<svg
							class="w-5 h-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							aria-hidden="true"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M6 18L18 6M6 6l12 12"
							></path>
						</svg>
					</button>
				</div>
				<nav class="flex gap-0 px-6 -mb-px" aria-label={$t('admin.courses.tab_content')}>
					<button
						type="button"
						onclick={() => (activeTab = 'content')}
						class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
							{activeTab === 'content'
							? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400'
							: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}"
					>
						<span class="flex items-center gap-1.5">
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
								></path>
							</svg>

							{$t('admin.courses.tab_content')}
						</span>
					</button>
					<button
						type="button"
						onclick={() => (activeTab = 'curriculum')}
						class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
							{activeTab === 'curriculum'
							? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400'
							: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}"
					>
						<span class="flex items-center gap-1.5">
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
								></path>
							</svg>

							{$t('admin.courses.tab_curriculum')}
							{#if editingCourse && curriculumCourseId === editingCourse.id}
								<span
									class="ml-1 text-xs px-1.5 py-0.5 rounded-full bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400"
								>
									{courseModules.length}
								</span>
							{/if}
						</span>
					</button>
					<button
						type="button"
						onclick={() => (activeTab = 'settings')}
						class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
							{activeTab === 'settings'
							? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400'
							: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}"
					>
						<span class="flex items-center gap-1.5">
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
								></path>

								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
								></path>
							</svg>

							{$t('admin.courses.tab_settings')}
						</span>
					</button>
					<button
						type="button"
						onclick={() => (activeTab = 'details')}
						class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
							{activeTab === 'details'
							? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400'
							: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}"
					>
						<span class="flex items-center gap-1.5">
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
								></path>
							</svg>

							{$t('admin.courses.section_details')}
						</span>
					</button>
					{#if editingCourse}
						<button
							type="button"
							onclick={() => (activeTab = 'students')}
							class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
								{activeTab === 'students'
								? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400'
								: 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}"
						>
							<span class="flex items-center gap-1.5">
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
									></path>
								</svg>

								{$t('admin.courses.enrolled_students')}
							</span>
						</button>
					{/if}
				</nav>
			</div>

			<!-- Tab panels -->
			<fieldset class="contents" disabled={courseFormLocked}>
				<div class="space-y-6 mt-0">
					<!-- Content tab -->
					{#if activeTab === 'content'}
						<div
							class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4"
						>
							<div>
								<label for="title" class="label">{$t('admin.courses.title')} *</label>
								<input
									type="text"
									id="title"
									bind:value={title}
									class="input"
									placeholder={$t('admin.courses.title_placeholder')}
									required
									onblur={() => !slug && generateSlug()}
								/>
							</div>

							<div>
								<label for="slug" class="label">
									{$t('admin.courses.slug')} *
									<button
										type="button"
										class="text-xs text-primary-700 ml-2"
										onclick={generateSlug}
									>
										{$t('admin.courses.generate_slug')}
									</button>
								</label>
								<input
									type="text"
									id="slug"
									bind:value={slug}
									class="input"
									placeholder={$t('admin.courses.slug_placeholder')}
									required
								/>
								<p class="text-xs text-gray-500 mt-1">
									{$t('admin.courses.slug_preview', {
										values: { slug: slug || 'course-url-slug' }
									})}
								</p>
							</div>

							<div>
								<div class="flex items-center justify-between mb-2">
									<label for="description" class="label mb-0"
										>{$t('admin.courses.description')}</label
									>
									<AIContentHelper
										fieldType="course_description"
										content={description}
										context={{ title, type: 'course_description' }}
										on:apply={(e) => (description = e.detail.content)}
									/>
								</div>
								<textarea
									id="description"
									bind:value={description}
									class="input"
									rows="4"
									placeholder={$t('admin.courses.description_placeholder')}></textarea>
							</div>

							<SingleMediaPicker
								label={$t('admin.courses.cover_image')}
								helpText={$t('admin.courses.cover_image_help')}
								bind:value={coverImageLibraryUrl}
								bind:fileInput={coverImageFile}
								bind:clearExisting={clearCoverImage}
								currentFileUrl={editingCourse?.cover_image_library_url || ''}
								currentFileName={editingCourse?.cover_image || ''}
								imagesOnly={true}
								onchange={handleFormChange}
							/>
						</div>
					{/if}

					<!-- Curriculum tab -->
					{#if activeTab === 'curriculum'}
						<div
							class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4"
						>
							<!-- Polite live region for keyboard module/lesson reorder announcements -->
							<ReorderAnnouncer bind:this={reorderAnnouncer} />
							<div class="flex items-center justify-between">
								<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
									{$t('admin.courses.tab_curriculum')}
								</h2>
								{#if editingCourse}
									<button
										type="button"
										class="btn btn-secondary text-sm"
										disabled={!curriculumReady}
										onclick={openNewModuleForm}
									>
										+ {$t('admin.courses.add_module')}
									</button>
								{/if}
							</div>

							{#if curriculumFailed}
								<div role="alert" class="rounded-lg border border-red-300 dark:border-red-700 p-4">
									<p class="text-sm text-red-700 dark:text-red-300">
										{$t('admin.courses.curriculum_failed')}
									</p>
									<button
										type="button"
										class="btn btn-secondary mt-3"
										disabled={loadingCurriculum}
										onclick={retryCurriculum}
									>
										{$t('admin.dashboard.retry')}
									</button>
								</div>
							{/if}

							{#if !editingCourse}
								<div class="text-center py-6 text-gray-500 dark:text-gray-400 text-sm">
									<p>{$t('admin.courses.save_first')}</p>
									<p class="text-xs mt-1">{$t('admin.courses.save_first_hint')}</p>
								</div>
							{:else if curriculumCourseId !== editingCourse.id}
								{#if loadingCurriculum}
									<div class="py-6 text-center">
										<div class="animate-pulse text-gray-500">
											{$t('admin.courses.loading_curriculum')}
										</div>
									</div>
								{/if}
							{:else}
								<!-- Inline Module Form -->
								{#if showModuleForm}
									<div
										class="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border-2 border-primary-200 dark:border-primary-800"
									>
										{#if moduleCreateIssue}
											<div
												role="alert"
												class="rounded-lg border border-red-300 dark:border-red-700 p-3 mb-3"
											>
												<p class="text-sm text-red-700 dark:text-red-300">
													{$t(
														moduleCreateIssue === 'unknown'
															? 'admin.courses.module_create_unknown'
															: 'admin.courses.module_create_differs'
													)}
												</p>
											</div>
										{/if}
										<div class="flex items-center justify-between mb-3">
											<h4 class="text-sm font-medium text-gray-900 dark:text-white">
												{editingModule
													? $t('admin.courses.edit_module')
													: $t('admin.courses.add_module')}
											</h4>

											<button
												type="button"
												class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
												onclick={closeModuleForm}
												aria-label={$t('admin.courses.cancel')}
											>
												<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="2"
														d="M6 18L18 6M6 6l12 12"
													></path>
												</svg>
											</button>
										</div>
										<div class="space-y-3">
											<div>
												<label for="module_title" class="label text-sm"
													>{$t('admin.courses.module_title')}</label
												>
												<input
													type="text"
													id="module_title"
													bind:value={moduleTitle}
													disabled={moduleFormLocked}
													class="input"
													placeholder={$t('admin.courses.module_title_placeholder')}
												/>
											</div>
											<div>
												<div class="flex items-center justify-between mb-2">
													<label for="module_description" class="label text-sm mb-0"
														>{$t('admin.courses.module_description')}</label
													>
													<AIContentHelper
														fieldType="module_description"
														content={moduleDescription}
														context={{
															title: moduleTitle,
															parent: title,
															type: 'module_description'
														}}
														on:apply={(e) => (moduleDescription = e.detail.content)}
													/>
												</div>
												<textarea
													id="module_description"
													bind:value={moduleDescription}
													disabled={moduleFormLocked}
													class="input"
													rows="3"></textarea>
											</div>
											<!-- Module Drip Settings -->
											<div
												class="border-t border-gray-200 dark:border-gray-700 pt-3 mt-3 space-y-2"
											>
												<p class="text-xs font-medium text-gray-700 dark:text-gray-300">
													{$t('admin.courses.drip_schedule')}
												</p>
												<div class="grid grid-cols-2 gap-3">
													<div>
														<label for="module_drip_days" class="label text-xs"
															>{$t('admin.courses.module_drip_days')}</label
														>
														<input
															type="number"
															id="module_drip_days"
															bind:value={moduleDripDaysAfterEnrollment}
															disabled={moduleFormLocked}
															class="input text-sm"
															placeholder="0"
															min="0"
														/>
													</div>
													<div>
														<label for="module_drip_date" class="label text-xs"
															>{$t('admin.courses.module_drip_date')}</label
														>
														<input
															type="date"
															id="module_drip_date"
															bind:value={moduleDripUnlockDate}
															disabled={moduleFormLocked}
															class="input text-sm"
														/>
													</div>
												</div>
												<p class="text-xs text-gray-600 dark:text-gray-400">
													{$t('admin.courses.module_drip_helper')}
												</p>
											</div>
											<div class="flex gap-2 justify-end">
												<button
													type="button"
													class="btn btn-ghost text-sm"
													onclick={closeModuleForm}>{$t('admin.courses.cancel')}</button
												>
												<button
													type="button"
													class="btn btn-primary text-sm"
													onclick={saveModule}
													disabled={saving || !moduleTitle.trim() || !curriculumReady}
												>
													{saving
														? $t('admin.courses.saving')
														: moduleCreateIssue === 'unknown'
															? $t('admin.dashboard.retry')
															: $t('admin.courses.save_module')}
												</button>
											</div>
										</div>
									</div>
								{/if}

								{#if courseModules.length === 0 && !showModuleForm}
									<div class="text-center py-8 text-gray-500 dark:text-gray-400">
										<svg
											class="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600 mb-3"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
											></path>
										</svg>
										<p class="text-sm">{$t('admin.courses.no_modules')}</p>
										<button
											type="button"
											class="btn btn-secondary text-sm mt-3"
											disabled={!curriculumReady}
											onclick={openNewModuleForm}
										>
											+ {$t('admin.courses.add_module')}
										</button>
									</div>
								{:else}
									<div class="space-y-4">
										{#each courseModules as module, moduleIndex (module.id)}
											<div
												class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden"
											>
												<!-- Module Header -->
												<div
													class="flex items-center gap-3 px-4 py-3 bg-gray-50 dark:bg-gray-800/50"
												>
													<button
														type="button"
														class="flex-1 text-left flex items-center gap-2"
														onclick={() => toggleModuleExpand(module.id)}
													>
														<svg
															class="w-4 h-4 text-gray-400 transition-transform {expandedModules.has(
																module.id
															)
																? 'rotate-90'
																: ''}"
															fill="none"
															viewBox="0 0 24 24"
															stroke="currentColor"
														>
															<path
																stroke-linecap="round"
																stroke-linejoin="round"
																stroke-width="2"
																d="M9 5l7 7-7 7"
															></path>
														</svg>
														<span class="font-medium text-gray-900 dark:text-white text-sm"
															>{module.title}</span
														>
														<span class="text-xs text-gray-500">
															{courseLessons[module.id]?.length || 0}
															{$t('admin.courses.lessons').toLowerCase()}
														</span>
														{#if module.drip_days_after_enrollment}
															<span
																class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400"
															>
																<svg
																	class="w-3 h-3"
																	fill="none"
																	viewBox="0 0 24 24"
																	stroke="currentColor"
																>
																	<path
																		stroke-linecap="round"
																		stroke-linejoin="round"
																		stroke-width="2"
																		d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
																	></path>
																</svg>

																{module.drip_days_after_enrollment}d
															</span>
														{:else if module.drip_unlock_date}
															<span
																class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400"
															>
																<svg
																	class="w-3 h-3"
																	fill="none"
																	viewBox="0 0 24 24"
																	stroke="currentColor"
																>
																	<path
																		stroke-linecap="round"
																		stroke-linejoin="round"
																		stroke-width="2"
																		d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
																	></path>
																</svg>

																drip
															</span>
														{/if}
													</button>
													<div class="flex items-center gap-1">
														<button
															type="button"
															class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30"
															onclick={() => moveModule(module, 'up')}
															disabled={moduleIndex === 0 || !curriculumReady}
															aria-label={$t('admin.courses.reorder_modules')}
														>
															<svg
																class="w-4 h-4"
																fill="none"
																viewBox="0 0 24 24"
																stroke="currentColor"
															>
																<path
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	stroke-width="2"
																	d="M5 15l7-7 7 7"
																></path>
															</svg>
														</button>

														<button
															type="button"
															class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30"
															onclick={() => moveModule(module, 'down')}
															disabled={moduleIndex === courseModules.length - 1 ||
																!curriculumReady}
															aria-label={$t('admin.courses.reorder_modules')}
														>
															<svg
																class="w-4 h-4"
																fill="none"
																viewBox="0 0 24 24"
																stroke="currentColor"
															>
																<path
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	stroke-width="2"
																	d="M19 9l-7 7-7-7"
																></path>
															</svg>
														</button>

														<button
															type="button"
															class="p-1 text-gray-400 hover:text-primary-600 dark:hover:text-primary-400"
															onclick={() => openEditModuleForm(module)}
															disabled={!curriculumReady || module.id === moduleSavingId}
															aria-label={$t('admin.courses.edit_module')}
														>
															<svg
																class="w-4 h-4"
																fill="none"
																viewBox="0 0 24 24"
																stroke="currentColor"
															>
																<path
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	stroke-width="2"
																	d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
																></path>
															</svg>
														</button>

														<button
															type="button"
															class="p-1 text-gray-400 hover:text-red-600 dark:hover:text-red-400"
															onclick={() => deleteModule(module)}
															disabled={!curriculumReady}
															aria-label={$t('admin.courses.delete_module')}
														>
															<svg
																class="w-4 h-4"
																fill="none"
																viewBox="0 0 24 24"
																stroke="currentColor"
															>
																<path
																	stroke-linecap="round"
																	stroke-linejoin="round"
																	stroke-width="2"
																	d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
																></path>
															</svg>
														</button>
													</div>
												</div>

												<!-- Module Content (Lessons) -->
												{#if expandedModules.has(module.id)}
													<div class="px-4 py-3 space-y-2">
														{#if module.description}
															<p class="text-xs text-gray-500 dark:text-gray-400 mb-2">
																{module.description}
															</p>
														{/if}

														<!-- Lesson List -->
														{#each courseLessons[module.id] || [] as lesson, lessonIndex (lesson.id)}
															<div class="border border-gray-100 dark:border-gray-700 rounded-lg">
																<!-- Lesson Header -->
																<div class="flex items-center gap-3 px-3 py-2">
																	<button
																		type="button"
																		class="flex-1 text-left flex items-center gap-2"
																		onclick={() => toggleLessonExpand(lesson.id)}
																	>
																		<svg
																			class="w-3 h-3 text-gray-400 transition-transform {expandedLessons.has(
																				lesson.id
																			)
																				? 'rotate-90'
																				: ''}"
																			fill="none"
																			viewBox="0 0 24 24"
																			stroke="currentColor"
																		>
																			<path
																				stroke-linecap="round"
																				stroke-linejoin="round"
																				stroke-width="2"
																				d="M9 5l7 7-7 7"
																			></path>
																		</svg>
																		<span class="text-sm text-gray-900 dark:text-white"
																			>{lesson.title}</span
																		>
																		<span
																			class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium
																{lesson.content_type === 'video'
																				? 'bg-sky-100 dark:bg-sky-900/30 text-sky-700 dark:text-sky-400'
																				: lesson.content_type === 'text'
																					? 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400'
																					: lesson.content_type === 'quiz'
																						? 'bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-400'
																						: 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400'}"
																		>
																			{$t('admin.courses.lesson_type_' + lesson.content_type)}
																		</span>
																		{#if lesson.estimated_minutes}
																			<span class="text-xs text-gray-500"
																				>{lesson.estimated_minutes}m</span
																			>
																		{/if}
																		{#if lesson.is_preview}
																			<span class="text-xs text-primary-700 dark:text-primary-400"
																				>{$t('admin.courses.preview_badge')}</span
																			>
																		{/if}
																	</button>
																	<div class="flex items-center gap-1">
																		<button
																			type="button"
																			class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30"
																			onclick={() => moveLesson(lesson, module.id, 'up')}
																			disabled={lessonIndex === 0 || !curriculumReady}
																			aria-label={$t('admin.courses.reorder_lessons')}
																		>
																			<svg
																				class="w-3 h-3"
																				fill="none"
																				viewBox="0 0 24 24"
																				stroke="currentColor"
																			>
																				<path
																					stroke-linecap="round"
																					stroke-linejoin="round"
																					stroke-width="2"
																					d="M5 15l7-7 7 7"
																				></path>
																			</svg>
																		</button>

																		<button
																			type="button"
																			class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30"
																			onclick={() => moveLesson(lesson, module.id, 'down')}
																			disabled={lessonIndex ===
																				(courseLessons[module.id]?.length || 0) - 1 ||
																				!curriculumReady}
																			aria-label={$t('admin.courses.reorder_lessons')}
																		>
																			<svg
																				class="w-3 h-3"
																				fill="none"
																				viewBox="0 0 24 24"
																				stroke="currentColor"
																			>
																				<path
																					stroke-linecap="round"
																					stroke-linejoin="round"
																					stroke-width="2"
																					d="M19 9l-7 7-7-7"
																				></path>
																			</svg>
																		</button>

																		<button
																			type="button"
																			class="p-1 text-gray-400 hover:text-primary-600 dark:hover:text-primary-400 disabled:opacity-30"
																			onclick={() => openEditLessonForm(lesson, module.id)}
																			disabled={!curriculumReady || lesson.id === lessonSavingId}
																			aria-label={$t('admin.courses.edit_lesson')}
																		>
																			<svg
																				class="w-3 h-3"
																				fill="none"
																				viewBox="0 0 24 24"
																				stroke="currentColor"
																			>
																				<path
																					stroke-linecap="round"
																					stroke-linejoin="round"
																					stroke-width="2"
																					d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
																				></path>
																			</svg>
																		</button>

																		<button
																			type="button"
																			class="p-1 text-gray-400 hover:text-red-600 dark:hover:text-red-400"
																			onclick={() => deleteLesson(lesson)}
																			disabled={!curriculumReady}
																			aria-label={$t('admin.courses.delete_lesson')}
																		>
																			<svg
																				class="w-3 h-3"
																				fill="none"
																				viewBox="0 0 24 24"
																				stroke="currentColor"
																			>
																				<path
																					stroke-linecap="round"
																					stroke-linejoin="round"
																					stroke-width="2"
																					d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
																				></path>
																			</svg>
																		</button>
																	</div>
																</div>

																<!-- Lesson Content (expanded inline) -->
																{#if expandedLessons.has(lesson.id)}
																	<div
																		class="px-3 pb-3 pt-1 border-t border-gray-100 dark:border-gray-700 space-y-3"
																	>
																		{#if lesson.video_url}
																			<div>
																				<span class="label text-xs"
																					>{$t('admin.courses.lesson_video_url')}</span
																				>
																				<p
																					class="text-sm text-gray-600 dark:text-gray-400 break-all"
																				>
																					{lesson.video_url}
																				</p>
																			</div>
																		{/if}
																		{#if lesson.content}
																			<div>
																				<span class="label text-xs"
																					>{$t('admin.courses.lesson_content')}</span
																				>
																				<div
																					class="text-sm text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-900/30 rounded p-2 max-h-32 overflow-y-auto font-mono text-xs whitespace-pre-wrap"
																				>
																					{lesson.content}
																				</div>
																			</div>
																		{/if}
																		{#if !lesson.video_url && !lesson.content}
																			<p class="text-xs text-gray-400 italic">
																				{$t('admin.courses.no_content_yet')}
																			</p>
																		{/if}
																	</div>
																{/if}
															</div>
														{/each}

														<!-- New Lesson Form (inline) -->
														{#if showLessonForm === module.id}
															<div
																class="bg-primary-50/50 dark:bg-primary-900/10 rounded-lg p-3 border border-primary-200 dark:border-primary-800 space-y-3"
															>
																<div class="flex items-center justify-between">
																	<h5 class="text-sm font-medium text-gray-900 dark:text-white">
																		{editingLesson
																			? $t('admin.courses.edit_lesson')
																			: $t('admin.courses.add_lesson')}
																	</h5>

																	<button
																		type="button"
																		class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
																		onclick={closeLessonForm}
																		aria-label={$t('admin.courses.cancel')}
																	>
																		<svg
																			class="w-4 h-4"
																			fill="none"
																			viewBox="0 0 24 24"
																			stroke="currentColor"
																		>
																			<path
																				stroke-linecap="round"
																				stroke-linejoin="round"
																				stroke-width="2"
																				d="M6 18L18 6M6 6l12 12"
																			></path>
																		</svg>
																	</button>
																</div>
																<fieldset class="min-w-0 space-y-3" disabled={lessonFormLocked}>
																	<div>
																		<label for="lesson_title" class="label text-xs"
																			>{$t('admin.courses.lesson_title')} *</label
																		>
																		<input
																			type="text"
																			id="lesson_title"
																			bind:value={lessonTitle}
																			class="input text-sm"
																			placeholder={$t('admin.courses.lesson_title_placeholder')}
																		/>
																	</div>
																	<div class="grid grid-cols-2 gap-3">
																		<div>
																			<label for="lesson_content_type" class="label text-xs"
																				>{$t('admin.courses.lesson_content_type')}</label
																			>
																			<select
																				id="lesson_content_type"
																				bind:value={lessonContentType}
																				class="input text-sm"
																			>
																				<option value="video"
																					>{$t('admin.courses.lesson_type_video')}</option
																				>
																				<option value="text"
																					>{$t('admin.courses.lesson_type_text')}</option
																				>
																				<option value="mixed"
																					>{$t('admin.courses.lesson_type_mixed')}</option
																				>
																				<option value="quiz"
																					>{$t('admin.courses.lesson_type_quiz')}</option
																				>
																			</select>
																		</div>
																		<div>
																			<label for="lesson_minutes" class="label text-xs"
																				>{$t('admin.courses.lesson_estimated_minutes')}</label
																			>
																			<input
																				type="number"
																				id="lesson_minutes"
																				bind:value={lessonEstimatedMinutes}
																				class="input text-sm"
																				min="0"
																			/>
																		</div>
																	</div>
																	{#if lessonContentType === 'video' || lessonContentType === 'mixed'}
																		<div>
																			<label for="lesson_video_url" class="label text-xs"
																				>{$t('admin.courses.lesson_video_url')}</label
																			>
																			<input
																				type="url"
																				id="lesson_video_url"
																				bind:value={lessonVideoUrl}
																				class="input text-sm"
																				placeholder={$t(
																					'admin.courses.lesson_video_url_placeholder'
																				)}
																			/>
																			{#if lessonVideoUrl && !lessonVideoUrl.match(/(?:youtube\.com\/watch\?v=|youtu\.be\/|vimeo\.com\/\d+)/)}
																				<p class="text-xs text-amber-600 dark:text-amber-400 mt-1">
																					{$t('admin.courses.video_url_warning')}
																				</p>
																			{/if}
																		</div>
																	{/if}
																	{#if lessonContentType === 'text' || lessonContentType === 'mixed'}
																		<div>
																			<div class="flex items-center justify-between mb-1">
																				<label for="lesson_content" class="label text-xs"
																					>{$t('admin.courses.lesson_content')} (Markdown)</label
																				>
																				<button
																					type="button"
																					class="text-xs text-primary-700 dark:text-primary-400 hover:underline"
																					onclick={() => (lessonPreviewMode = !lessonPreviewMode)}
																				>
																					{lessonPreviewMode
																						? $t('admin.courses.edit_mode')
																						: $t('admin.courses.preview_mode')}
																				</button>
																			</div>
																			{#if lessonPreviewMode}
																				<div
																					class="prose dark:prose-invert max-w-none text-sm bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700 min-h-[150px]"
																				>
																					{#if lessonContent.trim()}
																						{@html DOMPurify.sanitize(lessonContent)}
																					{:else}
																						<p class="text-gray-400 italic">
																							{$t('admin.courses.no_content_preview')}
																						</p>
																					{/if}
																				</div>
																			{:else}
																				<div class="flex justify-end mb-2">
																					<AIContentHelper
																						fieldType="lesson_content"
																						content={lessonContent}
																						context={{
																							title: lessonTitle,
																							parent: moduleTitle,
																							type: 'lesson_content'
																						}}
																						on:apply={(e) => (lessonContent = e.detail.content)}
																					/>
																				</div>
																				<textarea
																					id="lesson_content"
																					bind:value={lessonContent}
																					class="input text-sm"
																					rows="8"
																					placeholder={$t('admin.courses.description_placeholder')}
																				></textarea>
																			{/if}
																		</div>
																	{/if}
																	{#if lessonContentType === 'quiz'}
																		<!-- Quiz Builder -->
																		<div
																			class="space-y-4 border border-gray-200 dark:border-gray-700 rounded-lg p-4 bg-gray-50 dark:bg-gray-800/50"
																		>
																			{#if quizFailed}
																				<div
																					role="alert"
																					class="rounded-lg border border-red-300 dark:border-red-700 p-3"
																				>
																					<p class="text-sm text-red-700 dark:text-red-300">
																						{$t('admin.courses.quiz_read_failed')}
																					</p>
																					<button
																						type="button"
																						class="btn btn-secondary mt-2 text-sm"
																						disabled={loadingQuiz}
																						onclick={retryQuiz}
																					>
																						{$t('admin.dashboard.retry')}
																					</button>
																				</div>
																			{/if}
																			{#if quizSaveIssue}
																				<div
																					role="alert"
																					class="rounded-lg border border-red-300 dark:border-red-700 p-3"
																				>
																					<p class="text-sm text-red-700 dark:text-red-300">
																						{$t(
																							quizSaveIssue === 'conflict'
																								? 'admin.courses.quiz_save_conflict'
																								: 'admin.courses.quiz_save_partial'
																						)}
																					</p>
																				</div>
																			{/if}
																			{#if loadingQuiz && !quizEditable}
																				<div class="flex items-center justify-center py-6">
																					<svg
																						class="animate-spin h-5 w-5 text-primary-700"
																						fill="none"
																						viewBox="0 0 24 24"
																					>
																						<circle
																							class="opacity-25"
																							cx="12"
																							cy="12"
																							r="10"
																							stroke="currentColor"
																							stroke-width="4"
																						></circle>
																						<path
																							class="opacity-75"
																							fill="currentColor"
																							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
																						></path>
																					</svg>
																				</div>
																			{:else if quizEditable}
																				<h4
																					class="text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2"
																				>
																					<svg
																						class="w-4 h-4"
																						fill="none"
																						viewBox="0 0 24 24"
																						stroke="currentColor"
																					>
																						<path
																							stroke-linecap="round"
																							stroke-linejoin="round"
																							stroke-width="2"
																							d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
																						></path>
																					</svg>
																					{$t('admin.courses.quiz_title')}
																				</h4>
																				<div>
																					<label for="quiz_title" class="label text-xs"
																						>{$t('admin.courses.quiz_title')}</label
																					>
																					<input
																						type="text"
																						id="quiz_title"
																						bind:value={quizTitle}
																						class="input text-sm"
																						placeholder={lessonTitle || 'Quiz title...'}
																					/>
																				</div>
																				<div class="grid grid-cols-3 gap-3">
																					<div>
																						<label for="quiz_passing_score" class="label text-xs"
																							>{$t('admin.courses.quiz_passing_score')}</label
																						>
																						<div class="flex items-center gap-1">
																							<input
																								type="number"
																								id="quiz_passing_score"
																								bind:value={quizPassingScore}
																								class="input text-sm"
																								min="0"
																								max="100"
																							/>
																							<span class="text-xs text-gray-500">%</span>
																						</div>
																					</div>
																					<div>
																						<label for="quiz_max_attempts" class="label text-xs"
																							>{$t('admin.courses.quiz_max_attempts')}</label
																						>
																						<input
																							type="number"
																							id="quiz_max_attempts"
																							bind:value={quizMaxAttempts}
																							class="input text-sm"
																							min="0"
																						/>
																					</div>
																					<div class="flex items-end pb-1">
																						<Toggle
																							bind:checked={quizRandomize}
																							id="quiz_randomize"
																							label={$t('admin.courses.quiz_randomize')}
																						/>
																					</div>
																				</div>

																				<!-- Questions List -->
																				<div class="space-y-3">
																					{#each quizQuestions as question, qIndex}
																						<div
																							class="border border-gray-200 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 p-3 space-y-3"
																						>
																							<div class="flex items-center justify-between">
																								<span
																									class="text-xs font-medium text-gray-500 dark:text-gray-400"
																									>{$t('admin.courses.quiz_question_text')}
																									{qIndex + 1}</span
																								>
																								<div class="flex items-center gap-1">
																									<button
																										type="button"
																										class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30"
																										disabled={qIndex === 0}
																										onclick={() => moveQuizQuestion(qIndex, -1)}
																										title={$t('admin.courses.reorder_modules')}
																									>
																										<svg
																											class="w-3.5 h-3.5"
																											fill="none"
																											viewBox="0 0 24 24"
																											stroke="currentColor"
																										>
																											<path
																												stroke-linecap="round"
																												stroke-linejoin="round"
																												stroke-width="2"
																												d="M5 15l7-7 7 7"
																											></path>
																										</svg>
																									</button>

																									<button
																										type="button"
																										class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30"
																										disabled={qIndex === quizQuestions.length - 1}
																										onclick={() => moveQuizQuestion(qIndex, 1)}
																										title={$t('admin.courses.reorder_modules')}
																									>
																										<svg
																											class="w-3.5 h-3.5"
																											fill="none"
																											viewBox="0 0 24 24"
																											stroke="currentColor"
																										>
																											<path
																												stroke-linecap="round"
																												stroke-linejoin="round"
																												stroke-width="2"
																												d="M19 9l-7 7-7-7"
																											></path>
																										</svg>
																									</button>

																									<button
																										type="button"
																										class="p-1 text-red-400 hover:text-red-600"
																										onclick={() => removeQuizQuestion(qIndex)}
																										title={$t('admin.courses.delete')}
																									>
																										<svg
																											class="w-3.5 h-3.5"
																											fill="none"
																											viewBox="0 0 24 24"
																											stroke="currentColor"
																										>
																											<path
																												stroke-linecap="round"
																												stroke-linejoin="round"
																												stroke-width="2"
																												d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
																											></path>
																										</svg>
																									</button>
																								</div>
																							</div>

																							<div class="grid grid-cols-2 gap-3">
																								<label class="block">
																									<span class="label text-xs"
																										>{$t('admin.courses.quiz_question_type')}</span
																									>
																									<select
																										class="input text-sm"
																										value={question.question_type}
																										onchange={(e) =>
																											setQuestionType(
																												qIndex,
																												(e.target as HTMLSelectElement).value as
																													'multiple_choice' | 'true_false'
																											)}
																									>
																										<option value="multiple_choice"
																											>{$t(
																												'admin.courses.quiz_multiple_choice'
																											)}</option
																										>
																										<option value="true_false"
																											>{$t('admin.courses.quiz_true_false')}</option
																										>
																									</select>
																								</label>
																								<label class="block">
																									<span class="label text-xs"
																										>{$t('admin.courses.quiz_points')}</span
																									>
																									<input
																										type="number"
																										bind:value={question.points}
																										class="input text-sm"
																										min="0"
																									/>
																								</label>
																							</div>

																							<div>
																								<label
																									for="quiz-question-{qIndex}"
																									class="label text-xs"
																									>{$t('admin.courses.quiz_question_text')}</label
																								>
																								<textarea
																									id="quiz-question-{qIndex}"
																									bind:value={question.question_text}
																									class="input text-sm"
																									rows="2"
																									placeholder={$t(
																										'admin.courses.quiz_question_text'
																									)}></textarea>
																							</div>

																							<!-- Options -->
																							<div class="space-y-2">
																								<span
																									class="text-xs font-medium text-gray-500 dark:text-gray-400"
																									>{$t('admin.courses.quiz_correct_answer')}</span
																								>
																								{#each question.options as option, oIndex}
																									<div class="flex items-center gap-2">
																										<input
																											type="radio"
																											name="correct_{qIndex}"
																											checked={option.is_correct}
																											onchange={() =>
																												setCorrectOption(qIndex, oIndex)}
																											class="text-primary-700 focus:ring-primary-500"
																										/>
																										{#if question.question_type === 'true_false'}
																											<span
																												class="text-sm text-gray-700 dark:text-gray-300"
																												>{option.text}</span
																											>
																										{:else}
																											<input
																												type="text"
																												bind:value={option.text}
																												class="input text-sm flex-1"
																												placeholder="Option {oIndex + 1}"
																											/>
																											{#if question.options.length > 2}
																												<button
																													type="button"
																													class="p-1 text-gray-400 hover:text-red-500"
																													onclick={() =>
																														removeQuizOption(qIndex, oIndex)}
																													title={$t('admin.courses.delete')}
																												>
																													<svg
																														class="w-3.5 h-3.5"
																														fill="none"
																														viewBox="0 0 24 24"
																														stroke="currentColor"
																													>
																														<path
																															stroke-linecap="round"
																															stroke-linejoin="round"
																															stroke-width="2"
																															d="M6 18L18 6M6 6l12 12"
																														></path>
																													</svg>
																												</button>
																											{/if}
																										{/if}
																									</div>
																								{/each}
																								{#if question.question_type === 'multiple_choice' && question.options.length < 6}
																									<button
																										type="button"
																										class="text-xs text-primary-700 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
																										onclick={() => addQuizOption(qIndex)}
																									>
																										+ {$t('admin.courses.quiz_add_option')}
																									</button>
																								{/if}
																							</div>

																							<label class="block">
																								<span class="label text-xs"
																									>{$t('admin.courses.quiz_explanation')}</span
																								>
																								<textarea
																									bind:value={question.explanation}
																									class="input text-sm"
																									rows="2"
																									placeholder={$t('admin.courses.quiz_explanation')}
																								></textarea>
																							</label>
																						</div>
																					{/each}
																				</div>

																				<button
																					type="button"
																					class="w-full text-left text-sm text-primary-700 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 py-2 px-3 rounded hover:bg-primary-50 dark:hover:bg-primary-900/10 transition-colors border border-dashed border-gray-300 dark:border-gray-600"
																					onclick={addQuizQuestion}
																				>
																					+ {$t('admin.courses.quiz_add_question')}
																				</button>
																			{/if}
																		</div>
																	{/if}
																	<Toggle
																		bind:checked={lessonIsPreview}
																		id="lesson_is_preview"
																		label={$t('admin.courses.lesson_is_preview')}
																	/>
																	<!-- Attachments -->
																	<div>
																		<label for="lesson_attachments" class="label text-xs"
																			>{$t('admin.courses.attachments_label')}</label
																		>
																		<input
																			type="file"
																			id="lesson_attachments"
																			bind:this={lessonFileInput}
																			multiple
																			accept=".pdf,.pptx,.ppt,.zip,.tar.gz,.doc,.docx,.xls,.xlsx,.txt,.md,.csv"
																			onchange={(e) => {
																				lessonAttachmentFiles = (e.target as HTMLInputElement)
																					.files;
																			}}
																			class="block w-full text-sm text-gray-500 dark:text-gray-400
																file:mr-4 file:py-2 file:px-4
																file:rounded-md file:border-0
																file:text-sm file:font-medium
																file:bg-primary-50 file:text-primary-700
																dark:file:bg-primary-900/20 dark:file:text-primary-400
																hover:file:bg-primary-100 dark:hover:file:bg-primary-900/30
																file:cursor-pointer"
																		/>
																		{#if editingLesson?.lesson?.id}
																			{@const existingAttachments =
																				(editingLesson.lesson as any).attachments || []}
																			{#if existingAttachments.length > 0}
																				<div class="mt-2 space-y-1">
																					<span class="text-xs text-gray-500 dark:text-gray-400"
																						>{$t('admin.courses.attachments_label')}:</span
																					>
																					{#each existingAttachments as attachment}
																						<div
																							class="flex items-center gap-2 text-xs text-gray-600 dark:text-gray-400"
																						>
																							<svg
																								class="w-3.5 h-3.5"
																								fill="none"
																								viewBox="0 0 24 24"
																								stroke="currentColor"
																							>
																								<path
																									stroke-linecap="round"
																									stroke-linejoin="round"
																									stroke-width="2"
																									d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13"
																								></path>
																							</svg>
																							<span>{attachment}</span>
																						</div>
																					{/each}
																				</div>
																			{/if}
																		{/if}
																	</div>
																</fieldset>
																{#if lessonCreateIssue}
																	<div
																		role="alert"
																		class="rounded-lg border border-red-300 dark:border-red-700 p-3"
																	>
																		<p class="text-sm text-red-700 dark:text-red-300">
																			{$t(
																				lessonCreateIssue === 'unknown'
																					? 'admin.courses.lesson_create_unknown'
																					: 'admin.courses.lesson_create_differs'
																			)}
																		</p>
																		{#if lessonCreateIssue === 'unknown'}
																			<button
																				type="button"
																				class="btn btn-secondary mt-2 text-sm"
																				disabled={saving}
																				onclick={saveLesson}
																			>
																				{$t('admin.dashboard.retry')}
																			</button>
																		{/if}
																	</div>
																{/if}
																<div class="flex gap-2 justify-end">
																	<button
																		type="button"
																		class="btn btn-ghost text-sm"
																		onclick={closeLessonForm}>{$t('admin.courses.cancel')}</button
																	>
																	<button
																		type="button"
																		class="btn btn-primary text-sm"
																		onclick={saveLesson}
																		disabled={saving ||
																			!lessonTitle.trim() ||
																			!curriculumReady ||
																			lessonCreateIssue === 'unknown' ||
																			(lessonContentType === 'quiz' && !quizReady)}
																	>
																		{saving
																			? $t('admin.courses.saving')
																			: $t('admin.courses.save_lesson')}
																	</button>
																</div>
															</div>
														{:else}
															<button
																type="button"
																class="w-full text-left text-sm text-primary-700 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 py-2 px-3 rounded hover:bg-primary-50 dark:hover:bg-primary-900/10 transition-colors"
																onclick={() => openNewLessonForm(module.id)}
															>
																+ {$t('admin.courses.add_lesson')}
															</button>
														{/if}
													</div>
												{/if}
											</div>
										{/each}
									</div>
								{/if}
							{/if}
						</div>
					{/if}

					<!-- Students tab (enrollment dashboard) -->
					{#if activeTab === 'students' && editingCourse}
						<div
							class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4"
						>
							<div class="flex items-center justify-between">
								<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
									{$t('admin.courses.enrolled_students')}
								</h2>
								<button
									type="button"
									class="btn btn-secondary btn-sm"
									onclick={() => {
										showStudents = !showStudents;

										if (!showStudents) return;

										loadStudents(editingCourse!.id);
									}}>{$t('admin.courses.show_hide_enrollment')}</button
								>
							</div>

							{#if showStudents}
								{#if studentsFailed}
									<div
										role="alert"
										class="rounded-lg border border-red-300 dark:border-red-700 p-4"
									>
										<p class="text-sm text-red-700 dark:text-red-300">
											{$t('admin.courses.failed_load_enrollment')}
										</p>
										<button
											type="button"
											class="btn btn-secondary mt-3"
											disabled={loadingStudents}
											onclick={() => loadStudents(editingCourse!.id)}
										>
											{$t('admin.dashboard.retry')}
										</button>
									</div>
								{/if}
								{#if loadingStudents}
									<div class="flex items-center justify-center py-8">
										<svg
											class="animate-spin h-6 w-6 text-primary-700"
											fill="none"
											viewBox="0 0 24 24"
										>
											<circle
												class="opacity-25"
												cx="12"
												cy="12"
												r="10"
												stroke="currentColor"
												stroke-width="4"
											></circle>
											<path
												class="opacity-75"
												fill="currentColor"
												d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
											></path>
										</svg>
									</div>
								{:else if studentsCourseId === editingCourse.id}
									<!-- Stats -->
									{#if enrollmentStats}
										<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
											<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 text-center">
												<div class="text-2xl font-bold text-gray-900 dark:text-white">
													{enrollmentStats.total_enrolled}
												</div>
												<div class="text-xs text-gray-500 dark:text-gray-400">
													{$t('admin.courses.stat_enrolled')}
												</div>
											</div>
											<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 text-center">
												<div class="text-2xl font-bold text-green-600 dark:text-green-400">
													{enrollmentStats.completed}
												</div>
												<div class="text-xs text-gray-500 dark:text-gray-400">
													{$t('admin.courses.stat_completed')}
												</div>
											</div>
											<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 text-center">
												<div class="text-2xl font-bold text-primary-700 dark:text-primary-400">
													{Math.round(enrollmentStats.average_progress)}%
												</div>
												<div class="text-xs text-gray-500 dark:text-gray-400">
													{$t('admin.courses.stat_avg_progress')}
												</div>
											</div>
											<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 text-center">
												<div class="text-2xl font-bold text-purple-600 dark:text-purple-400">
													{Math.round(enrollmentStats.completion_rate)}%
												</div>
												<div class="text-xs text-gray-500 dark:text-gray-400">
													{$t('admin.courses.stat_completion_rate')}
												</div>
											</div>
										</div>
									{/if}

									<!-- Student table -->
									{#if students.length > 0}
										<div class="overflow-x-auto">
											<table class="w-full text-sm">
												<thead>
													<tr class="border-b border-gray-200 dark:border-gray-700">
														<th
															class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400"
															>{$t('admin.courses.col_email')}</th
														>
														<th
															class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400"
															>{$t('admin.courses.col_progress')}</th
														>
														<th
															class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400"
															>{$t('admin.courses.col_lessons')}</th
														>
														<th
															class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400"
															>{$t('admin.courses.col_last_active')}</th
														>
														<th
															class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400"
															>{$t('admin.courses.col_certificate')}</th
														>
													</tr>
												</thead>
												<tbody>
													{#each students as student}
														<tr class="border-b border-gray-100 dark:border-gray-800">
															<td class="py-2 px-3 text-gray-900 dark:text-gray-100"
																>{student.email}</td
															>
															<td class="py-2 px-3">
																<div class="flex items-center gap-2">
																	<div class="w-20 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
																		<div
																			class="h-2 rounded-full transition-all {student.progress_percent >=
																			100
																				? 'bg-green-500'
																				: 'bg-primary-600'}"
																			style="width: {Math.min(student.progress_percent, 100)}%"
																		></div>
																	</div>
																	<span class="text-xs text-gray-500 dark:text-gray-400"
																		>{Math.round(student.progress_percent)}%</span
																	>
																</div>
															</td>
															<td class="py-2 px-3 text-gray-600 dark:text-gray-400"
																>{student.lessons_completed}/{student.total_lessons}</td
															>
															<td class="py-2 px-3 text-gray-600 dark:text-gray-400">
																{#if student.last_accessed}
																	{new Date(student.last_accessed).toLocaleDateString()}
																{:else}
																	-
																{/if}
															</td>
															<td class="py-2 px-3">
																{#if student.has_certificate}
																	<span
																		class="inline-flex items-center gap-1 text-green-600 dark:text-green-400"
																	>
																		<svg
																			class="w-4 h-4"
																			fill="none"
																			viewBox="0 0 24 24"
																			stroke="currentColor"
																		>
																			<path
																				stroke-linecap="round"
																				stroke-linejoin="round"
																				stroke-width="2"
																				d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
																			></path>
																		</svg>
																		<span class="text-xs">{$t('admin.courses.cert_issued')}</span>
																	</span>
																{:else}
																	<span class="text-xs text-gray-400">-</span>
																{/if}
															</td>
														</tr>
													{/each}
												</tbody>
											</table>
										</div>
									{:else}
										<p class="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
											{$t('admin.courses.no_students')}
										</p>
									{/if}
								{/if}
							{/if}
						</div>
					{/if}

					<!-- Settings tab -->
					{#if activeTab === 'settings'}
						<div
							class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4"
						>
							<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
								{$t('admin.courses.section_publishing')}
							</h2>

							<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
								<VisibilitySelector bind:value={visibility} />
							</div>

							<!-- Course sales are Facet Cloud-only, so there is no price or
						     paid-tier control here. An imported paid course keeps its
						     tier and price untouched; the server refuses any change. -->
							{#if accessTier === 'paid'}
								<p
									class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200"
									data-testid="course-sales-unavailable-admin"
								>
									{$t('admin.courses.sales_unavailable_imported')}
								</p>
							{:else}
								<p class="text-sm text-gray-500 dark:text-gray-400" data-testid="course-free-note">
									{$t('admin.courses.sales_cloud_only')}
								</p>
							{/if}

							<div class="flex items-center gap-6">
								<Toggle bind:checked={isDraft} id="is_draft" label={$t('admin.courses.is_draft')} />
								<Toggle
									bind:checked={sequentialAccess}
									id="sequential_access"
									label={$t('admin.courses.sequential_access')}
								/>
							</div>
							<div class="flex items-center gap-6">
								<Toggle
									bind:checked={enableProgress}
									id="enable_progress"
									label={$t('admin.courses.track_progress')}
								/>
								<Toggle
									bind:checked={showProgressBar}
									id="show_progress_bar"
									label={$t('admin.courses.show_progress_bar')}
									disabled={!enableProgress}
								/>
								<Toggle
									bind:checked={requireEnrollment}
									id="require_enrollment"
									label={$t('admin.courses.require_enrollment')}
								/>
								<Toggle
									bind:checked={enableCertificates}
									id="enable_certificates"
									label={$t('admin.courses.issue_certificates')}
									disabled={!enableProgress}
								/>
							</div>
							<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
								{$t('admin.courses.enrollment_help')}
							</p>
							<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
								{$t('admin.courses.certificates_help')}
							</p>
						</div>
					{/if}

					<!-- Details tab -->
					{#if activeTab === 'details'}
						<div
							class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4"
						>
							<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
								{$t('admin.courses.section_details')}
							</h2>

							<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
								<div>
									<label for="difficulty" class="label">{$t('admin.courses.difficulty')}</label>
									<select id="difficulty" bind:value={difficulty} class="input">
										<option value="beginner">{$t('admin.courses.difficulty_beginner')}</option>
										<option value="intermediate"
											>{$t('admin.courses.difficulty_intermediate')}</option
										>
										<option value="advanced">{$t('admin.courses.difficulty_advanced')}</option>
									</select>
								</div>
								<div>
									<label for="estimated_hours" class="label"
										>{$t('admin.courses.estimated_hours')}</label
									>
									<input
										type="number"
										id="estimated_hours"
										bind:value={estimatedHours}
										class="input"
										min="0"
									/>
								</div>
								<div>
									<label for="instructor_name" class="label"
										>{$t('admin.courses.instructor_name')}</label
									>
									<input
										type="text"
										id="instructor_name"
										bind:value={instructorName}
										class="input"
										placeholder={$t('admin.courses.instructor_name_placeholder')}
									/>
								</div>
							</div>

							<div>
								<div class="flex items-center justify-between mb-2">
									<label for="instructor_bio" class="label mb-0"
										>{$t('admin.courses.instructor_bio')}</label
									>
									<AIContentHelper
										fieldType="instructor_bio"
										content={instructorBio}
										context={{ title: instructorName, type: 'instructor_bio' }}
										on:apply={(e) => (instructorBio = e.detail.content)}
									/>
								</div>
								<textarea id="instructor_bio" bind:value={instructorBio} class="input" rows="3"
								></textarea>
							</div>

							<div>
								<span class="label">{$t('admin.courses.tags')}</span>
								<div class="flex flex-wrap gap-2 mb-2">
									{#each tags as tag}
										<span
											class="inline-flex items-center gap-1 px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded text-sm"
										>
											{tag}

											<button
												type="button"
												class="text-gray-500 hover:text-red-500"
												onclick={() => removeTag(tag)}
												aria-label={$t('admin.courses.delete')}
											>
												<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="2"
														d="M6 18L18 6M6 6l12 12"
													></path>
												</svg>
											</button>
										</span>
									{/each}
								</div>
								<div class="flex gap-2">
									<input
										type="text"
										bind:value={tagInput}
										class="input flex-1"
										placeholder={$t('admin.courses.tag_placeholder')}
										onkeydown={handleTagKeydown}
									/>
									<button type="button" class="btn btn-secondary" onclick={addTag}
										>{$t('admin.courses.add_tag')}</button
									>
								</div>
							</div>

							<div>
								<span class="label">{$t('admin.courses.learning_outcomes')}</span>
								<div class="space-y-1 mb-2">
									{#each learningOutcomes as outcome}
										<div class="flex items-center gap-2">
											<span class="flex-1 text-sm text-gray-700 dark:text-gray-300">{outcome}</span>

											<button
												type="button"
												onclick={() => removeOutcome(outcome)}
												class="text-red-600 hover:text-red-700 dark:text-red-400"
												aria-label={$t('admin.courses.delete')}
											>
												<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="2"
														d="M6 18L18 6M6 6l12 12"
													></path>
												</svg>
											</button>
										</div>
									{/each}
								</div>
								<div class="flex gap-2">
									<input
										type="text"
										bind:value={outcomeInput}
										class="input flex-1"
										placeholder={$t('admin.courses.learning_outcomes_placeholder')}
										onkeydown={handleOutcomeKeydown}
									/>
									<button type="button" class="btn btn-secondary" onclick={addOutcome}
										>{$t('admin.courses.add_outcome')}</button
									>
								</div>
							</div>
						</div>
					{/if}
				</div>

				<!-- /Tab panels -->
			</fieldset>
			{#if courseCreateIssue}
				<div role="alert" class="rounded-lg border border-red-300 dark:border-red-700 p-3 mt-4">
					<p class="text-sm text-red-700 dark:text-red-300">
						{$t(
							courseCreateIssue === 'unknown'
								? 'admin.courses.course_create_unknown'
								: 'admin.courses.course_create_differs'
						)}
					</p>
				</div>
			{/if}
			<!-- Sticky form actions -->
			<div
				class="sticky bottom-0 z-10 bg-white/80 dark:bg-gray-900/80 backdrop-blur-sm border-t border-gray-200 dark:border-gray-700 -mx-6 px-6 py-3 mt-6 rounded-b-lg flex items-center justify-between"
			>
				<div class="flex items-center gap-3">
					<button type="button" class="btn btn-ghost" onclick={closeForm}
						>{$t('admin.courses.back_to_list')}</button
					>
					{#if editingCourse}
						<button
							type="button"
							class="btn btn-danger-ghost"
							onclick={() => editingCourse && deleteCourse(editingCourse)}
							>{$t('admin.courses.delete_course')}</button
						>
					{/if}
				</div>
				<button type="submit" class="btn btn-primary" disabled={saving || $listLoad.failed}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
					{/if}
					{courseCreateIssue === 'unknown' ? $t('admin.dashboard.retry') : $t('admin.courses.save')}
				</button>
			</div>
		</form>
	{:else if !$listLoad.loaded}
		<!-- ListLoadStatus above owns the unavailable state. -->
	{:else if courses.length === 0}
		<div class="card p-12 text-center">
			<svg
				class="mx-auto h-12 w-12 text-gray-400"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				aria-hidden="true"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 14l9-5-9-5-9 5 9 5z"
				></path>

				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z"
				></path>
			</svg>
			<h3 class="mt-4 text-lg font-medium text-gray-900 dark:text-white">
				{$t('admin.courses.no_courses')}
			</h3>
			<button class="btn btn-primary mt-6" onclick={openNewForm}>
				{$t('admin.courses.new_course')}
			</button>
		</div>
	{:else if filteredCourses.length === 0}
		<div class="card p-8 text-center">
			<svg
				class="w-12 h-12 mx-auto text-gray-400 mb-4"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
				></path>
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
				{$t('admin.courses.no_results')}
			</h3>
			<button class="btn btn-secondary" onclick={() => filterStore.clearAllFilters()}>
				{$t('admin.courses.clear_filters')}
			</button>
		</div>
	{:else}
		<fieldset class="contents" disabled={$listLoad.failed || $listLoad.loading}>
			<!-- Course List (grid layout) -->
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
				{#each filteredCourses as course (course.id)}
					<div
						class="card group relative hover:shadow-lg transition-all duration-200 overflow-hidden {selectMode &&
						selectedIds.has(course.id)
							? 'ring-2 ring-primary-500'
							: ''}"
					>
						{#if selectMode}
							<div class="absolute top-3 left-3 z-10">
								<input
									type="checkbox"
									aria-label={$t('admin.bulk_actions.select_item', {
										values: { item: course.title }
									})}
									checked={selectedIds.has(course.id)}
									onchange={() => toggleSelect(course.id)}
									class="w-5 h-5 text-primary-700 rounded border-gray-300 shadow-sm"
								/>
							</div>
						{/if}

						<!-- Cover image or gradient placeholder -->
						{#if course.cover_image_library_url || course.cover_image}
							<div class="h-36 overflow-hidden bg-gray-100 dark:bg-gray-800 relative">
								<img
									src={course.cover_image_library_url || course.cover_image}
									alt={course.title}
									class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
								/>
								<div class="absolute inset-0 bg-gradient-to-t from-black/30 to-transparent"></div>
							</div>
						{:else}
							<div
								class="h-20 bg-gradient-to-br from-primary-100 to-primary-50 dark:from-primary-900 dark:to-gray-800 flex items-center justify-center"
							>
								<svg
									class="w-8 h-8 text-primary-300 dark:text-primary-700"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M12 14l9-5-9-5-9 5 9 5z"
									></path>

									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z"
									></path>
								</svg>
							</div>
						{/if}

						<div class="p-4">
							<!-- Status badges -->
							<div class="flex items-center gap-1.5 mb-2 flex-wrap">
								{#if course.is_draft}
									<span
										class="px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/50 dark:text-yellow-300 rounded-full"
									>
										{$t('admin.courses.status_draft')}
									</span>
								{:else}
									<span
										class="px-2 py-0.5 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300 rounded-full"
									>
										{$t('admin.courses.status_published')}
									</span>
								{/if}
								<span
									class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-full capitalize"
								>
									{course.visibility}
								</span>
								{#if course.access_tier === 'paid'}
									<span
										class="px-2 py-0.5 text-xs font-semibold bg-purple-100 text-purple-800 dark:bg-purple-900/50 dark:text-purple-300 rounded-full"
									>
										{$t('admin.courses.imported_paid_badge')}
									</span>
								{/if}
							</div>

							<!-- Title -->
							<h3 class="text-base font-semibold text-gray-900 dark:text-white mb-1 line-clamp-1">
								{course.title}
							</h3>

							{#if course.description}
								<p class="text-sm text-gray-500 dark:text-gray-400 mb-3 line-clamp-2">
									{course.description}
								</p>
							{/if}

							<!-- Stats row -->
							<div class="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400 mb-3">
								<span class="inline-flex items-center gap-1 capitalize">
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M13 10V3L4 14h7v7l9-11h-7z"
										></path>
									</svg>

									{course.difficulty}
								</span>
								<span class="inline-flex items-center gap-1">
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
										></path>
									</svg>

									{getCourseModuleCount(course.id)} / {getCourseLessonCount(course.id)}
								</span>
								{#if course.estimated_hours}
									<span class="inline-flex items-center gap-1">
										<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
											></path>
										</svg>

										{course.estimated_hours}h
									</span>
								{/if}
							</div>

							<!-- Actions -->
							<div
								class="flex items-center justify-between pt-3 border-t border-gray-100 dark:border-gray-800"
							>
								<button
									class="text-sm font-medium text-primary-700 dark:text-primary-300 hover:text-primary-800 dark:hover:text-primary-200"
									onclick={() => openEditForm(course)}
									disabled={course.id === courseSavingId}
								>
									{$t('admin.courses.edit')}
								</button>
								<div class="flex items-center gap-1">
									<button
										class="btn btn-ghost btn-sm"
										title={course.is_draft
											? $t('admin.courses.publish')
											: $t('admin.courses.unpublish')}
										onclick={() => togglePublish(course)}
									>
										{#if course.is_draft}
											<svg
												class="w-4 h-4 text-green-600"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M5 13l4 4L19 7"
												></path>
											</svg>
										{:else}
											<svg
												class="w-4 h-4 text-yellow-600"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
												></path>
											</svg>
										{/if}
									</button>
									<button
										class="btn btn-danger-ghost btn-sm"
										title={$t('admin.courses.delete')}
										onclick={() => deleteCourse(course)}
									>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
											></path>
										</svg>
									</button>
								</div>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</fieldset>
	{/if}
</div>
