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
	const pbUrl = process.env.POCKETBASE_URL || 'http://localhost:8090';

	try {
		// Check if there's a default view configured
		const defaultViewResponse = await fetch(`${pbUrl}/api/default-view`);

		if (defaultViewResponse.ok) {
			const defaultViewInfo = await defaultViewResponse.json();

			if (defaultViewInfo.has_default && defaultViewInfo.slug) {
				// Fetch the default view's data
				const viewDataResponse = await fetch(`${pbUrl}/api/view/${defaultViewInfo.slug}/data`);

				if (viewDataResponse.ok) {
					const viewData = await viewDataResponse.json();

					// Return data in a format compatible with both view and homepage layouts
					const profile = viewData.profile || null;

					return {
						// View-specific data
						view: {
							id: viewData.id,
							slug: viewData.slug,
							name: viewData.name,
							hero_headline: viewData.hero_headline,
							hero_summary: viewData.hero_summary,
							cta_text: viewData.cta_text,
							cta_url: viewData.cta_url
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
						skills: viewData.sections?.skills || [],
						posts: viewData.sections?.posts || [],
						// Indicate this is a view-based homepage
						isDefaultView: true
					};
				}
			}
		}

		// Fallback: No default view or failed to load - use legacy homepage
		const response = await fetch(`${pbUrl}/api/homepage`);

		if (!response.ok) {
			console.error('Homepage API error:', response.status);
			return {
				profile: null,
				experience: [],
				projects: [],
				education: [],
				skills: [],
				posts: [],
				error: 'Failed to load profile'
			};
		}

		const data = await response.json();

		// Check if profile is private
		if (!data.profile) {
			return {
				profile: null,
				experience: [],
				projects: [],
				education: [],
				skills: [],
				posts: [],
				error: 'Profile is private'
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

		return {
			profile,
			experience: data.experience || [],
			projects,
			education: data.education || [],
			skills: data.skills || [],
			posts,
			// Indicate this is legacy homepage mode
			isDefaultView: false
		};
	} catch (error) {
		console.error('Failed to load profile data:', error);
		return {
			profile: null,
			experience: [],
			projects: [],
			education: [],
			skills: [],
			posts: [],
			error: 'Failed to load profile'
		};
	}
};
