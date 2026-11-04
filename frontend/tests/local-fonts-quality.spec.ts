import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import { FONT_PACKS, type FontPack } from '../src/lib/fonts';
import { loginAsAdmin } from './helpers';
import { apiBaseURL } from './config';

const PACK_FACES: Record<FontPack, Array<[string, number, string]>> = {
	editorial: [
		...['normal', 'italic'].flatMap((style) =>
			[400, 500, 600, 700].map((weight) => ['Lora', weight, style] as [string, number, string])
		),
		...[400, 500, 600, 700].map(
			(weight) => ['Plus Jakarta Sans', weight, 'normal'] as [string, number, string]
		),
		...[400, 500].map((weight) => ['JetBrains Mono', weight, 'normal'] as [string, number, string])
	],
	modern: [
		...[400, 500, 600, 700].map(
			(weight) => ['Inter', weight, 'normal'] as [string, number, string]
		),
		...[400, 500].map((weight) => ['JetBrains Mono', weight, 'normal'] as [string, number, string])
	],
	classic: [
		...[400, 500, 600, 700].map(
			(weight) => ['Playfair Display', weight, 'normal'] as [string, number, string]
		),
		...[400, 500, 600, 700].map(
			(weight) => ['Source Sans 3', weight, 'normal'] as [string, number, string]
		),
		...[400, 500].map((weight) => ['Source Code Pro', weight, 'normal'] as [string, number, string])
	],
	technical: [
		...[400, 500, 600, 700].map(
			(weight) => ['IBM Plex Sans', weight, 'normal'] as [string, number, string]
		),
		...[400, 500].map((weight) => ['IBM Plex Mono', weight, 'normal'] as [string, number, string])
	],
	'editorial-bold': [
		...[400, 500, 600, 700].map((weight) => ['Sora', weight, 'normal'] as [string, number, string]),
		...[400, 500, 600, 700].map(
			(weight) => ['Nunito Sans', weight, 'normal'] as [string, number, string]
		),
		...[400, 500].map((weight) => ['Fira Code', weight, 'normal'] as [string, number, string])
	],
	humanist: [
		...[400, 700].map((weight) => ['Merriweather', weight, 'normal'] as [string, number, string]),
		...[400, 700].map((weight) => ['Lato', weight, 'normal'] as [string, number, string]),
		...[400, 500].map((weight) => ['Source Code Pro', weight, 'normal'] as [string, number, string])
	],
	geometric: [
		...[400, 500, 600, 700].map(
			(weight) => ['Space Grotesk', weight, 'normal'] as [string, number, string]
		),
		...[400, 500, 600, 700].map(
			(weight) => ['DM Sans', weight, 'normal'] as [string, number, string]
		),
		...[400, 500].map((weight) => ['DM Mono', weight, 'normal'] as [string, number, string])
	],
	'soft-premium': [
		...[400, 500, 600, 700].map(
			(weight) => ['Hanken Grotesk', weight, 'normal'] as [string, number, string]
		),
		...['normal', 'italic'].flatMap((style) =>
			[400, 500].map((weight) => ['Newsreader', weight, style] as [string, number, string])
		),
		...[400, 500].map((weight) => ['JetBrains Mono', weight, 'normal'] as [string, number, string])
	]
};

async function setOwnerFontPack(
	request: APIRequestContext,
	token: string,
	profileId: string,
	fontPack: string
) {
	const response = await request.patch(
		apiBaseURL + '/api/collections/profile/records/' + profileId,
		{ headers: { Authorization: token }, data: { font_pack: fontPack } }
	);
	expect(response.ok(), `owner font-pack update failed: ${response.status()}`).toBe(true);
}

async function expectLoadedLocalFont(page: Page, family: string, weight: number, style: string) {
	const alreadyLoaded = await page.evaluate(
		({ family, weight, style }) =>
			[...document.fonts].some(
				(face) =>
					face.family.replace(/^['"]|['"]$/g, '') === family &&
					face.weight === String(weight) &&
					face.style === style &&
					face.status === 'loaded'
			),
		{ family, weight, style }
	);
	if (alreadyLoaded) return;

	const fontResponse = page.waitForResponse(
		(response) =>
			response.request().resourceType() === 'font' &&
			new URL(response.url()).origin === new URL(page.url()).origin &&
			response.ok()
	);
	const loaded = await page.evaluate(
		async ({ family, weight, style }) => {
			const cssFont = `${style} ${weight} 16px "${family}"`;
			await document.fonts.load(cssFont, 'Facet local font regression');
			await document.fonts.ready;
			return [...document.fonts].some(
				(face) =>
					face.family.replace(/^['"]|['"]$/g, '') === family &&
					face.weight === String(weight) &&
					face.style === style &&
					face.status === 'loaded'
			);
		},
		{ family, weight, style }
	);
	expect(loaded, `${family} ${weight} ${style} is not a loaded FontFace`).toBe(true);
	await (await fontResponse).finished();
}

test('all typography is loaded from local assets without Google Fonts', async ({
	browser,
	baseURL,
	request: ownerRequest
}) => {
	test.setTimeout(180_000);
	const { token } = await loginAsAdmin(ownerRequest);
	const profileResponse = await ownerRequest.get(
		apiBaseURL + '/api/collections/profile/records?perPage=1',
		{ headers: { Authorization: token } }
	);
	expect(profileResponse.ok()).toBe(true);
	const profileList = (await profileResponse.json()) as {
		items: Array<{ id: string; font_pack?: string }>;
	};
	const profile = profileList.items[0];
	expect(profile?.id).toBeTruthy();
	const originalFontPack = profile.font_pack || '';

	const googleRequests: string[] = [];
	const context = await browser.newContext({ baseURL });
	await context.route(/https:\/\/fonts\.(googleapis|gstatic)\.com\//, async (route) => {
		googleRequests.push(route.request().url());
		await route.abort();
	});

	try {
		const page = await context.newPage();
		await page.goto('/admin/login');
		for (const family of ['Lora', 'Plus Jakarta Sans', 'Inter', 'JetBrains Mono']) {
			await expectLoadedLocalFont(page, family, 400, 'normal');
		}

		for (const packName of Object.keys(FONT_PACKS) as FontPack[]) {
			await setOwnerFontPack(ownerRequest, token, profile.id, packName);
			await page.goto(`/?local-font-pack=${packName}`);
			const expectedFamily = FONT_PACKS[packName].body;
			await expect
				.poll(() =>
					page.evaluate(() =>
						getComputedStyle(document.documentElement).getPropertyValue('--font-body')
					)
				)
				.toContain(expectedFamily);
			for (const [family, weight, style] of PACK_FACES[packName]) {
				await expectLoadedLocalFont(page, family, weight, style);
			}
		}
		expect(googleRequests).toEqual([]);
	} finally {
		await setOwnerFontPack(ownerRequest, token, profile.id, originalFontPack);
		await context.close();
	}
});
