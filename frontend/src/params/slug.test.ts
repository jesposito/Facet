/**
 * Tests for the slug param matcher
 *
 * This file documents expected behavior of the slug matcher.
 * Run with: npx tsx src/params/slug.test.ts
 * Or use with vitest when available: npx vitest run src/params/slug.test.ts
 */

import { match, RESERVED_SLUGS } from './slug';

// Simple test runner
let passed = 0;
let failed = 0;

function test(name: string, fn: () => void) {
	try {
		fn();
		passed++;
		console.log(`✓ ${name}`);
	} catch (e) {
		failed++;
		console.log(`✗ ${name}`);
		console.log(`  ${e}`);
	}
}

function expect(value: boolean) {
	return {
		toBe(expected: boolean) {
			if (value !== expected) {
				throw new Error(`Expected ${expected} but got ${value}`);
			}
		}
	};
}

console.log('\n=== Slug Param Matcher Tests ===\n');

// Reserved slugs tests
console.log('Reserved slugs:');
test('should reject all reserved slugs', () => {
	const reservedList = Array.from(RESERVED_SLUGS);
	for (const slug of reservedList) {
		expect(match(slug)).toBe(false);
	}
});

test('should reject reserved slugs case-insensitively', () => {
	expect(match('ADMIN')).toBe(false);
	expect(match('Admin')).toBe(false);
	expect(match('API')).toBe(false);
	expect(match('Api')).toBe(false);
});

// Valid slugs tests
console.log('\nValid slugs:');
test('should accept simple slugs', () => {
	expect(match('recruiter')).toBe(true);
	expect(match('investor')).toBe(true);
	expect(match('my-resume')).toBe(true);
	expect(match('portfolio2024')).toBe(true);
});

test('should accept slugs with hyphens and underscores', () => {
	expect(match('my-resume')).toBe(true);
	expect(match('my_resume')).toBe(true);
	expect(match('my-resume-2024')).toBe(true);
	expect(match('portfolio_v2')).toBe(true);
});

test('should accept slugs starting with numbers', () => {
	expect(match('2024-resume')).toBe(true);
	expect(match('123abc')).toBe(true);
});

// Invalid slugs tests
console.log('\nInvalid slugs:');
test('should reject empty slugs', () => {
	expect(match('')).toBe(false);
});

test('should reject slugs starting with underscore', () => {
	expect(match('_hidden')).toBe(false);
	expect(match('_internal')).toBe(false);
});

test('should reject slugs starting with hyphen', () => {
	expect(match('-invalid')).toBe(false);
});

test('should reject slugs with special characters', () => {
	expect(match('my/slug')).toBe(false);
	expect(match('my.slug')).toBe(false);
	expect(match('my@slug')).toBe(false);
	expect(match('my slug')).toBe(false);
	expect(match('my#slug')).toBe(false);
});

test('should reject slugs that are too long', () => {
	const longSlug = 'a'.repeat(101);
	expect(match(longSlug)).toBe(false);
});

test('should accept slugs at the max length', () => {
	const maxSlug = 'a'.repeat(100);
	expect(match(maxSlug)).toBe(true);
});

// Security-critical tests
console.log('\nSecurity-critical reserved slugs:');
test('should reject admin paths', () => {
	expect(match('admin')).toBe(false);
});

test('should reject api path', () => {
	expect(match('api')).toBe(false);
});

test('should reject share link path', () => {
	expect(match('s')).toBe(false);
});

test('should reject legacy view path', () => {
	expect(match('v')).toBe(false);
});

test('should reject auth-related paths', () => {
	expect(match('login')).toBe(false);
	expect(match('logout')).toBe(false);
	expect(match('auth')).toBe(false);
	expect(match('oauth')).toBe(false);
	expect(match('callback')).toBe(false);
});

// Summary
console.log(`\n=== Results: ${passed} passed, ${failed} failed ===\n`);

if (failed > 0) {
	process.exit(1);
}
