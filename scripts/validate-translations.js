#!/usr/bin/env node
/**
 * Translation Validation Script
 *
 * Validates translation files against the English source.
 * Reports missing keys, extra keys, and coverage percentage.
 *
 * Usage:
 *   node scripts/validate-translations.js           # Validate all locales
 *   node scripts/validate-translations.js es        # Validate specific locale
 *   node scripts/validate-translations.js --ci      # Exit with error code if issues found
 */

import { readFileSync, readdirSync, existsSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const localesDir = join(__dirname, '../frontend/src/locales');
const sourceLocale = 'en';

/**
 * Recursively extract all keys from a nested object
 */
function getKeys(obj, prefix = '') {
	return Object.entries(obj).flatMap(([key, value]) => {
		const fullKey = prefix ? `${prefix}.${key}` : key;
		if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
			return getKeys(value, fullKey);
		}
		return [fullKey];
	});
}

/**
 * Validate a single locale against the source
 */
function validateLocale(locale) {
	const sourcePath = join(localesDir, `${sourceLocale}.json`);
	const targetPath = join(localesDir, `${locale}.json`);

	if (!existsSync(sourcePath)) {
		console.error(`Error: Source file ${sourceLocale}.json not found`);
		return { valid: false, missing: [], extra: [] };
	}

	if (!existsSync(targetPath)) {
		console.error(`Error: Target file ${locale}.json not found`);
		return { valid: false, missing: [], extra: [] };
	}

	let source, target;
	try {
		source = JSON.parse(readFileSync(sourcePath, 'utf-8'));
		target = JSON.parse(readFileSync(targetPath, 'utf-8'));
	} catch (err) {
		console.error(`Error parsing JSON: ${err.message}`);
		return { valid: false, missing: [], extra: [] };
	}

	const sourceKeys = new Set(getKeys(source));
	const targetKeys = new Set(getKeys(target));

	const missing = [...sourceKeys].filter((k) => !targetKeys.has(k));
	const extra = [...targetKeys].filter((k) => !sourceKeys.has(k));
	const translated = [...sourceKeys].filter((k) => targetKeys.has(k));
	const coverage = ((translated.length / sourceKeys.size) * 100).toFixed(1);

	console.log(`\n${'='.repeat(50)}`);
	console.log(`  ${locale}.json`);
	console.log(`${'='.repeat(50)}`);
	console.log(`  Source keys:     ${sourceKeys.size}`);
	console.log(`  Translated:      ${translated.length}`);
	console.log(`  Coverage:        ${coverage}%`);

	if (missing.length > 0) {
		console.log(`\n  Missing keys (${missing.length}):`);
		missing.slice(0, 20).forEach((k) => console.log(`    - ${k}`));
		if (missing.length > 20) {
			console.log(`    ... and ${missing.length - 20} more`);
		}
	}

	if (extra.length > 0) {
		console.log(`\n  Extra keys (${extra.length}):`);
		extra.slice(0, 10).forEach((k) => console.log(`    - ${k}`));
		if (extra.length > 10) {
			console.log(`    ... and ${extra.length - 10} more`);
		}
	}

	const valid = missing.length === 0 && extra.length === 0;
	if (valid) {
		console.log(`\n  Status: OK`);
	} else {
		console.log(`\n  Status: Issues found`);
	}

	return { valid, missing, extra, coverage };
}

/**
 * Get all locale files (excluding source)
 */
function getLocales() {
	return readdirSync(localesDir)
		.filter((f) => f.endsWith('.json') && f !== `${sourceLocale}.json`)
		.map((f) => f.replace('.json', ''));
}

// Main execution
const args = process.argv.slice(2);
const ciMode = args.includes('--ci');
const specificLocale = args.find((a) => !a.startsWith('--'));

console.log('\nFacet Translation Validator');
console.log(`Source: ${sourceLocale}.json`);

let allValid = true;

if (specificLocale) {
	const result = validateLocale(specificLocale);
	allValid = result.valid;
} else {
	const locales = getLocales();
	if (locales.length === 0) {
		console.log('\nNo translation files found (only source exists).');
		console.log('Add translations by copying en.json to a new locale file.');
	} else {
		for (const locale of locales) {
			const result = validateLocale(locale);
			if (!result.valid) allValid = false;
		}
	}
}

console.log('\n');

if (ciMode && !allValid) {
	process.exit(1);
}
