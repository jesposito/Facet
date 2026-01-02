/**
 * Homepage route: /
 *
 * Renders the default public profile view. Behavior:
 * 1. Check if a default view is configured (is_default = true)
 * 2. If yes: render that view's curated content
 * 3. If no: fallback to legacy homepage (all public content aggregated)
 *
 * This route does NOT handle password or unlisted views at the root.
 * Those must be accessed via /<slug> with proper tokens.
 */

import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	console.log('[ROOT PAGE] ========== LOAD START ==========');

	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';
	console.log('[ROOT PAGE] pbUrl:', pbUrl);

	try {
		// Check if there's a default view configured
		console.log('[ROOT PAGE] Fetching default-view...');
		const defaultViewResponse = await fetch(`${pbUrl}/api/default-view`);
		console.log('[ROOT PAGE] default-view response status:', defaultViewResponse.status);

		if (defaultViewResponse.ok) {
			const defaultViewInfo = await defaultViewResponse.json();
			console.log('[ROOT PAGE] default-view info:', JSON.stringify(defaultViewInfo));

			if (defaultViewInfo.homepage_enabled === false) {
				console.log('[ROOT PAGE] Homepage disabled via settings');
				return {
					homepageDisabled: true,
					landingPageMessage: defaultViewInfo.landing_page_message || ''
				};
			}

			if (defaultViewInfo.has_default && defaultViewInfo.slug) {
				console.log('[ROOT PAGE] Has default view, fetching view data for slug:', defaultViewInfo.slug);
				// Fetch the default view's data
				const viewDataResponse = await fetch(`${pbUrl}/api/view/${defaultViewInfo.slug}/data`);
				console.log('[ROOT PAGE] view data response status:', viewDataResponse.status);

				if (viewDataResponse.ok) {
					const viewData = await viewDataResponse.json();
					console.log('[ROOT PAGE] view data received, sections:', Object.keys(viewData.sections || {}));

					// Return data in a format compatible with both view and homepage layouts
					const profile = viewData.profile || null;
					console.log('[ROOT PAGE] Profile from view:', profile ? profile.name : 'null');

					// Get posts and talks from view sections
					let posts = viewData.sections?.posts || [];
					let talks = viewData.sections?.talks || [];
					console.log('[ROOT PAGE] From view sections - posts:', posts.length, 'talks:', talks.length);

					// If posts/talks aren't in the view's sections, fetch them from the homepage API
					// This ensures posts/talks always appear on the profile even if the view
					// doesn't explicitly include them as sections
					if (posts.length === 0 || talks.length === 0) {
						console.log('[ROOT PAGE] Posts/talks empty, fetching from homepage API...');
						try {
							const homepageResponse = await fetch(`${pbUrl}/api/homepage`);
							console.log('[ROOT PAGE] homepage fallback response status:', homepageResponse.status);
							if (homepageResponse.ok) {
								const homepageData = await homepageResponse.json();
								console.log('[ROOT PAGE] homepage data - posts:', (homepageData.posts || []).length, 'talks:', (homepageData.talks || []).length);
								if (posts.length === 0 && homepageData.posts) {
									posts = homepageData.posts.map((p: Record<string, unknown>) => ({
										...p,
										cover_image: p.cover_image_url || null
									}));
									console.log('[ROOT PAGE] Populated posts from homepage:', posts.length);
								}
								if (talks.length === 0 && homepageData.talks) {
									talks = homepageData.talks;
									console.log('[ROOT PAGE] Populated talks from homepage:', talks.length);
								}
							}
						} catch (fallbackErr) {
							console.log('[ROOT PAGE] Homepage fallback error (ignored):', fallbackErr);
						}
					}

					const result = {
						// View-specific data
						view: {
							id: viewData.id,
							slug: viewData.slug,
							name: viewData.name,
							hero_headline: viewData.hero_headline,
							hero_summary: viewData.hero_summary,
							cta_text: viewData.cta_text,
							cta_url: viewData.cta_url,
							accent_color: viewData.accent_color || null
						},
						// Profile with resolved file URLs
						profile: profile
							? {
									...profile,
									hero_image: profile.hero_image_url || null,
									avatar: profile.avatar_url || null
								}
							: null,
						// Section data from view
						sections: viewData.sections || {},
						// Legacy fields for backwards compat (from sections)
						experience: viewData.sections?.experience || [],
						projects: viewData.sections?.projects || [],
						education: viewData.sections?.education || [],
						certifications: viewData.sections?.certifications || [],
						skills: viewData.sections?.skills || [],
						posts,
						talks,
						// Indicate this is a view-based homepage
						isDefaultView: true,
						homepageDisabled: false,
						landingPageMessage: defaultViewInfo.landing_page_message || ''
					};
					console.log('[ROOT PAGE] Returning DEFAULT VIEW data:');
					console.log('[ROOT PAGE]   view.slug:', result.view.slug);
					console.log('[ROOT PAGE]   profile:', result.profile?.name);
					console.log('[ROOT PAGE]   experience:', result.experience.length);
					console.log('[ROOT PAGE]   projects:', result.projects.length);
					console.log('[ROOT PAGE]   education:', result.education.length);
					console.log('[ROOT PAGE]   certifications:', result.certifications.length);
					console.log('[ROOT PAGE]   skills:', result.skills.length);
					console.log('[ROOT PAGE]   posts:', result.posts.length);
					console.log('[ROOT PAGE]   talks:', result.talks.length);
					console.log('[ROOT PAGE] ========== LOAD END (default view) ==========');
					return result;
				}
			}
		}

		// Fallback: No default view or failed to load - use legacy homepage
		console.log('[ROOT PAGE] No default view, using legacy homepage API...');
		const response = await fetch(`${pbUrl}/api/homepage`);
		console.log('[ROOT PAGE] homepage response status:', response.status);

		if (!response.ok) {
			const errorText = await response.text();
			console.error('[ROOT PAGE] Homepage API error:', response.status, errorText);
			return {
				profile: null,
				experience: [],
				projects: [],
				education: [],
				certifications: [],
				skills: [],
				posts: [],
				talks: [],
				view: null,
				error: 'Failed to load profile',
				isDefaultView: false,
				homepageDisabled: false,
				landingPageMessage: ''
			};
		}

		const data = await response.json();
		if (data.homepage_enabled === false) {
			return {
				homepageDisabled: true,
				landingPageMessage: data.landing_page_message || '',
				profile: null,
				experience: [],
				projects: [],
				education: [],
				certifications: [],
				skills: [],
				posts: [],
				talks: [],
				view: null,
				isDefaultView: false
			};
		}
		console.log('[ROOT PAGE] Homepage data received:');
		console.log('[ROOT PAGE]   profile:', data.profile?.name || 'null');
		console.log('[ROOT PAGE]   experience:', (data.experience || []).length);
		console.log('[ROOT PAGE]   projects:', (data.projects || []).length);
		console.log('[ROOT PAGE]   education:', (data.education || []).length);
		console.log('[ROOT PAGE]   skills:', (data.skills || []).length);
		console.log('[ROOT PAGE]   posts:', (data.posts || []).length);
		console.log('[ROOT PAGE]   talks:', (data.talks || []).length);
		console.log('[ROOT PAGE]   certifications:', (data.certifications || []).length);

		// Check if profile is private
		if (!data.profile) {
			console.log('[ROOT PAGE] No profile in response - profile is private or missing');
			return {
				profile: null,
				experience: [],
				projects: [],
				education: [],
				certifications: [],
				skills: [],
				posts: [],
				talks: [],
				view: null,
				error: 'Profile is private',
				isDefaultView: false
			};
		}

		// Map API response to expected format
		const profile = data.profile
			? {
					...data.profile,
					hero_image: data.profile.hero_image_url || null,
					avatar: data.profile.avatar_url || null
				}
			: null;

		const projects = (data.projects || []).map((p: Record<string, unknown>) => ({
			...p,
			cover_image: p.cover_image_url || null
		}));

		const posts = (data.posts || []).map((p: Record<string, unknown>) => ({
			...p,
			cover_image: p.cover_image_url || null
		}));

		const result = {
			profile,
			experience: data.experience || [],
			projects,
			education: data.education || [],
			certifications: data.certifications || [],
			skills: data.skills || [],
			posts,
			talks: data.talks || [],
			// Indicate this is legacy homepage mode
			view: null,
			isDefaultView: false
		};
		console.log('[ROOT PAGE] Returning LEGACY homepage data:');
		console.log('[ROOT PAGE]   isDefaultView:', result.isDefaultView);
		console.log('[ROOT PAGE]   profile:', result.profile?.name);
		console.log('[ROOT PAGE]   experience:', result.experience.length);
		console.log('[ROOT PAGE]   projects:', result.projects.length);
		console.log('[ROOT PAGE]   posts:', result.posts.length);
		console.log('[ROOT PAGE]   talks:', result.talks.length);
		console.log('[ROOT PAGE] ========== LOAD END (legacy) ==========');
		return result;
	} catch (error) {
		console.error('[ROOT PAGE] EXCEPTION:', error);
		return {
			profile: null,
			experience: [],
			projects: [],
			education: [],
			certifications: [],
			skills: [],
			posts: [],
			talks: [],
			view: null,
			error: 'Failed to load profile',
			isDefaultView: false
		};
	}
};
