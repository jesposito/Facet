export const apiBaseURL =
	process.env.API_BASE_URL ||
	process.env.POCKETBASE_URL ||
	'http://localhost:8090';

export const adminEmail = process.env.ADMIN_EMAIL || '';
export const adminPassword = process.env.ADMIN_PASSWORD || '';
