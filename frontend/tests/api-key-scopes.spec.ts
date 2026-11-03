import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import {
	availableScopes,
	readScopes,
	writeScopes,
	defaultScopes,
	isWriteScope,
	parseOriginsInput
} from '../src/lib/apiKeyScopes';

// These checks need no browser or running server.
const here = (p: string) => fileURLToPath(new URL(p, import.meta.url));
const en = JSON.parse(readFileSync(here('../src/locales/en.json'), 'utf8'));
const pageSource = readFileSync(here('../src/routes/admin/api/+page.svelte'), 'utf8');

function backendScopes(): string[] {
	const go = readFileSync(here('../../backend/hooks/api_keys.go'), 'utf8');
	const block = go.match(/var validScopes = map\[string\]bool\{([\s\S]*?)\n\}/);
	expect(block, 'validScopes block in api_keys.go').not.toBeNull();
	return [...block![1].matchAll(/"([a-z]+:[a-z]+)":\s*true/g)].map((m) => m[1]).sort();
}

test.describe('API key scopes', () => {
	test('admin UI offers exactly the scopes the backend accepts', () => {
		expect([...availableScopes].sort()).toEqual(backendScopes());
	});

	test('write scopes are offered, grouped apart from read scopes', () => {
		expect(writeScopes.length).toBeGreaterThan(0);
		expect(writeScopes.every(isWriteScope)).toBe(true);
		expect(readScopes.some(isWriteScope)).toBe(false);
		expect(pageSource).toContain('scopeGroups');
		expect(pageSource).not.toMatch(/const availableScopes\s*=/);
	});

	test('a new key grants no write access until it is ticked', () => {
		expect(defaultScopes.length).toBeGreaterThan(0);
		expect(defaultScopes.some(isWriteScope)).toBe(false);
	});

	test('every scope and scope group has an English label', () => {
		const keys = en.admin.api_keys;
		for (const scope of availableScopes) {
			expect(keys.scope_descriptions[scope], scope).toBeTruthy();
		}
		for (const kind of ['read', 'write']) {
			expect(keys[`scope_group_${kind}`]).toBeTruthy();
			expect(keys[`scope_group_${kind}_help`]).toBeTruthy();
		}
		expect(keys.write_scope_warning).toBeTruthy();
		expect(keys.field_origins_help).toMatch(/Origin header/);
	});

	test('allowed origins input splits on lines and commas', () => {
		expect(
			parseOriginsInput(' https://a.example \n\nhttps://b.example, http://localhost:5173 ')
		).toEqual(['https://a.example', 'https://b.example', 'http://localhost:5173']);
		expect(parseOriginsInput('   ')).toEqual([]);
	});
});
