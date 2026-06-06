// Package services — content recovery for issue #443.
//
// The TipTap markdown editor (MarkdownEditor.svelte) shipped with a bug where
// stored markdown was parsed as HTML on edit-load, collapsing every newline to a
// space. Re-saving then persisted the flattened, single-line content, destroying
// paragraph/line structure (the words survived; only whitespace was lost).
//
// This recovers that lost structure from Facet's daily auto-backups, which still
// hold the pre-corruption values for recently-edited records. It is intentionally
// conservative: a field is only rewritten when the backup value collapses to the
// exact same text as the current value (same words, only whitespace differs) and
// the backup has more line structure. Anything the user genuinely re-edited will
// not collapse-match and is left untouched.
package services

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver (same one PocketBase uses)
)

// wsRun matches runs of any whitespace, including the non-breaking space that the
// markdown serializer uses for blank paragraphs.
var wsRun = regexp.MustCompile(`[\s\x{00A0}]+`)

// identRe guards collection/field names before they are interpolated into SQL.
// All names come from the trusted PocketBase schema, but validating keeps the
// query construction injection-proof regardless.
var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func safeIdent(s string) bool { return identRe.MatchString(s) }

// collapseWS reduces every whitespace run to a single space and trims the ends.
// Two strings with equal collapseWS differ only in whitespace, never in words.
func collapseWS(s string) string {
	return strings.TrimSpace(wsRun.ReplaceAllString(strings.ReplaceAll(s, "\r\n", "\n"), " "))
}

func lineBreaks(s string) int {
	return strings.Count(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

// shouldRestore reports whether backup is the structure-preserving original of a
// flattened current value. Three independent gates make this safe:
//  1. same words            -> collapseWS(current) == collapseWS(backup)
//  2. backup has structure  -> lineBreaks(backup) > lineBreaks(current)
//  3. actually different     -> backup != current
func shouldRestore(current, backup string) bool {
	if current == backup {
		return false
	}
	if collapseWS(current) == "" {
		return false
	}
	if lineBreaks(backup) <= lineBreaks(current) {
		return false
	}
	return collapseWS(current) == collapseWS(backup)
}

// shouldRestoreJSONArray applies the same collapse-match safety to a JSON string
// array stored as line-joined markdown (e.g. experience.bullets, where #443 turns
// ["a","b","c"] into ["a b c"]). It restores the backup array only when both parse
// as []string, the backup has more elements (more structure), and joining each
// with newlines yields the same collapsed text (same words, only structure lost).
func shouldRestoreJSONArray(currentRaw, backupRaw string) bool {
	if currentRaw == backupRaw {
		return false
	}
	var cur, bak []string
	if json.Unmarshal([]byte(currentRaw), &cur) != nil {
		return false
	}
	if json.Unmarshal([]byte(backupRaw), &bak) != nil {
		return false
	}
	if len(bak) <= len(cur) {
		return false
	}
	curJoined := collapseWS(strings.Join(cur, "\n"))
	if curJoined == "" {
		return false
	}
	return curJoined == collapseWS(strings.Join(bak, "\n"))
}

// RestoreItem records a single planned/applied field restoration.
type RestoreItem struct {
	Collection string
	Field      string
	RecordID   string
	Current    string // value scanned from the live DB; used as an optimistic-update guard
	FromBackup string
}

// RecoveryReport summarizes a recovery pass.
type RecoveryReport struct {
	Applied    bool
	Scanned    int // candidate (flattened) values examined
	Restored   int
	Skipped    int // planned but skipped because the field changed under us (live edit)
	BackupUsed string // safety backup taken before applying, if any
	Items      []RestoreItem
}

type fieldTarget struct {
	collection string
	field      string
	isJSON     bool // line-joined JSON string array (e.g. experience.bullets) vs plain editor/text
}

// jsonArrayTargets are JSON fields edited via MarkdownEditor as line-joined text.
// #443 can flatten e.g. ["a","b","c"] into ["a b c"]. They are type `json`, so the
// editor/text sweep skips them; they need array-aware comparison.
var jsonArrayTargets = []fieldTarget{
	{collection: "experience", field: "bullets", isJSON: true},
}

type idVal struct {
	ID  string `db:"id"`
	Val string `db:"val"`
}

// RunContentRecovery scans editor/text fields for content flattened by #443 and,
// when apply is true, restores the structured value from the newest matching
// backup. It never returns a fatal error for recoverable conditions; callers
// (the startup hook) treat a returned error as "retry next boot".
func RunContentRecovery(app *pocketbase.PocketBase, apply bool) (*RecoveryReport, error) {
	report := &RecoveryReport{Applied: apply}

	targets, err := editorTextTargets(app)
	if err != nil {
		return report, fmt.Errorf("enumerate targets: %w", err)
	}
	if len(targets) == 0 {
		return report, nil
	}

	// Collect the current flattened candidates per target: non-empty values that
	// currently have no line breaks (the #443 signature is total newline loss).
	candidates := make(map[fieldTarget]map[string]candidateValue)
	for _, t := range targets {
		rows := []idVal{}
		q := fmt.Sprintf("SELECT id, %s AS val FROM %s WHERE %s <> ''", t.field, t.collection, t.field)
		if err := app.DB().NewQuery(q).All(&rows); err != nil {
			continue // collection/field not present in this instance — skip
		}
		for _, r := range rows {
			if lineBreaks(r.Val) == 0 && collapseWS(r.Val) != "" {
				if candidates[t] == nil {
					candidates[t] = map[string]candidateValue{}
				}
				candidates[t][r.ID] = candidateValue{current: r.Val}
				report.Scanned++
			}
		}
	}
	if report.Scanned == 0 {
		return report, nil
	}

	// Walk backups newest-first; the first backup whose value collapse-matches a
	// candidate (and has more structure) is the freshest pre-corruption original.
	backups, _ := ListBackups(app)
	sort.Slice(backups, func(i, j int) bool { return backups[i].Created.After(backups[j].Created) })
	remaining := report.Scanned
	for _, b := range backups {
		if remaining == 0 {
			break
		}
		planned := scanBackup(app, b.Filename, targets, candidates)
		for _, item := range planned {
			// scanBackup already removed satisfied ids from the shared candidate map.
			report.Items = append(report.Items, item)
			remaining--
		}
	}

	if len(report.Items) == 0 {
		return report, nil
	}

	if !apply {
		return report, nil
	}

	// Take a safety backup before mutating anything, so the recovery is reversible.
	if res, err := RunBackup(app, fmt.Sprintf("facet_pre_recovery_%s.zip", time.Now().UTC().Format("20060102_150405"))); err == nil {
		report.BackupUsed = res.Filename
	} else {
		app.Logger().Warn("content-recovery: pre-recovery backup failed; proceeding", "error", err)
	}

	restoreFailed := false
	for _, item := range report.Items {
		// Surgical + race-safe: set only the field, and only if it STILL holds the
		// exact value we scanned. The `AND field = {:cur}` guard means that if an
		// admin edited this field during the background pass, the UPDATE matches no
		// rows and we skip it — a live edit is never clobbered. `updated` is left
		// untouched (this is a restoration, not a user edit).
		q := fmt.Sprintf("UPDATE %s SET %s = {:v} WHERE id = {:id} AND %s = {:cur}", item.Collection, item.Field, item.Field)
		res, err := app.DB().NewQuery(q).Bind(dbx.Params{"v": item.FromBackup, "id": item.RecordID, "cur": item.Current}).Execute()
		if err != nil {
			// e.g. SQLite busy during a concurrent backup/write. Don't write the
			// completion marker this pass so the record is retried next boot.
			restoreFailed = true
			app.Logger().Error("content-recovery: restore failed", "collection", item.Collection, "field", item.Field, "id", item.RecordID, "error", err)
			continue
		}
		if n, _ := res.RowsAffected(); n == 0 {
			report.Skipped++ // field changed since scan (live edit) — leave it intact
			continue
		}
		report.Restored++
	}

	if restoreFailed {
		return report, fmt.Errorf("content-recovery: %d restored, %d skipped, but some updates failed; marker withheld for retry", report.Restored, report.Skipped)
	}
	return report, nil
}

// editorTextTargets returns every editor/text field of every base collection.
func editorTextTargets(app *pocketbase.PocketBase) ([]fieldTarget, error) {
	collections, err := app.FindAllCollections(core.CollectionTypeBase)
	if err != nil {
		return nil, err
	}
	var targets []fieldTarget
	for _, c := range collections {
		if !safeIdent(c.Name) {
			continue
		}
		for _, f := range c.Fields {
			ft := f.Type()
			if ft != core.FieldTypeEditor && ft != core.FieldTypeText {
				continue
			}
			name := f.GetName()
			if name == "id" || !safeIdent(name) {
				continue
			}
			targets = append(targets, fieldTarget{collection: c.Name, field: name})
		}
	}
	targets = append(targets, jsonArrayTargets...)
	return targets, nil
}

// scanBackup opens one backup archive and returns the restorations it can satisfy
// for the still-remaining candidates.
func scanBackup(app *pocketbase.PocketBase, name string, targets []fieldTarget, candidates map[fieldTarget]map[string]candidateValue) []RestoreItem {
	tmp, cleanup, err := extractBackupDB(app, name)
	if err != nil {
		app.Logger().Warn("content-recovery: could not read backup", "backup", name, "error", err)
		return nil
	}
	defer cleanup()

	db, err := sql.Open("sqlite", "file:"+tmp+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil
	}
	defer db.Close()

	var planned []RestoreItem
	for _, t := range targets {
		pending := candidates[t]
		if len(pending) == 0 {
			continue
		}
		rows, err := db.Query(fmt.Sprintf("SELECT id, %s FROM %s", t.field, t.collection))
		if err != nil {
			continue // table/column absent in this (older) backup schema
		}
		func() {
			defer rows.Close()
			for rows.Next() {
				var id string
				var val sql.NullString
				if err := rows.Scan(&id, &val); err != nil {
					continue
				}
				cand, ok := pending[id]
				if !ok || !val.Valid {
					continue
				}
				var match bool
				if t.isJSON {
					match = shouldRestoreJSONArray(cand.current, val.String)
				} else {
					match = shouldRestore(cand.current, val.String)
				}
				if match {
					planned = append(planned, RestoreItem{
						Collection: t.collection,
						Field:      t.field,
						RecordID:   id,
						Current:    cand.current,
						FromBackup: val.String,
					})
					delete(pending, id) // satisfied by this (newest) backup
				}
			}
		}()
	}
	return planned
}

// candidateValue is the map value type for in-flight candidates.
type candidateValue = struct{ current string }

// extractBackupDB pulls data.db out of a backup zip into a temp file and returns
// its path plus a cleanup func.
func extractBackupDB(app *pocketbase.PocketBase, name string) (string, func(), error) {
	fsys, err := app.NewBackupsFilesystem()
	if err != nil {
		return "", nil, err
	}
	defer fsys.Close()

	reader, err := fsys.GetFile(name)
	if err != nil {
		return "", nil, err
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		return "", nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", nil, err
	}

	var entry *zip.File
	for _, f := range zr.File {
		// Match the archive-root data.db exactly. A PocketBase backup places the
		// primary DB at the zip root; matching on basename could pick up a nested
		// remnant (e.g. pb_data/data.db) and scan a stale snapshot.
		if f.Name == "data.db" {
			entry = f
			break
		}
	}
	if entry == nil {
		return "", nil, fmt.Errorf("data.db not found in backup %s", name)
	}

	src, err := entry.Open()
	if err != nil {
		return "", nil, err
	}
	defer src.Close()

	tmp, err := os.CreateTemp("", "facet-recovery-*.db")
	if err != nil {
		return "", nil, err
	}
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", nil, err
	}
	tmp.Close()

	path := tmp.Name()
	cleanup := func() { os.Remove(path) }
	return path, cleanup, nil
}
