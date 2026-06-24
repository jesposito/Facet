import { test, expect, type APIRequestContext } from '@playwright/test';
import { apiBaseURL } from './config';

/**
 * AAA proof for the operator-settable text (font) color.
 *
 * The whole promise of the feature is: whatever hue the operator picks — even a
 * deliberately terrible one (pale yellow, mid gray) — the ACTUAL rendered text
 * on the public page clears WCAG AAA (7:1) against its background, in BOTH light
 * and dark mode, BY CONSTRUCTION.
 *
 * This test drives the real bundled CSS + the real SSR injection (no mocks):
 *  1. PATCH profile.text_color to a bad pick.
 *  2. Load the public page.
 *  3. Read the COMPUTED color of a heading AND a body paragraph and compute the
 *     contrast ratio in-test against their COMPUTED background.
 *  4. Repeat in dark mode.
 *  5. Assert the default (no text_color) is unchanged, and that a known surface
 *     element is NOT tinted (stays the neutral stone value).
 *
 * The PB superuser PATCHes the field; the public render re-derives the AAA clamp.
 */

const PB_SUPERUSER = process.env.PB_SUPERUSER_EMAIL || 'admin@localhost.dev';
const PB_SUPERUSER_PW = process.env.PB_SUPERUSER_PASSWORD || 'admin123';

// Two deliberately bad picks: pale yellow (worst case for dark-on-light) and a
// mid gray (near the worst case for BOTH modes). Both must end up AAA.
const BAD_PICKS = ['#ffe066', '#808080'];

function srgbLuminance(r: number, g: number, b: number): number {
	const f = (c: number) => {
		const cc = c / 255;
		return cc <= 0.03928 ? cc / 12.92 : Math.pow((cc + 0.055) / 1.055, 2.4);
	};
	return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
}

function parseRgb(s: string): [number, number, number] {
	const m = s.match(/\d+(\.\d+)?/g);
	if (!m || m.length < 3) throw new Error(`Cannot parse color: ${s}`);
	return [Number(m[0]), Number(m[1]), Number(m[2])];
}

function contrast(c1: string, c2: string): number {
	const l1 = srgbLuminance(...parseRgb(c1));
	const l2 = srgbLuminance(...parseRgb(c2));
	const hi = Math.max(l1, l2);
	const lo = Math.min(l1, l2);
	return (hi + 0.05) / (lo + 0.05);
}

async function superuserToken(request: APIRequestContext): Promise<string> {
	const res = await request.post(
		`${apiBaseURL}/api/collections/_superusers/auth-with-password`,
		{ data: { identity: PB_SUPERUSER, password: PB_SUPERUSER_PW } }
	);
	expect(res.ok(), 'PB superuser auth must succeed').toBeTruthy();
	return (await res.json()).token as string;
}

async function profileId(request: APIRequestContext, token: string): Promise<string> {
	const res = await request.get(`${apiBaseURL}/api/collections/profile/records?perPage=1`, {
		headers: { Authorization: token }
	});
	expect(res.ok()).toBeTruthy();
	const items = (await res.json()).items as Array<{ id: string }>;
	expect(items.length, 'a profile record must exist').toBeGreaterThan(0);
	return items[0].id;
}

async function setTextColor(
	request: APIRequestContext,
	token: string,
	id: string,
	hex: string
): Promise<void> {
	const res = await request.patch(`${apiBaseURL}/api/collections/profile/records/${id}`, {
		headers: { Authorization: token },
		data: { text_color: hex }
	});
	expect(res.ok(), `PATCH text_color=${hex || '(empty)'} must succeed`).toBeTruthy();
}

/**
 * Measure computed color + background for a heading and a body paragraph on the
 * currently loaded public page, in the given theme. Appends a real product-class
 * body paragraph into <main> so the assertion does not depend on seeded content,
 * while still exercising the exact scoped CSS rule the product ships.
 */
async function measure(page: import('@playwright/test').Page, dark: boolean) {
	return page.evaluate((isDark) => {
		document.documentElement.classList.toggle('dark', isDark);

		const bgOf = (el: Element | null): string => {
			let e: Element | null = el;
			while (e) {
				const c = getComputedStyle(e).backgroundColor;
				if (c && c !== 'rgba(0, 0, 0, 0)' && c !== 'transparent') return c;
				e = e.parentElement;
			}
			return getComputedStyle(document.documentElement).backgroundColor;
		};

		// Heading: a real section/article heading (text-gray-900 / dark:text-white).
		const heading = document.querySelector('article h3, h2.section-title, main h1');

		// Body: append a product-class paragraph into <main> so we always have a
		// body target on a real surface, styled by the real scoped CSS rules.
		const main = document.querySelector('main') || document.body;
		let para = document.getElementById('aaa-test-body') as HTMLParagraphElement | null;
		if (!para) {
			para = document.createElement('p');
			para.id = 'aaa-test-body';
			para.className = 'prose-custom text-gray-700 dark:text-gray-300';
			para.textContent = 'Body paragraph text used for the AAA contrast assertion.';
			const card = document.createElement('div');
			card.className = 'card p-4';
			card.appendChild(para);
			main.appendChild(card);
		}

		// On-bg body: append a SECOND product-class paragraph directly to <main>
		// (no .card wrapper) so it inherits the warm page --bg, which is the
		// worst-case (lowest-contrast) surface for dark-on-light text. bgOf walks
		// up past <main> (no background) to documentElement, which carries --bg.
		let paraOnBg = document.getElementById('aaa-test-body-onbg') as HTMLParagraphElement | null;
		if (!paraOnBg) {
			paraOnBg = document.createElement('p');
			paraOnBg.id = 'aaa-test-body-onbg';
			paraOnBg.className = 'prose-custom text-gray-700 dark:text-gray-300';
			paraOnBg.textContent = 'Body paragraph text on the warm page background for the AAA contrast assertion.';
			main.appendChild(paraOnBg);
		}

		const surface = document.querySelector('.card, article');

		return {
			headingColor: heading ? getComputedStyle(heading).color : null,
			headingBg: bgOf(heading),
			bodyColor: getComputedStyle(para).color,
			bodyBg: bgOf(para),
			bodyOnBgColor: getComputedStyle(paraOnBg).color,
			bodyOnBgBg: bgOf(paraOnBg),
			surfaceBg: surface ? getComputedStyle(surface).backgroundColor : null,
			textInk: getComputedStyle(document.documentElement).getPropertyValue('--text-ink').trim(),
			hasAttr: document.documentElement.hasAttribute('data-text-ink')
		};
	}, dark);
}

test.describe('operator text color is AAA by construction', () => {
	let token: string;
	let id: string;

	test.beforeAll(async ({ request }) => {
		token = await superuserToken(request);
		id = await profileId(request, token);
	});

	test.afterAll(async ({ request }) => {
		// Always reset so the live instance / other specs see the default.
		await setTextColor(request, token, id, '');
	});

	test('default (no text_color) leaves text + surfaces untinted', async ({ request, page }) => {
		await setTextColor(request, token, id, '');
		await page.goto('/', { waitUntil: 'networkidle' });

		const light = await measure(page, false);
		expect(light.hasAttr, 'no data-text-ink attribute when unset').toBeFalsy();
		expect(light.textInk, '--text-ink unset by default').toBe('');
		// Surface stays the neutral stone white (#ffffff) — not tinted.
		expect(light.surfaceBg).toBe('rgb(255, 255, 255)');
	});

	for (const pick of BAD_PICKS) {
		test(`bad pick ${pick}: heading + body clear 7:1 in light AND dark`, async ({
			request,
			page
		}) => {
			await setTextColor(request, token, id, pick);
			await page.goto('/', { waitUntil: 'networkidle' });

			// ---- Light mode ----
			const light = await measure(page, false);
			expect(light.hasAttr, 'data-text-ink present when color set').toBeTruthy();
			expect(light.textInk, '--text-ink injected').toMatch(/^#[0-9a-fA-F]{6}$/);
			expect(light.headingColor, 'heading present').not.toBeNull();

			const lightHeading = contrast(light.headingColor!, light.headingBg);
			const lightBody = contrast(light.bodyColor, light.bodyBg);
			expect(
				lightHeading,
				`light heading ${light.headingColor} on ${light.headingBg} = ${lightHeading.toFixed(2)}:1`
			).toBeGreaterThanOrEqual(6.95);
			expect(
				lightBody,
				`light body ${light.bodyColor} on ${light.bodyBg} = ${lightBody.toFixed(2)}:1`
			).toBeGreaterThanOrEqual(6.95);

			// Worst-case surface: body text directly on the warm page --bg (#fbf8f4),
			// not a white card. The clamp targets exactly this surface, so hold it to
			// the strict 7.0 floor.
			const lightBodyOnBg = contrast(light.bodyOnBgColor, light.bodyOnBgBg);
			expect(
				lightBodyOnBg,
				`light body on page bg ${light.bodyOnBgColor} on ${light.bodyOnBgBg} = ${lightBodyOnBg.toFixed(2)}:1`
			).toBeGreaterThanOrEqual(7.0);

			// Surface must NOT be tinted — still neutral stone white.
			expect(light.surfaceBg, 'surface stays neutral in light mode').toBe('rgb(255, 255, 255)');

			// ---- Dark mode ----
			const dark = await measure(page, true);
			const darkHeading = contrast(dark.headingColor!, dark.headingBg);
			const darkBody = contrast(dark.bodyColor, dark.bodyBg);
			expect(
				darkHeading,
				`dark heading ${dark.headingColor} on ${dark.headingBg} = ${darkHeading.toFixed(2)}:1`
			).toBeGreaterThanOrEqual(6.95);
			expect(
				darkBody,
				`dark body ${dark.bodyColor} on ${dark.bodyBg} = ${darkBody.toFixed(2)}:1`
			).toBeGreaterThanOrEqual(6.95);

			// Worst-case surface in dark mode: body text directly on the page --bg
			// (#1a1613), not a card. Hold it to the strict 7.0 floor.
			const darkBodyOnBg = contrast(dark.bodyOnBgColor, dark.bodyOnBgBg);
			expect(
				darkBodyOnBg,
				`dark body on page bg ${dark.bodyOnBgColor} on ${dark.bodyOnBgBg} = ${darkBodyOnBg.toFixed(2)}:1`
			).toBeGreaterThanOrEqual(7.0);
		});
	}
});
