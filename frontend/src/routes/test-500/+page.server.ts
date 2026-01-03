import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
	// Intentionally throw a 500 error for testing
	throw error(500, 'Test server error - this route intentionally triggers a 500');
};
