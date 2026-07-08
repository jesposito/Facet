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
	import SingleMediaPicker from '$lib/components/admin/SingleMediaPicker.svelte';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';
	import AutosaveRecoveryBanner from '$components/admin/AutosaveRecoveryBanner.svelte';
	import BulkActionBar from '$components/admin/BulkActionBar.svelte';
	import AdminFilters from '$components/admin/AdminFilters.svelte';
	import Toggle from '$components/admin/Toggle.svelte';
	import VisibilitySelector from '$components/admin/VisibilitySelector.svelte';
	import AccessTierSelect from '$components/admin/AccessTierSelect.svelte';
	import ReorderAnnouncer from '$lib/components/admin/ReorderAnnouncer.svelte';
	import { pb } from '$lib/pocketbase';

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

	let courses: Course[] = $state([]);
	let allModules: Module[] = $state([]);
	let allLessons: Lesson[] = $state([]);
	let loading = $state(true);
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
			(course) => `${course.title} ${course.description || ''} ${(course.tags || []).join(' ')} ${course.instructor_name || ''}`,
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
	let accessTier: 'free' | 'paid' = $state('free');
	let priceInDollars = $state('');
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

	// Curriculum state
	let courseModules: Module[] = $state([]);
	let courseLessons: Record<string, Lesson[]> = $state({});
	let expandedModules: Set<string> = $state(new Set());
	let expandedLessons: Set<string> = $state(new Set());
	let loadingCurriculum = $state(false);

	// Inline module form
	let showModuleForm = $state(false);
	let editingModule: Module | null = $state(null);
	let moduleTitle = $state('');
	let moduleDescription = $state('');
	let moduleDripDaysAfterEnrollment = $state('');
	let moduleDripUnlockDate = $state('');

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
	let lessonPreviewMode = $state(false);

	// Quiz builder state
	let quizTitle = $state('');
	let quizPassingScore = $state(70);
	let quizMaxAttempts = $state(0);
	let quizRandomize = $state(false);
	let quizQuestions: QuizQuestion[] = $state([]);
	let quizId: string | null = $state(null);
	let loadingQuiz = $state(false);

	// Enrollment dashboard
	let showStudents = $state(true);
	let loadingStudents = $state(false);
	let studentsLoaded = $state(false);
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

	function getFormData() {
		return {
			title, slug, description, instructorName, instructorBio,
			accessTier, priceInDollars, difficulty, estimatedHours,
			tags, learningOutcomes, visibility, isDraft, sequentialAccess,
			enableProgress, showProgressBar, requireEnrollment, enableCertificates,
			coverImageLibraryUrl
		};
	}

	function restoreFromDraft(data: Record<string, any>) {
		title = data.title || '';
		slug = data.slug || '';
		description = data.description || '';
		instructorName = data.instructorName || '';
		instructorBio = data.instructorBio || '';
		accessTier = data.accessTier || 'free';
		priceInDollars = data.priceInDollars || '';
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
		}
	});

	afterNavigate(() => {
		showForm = false;
		editingCourse = null;
		selectMode = false;
		selectedIds = new Set();

		const draft = autosave.loadDraft();
		if (draft?.data && Object.values(draft.data).some(v => v !== '' && v !== false && v !== 0 && !(Array.isArray(v) && v.length === 0))) {
			recoveryData = { savedAt: draft.savedAt, isEditing: draft.isEditing || false };
			showRecoveryBanner = true;
		} else {
			showRecoveryBanner = false;
			recoveryData = null;
		}
	});

	async function loadCourses() {
		loading = true;
		try {
			const [coursesResult, modulesResult, lessonsResult] = await Promise.all([
				pb.collection('courses').getList(1, 100, { sort: 'sort_order' }),
				pb.collection('modules').getList(1, 500, { sort: 'sort_order' }),
				pb.collection('lessons').getList(1, 500, { sort: 'sort_order' })
			]);
			courses = coursesResult.items as unknown as Course[];
			allModules = modulesResult.items as unknown as Module[];
			allLessons = lessonsResult.items as unknown as Lesson[];
		} catch (err) {
			console.error('Failed to load courses:', err);
			toasts.add('error', $t('admin.courses.error_load'));
		} finally {
			loading = false;
		}
	}

	async function loadCurriculum(courseId: string) {
		loadingCurriculum = true;
		try {
			const [modulesResult, lessonsResult] = await Promise.all([
				pb.collection('modules').getList(1, 100, {
					filter: `course="${courseId}"`,
					sort: 'sort_order'
				}),
				pb.collection('lessons').getList(1, 100, {
					filter: `module.course="${courseId}"`,
					sort: 'module.sort_order,sort_order'
				})
			]);

			courseModules = modulesResult.items as unknown as Module[];

			const lessonsByModule: Record<string, Lesson[]> = {};
			for (const module of courseModules) {
				lessonsByModule[module.id] = [];
			}
			for (const lesson of lessonsResult.items as unknown as Lesson[]) {
				if (lessonsByModule[lesson.module]) {
					lessonsByModule[lesson.module].push(lesson);
				}
			}
			courseLessons = lessonsByModule;

			expandedModules = new Set(courseModules.map(m => m.id));
		} catch (err) {
			console.error('Failed to load curriculum:', err);
			toasts.add('error', $t('admin.courses.error_load_curriculum'));
		} finally {
			loadingCurriculum = false;
		}
	}

	function resetForm() {
		title = '';
		slug = '';
		description = '';
		coverImageFile = null;
		coverImageLibraryUrl = '';
		clearCoverImage = false;
		instructorName = '';
		instructorBio = '';
		accessTier = 'free';
		priceInDollars = '';
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
		courseModules = [];
		courseLessons = {};
		expandedModules = new Set();
		expandedLessons = new Set();
		students = [];
		enrollmentStats = null;
		studentsLoaded = false;
	}

	function openNewForm() {
		resetForm();
		showForm = true;
		showRecoveryBanner = false;
	}

	function handleRestoreDraft() {
		const draft = autosave.loadDraft();
		if (draft?.data) {
			restoreFromDraft(draft.data);
			if (draft.isEditing && draft.editingId) {
				const course = courses.find(c => c.id === draft.editingId);
				if (course) {
					editingCourse = course;
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
		editingCourse = course;
		students = [];
		enrollmentStats = null;
		studentsLoaded = false;
		title = course.title;
		slug = course.slug || '';
		description = course.description || '';
		coverImageFile = null;
		coverImageLibraryUrl = course.cover_image_library_url || '';
		clearCoverImage = false;
		instructorName = course.instructor_name || '';
		instructorBio = course.instructor_bio || '';
		accessTier = course.access_tier || 'free';
		priceInDollars = course.price ? (course.price / 100).toFixed(2) : '';
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

	async function handleSubmit() {
		if (!title.trim()) {
			toasts.add('error', $t('admin.courses.error_title_required'));
			return;
		}
		if (!slug.trim()) {
			toasts.add('error', $t('admin.courses.error_slug_required'));
			return;
		}

		saving = true;
		try {
			const formData = new FormData();
			formData.append('title', title.trim());
			formData.append('slug', slug.trim());
			formData.append('description', description.trim());
			formData.append('instructor_name', instructorName.trim());
			formData.append('instructor_bio', instructorBio.trim());
			formData.append('access_tier', accessTier);

			if (accessTier === 'paid' && priceInDollars) {
				const priceInCents = Math.round(parseFloat(priceInDollars) * 100);
				formData.append('price', String(priceInCents));
			} else {
				formData.append('price', '0');
			}

			formData.append('difficulty', difficulty);
			formData.append('estimated_hours', String(estimatedHours));
			formData.append('tags', JSON.stringify(tags));
			formData.append('learning_outcomes', JSON.stringify(learningOutcomes));
			formData.append('visibility', visibility);
			formData.append('is_draft', String(isDraft));
			formData.append('sequential_access', String(sequentialAccess));
			formData.append('enable_progress', String(enableProgress));
			formData.append('show_progress_bar', String(showProgressBar));
			formData.append('require_enrollment', String(requireEnrollment));
			formData.append('enable_certificates', String(enableCertificates));

			if (coverImageFile && coverImageFile.length > 0) {
				formData.append('cover_image', coverImageFile[0]);
				formData.append('cover_image_library_url', '');
			} else if (clearCoverImage && editingCourse?.cover_image) {
				formData.append('cover_image', '');
				formData.append('cover_image_library_url', coverImageLibraryUrl || '');
			} else {
				formData.append('cover_image_library_url', coverImageLibraryUrl || '');
			}

			if (editingCourse) {
				await pb.collection('courses').update(editingCourse.id, formData);
				toasts.add('success', $t('admin.courses.saved'));
			} else {
				const created = await pb.collection('courses').create(formData);
				editingCourse = created as unknown as Course;
				toasts.add('success', $t('admin.courses.saved'));
				await loadCurriculum(editingCourse.id);
			}

			autosave.clearDraft();
			await loadCourses();
		} catch (err: unknown) {
			console.error('Failed to save course:', err);
			const error = err as { data?: { data?: { slug?: { message?: string } } } };
			if (error.data?.data?.slug?.message) {
				toasts.add('error', $t('admin.courses.error_slug_exists'));
			} else {
				toasts.add('error', $t('admin.courses.error_save'));
			}
		} finally {
			saving = false;
		}
	}

	async function deleteCourse(course: Course) {
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
	}

	async function togglePublish(course: Course) {
		try {
			const newDraftState = !course.is_draft;
			await pb.collection('courses').update(course.id, { is_draft: newDraftState });
			toasts.add('success', newDraftState ? $t('admin.courses.unpublished') : $t('admin.courses.published'));
			await loadCourses();
		} catch (err) {
			console.error('Failed to toggle publish:', err);
			toasts.add('error', $t('admin.courses.error_save'));
		}
	}

	function getCourseModuleCount(courseId: string): number {
		return allModules.filter(m => m.course === courseId).length;
	}

	function getCourseLessonCount(courseId: string): number {
		return allLessons.filter(l => {
			const mod = allModules.find(m => m.id === l.module);
			return mod?.course === courseId;
		}).length;
	}

	// Select mode
	function toggleSelectMode() {
		selectMode = !selectMode;
		if (!selectMode) selectedIds = new Set();
	}

	function toggleSelect(id: string) {
		if (selectedIds.has(id)) selectedIds.delete(id);
		else selectedIds.add(id);
		selectedIds = selectedIds;
	}

	function selectAll() { selectedIds = new Set(courses.map(c => c.id)); }
	function clearSelection() { selectedIds = new Set(); }

	async function bulkSetVisibility(visibility: 'public' | 'unlisted' | 'private') {
		const ids = Array.from(selectedIds);
		let succeeded = 0;
		const failures: string[] = [];
		for (const id of ids) {
			try {
				await pb.collection('courses').update(id, { visibility });
				succeeded++;
			} catch (err) {
				failures.push(id);
			}
		}
		if (failures.length === 0) {
			toasts.add('success', $t('admin.courses.bulk_visibility_updated', { values: { count: ids.length, visibility } }));
		} else if (succeeded === 0) {
			toasts.add('error', $t('admin.courses.error_save'));
		} else {
			toasts.add('error', `${succeeded} of ${ids.length} updated, ${failures.length} failed`);
		}
		selectedIds = new Set();
		selectMode = false;
		await loadCourses();
	}

	async function bulkDelete() {
		const ids = Array.from(selectedIds);
		const confirmed = await confirm({
			title: $t('admin.courses.delete_course'),
			message: $t('admin.courses.bulk_delete_confirm', { values: { count: ids.length } }),
			confirmText: $t('admin.courses.confirm_delete'),
			danger: true
		});
		if (!confirmed) return;
		let succeeded = 0;
		const failures: string[] = [];
		for (const id of ids) {
			try {
				await pb.collection('courses').delete(id);
				succeeded++;
			} catch (err) {
				failures.push(id);
			}
		}
		if (failures.length === 0) {
			toasts.add('success', $t('admin.courses.bulk_deleted', { values: { count: ids.length } }));
		} else if (succeeded === 0) {
			toasts.add('error', $t('admin.courses.error_delete'));
		} else {
			toasts.add('error', `${succeeded} of ${ids.length} deleted, ${failures.length} failed`);
		}
		selectedIds = new Set();
		selectMode = false;
		await loadCourses();
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
		editingModule = null;
		moduleTitle = '';
		moduleDescription = '';
		moduleDripDaysAfterEnrollment = '';
		moduleDripUnlockDate = '';
		showModuleForm = true;
	}

	function openEditModuleForm(module: Module) {
		editingModule = module;
		moduleTitle = module.title;
		moduleDescription = module.description || '';
		moduleDripDaysAfterEnrollment = module.drip_days_after_enrollment ? String(module.drip_days_after_enrollment) : '';
		moduleDripUnlockDate = module.drip_unlock_date || '';
		showModuleForm = true;
	}

	function closeModuleForm() {
		showModuleForm = false;
		editingModule = null;
		moduleTitle = '';
		moduleDescription = '';
		moduleDripDaysAfterEnrollment = '';
		moduleDripUnlockDate = '';
	}

	async function saveModule() {
		if (!moduleTitle.trim() || !editingCourse) return;

		saving = true;
		try {
			const data: Record<string, any> = {
				title: moduleTitle.trim(),
				description: moduleDescription.trim(),
				sort_order: editingModule ? editingModule.sort_order : courseModules.length + 1,
				course: editingCourse.id
			};

			if (moduleDripDaysAfterEnrollment) {
				data.drip_days_after_enrollment = parseInt(moduleDripDaysAfterEnrollment);
			} else {
				data.drip_days_after_enrollment = 0;
			}
			if (moduleDripUnlockDate) {
				data.drip_unlock_date = new Date(moduleDripUnlockDate).toISOString();
			} else {
				data.drip_unlock_date = '';
			}

			if (editingModule) {
				await pb.collection('modules').update(editingModule.id, data);
				toasts.add('success', $t('admin.courses.module_updated'));
			} else {
				await pb.collection('modules').create(data);
				toasts.add('success', $t('admin.courses.module_created'));
			}

			closeModuleForm();
			await loadCurriculum(editingCourse.id);
			await loadCourses();
		} catch (err) {
			console.error('Failed to save module:', err);
			toasts.add('error', $t('admin.courses.error_save_module'));
		} finally {
			saving = false;
		}
	}

	async function deleteModule(module: Module) {
		if (!editingCourse) return;

		const confirmed = await confirm({
			title: $t('admin.courses.delete_module'),
			message: $t('admin.courses.delete_module_confirm'),
			confirmText: $t('admin.courses.confirm_delete'),
			danger: true
		});
		if (!confirmed) return;

		try {
			await pb.collection('modules').delete(module.id);
			toasts.add('success', $t('admin.courses.module_deleted'));
			await loadCurriculum(editingCourse.id);
			await loadCourses();
		} catch (err) {
			console.error('Failed to delete module:', err);
			toasts.add('error', $t('admin.courses.error_delete_module'));
		}
	}

	async function moveModule(module: Module, direction: 'up' | 'down') {
		if (!editingCourse) return;

		const currentIndex = courseModules.findIndex((m) => m.id === module.id);
		if (currentIndex === -1) return;
		if (direction === 'up' && currentIndex === 0) return;
		if (direction === 'down' && currentIndex === courseModules.length - 1) return;

		const targetIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1;
		const targetModule = courseModules[targetIndex];

		try {
			await Promise.all([
				pb.collection('modules').update(module.id, { sort_order: targetModule.sort_order }),
				pb.collection('modules').update(targetModule.id, { sort_order: module.sort_order })
			]);
			await loadCurriculum(editingCourse.id);
			// Announce the new position (1-based) for screen-reader users.
			reorderAnnouncer?.announce(
				$t('admin.courses.module_moved_announce', {
					values: { title: module.title, position: targetIndex + 1, total: courseModules.length }
				})
			);
		} catch (err) {
			console.error('Failed to reorder modules:', err);
			toasts.add('error', $t('admin.courses.error_reorder_modules'));
		}
	}

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
		if (lesson.content_type === 'quiz' && editingCourse) {
			loadQuizData(editingCourse.id, lesson.id);
		}
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
		lessonAttachmentFiles = null;
		lessonPreviewMode = false;
		resetQuizState();
	}

	async function loadStudents(courseId: string) {
		loadingStudents = true;
		try {
			const response = await fetch(`/api/admin/courses/${courseId}/students`, {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (!response.ok) throw new Error('Failed to load students');
			const data = await response.json();
			students = data.students || [];
			enrollmentStats = data.stats || null;
			studentsLoaded = true;
		} catch (err) {
			console.error('Failed to load students:', err);
			toasts.add('error', $t('admin.courses.failed_load_enrollment'));
			studentsLoaded = true;
		} finally {
			loadingStudents = false;
		}
	}

	// Auto-load students when switching to students tab
	$effect(() => {
		if (activeTab === 'students' && editingCourse && !loadingStudents && !studentsLoaded) {
			loadStudents(editingCourse.id);
		}
	});

	function resetQuizState() {
		quizTitle = '';
		quizPassingScore = 70;
		quizMaxAttempts = 0;
		quizRandomize = false;
		quizQuestions = [];
		quizId = null;
		loadingQuiz = false;
	}

	async function loadQuizData(courseId: string, lessonId: string) {
		loadingQuiz = true;
		try {
			const response = await fetch(`/api/admin/courses/${courseId}/lessons/${lessonId}/quiz`, {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (response.ok) {
				const data = await response.json();
				quizId = data.id || null;
				quizTitle = data.title || '';
				quizPassingScore = data.passing_score ?? 70;
				quizMaxAttempts = data.max_attempts ?? 0;
				quizRandomize = data.randomize_questions ?? false;
				quizQuestions = (data.questions || []).map((q: any, i: number) => ({
					id: q.id,
					question_text: q.question_text || '',
					question_type: q.question_type || 'multiple_choice',
					options: q.options || [{ text: '', is_correct: true }, { text: '', is_correct: false }],
					explanation: q.explanation || '',
					points: q.points ?? 10,
					sort_order: q.sort_order ?? i
				}));
			}
		} catch (err) {
			console.error('Failed to load quiz data:', err);
		} finally {
			loadingQuiz = false;
		}
	}

	async function saveQuizData(courseId: string, lessonId: string) {
		if (quizQuestions.length === 0) {
			throw new Error('Quiz must have at least one question');
		}
		const quizPayload = {
			title: quizTitle || lessonTitle,
			passing_score: quizPassingScore,
			max_attempts: quizMaxAttempts,
			randomize_questions: quizRandomize,
			questions: quizQuestions.map((q, i) => ({
				...q,
				sort_order: i
			}))
		};

		const method = quizId ? 'PATCH' : 'POST';
		const response = await fetch(`/api/admin/courses/${courseId}/lessons/${lessonId}/quiz`, {
			method,
			headers: {
				'Content-Type': 'application/json',
				...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
			},
			body: JSON.stringify(quizPayload)
		});
		if (!response.ok) throw new Error('Failed to save quiz');
		const data = await response.json();
		quizId = data.id || quizId;
	}

	function addQuizQuestion() {
		quizQuestions = [...quizQuestions, {
			question_text: '',
			question_type: 'multiple_choice',
			options: [
				{ text: '', is_correct: true },
				{ text: '', is_correct: false }
			],
			explanation: '',
			points: 10,
			sort_order: quizQuestions.length
		}];
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
		if (!q.options.some(o => o.is_correct)) {
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

	async function saveLesson() {
		if (!lessonTitle.trim() || !showLessonForm || !editingCourse) return;

		saving = true;
		try {
			const formData = new FormData();
			formData.append('title', lessonTitle.trim());
			formData.append('content_type', lessonContentType);
			formData.append('video_url', lessonVideoUrl.trim());
			formData.append('content', lessonContent.trim());
			formData.append('estimated_minutes', String(lessonEstimatedMinutes));
			formData.append('is_preview', String(lessonIsPreview));
			formData.append('sort_order', String(editingLesson ? editingLesson.lesson.sort_order : (courseLessons[showLessonForm]?.length || 0) + 1));
			formData.append('module', showLessonForm);

			if (lessonAttachmentFiles) {
				for (let i = 0; i < lessonAttachmentFiles.length; i++) {
					formData.append('attachments', lessonAttachmentFiles[i]);
				}
			}

			let savedLessonId: string;
			if (editingLesson) {
				await pb.collection('lessons').update(editingLesson.lesson.id, formData);
				savedLessonId = editingLesson.lesson.id;
				toasts.add('success', $t('admin.courses.lesson_updated'));
			} else {
				const created = await pb.collection('lessons').create(formData);
				savedLessonId = created.id;
				toasts.add('success', $t('admin.courses.lesson_created'));
			}

			if (lessonContentType === 'quiz' && editingCourse) {
				try {
					await saveQuizData(editingCourse.id, savedLessonId);
				} catch (quizErr) {
					console.error('Failed to save quiz data:', quizErr);
					toasts.add('error', $t('admin.courses.error_save_lesson'));
				}
			}

			closeLessonForm();
			await loadCurriculum(editingCourse.id);
			await loadCourses();
		} catch (err) {
			console.error('Failed to save lesson:', err);
			toasts.add('error', $t('admin.courses.error_save_lesson'));
		} finally {
			saving = false;
		}
	}

	async function deleteLesson(lesson: Lesson) {
		if (!editingCourse) return;

		const confirmed = await confirm({
			title: $t('admin.courses.delete_lesson'),
			message: $t('admin.courses.delete_lesson_confirm'),
			confirmText: $t('admin.courses.confirm_delete'),
			danger: true
		});
		if (!confirmed) return;

		try {
			await pb.collection('lessons').delete(lesson.id);
			toasts.add('success', $t('admin.courses.lesson_deleted'));
			await loadCurriculum(editingCourse.id);
			await loadCourses();
		} catch (err) {
			console.error('Failed to delete lesson:', err);
			toasts.add('error', $t('admin.courses.error_delete_lesson'));
		}
	}

	async function moveLesson(lesson: Lesson, moduleId: string, direction: 'up' | 'down') {
		if (!editingCourse) return;

		const moduleLessons = courseLessons[moduleId] || [];
		const currentIndex = moduleLessons.findIndex((l) => l.id === lesson.id);
		if (currentIndex === -1) return;
		if (direction === 'up' && currentIndex === 0) return;
		if (direction === 'down' && currentIndex === moduleLessons.length - 1) return;

		const targetIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1;
		const targetLesson = moduleLessons[targetIndex];

		try {
			await Promise.all([
				pb.collection('lessons').update(lesson.id, { sort_order: targetLesson.sort_order }),
				pb.collection('lessons').update(targetLesson.id, { sort_order: lesson.sort_order })
			]);
			await loadCurriculum(editingCourse.id);
			// Announce the new position (1-based) for screen-reader users.
			reorderAnnouncer?.announce(
				$t('admin.courses.lesson_moved_announce', {
					values: { title: lesson.title, position: targetIndex + 1, total: moduleLessons.length }
				})
			);
		} catch (err) {
			console.error('Failed to reorder lessons:', err);
			toasts.add('error', $t('admin.courses.error_reorder_lessons'));
		}
	}

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
	{#if showRecoveryBanner && recoveryData}
		<AutosaveRecoveryBanner
			savedAt={recoveryData.savedAt}
			isEditing={recoveryData.isEditing}
			visible={true}
			on:restore={handleRestoreDraft}
			on:dismiss={handleDismissDraft}
		/>
	{/if}

	{#if selectMode && selectedIds.size > 0}
		<BulkActionBar
			selectedCount={selectedIds.size}
			totalCount={courses.length}
			on:selectAll={selectAll}
			on:clearSelection={clearSelection}
			on:setVisibility={(e) => bulkSetVisibility(e.detail)}
			on:delete={bulkDelete}
			on:cancel={toggleSelectMode}
		/>
	{/if}

	<div class="flex items-center justify-between mb-8">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{$t('admin.courses.list_title')}</h1>
			<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{$t('admin.courses.course_count', { values: { count: courses.length } })}</p>
		</div>
		<div class="flex items-center gap-2">
			{#if courses.length > 0 && !showForm}
				<button
					class="btn {selectMode ? 'btn-secondary' : 'btn-ghost'}"
					onclick={toggleSelectMode}
				>
					{selectMode ? $t('admin.courses.cancel') : $t('admin.courses.select')}
				</button>
			{/if}
			{#if !showForm}
				<button class="btn btn-primary" onclick={openNewForm}>
					{$t('admin.courses.new_course')}
				</button>
			{/if}
		</div>
	</div>

	{#if !showForm}
		<AdminFilters bind:showAdvanced={showAdvancedFilters} {filterStore} availableTags={[]} />
	{/if}

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
					<button type="button" class="p-2 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-800 transition-colors" onclick={closeForm} aria-label={$t('admin.courses.cancel')}>
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
				<nav class="flex gap-0 px-6 -mb-px" aria-label={$t('admin.courses.tab_content')}>
					<button type="button" onclick={() => activeTab = 'content'}
						class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
							{activeTab === 'content' ? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}">
						<span class="flex items-center gap-1.5">
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>
							{$t('admin.courses.tab_content')}
						</span>
					</button>
					<button type="button" onclick={() => activeTab = 'curriculum'}
						class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
							{activeTab === 'curriculum' ? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}">
						<span class="flex items-center gap-1.5">
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" /></svg>
							{$t('admin.courses.tab_curriculum')}
							{#if editingCourse}
								<span class="ml-1 text-xs px-1.5 py-0.5 rounded-full bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400">
									{courseModules.length}
								</span>
							{/if}
						</span>
					</button>
					<button type="button" onclick={() => activeTab = 'settings'}
						class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
							{activeTab === 'settings' ? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}">
						<span class="flex items-center gap-1.5">
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
							{$t('admin.courses.tab_settings')}
						</span>
					</button>
					<button type="button" onclick={() => activeTab = 'details'}
						class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
							{activeTab === 'details' ? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}">
						<span class="flex items-center gap-1.5">
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" /></svg>
							{$t('admin.courses.section_details')}
						</span>
					</button>
					{#if editingCourse}
						<button type="button" onclick={() => activeTab = 'students'}
							class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors
								{activeTab === 'students' ? 'border-primary-600 text-primary-700 dark:text-primary-400 dark:border-primary-400' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'}">
							<span class="flex items-center gap-1.5">
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" /></svg>
								{$t('admin.courses.enrolled_students')}
							</span>
						</button>
					{/if}
				</nav>
			</div>

			<!-- Tab panels -->
			<div class="space-y-6 mt-0">

			<!-- Content tab -->
			{#if activeTab === 'content'}
			<div class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4">

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
						<button type="button" class="text-xs text-primary-600 ml-2" onclick={generateSlug}>
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
					<p class="text-xs text-gray-500 mt-1">{$t('admin.courses.slug_preview', { values: { slug: slug || 'course-url-slug' } })}</p>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="description" class="label mb-0">{$t('admin.courses.description')}</label>
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
						placeholder={$t('admin.courses.description_placeholder')}
					></textarea>
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
			<div class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4">
				<!-- Polite live region for keyboard module/lesson reorder announcements -->
				<ReorderAnnouncer bind:this={reorderAnnouncer} />
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.courses.tab_curriculum')}</h2>
					{#if editingCourse}
						<button type="button" class="btn btn-secondary text-sm" onclick={openNewModuleForm}>
							+ {$t('admin.courses.add_module')}
						</button>
					{/if}
				</div>

				{#if !editingCourse}
					<div class="text-center py-6 text-gray-500 dark:text-gray-400 text-sm">
						<p>{$t('admin.courses.save_first')}</p>
						<p class="text-xs mt-1">{$t('admin.courses.save_first_hint')}</p>
					</div>
				{:else if loadingCurriculum}
					<div class="py-6 text-center">
						<div class="animate-pulse text-gray-500">{$t('admin.courses.loading_curriculum')}</div>
					</div>
				{:else}
					<!-- Inline Module Form -->
					{#if showModuleForm}
						<div class="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border-2 border-primary-200 dark:border-primary-800">
							<div class="flex items-center justify-between mb-3">
								<h4 class="text-sm font-medium text-gray-900 dark:text-white">
									{editingModule ? $t('admin.courses.edit_module') : $t('admin.courses.add_module')}
								</h4>
								<button type="button" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" onclick={closeModuleForm} aria-label={$t('admin.courses.cancel')}>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</div>
							<div class="space-y-3">
								<div>
									<label for="module_title" class="label text-sm">{$t('admin.courses.module_title')}</label>
									<input type="text" id="module_title" bind:value={moduleTitle} class="input" placeholder={$t('admin.courses.module_title_placeholder')} />
								</div>
								<div>
									<div class="flex items-center justify-between mb-2">
										<label for="module_description" class="label text-sm mb-0">{$t('admin.courses.module_description')}</label>
										<AIContentHelper
											fieldType="module_description"
											content={moduleDescription}
											context={{ title: moduleTitle, parent: title, type: 'module_description' }}
											on:apply={(e) => (moduleDescription = e.detail.content)}
										/>
									</div>
									<textarea id="module_description" bind:value={moduleDescription} class="input" rows="3"></textarea>
								</div>
								<!-- Module Drip Settings -->
								<div class="border-t border-gray-200 dark:border-gray-700 pt-3 mt-3 space-y-2">
									<p class="text-xs font-medium text-gray-700 dark:text-gray-300">{$t('admin.courses.drip_schedule')}</p>
									<div class="grid grid-cols-2 gap-3">
										<div>
											<label for="module_drip_days" class="label text-xs">{$t('admin.courses.module_drip_days')}</label>
											<input
												type="number"
												id="module_drip_days"
												bind:value={moduleDripDaysAfterEnrollment}
												class="input text-sm"
												placeholder="0"
												min="0"
											/>
										</div>
										<div>
											<label for="module_drip_date" class="label text-xs">{$t('admin.courses.module_drip_date')}</label>
											<input
												type="date"
												id="module_drip_date"
												bind:value={moduleDripUnlockDate}
												class="input text-sm"
											/>
										</div>
									</div>
									<p class="text-xs text-gray-400 dark:text-gray-500">{$t('admin.courses.module_drip_helper')}</p>
								</div>
								<div class="flex gap-2 justify-end">
									<button type="button" class="btn btn-ghost text-sm" onclick={closeModuleForm}>{$t('admin.courses.cancel')}</button>
									<button type="button" class="btn btn-primary text-sm" onclick={saveModule} disabled={saving || !moduleTitle.trim()}>
										{saving ? $t('admin.courses.saving') : $t('admin.courses.save_module')}
									</button>
								</div>
							</div>
						</div>
					{/if}

					{#if courseModules.length === 0 && !showModuleForm}
						<div class="text-center py-8 text-gray-500 dark:text-gray-400">
							<svg class="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
							</svg>
							<p class="text-sm">{$t('admin.courses.no_modules')}</p>
							<button type="button" class="btn btn-secondary text-sm mt-3" onclick={openNewModuleForm}>
								+ {$t('admin.courses.add_module')}
							</button>
						</div>
					{:else}
						<div class="space-y-4">
							{#each courseModules as module, moduleIndex (module.id)}
								<div class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
									<!-- Module Header -->
									<div class="flex items-center gap-3 px-4 py-3 bg-gray-50 dark:bg-gray-800/50">
										<button
											type="button"
											class="flex-1 text-left flex items-center gap-2"
											onclick={() => toggleModuleExpand(module.id)}
										>
											<svg
												class="w-4 h-4 text-gray-400 transition-transform {expandedModules.has(module.id) ? 'rotate-90' : ''}"
												fill="none" viewBox="0 0 24 24" stroke="currentColor"
											>
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
											</svg>
											<span class="font-medium text-gray-900 dark:text-white text-sm">{module.title}</span>
											<span class="text-xs text-gray-500">
												{courseLessons[module.id]?.length || 0} {$t('admin.courses.lessons').toLowerCase()}
											</span>
											{#if module.drip_days_after_enrollment}
												<span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400">
													<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
													{module.drip_days_after_enrollment}d
												</span>
											{:else if module.drip_unlock_date}
												<span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400">
													<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
													drip
												</span>
											{/if}
										</button>
										<div class="flex items-center gap-1">
											<button type="button" class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30" onclick={() => moveModule(module, 'up')} disabled={moduleIndex === 0} aria-label={$t('admin.courses.reorder_modules')}>
												<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" /></svg>
											</button>
											<button type="button" class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30" onclick={() => moveModule(module, 'down')} disabled={moduleIndex === courseModules.length - 1} aria-label={$t('admin.courses.reorder_modules')}>
												<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" /></svg>
											</button>
											<button type="button" class="p-1 text-gray-400 hover:text-primary-600 dark:hover:text-primary-400" onclick={() => openEditModuleForm(module)} aria-label={$t('admin.courses.edit_module')}>
												<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>
											</button>
											<button type="button" class="p-1 text-gray-400 hover:text-red-600 dark:hover:text-red-400" onclick={() => deleteModule(module)} aria-label={$t('admin.courses.delete_module')}>
												<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
											</button>
										</div>
									</div>

									<!-- Module Content (Lessons) -->
									{#if expandedModules.has(module.id)}
										<div class="px-4 py-3 space-y-2">
											{#if module.description}
												<p class="text-xs text-gray-500 dark:text-gray-400 mb-2">{module.description}</p>
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
																class="w-3 h-3 text-gray-400 transition-transform {expandedLessons.has(lesson.id) ? 'rotate-90' : ''}"
																fill="none" viewBox="0 0 24 24" stroke="currentColor"
															>
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
															</svg>
															<span class="text-sm text-gray-900 dark:text-white">{lesson.title}</span>
															<span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium
																{lesson.content_type === 'video' ? 'bg-sky-100 dark:bg-sky-900/30 text-sky-700 dark:text-sky-400' :
																 lesson.content_type === 'text' ? 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400' :
																 lesson.content_type === 'quiz' ? 'bg-violet-100 dark:bg-violet-900/30 text-violet-700 dark:text-violet-400' :
																 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400'}">
																{$t('admin.courses.lesson_type_' + lesson.content_type)}
															</span>
															{#if lesson.estimated_minutes}
																<span class="text-xs text-gray-500">{lesson.estimated_minutes}m</span>
															{/if}
															{#if lesson.is_preview}
																<span class="text-xs text-primary-600 dark:text-primary-400">{$t('admin.courses.preview_badge')}</span>
															{/if}
														</button>
														<div class="flex items-center gap-1">
															<button type="button" class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30" onclick={() => moveLesson(lesson, module.id, 'up')} disabled={lessonIndex === 0} aria-label={$t('admin.courses.reorder_lessons')}>
																<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" /></svg>
															</button>
															<button type="button" class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30" onclick={() => moveLesson(lesson, module.id, 'down')} disabled={lessonIndex === (courseLessons[module.id]?.length || 0) - 1} aria-label={$t('admin.courses.reorder_lessons')}>
																<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" /></svg>
															</button>
															<button type="button" class="p-1 text-gray-400 hover:text-primary-600 dark:hover:text-primary-400" onclick={() => openEditLessonForm(lesson, module.id)} aria-label={$t('admin.courses.edit_lesson')}>
																<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>
															</button>
															<button type="button" class="p-1 text-gray-400 hover:text-red-600 dark:hover:text-red-400" onclick={() => deleteLesson(lesson)} aria-label={$t('admin.courses.delete_lesson')}>
																<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
															</button>
														</div>
													</div>

													<!-- Lesson Content (expanded inline) -->
													{#if expandedLessons.has(lesson.id)}
														<div class="px-3 pb-3 pt-1 border-t border-gray-100 dark:border-gray-700 space-y-3">
															{#if lesson.video_url}
																<div>
																	<span class="label text-xs">{$t('admin.courses.lesson_video_url')}</span>
																	<p class="text-sm text-gray-600 dark:text-gray-400 break-all">{lesson.video_url}</p>
																</div>
															{/if}
															{#if lesson.content}
																<div>
																	<span class="label text-xs">{$t('admin.courses.lesson_content')}</span>
																	<div class="text-sm text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-900/30 rounded p-2 max-h-32 overflow-y-auto font-mono text-xs whitespace-pre-wrap">{lesson.content}</div>
																</div>
															{/if}
															{#if !lesson.video_url && !lesson.content}
																<p class="text-xs text-gray-400 italic">{$t('admin.courses.no_content_yet')}</p>
															{/if}
														</div>
													{/if}
												</div>
											{/each}

											<!-- New Lesson Form (inline) -->
											{#if showLessonForm === module.id}
												<div class="bg-primary-50/50 dark:bg-primary-900/10 rounded-lg p-3 border border-primary-200 dark:border-primary-800 space-y-3">
													<div class="flex items-center justify-between">
														<h5 class="text-sm font-medium text-gray-900 dark:text-white">
															{editingLesson ? $t('admin.courses.edit_lesson') : $t('admin.courses.add_lesson')}
														</h5>
														<button type="button" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" onclick={closeLessonForm} aria-label={$t('admin.courses.cancel')}>
															<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
																<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
															</svg>
														</button>
													</div>
													<div>
														<label for="lesson_title" class="label text-xs">{$t('admin.courses.lesson_title')} *</label>
														<input type="text" id="lesson_title" bind:value={lessonTitle} class="input text-sm" placeholder={$t('admin.courses.lesson_title_placeholder')} />
													</div>
													<div class="grid grid-cols-2 gap-3">
														<div>
															<label for="lesson_content_type" class="label text-xs">{$t('admin.courses.lesson_content_type')}</label>
															<select id="lesson_content_type" bind:value={lessonContentType} class="input text-sm">
																<option value="video">{$t('admin.courses.lesson_type_video')}</option>
																<option value="text">{$t('admin.courses.lesson_type_text')}</option>
																<option value="mixed">{$t('admin.courses.lesson_type_mixed')}</option>
																<option value="quiz">{$t('admin.courses.lesson_type_quiz')}</option>
															</select>
														</div>
														<div>
															<label for="lesson_minutes" class="label text-xs">{$t('admin.courses.lesson_estimated_minutes')}</label>
															<input type="number" id="lesson_minutes" bind:value={lessonEstimatedMinutes} class="input text-sm" min="0" />
														</div>
													</div>
													{#if lessonContentType === 'video' || lessonContentType === 'mixed'}
														<div>
															<label for="lesson_video_url" class="label text-xs">{$t('admin.courses.lesson_video_url')}</label>
															<input type="url" id="lesson_video_url" bind:value={lessonVideoUrl} class="input text-sm" placeholder={$t('admin.courses.lesson_video_url_placeholder')} />
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
																<label for="lesson_content" class="label text-xs">{$t('admin.courses.lesson_content')} (Markdown)</label>
																<button
																	type="button"
																	class="text-xs text-primary-600 dark:text-primary-400 hover:underline"
																	onclick={() => lessonPreviewMode = !lessonPreviewMode}
																>
																	{lessonPreviewMode ? $t('admin.courses.edit_mode') : $t('admin.courses.preview_mode')}
																</button>
															</div>
															{#if lessonPreviewMode}
																<div class="prose dark:prose-invert max-w-none text-sm bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700 min-h-[150px]">
																	{#if lessonContent.trim()}
																		{@html DOMPurify.sanitize(lessonContent)}
																	{:else}
																		<p class="text-gray-400 italic">{$t('admin.courses.no_content_preview')}</p>
																	{/if}
																</div>
															{:else}
																<div class="flex justify-end mb-2">
																	<AIContentHelper
																		fieldType="lesson_content"
																		content={lessonContent}
																		context={{ title: lessonTitle, parent: moduleTitle, type: 'lesson_content' }}
																		on:apply={(e) => (lessonContent = e.detail.content)}
																	/>
																</div>
																<textarea id="lesson_content" bind:value={lessonContent} class="input text-sm" rows="8" placeholder={$t('admin.courses.description_placeholder')}></textarea>
															{/if}
														</div>
													{/if}
													{#if lessonContentType === 'quiz'}
														<!-- Quiz Builder -->
														<div class="space-y-4 border border-gray-200 dark:border-gray-700 rounded-lg p-4 bg-gray-50 dark:bg-gray-800/50">
															{#if loadingQuiz}
																<div class="flex items-center justify-center py-6">
																	<svg class="animate-spin h-5 w-5 text-primary-600" fill="none" viewBox="0 0 24 24">
																		<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
																		<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
																	</svg>
																</div>
															{:else}
																<h4 class="text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2">
																	<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
																		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
																	</svg>
																	{$t('admin.courses.quiz_title')}
																</h4>
																<div>
																	<label for="quiz_title" class="label text-xs">{$t('admin.courses.quiz_title')}</label>
																	<input type="text" id="quiz_title" bind:value={quizTitle} class="input text-sm" placeholder={lessonTitle || 'Quiz title...'} />
																</div>
																<div class="grid grid-cols-3 gap-3">
																	<div>
																		<label for="quiz_passing_score" class="label text-xs">{$t('admin.courses.quiz_passing_score')}</label>
																		<div class="flex items-center gap-1">
																			<input type="number" id="quiz_passing_score" bind:value={quizPassingScore} class="input text-sm" min="0" max="100" />
																			<span class="text-xs text-gray-500">%</span>
																		</div>
																	</div>
																	<div>
																		<label for="quiz_max_attempts" class="label text-xs">{$t('admin.courses.quiz_max_attempts')}</label>
																		<input type="number" id="quiz_max_attempts" bind:value={quizMaxAttempts} class="input text-sm" min="0" />
																	</div>
																	<div class="flex items-end pb-1">
																		<Toggle bind:checked={quizRandomize} id="quiz_randomize" label={$t('admin.courses.quiz_randomize')} />
																	</div>
																</div>

																<!-- Questions List -->
																<div class="space-y-3">
																	{#each quizQuestions as question, qIndex}
																		<div class="border border-gray-200 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 p-3 space-y-3">
																			<div class="flex items-center justify-between">
																				<span class="text-xs font-medium text-gray-500 dark:text-gray-400">{$t('admin.courses.quiz_question_text')} {qIndex + 1}</span>
																				<div class="flex items-center gap-1">
																					<button type="button" class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30" disabled={qIndex === 0} onclick={() => moveQuizQuestion(qIndex, -1)} title={$t('admin.courses.reorder_modules')}>
																						<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" /></svg>
																					</button>
																					<button type="button" class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30" disabled={qIndex === quizQuestions.length - 1} onclick={() => moveQuizQuestion(qIndex, 1)} title={$t('admin.courses.reorder_modules')}>
																						<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" /></svg>
																					</button>
																					<button type="button" class="p-1 text-red-400 hover:text-red-600" onclick={() => removeQuizQuestion(qIndex)} title={$t('admin.courses.delete')}>
																						<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
																					</button>
																				</div>
																			</div>

																			<div class="grid grid-cols-2 gap-3">
																				<label class="block">
																					<span class="label text-xs">{$t('admin.courses.quiz_question_type')}</span>
																					<select class="input text-sm" value={question.question_type} onchange={(e) => setQuestionType(qIndex, (e.target as HTMLSelectElement).value as 'multiple_choice' | 'true_false')}>
																						<option value="multiple_choice">{$t('admin.courses.quiz_multiple_choice')}</option>
																						<option value="true_false">{$t('admin.courses.quiz_true_false')}</option>
																					</select>
																				</label>
																				<label class="block">
																					<span class="label text-xs">{$t('admin.courses.quiz_points')}</span>
																					<input type="number" bind:value={question.points} class="input text-sm" min="0" />
																				</label>
																			</div>

																			<div>
																				<label for="quiz-question-{qIndex}" class="label text-xs">{$t('admin.courses.quiz_question_text')}</label>
																				<textarea id="quiz-question-{qIndex}" bind:value={question.question_text} class="input text-sm" rows="2" placeholder={$t('admin.courses.quiz_question_text')}></textarea>
																			</div>

																			<!-- Options -->
																			<div class="space-y-2">
																				<span class="text-xs font-medium text-gray-500 dark:text-gray-400">{$t('admin.courses.quiz_correct_answer')}</span>
																				{#each question.options as option, oIndex}
																					<div class="flex items-center gap-2">
																						<input
																							type="radio"
																							name="correct_{qIndex}"
																							checked={option.is_correct}
																							onchange={() => setCorrectOption(qIndex, oIndex)}
																							class="text-primary-600 focus:ring-primary-500"
																						/>
																						{#if question.question_type === 'true_false'}
																							<span class="text-sm text-gray-700 dark:text-gray-300">{option.text}</span>
																						{:else}
																							<input type="text" bind:value={option.text} class="input text-sm flex-1" placeholder="Option {oIndex + 1}" />
																							{#if question.options.length > 2}
																								<button type="button" class="p-1 text-gray-400 hover:text-red-500" onclick={() => removeQuizOption(qIndex, oIndex)} title={$t('admin.courses.delete')}>
																									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
																								</button>
																							{/if}
																						{/if}
																					</div>
																				{/each}
																				{#if question.question_type === 'multiple_choice' && question.options.length < 6}
																					<button type="button" class="text-xs text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300" onclick={() => addQuizOption(qIndex)}>
																						+ {$t('admin.courses.quiz_add_option')}
																					</button>
																				{/if}
																			</div>

																			<label class="block">
																				<span class="label text-xs">{$t('admin.courses.quiz_explanation')}</span>
																				<textarea bind:value={question.explanation} class="input text-sm" rows="2" placeholder={$t('admin.courses.quiz_explanation')}></textarea>
																			</label>
																		</div>
																	{/each}
																</div>

																<button
																	type="button"
																	class="w-full text-left text-sm text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 py-2 px-3 rounded hover:bg-primary-50 dark:hover:bg-primary-900/10 transition-colors border border-dashed border-gray-300 dark:border-gray-600"
																	onclick={addQuizQuestion}
																>
																	+ {$t('admin.courses.quiz_add_question')}
																</button>
															{/if}
														</div>
													{/if}
													<Toggle bind:checked={lessonIsPreview} id="lesson_is_preview" label={$t('admin.courses.lesson_is_preview')} />
													<!-- Attachments -->
													<div>
														<label for="lesson_attachments" class="label text-xs">{$t('admin.courses.attachments_label')}</label>
														<input
															type="file"
															id="lesson_attachments"
															multiple
															accept=".pdf,.pptx,.ppt,.zip,.tar.gz,.doc,.docx,.xls,.xlsx,.txt,.md,.csv"
															onchange={(e) => { lessonAttachmentFiles = (e.target as HTMLInputElement).files; }}
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
															{@const existingAttachments = (editingLesson.lesson as any).attachments || []}
															{#if existingAttachments.length > 0}
																<div class="mt-2 space-y-1">
																	<span class="text-xs text-gray-500 dark:text-gray-400">{$t('admin.courses.attachments_label')}:</span>
																	{#each existingAttachments as attachment}
																		<div class="flex items-center gap-2 text-xs text-gray-600 dark:text-gray-400">
																			<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
																				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
																			</svg>
																			<span>{attachment}</span>
																		</div>
																	{/each}
																</div>
															{/if}
														{/if}
													</div>
													<div class="flex gap-2 justify-end">
														<button type="button" class="btn btn-ghost text-sm" onclick={closeLessonForm}>{$t('admin.courses.cancel')}</button>
														<button type="button" class="btn btn-primary text-sm" onclick={saveLesson} disabled={saving || !lessonTitle.trim()}>
															{saving ? $t('admin.courses.saving') : $t('admin.courses.save_lesson')}
														</button>
													</div>
												</div>
											{:else}
												<button
													type="button"
													class="w-full text-left text-sm text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 py-2 px-3 rounded hover:bg-primary-50 dark:hover:bg-primary-900/10 transition-colors"
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
			<div class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.courses.enrolled_students')}</h2>
					<button
						type="button"
						class="btn btn-secondary btn-sm"
						onclick={() => { showStudents = !showStudents; if (!showStudents) return; loadStudents(editingCourse!.id); }}
					>
						{$t('admin.courses.show_hide_enrollment')}
					</button>
				</div>

					{#if showStudents}
						{#if loadingStudents}
							<div class="flex items-center justify-center py-8">
								<svg class="animate-spin h-6 w-6 text-primary-600" fill="none" viewBox="0 0 24 24">
									<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
									<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
								</svg>
							</div>
						{:else}
							<!-- Stats -->
							{#if enrollmentStats}
								<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
									<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 text-center">
										<div class="text-2xl font-bold text-gray-900 dark:text-white">{enrollmentStats.total_enrolled}</div>
										<div class="text-xs text-gray-500 dark:text-gray-400">{$t('admin.courses.stat_enrolled')}</div>
									</div>
									<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 text-center">
										<div class="text-2xl font-bold text-green-600 dark:text-green-400">{enrollmentStats.completed}</div>
										<div class="text-xs text-gray-500 dark:text-gray-400">{$t('admin.courses.stat_completed')}</div>
									</div>
									<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 text-center">
										<div class="text-2xl font-bold text-primary-600 dark:text-primary-400">{Math.round(enrollmentStats.average_progress)}%</div>
										<div class="text-xs text-gray-500 dark:text-gray-400">{$t('admin.courses.stat_avg_progress')}</div>
									</div>
									<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-3 text-center">
										<div class="text-2xl font-bold text-purple-600 dark:text-purple-400">{Math.round(enrollmentStats.completion_rate)}%</div>
										<div class="text-xs text-gray-500 dark:text-gray-400">{$t('admin.courses.stat_completion_rate')}</div>
									</div>
								</div>
							{/if}

							<!-- Student table -->
							{#if students.length > 0}
								<div class="overflow-x-auto">
									<table class="w-full text-sm">
										<thead>
											<tr class="border-b border-gray-200 dark:border-gray-700">
												<th class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400">{$t('admin.courses.col_email')}</th>
												<th class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400">{$t('admin.courses.col_progress')}</th>
												<th class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400">{$t('admin.courses.col_lessons')}</th>
												<th class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400">{$t('admin.courses.col_last_active')}</th>
												<th class="text-left py-2 px-3 font-medium text-gray-500 dark:text-gray-400">{$t('admin.courses.col_certificate')}</th>
											</tr>
										</thead>
										<tbody>
											{#each students as student}
												<tr class="border-b border-gray-100 dark:border-gray-800">
													<td class="py-2 px-3 text-gray-900 dark:text-gray-100">{student.email}</td>
													<td class="py-2 px-3">
														<div class="flex items-center gap-2">
															<div class="w-20 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
																<div
																	class="h-2 rounded-full transition-all {student.progress_percent >= 100 ? 'bg-green-500' : 'bg-primary-600'}"
																	style="width: {Math.min(student.progress_percent, 100)}%"
																></div>
															</div>
															<span class="text-xs text-gray-500 dark:text-gray-400">{Math.round(student.progress_percent)}%</span>
														</div>
													</td>
													<td class="py-2 px-3 text-gray-600 dark:text-gray-400">{student.lessons_completed}/{student.total_lessons}</td>
													<td class="py-2 px-3 text-gray-600 dark:text-gray-400">
														{#if student.last_accessed}
															{new Date(student.last_accessed).toLocaleDateString()}
														{:else}
															-
														{/if}
													</td>
													<td class="py-2 px-3">
														{#if student.has_certificate}
															<span class="inline-flex items-center gap-1 text-green-600 dark:text-green-400">
																<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
																	<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
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
								<p class="text-sm text-gray-500 dark:text-gray-400 text-center py-4">{$t('admin.courses.no_students')}</p>
							{/if}
						{/if}
					{/if}
				</div>
			{/if}

			<!-- Settings tab -->
			{#if activeTab === 'settings'}
			<div class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.courses.section_publishing')}</h2>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<VisibilitySelector bind:value={visibility} />
					<AccessTierSelect bind:value={accessTier} />
				</div>

				{#if accessTier === 'paid'}
					<div>
						<label for="price" class="label">{$t('admin.courses.price')}</label>
						<div class="relative">
							<span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
							<input type="number" id="price" bind:value={priceInDollars} class="input pl-7" placeholder={$t('admin.courses.price_placeholder')} step="0.01" min="0" />
						</div>
					</div>
				{/if}

				<div class="flex items-center gap-6">
					<Toggle bind:checked={isDraft} id="is_draft" label={$t('admin.courses.is_draft')} />
					<Toggle bind:checked={sequentialAccess} id="sequential_access" label={$t('admin.courses.sequential_access')} />
				</div>
				<div class="flex items-center gap-6">
					<Toggle bind:checked={enableProgress} id="enable_progress" label={$t('admin.courses.track_progress')} />
					<Toggle bind:checked={showProgressBar} id="show_progress_bar" label={$t('admin.courses.show_progress_bar')} disabled={!enableProgress} />
					<Toggle bind:checked={requireEnrollment} id="require_enrollment" label={$t('admin.courses.require_enrollment')} />
					<Toggle bind:checked={enableCertificates} id="enable_certificates" label={$t('admin.courses.issue_certificates')} disabled={!enableProgress} />
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
			<div class="card rounded-t-none border-t border-gray-200 dark:border-gray-700 p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('admin.courses.section_details')}</h2>

				<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
					<div>
						<label for="difficulty" class="label">{$t('admin.courses.difficulty')}</label>
						<select id="difficulty" bind:value={difficulty} class="input">
							<option value="beginner">{$t('admin.courses.difficulty_beginner')}</option>
							<option value="intermediate">{$t('admin.courses.difficulty_intermediate')}</option>
							<option value="advanced">{$t('admin.courses.difficulty_advanced')}</option>
						</select>
					</div>
					<div>
						<label for="estimated_hours" class="label">{$t('admin.courses.estimated_hours')}</label>
						<input type="number" id="estimated_hours" bind:value={estimatedHours} class="input" min="0" />
					</div>
					<div>
						<label for="instructor_name" class="label">{$t('admin.courses.instructor_name')}</label>
						<input type="text" id="instructor_name" bind:value={instructorName} class="input" placeholder={$t('admin.courses.instructor_name_placeholder')} />
					</div>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="instructor_bio" class="label mb-0">{$t('admin.courses.instructor_bio')}</label>
						<AIContentHelper
							fieldType="instructor_bio"
							content={instructorBio}
							context={{ title: instructorName, type: 'instructor_bio' }}
							on:apply={(e) => (instructorBio = e.detail.content)}
						/>
					</div>
					<textarea id="instructor_bio" bind:value={instructorBio} class="input" rows="3"></textarea>
				</div>

				<div>
					<span class="label">{$t('admin.courses.tags')}</span>
					<div class="flex flex-wrap gap-2 mb-2">
						{#each tags as tag}
							<span class="inline-flex items-center gap-1 px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded text-sm">
								{tag}
								<button type="button" class="text-gray-500 hover:text-red-500" onclick={() => removeTag(tag)} aria-label={$t('admin.courses.delete')}>
									<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
								</button>
							</span>
						{/each}
					</div>
					<div class="flex gap-2">
						<input type="text" bind:value={tagInput} class="input flex-1" placeholder={$t('admin.courses.tag_placeholder')} onkeydown={handleTagKeydown} />
						<button type="button" class="btn btn-secondary" onclick={addTag}>{$t('admin.courses.add_tag')}</button>
					</div>
				</div>

				<div>
					<span class="label">{$t('admin.courses.learning_outcomes')}</span>
					<div class="space-y-1 mb-2">
						{#each learningOutcomes as outcome}
							<div class="flex items-center gap-2">
								<span class="flex-1 text-sm text-gray-700 dark:text-gray-300">{outcome}</span>
								<button type="button" onclick={() => removeOutcome(outcome)} class="text-red-600 hover:text-red-700 dark:text-red-400" aria-label={$t('admin.courses.delete')}>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
								</button>
							</div>
						{/each}
					</div>
					<div class="flex gap-2">
						<input type="text" bind:value={outcomeInput} class="input flex-1" placeholder={$t('admin.courses.learning_outcomes_placeholder')} onkeydown={handleOutcomeKeydown} />
						<button type="button" class="btn btn-secondary" onclick={addOutcome}>{$t('admin.courses.add_outcome')}</button>
					</div>
				</div>
			</div>
			{/if}

			</div><!-- /Tab panels -->

			<!-- Sticky form actions -->
			<div class="sticky bottom-0 z-10 bg-white/80 dark:bg-gray-900/80 backdrop-blur-sm border-t border-gray-200 dark:border-gray-700 -mx-6 px-6 py-3 mt-6 rounded-b-lg flex items-center justify-between">
				<div class="flex items-center gap-3">
					<button type="button" class="btn btn-ghost" onclick={closeForm}>{$t('admin.courses.back_to_list')}</button>
					{#if editingCourse}
						<button type="button" class="btn btn-danger-ghost" onclick={() => editingCourse && deleteCourse(editingCourse)}>{$t('admin.courses.delete_course')}</button>
					{/if}
				</div>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					{$t('admin.courses.save')}
				</button>
			</div>
		</form>
	{:else if courses.length === 0}
		<div class="card p-12 text-center">
			<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5z" />
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
			</svg>
			<h3 class="mt-4 text-lg font-medium text-gray-900 dark:text-white">{$t('admin.courses.no_courses')}</h3>
			<button class="btn btn-primary mt-6" onclick={openNewForm}>
				{$t('admin.courses.new_course')}
			</button>
		</div>
	{:else if filteredCourses.length === 0}
		<div class="card p-8 text-center">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('admin.courses.no_results')}</h3>
			<button class="btn btn-secondary" onclick={() => filterStore.clearAllFilters()}>
				{$t('admin.courses.clear_filters')}
			</button>
		</div>
	{:else}
		<!-- Course List (grid layout) -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each filteredCourses as course (course.id)}
				<div class="card group hover:shadow-lg transition-all duration-200 overflow-hidden {selectMode && selectedIds.has(course.id) ? 'ring-2 ring-primary-500' : ''}">
					{#if selectMode}
						<div class="absolute top-3 left-3 z-10">
							<input
								type="checkbox"
								checked={selectedIds.has(course.id)}
								onchange={() => toggleSelect(course.id)}
								class="w-5 h-5 text-primary-600 rounded border-gray-300 shadow-sm"
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
						<div class="h-20 bg-gradient-to-br from-primary-100 to-primary-50 dark:from-primary-900 dark:to-gray-800 flex items-center justify-center">
							<svg class="w-8 h-8 text-primary-300 dark:text-primary-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l9-5-9-5-9 5 9 5z" />
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 14l6.16-3.422a12.083 12.083 0 01.665 6.479A11.952 11.952 0 0012 20.055a11.952 11.952 0 00-6.824-2.998 12.078 12.078 0 01.665-6.479L12 14z" />
							</svg>
						</div>
					{/if}

					<div class="p-4">
						<!-- Status badges -->
						<div class="flex items-center gap-1.5 mb-2 flex-wrap">
							{#if course.is_draft}
								<span class="px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/50 dark:text-yellow-300 rounded-full">
									{$t('admin.courses.status_draft')}
								</span>
							{:else}
								<span class="px-2 py-0.5 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300 rounded-full">
									{$t('admin.courses.status_published')}
								</span>
							{/if}
							<span class="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-full capitalize">
								{course.visibility}
							</span>
							{#if course.access_tier === 'paid'}
								<span class="px-2 py-0.5 text-xs font-semibold bg-purple-100 text-purple-800 dark:bg-purple-900/50 dark:text-purple-300 rounded-full">
									${(course.price / 100).toFixed(2)}
								</span>
							{/if}
						</div>

						<!-- Title -->
						<h3 class="text-base font-semibold text-gray-900 dark:text-white mb-1 line-clamp-1">{course.title}</h3>

						{#if course.description}
							<p class="text-sm text-gray-500 dark:text-gray-400 mb-3 line-clamp-2">{course.description}</p>
						{/if}

						<!-- Stats row -->
						<div class="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400 mb-3">
							<span class="inline-flex items-center gap-1 capitalize">
								<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
								{course.difficulty}
							</span>
							<span class="inline-flex items-center gap-1">
								<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" /></svg>
								{getCourseModuleCount(course.id)} / {getCourseLessonCount(course.id)}
							</span>
							{#if course.estimated_hours}
								<span class="inline-flex items-center gap-1">
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
									{course.estimated_hours}h
								</span>
							{/if}
						</div>

						<!-- Actions -->
						<div class="flex items-center justify-between pt-3 border-t border-gray-100 dark:border-gray-800">
							<button
								class="text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
								onclick={() => openEditForm(course)}
							>
								{$t('admin.courses.edit')}
							</button>
							<div class="flex items-center gap-1">
								<button
									class="btn btn-ghost btn-sm"
									title={course.is_draft ? $t('admin.courses.publish') : $t('admin.courses.unpublish')}
									onclick={() => togglePublish(course)}
								>
									{#if course.is_draft}
										<svg class="w-4 h-4 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
										</svg>
									{:else}
										<svg class="w-4 h-4 text-yellow-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
										</svg>
									{/if}
								</button>
								<button
									class="btn btn-danger-ghost btn-sm"
									title={$t('admin.courses.delete')}
									onclick={() => deleteCourse(course)}
								>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
									</svg>
								</button>
							</div>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
