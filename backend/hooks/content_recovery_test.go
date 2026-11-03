package hooks

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"facet/services"

	"github.com/pocketbase/dbx"
)

func TestContentRecoveryFailureWithholdsMarkerAndAlertsOnce(t *testing.T) {
	app := newMigratedTestApp(t)
	marker := filepath.Join(t.TempDir(), contentRecoveryMarker)

	fail := func() (*services.RecoveryReport, error) {
		return &services.RecoveryReport{Applied: true}, errors.New("scan backup facet_auto_x.zip: zip: not a valid zip file")
	}

	// Two failed boots in a row.
	finishContentRecovery(app, marker, fail)
	finishContentRecovery(app, marker, fail)

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("marker exists after a failed pass (stat err %v); recovery would never retry", err)
	}
	open, err := app.CountRecords("system_alerts", dbx.HashExp{"event": contentRecoveryFailedEvent, "acknowledged_at": ""})
	if err != nil {
		t.Fatal(err)
	}
	if open != 1 {
		t.Fatalf("open %s alerts = %d, want exactly 1 after repeated failures", contentRecoveryFailedEvent, open)
	}

	// Once acknowledged, a later failure raises a fresh alert.
	if _, err := app.DB().NewQuery("UPDATE system_alerts SET acknowledged_at = '2026-11-03 00:00:00.000Z'").Execute(); err != nil {
		t.Fatal(err)
	}
	finishContentRecovery(app, marker, fail)
	total, err := app.CountRecords("system_alerts", dbx.HashExp{"event": contentRecoveryFailedEvent})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("total %s alerts = %d, want 2 after acknowledging the first", contentRecoveryFailedEvent, total)
	}
}

func TestContentRecoverySuccessWritesPrivateMarker(t *testing.T) {
	app := newMigratedTestApp(t)
	marker := filepath.Join(t.TempDir(), contentRecoveryMarker)

	finishContentRecovery(app, marker, func() (*services.RecoveryReport, error) {
		return &services.RecoveryReport{Applied: true}, nil
	})

	info, err := os.Stat(marker)
	if err != nil {
		t.Fatalf("marker missing after a successful pass: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("marker mode = %o, want 600", perm)
	}
	if n, _ := app.CountRecords("system_alerts", dbx.HashExp{"event": contentRecoveryFailedEvent}); n != 0 {
		t.Fatalf("success raised %d failure alerts", n)
	}
}
