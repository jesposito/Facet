#!/usr/bin/env node
/**
 * i18n values-param guard.
 *
 * Scans frontend/src/locales/en.json for interpolation placeholders, then
 * checks literal $t('key') callsites under frontend/src. If a translated string
 * declares {count}, {name}, etc., the callsite must pass values for those names.
 */

import { readFileSync, readdirSync, statSync } from 'fs';
import { dirname, join, relative, resolve } from 'path';
import { fileURLToPath } from 'url';

const REPO_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const args = process.argv.slice(2);
const JSON_OUTPUT = args.includes('--json');
const QUIET = args.includes('--quiet');

const APP = {
	name: 'frontend',
	localeFile: 'frontend/src/locales/en.json',
	sourceDir: 'frontend/src'
};

const ALLOWLISTED_LITERAL_KEYS = [
	/\.media_shortcode_help$/,
	/\.shortcodes_description$/,
	/\.shortcodes_hint$/,
	/\.a_shortcodes$/,
	/\.slug_example$/,
	/\.url_pattern_example$/
];

const SOURCE_EXTENSIONS = new Set(['.svelte', '.ts', '.js']);
const SKIP_DIRS = new Set(['node_modules', '.svelte-kit', 'dist', 'build', 'coverage', '.cache']);

function walkSourceFiles(dir, out = []) {
	for (const entry of readdirSync(dir)) {
		if (SKIP_DIRS.has(entry)) continue;
		const full = join(dir, entry);
		const stat = statSync(full);
		if (stat.isDirectory()) {
			walkSourceFiles(full, out);
			continue;
		}
		const dot = entry.lastIndexOf('.');
		if (dot >= 0 && SOURCE_EXTENSIONS.has(entry.substring(dot))) {
			out.push(full);
		}
	}
	return out;
}

function extractPlaceholderKeys(json, prefix = '', out = []) {
	for (const [key, value] of Object.entries(json)) {
		const fullKey = prefix ? `${prefix}.${key}` : key;
		if (typeof value === 'string') {
			const placeholders = collectPlaceholders(value);
			if (placeholders.length > 0) {
				out.push({ key: fullKey, placeholders });
			}
			continue;
		}
		if (value && typeof value === 'object' && !Array.isArray(value)) {
			extractPlaceholderKeys(value, fullKey, out);
		}
	}
	return out;
}

function collectPlaceholders(str) {
	const stripped = str.replace(/\{([a-z_][a-z0-9_]*)\}[\s\S]*?\{\/\1\}/gi, '');
	const names = new Set();
	const re = /\{([a-z_][a-z0-9_]*)(?:,[^}]*)?\}/gi;
	for (const match of stripped.matchAll(re)) {
		names.add(match[1]);
	}
	return [...names];
}

function findCallsites(source, filePath) {
	const found = [];
	const opener = /\$(?:t|_)\(\s*(['"`])([^'"`\n]+)\1/g;
	for (const match of source.matchAll(opener)) {
		const key = match[2];
		const startIdx = match.index;
		const afterKeyIdx = match.index + match[0].length;
		let depth = 1;
		let i = afterKeyIdx;
		let inString = null;
		while (i < source.length && depth > 0) {
			const c = source[i];
			if (inString) {
				if (c === '\\') {
					i += 2;
					continue;
				}
				if (c === inString) inString = null;
			} else {
				if (c === '"' || c === "'" || c === '`') inString = c;
				else if (c === '(') depth++;
				else if (c === ')') depth--;
			}
			i++;
		}
		const callBody = source.substring(afterKeyIdx, i - 1);
		const valuesNames = extractValuesNames(callBody);
		const line = source.substring(0, startIdx).split('\n').length;
		found.push({ key, valuesNames, file: filePath, line });
	}
	return found;
}

function extractValuesNames(callBody) {
	const matches = [...callBody.matchAll(/\bvalues\s*:\s*\{/g)];
	if (matches.length === 0) return [];

	let depth = 1;
	let i = matches[0].index + matches[0][0].length;
	const start = i;
	while (i < callBody.length && depth > 0) {
		const c = callBody[i];
		if (c === '{') depth++;
		else if (c === '}') depth--;
		i++;
	}

	const block = callBody.substring(start, i - 1);
	const names = new Set();
	let nestedDepth = 0;
	let token = '';
	for (let j = 0; j < block.length; j++) {
		const c = block[j];
		if (c === '{') {
			nestedDepth++;
			token = '';
			continue;
		}
		if (c === '}') {
			nestedDepth--;
			token = '';
			continue;
		}
		if (nestedDepth > 0) continue;
		if (c === ':' || c === ',' || c === '\n') {
			const trimmed = token.trim();
			if (/^[a-z_][a-z0-9_]*$/i.test(trimmed)) names.add(trimmed);
			token = '';
			continue;
		}
		token += c;
	}
	const last = token.trim();
	if (/^[a-z_][a-z0-9_]*$/i.test(last)) names.add(last);
	return [...names];
}

function isAllowlisted(key) {
	return ALLOWLISTED_LITERAL_KEYS.some((pattern) => pattern.test(key));
}

function checkApp(app) {
	const localePath = join(REPO_ROOT, app.localeFile);
	const localeData = JSON.parse(readFileSync(localePath, 'utf8'));
	const requiredByKey = new Map();
	for (const { key, placeholders } of extractPlaceholderKeys(localeData)) {
		if (!isAllowlisted(key)) {
			requiredByKey.set(key, placeholders);
		}
	}

	const violations = [];
	let scannedFiles = 0;
	let totalCallsites = 0;
	for (const file of walkSourceFiles(join(REPO_ROOT, app.sourceDir))) {
		scannedFiles++;
		const content = readFileSync(file, 'utf8');
		const callsites = findCallsites(content, file);
		totalCallsites += callsites.length;
		for (const callsite of callsites) {
			const required = requiredByKey.get(callsite.key);
			if (!required) continue;
			const missing = required.filter((name) => !callsite.valuesNames.includes(name));
			if (missing.length > 0) {
				violations.push({
					app: app.name,
					file: relative(REPO_ROOT, callsite.file),
					line: callsite.line,
					key: callsite.key,
					missing,
					required,
					got: callsite.valuesNames
				});
			}
		}
	}
	return { violations, scannedFiles, totalCallsites };
}

const result = checkApp(APP);

if (JSON_OUTPUT) {
	console.log(JSON.stringify(result, null, 2));
} else {
	if (!QUIET) {
		console.log('i18n values-param guard\n');
		console.log(
			`  ${APP.name}: scanned ${result.scannedFiles} files, ${result.totalCallsites} callsites, ${result.violations.length} violations\n`
		);
	}
	if (result.violations.length === 0) {
		if (!QUIET) console.log('OK - no callsites missing required values params.');
	} else {
		console.error(`FAIL - ${result.violations.length} violation(s):\n`);
		for (const violation of result.violations) {
			console.error(
				`  ${violation.file}:${violation.line}  $t('${violation.key}')\n` +
					`    required: { ${violation.required.join(', ')} }\n` +
					`    missing:  ${violation.missing.join(', ')}\n` +
					`    got:      ${violation.got.length ? violation.got.join(', ') : '(no values object)'}\n`
			);
		}
	}
}

process.exit(result.violations.length === 0 ? 0 : 1);
