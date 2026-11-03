// Scopes an API key can carry. Must match `validScopes` in
// backend/hooks/api_keys.go (tests/api-key-scopes.spec.ts checks this).
//
// Write scopes are separate from read scopes: `write:posts` can create, edit
// and delete posts but can't list them, so a sync script usually needs both.
export const readScopes = [
	'read:profile',
	'read:posts',
	'read:projects',
	'read:skills',
	'read:experience'
] as const;

export const writeScopes = [
	'write:profile',
	'write:posts',
	'write:projects',
	'write:skills',
	'write:experience'
] as const;

export const availableScopes = [...readScopes, ...writeScopes] as const;
export type Scope = (typeof availableScopes)[number];

export const scopeGroups = [
	{ kind: 'read', scopes: readScopes },
	{ kind: 'write', scopes: writeScopes }
] as const;

// A new key starts with the least it could usefully do. Write access is only
// ever added by ticking it.
export const defaultScopes: Scope[] = ['read:profile'];

export function isWriteScope(scope: string): boolean {
	return scope.startsWith('write:');
}

// Turns the allowed-origins textarea (one per line, commas also accepted)
// into a list. The server validates and normalises each entry.
export function parseOriginsInput(text: string): string[] {
	return text
		.split(/[\n,]/)
		.map((s) => s.trim())
		.filter(Boolean);
}
