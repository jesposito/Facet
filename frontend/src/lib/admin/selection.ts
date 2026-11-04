export function toggleSelection(selectedIds: ReadonlySet<string>, id: string): Set<string> {
	const next = new Set(selectedIds);
	if (next.has(id)) next.delete(id);
	else next.add(id);
	return next;
}

export function pruneSelection(
	selectedIds: Set<string>,
	visibleIds: readonly string[]
): Set<string> {
	const visible = new Set(visibleIds);
	const next = new Set([...selectedIds].filter((id) => visible.has(id)));
	return next.size === selectedIds.size ? selectedIds : next;
}
