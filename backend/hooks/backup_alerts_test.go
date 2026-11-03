package hooks

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"facet/services"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func countAlerts(t *testing.T, app core.App, event string, openOnly bool) int64 {
	t.Helper()
	exp := dbx.HashExp{"event": event}
	if openOnly {
		exp["acknowledged_at"] = ""
	}
	n, err := app.CountRecords("system_alerts", exp)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func ackAllAlerts(t *testing.T, app core.App) {
	t.Helper()
	if _, err := app.DB().NewQuery("UPDATE system_alerts SET acknowledged_at = '2026-11-03 00:00:00.000Z'").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSystemAlertDedupUnderConcurrentEmits(t *testing.T) {
	app := newMigratedTestApp(t)
	input := SystemAlertInput{Severity: SeverityCritical, Event: "dedup_probe", Title: "Probe"}

	const workers = 24
	var created atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if createSystemAlert(app, input) {
				created.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := created.Load(); got != 1 {
		t.Fatalf("%d concurrent emits created %d alerts, want 1", workers, got)
	}
	if n := countAlerts(t, app, "dedup_probe", false); n != 1 {
		t.Fatalf("stored alerts = %d, want 1", n)
	}

	// Sequential repeats while still open are dropped too.
	for i := 0; i < 3; i++ {
		if createSystemAlert(app, input) {
			t.Fatal("repeat emit created a second open alert")
		}
	}

	// After acknowledgement a renewed failure raises a fresh alert, once.
	ackAllAlerts(t, app)
	if !createSystemAlert(app, input) {
		t.Fatal("emit after acknowledgement was dropped")
	}
	if createSystemAlert(app, input) {
		t.Fatal("second emit after acknowledgement was not deduplicated")
	}
	if n := countAlerts(t, app, "dedup_probe", false); n != 2 {
		t.Fatalf("total alerts = %d, want 2", n)
	}
	if n := countAlerts(t, app, "dedup_probe", true); n != 1 {
		t.Fatalf("open alerts = %d, want 1", n)
	}
}

func TestPerformBackupFailureAlertsOnceWithoutSecrets(t *testing.T) {
	app := newMigratedTestApp(t)
	enforced := false
	ops := backupOps{
		run: func(core.App, string) (*services.BackupResult, error) {
			return nil, errors.New("upload to s3://AKIAEXAMPLE:topsecret@backups.example.test/b failed: password=hunter2\ngoroutine 1 [running]:\nmain.main()")
		},
		enforce: func(core.App, int) error { enforced = true; return nil },
	}

	for i := 0; i < 3; i++ {
		out := performBackup(app, ops, "scheduled", "facet_auto_x.zip", 5)
		if out.err == nil || out.result != nil {
			t.Fatalf("outcome = %+v, want a creation failure", out)
		}
	}
	if enforced {
		t.Fatal("retention ran after a failed backup")
	}
	if n := countAlerts(t, app, backupFailedEvent, true); n != 1 {
		t.Fatalf("open %s alerts = %d, want 1 after repeated failures", backupFailedEvent, n)
	}

	rec, err := app.FindFirstRecordByData("system_alerts", "event", backupFailedEvent)
	if err != nil {
		t.Fatal(err)
	}
	details := rec.GetString("details")
	for _, leak := range []string{"topsecret", "AKIAEXAMPLE", "hunter2", "\n", "goroutine 1 [running]:\nmain"} {
		if strings.Contains(details, leak) {
			t.Fatalf("alert details leak %q: %s", leak, details)
		}
	}
	if !strings.Contains(details, "facet_auto_x.zip") || !strings.Contains(details, "scheduled") {
		t.Fatalf("alert details do not name the failed backup: %s", details)
	}
	if rec.GetString("severity") != SeverityCritical {
		t.Fatalf("severity = %q, want critical", rec.GetString("severity"))
	}

	ackAllAlerts(t, app)
	performBackup(app, ops, "manual", "facet_manual_y.zip", 5)
	if n := countAlerts(t, app, backupFailedEvent, false); n != 2 {
		t.Fatalf("total %s alerts = %d, want 2 after acknowledging the first", backupFailedEvent, n)
	}
}

func TestPerformBackupRetentionFailureIsReportedSeparately(t *testing.T) {
	app := newMigratedTestApp(t)
	ops := backupOps{
		run: func(_ core.App, name string) (*services.BackupResult, error) {
			return &services.BackupResult{Filename: name, Size: 42}, nil
		},
		enforce: func(core.App, int) error { return errors.New("delete facet_auto_old.zip: permission denied") },
	}

	out := performBackup(app, ops, "manual", "facet_manual_ok.zip", 3)
	if out.err != nil {
		t.Fatalf("backup reported as failed although the archive was created: %v", out.err)
	}
	if out.result == nil || out.result.Filename != "facet_manual_ok.zip" {
		t.Fatalf("result = %+v, want the created archive", out.result)
	}
	if out.retentionErr == nil {
		t.Fatal("retention error was swallowed")
	}
	if n := countAlerts(t, app, backupFailedEvent, false); n != 0 {
		t.Fatalf("retention failure raised %d backup_failed alerts, want 0", n)
	}
	if n := countAlerts(t, app, backupRetentionFailedEvent, true); n != 1 {
		t.Fatalf("open %s alerts = %d, want 1", backupRetentionFailedEvent, n)
	}
}

// Uses PocketBase's real local backup against the throwaway test data dir.
func TestPerformBackupWithTestAppStorage(t *testing.T) {
	app := newMigratedTestApp(t)

	for _, name := range []string{"facet_manual_a.zip", "facet_manual_b.zip", "facet_manual_c.zip"} {
		out := performBackup(app, defaultBackupOps, "manual", name, 2)
		if out.err != nil || out.retentionErr != nil {
			t.Fatalf("%s: err=%v retentionErr=%v", name, out.err, out.retentionErr)
		}
		time.Sleep(10 * time.Millisecond) // distinct mod times for newest-first ordering
	}
	backups, err := services.ListBackups(app)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 2 {
		t.Fatalf("backups after retention = %d, want 2", len(backups))
	}
	if n := countAlerts(t, app, backupFailedEvent, false) + countAlerts(t, app, backupRetentionFailedEvent, false); n != 0 {
		t.Fatalf("successful backups raised %d alerts", n)
	}

	// While another backup/restore holds the lock, CreateBackup fails for real.
	app.Store().Set(core.StoreKeyActiveBackup, "other")
	defer app.Store().Remove(core.StoreKeyActiveBackup)
	if out := performBackup(app, defaultBackupOps, "scheduled", "facet_auto_locked.zip", 2); out.err == nil {
		t.Fatal("backup succeeded while another backup was active")
	}
	if n := countAlerts(t, app, backupFailedEvent, true); n != 1 {
		t.Fatalf("open %s alerts = %d, want 1", backupFailedEvent, n)
	}
}

func TestReportRestoreFailureAlertsOnce(t *testing.T) {
	app := newMigratedTestApp(t)
	err := errors.New("failed to extract backup: zip: not a valid zip file")

	reportRestoreFailure(app, "facet_manual_bad.zip", err)
	reportRestoreFailure(app, "facet_manual_bad.zip", err)

	if n := countAlerts(t, app, backupRestoreFailedEvent, true); n != 1 {
		t.Fatalf("open %s alerts = %d, want 1", backupRestoreFailedEvent, n)
	}
	rec, findErr := app.FindFirstRecordByData("system_alerts", "event", backupRestoreFailedEvent)
	if findErr != nil {
		t.Fatal(findErr)
	}
	if !strings.Contains(rec.GetString("details"), "facet_manual_bad.zip") {
		t.Fatalf("restore alert does not name the archive: %s", rec.GetString("details"))
	}
}

func TestAlertSafeErrorTruncates(t *testing.T) {
	got := alertSafeError(errors.New(strings.Repeat("x", 3000)))
	if n := len([]rune(got)); n != alertMaxErrorLen+1 {
		t.Fatalf("length = %d, want %d (limit plus ellipsis)", n, alertMaxErrorLen+1)
	}
}

func TestNextBackupTime(t *testing.T) {
	cases := []struct {
		now  time.Time
		want time.Time
	}{
		{time.Date(2026, 11, 3, 1, 59, 0, 0, time.UTC), time.Date(2026, 11, 3, 2, 0, 0, 0, time.UTC)},
		{time.Date(2026, 11, 3, 2, 0, 0, 0, time.UTC), time.Date(2026, 11, 4, 2, 0, 0, 0, time.UTC)},
		{time.Date(2026, 11, 3, 23, 0, 0, 0, time.UTC), time.Date(2026, 11, 4, 2, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		if got := nextBackupTime(c.now, 2); !got.Equal(c.want) {
			t.Errorf("nextBackupTime(%s) = %s, want %s", c.now, got, c.want)
		}
	}
}

// fakeScheduler records each wait the loop asks for and lets the test decide
// when (or whether) that wait ends.
type fakeScheduler struct {
	waits chan time.Duration
	fire  chan time.Time
	runs  atomic.Int32
}

func newFakeScheduler() (*fakeScheduler, *backupScheduler) {
	f := &fakeScheduler{waits: make(chan time.Duration, 8), fire: make(chan time.Time)}
	s := &backupScheduler{
		startDelay: time.Minute,
		hour:       2,
		now:        func() time.Time { return time.Date(2026, 11, 3, 1, 0, 0, 0, time.UTC) },
		after: func(d time.Duration) <-chan time.Time {
			f.waits <- d
			return f.fire
		},
		runOnce: func() { f.runs.Add(1) },
		logNext: func(time.Time, time.Duration) {},
		onPanic: func(any) {},
	}
	return f, s
}

func runLoop(s *backupScheduler, ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() { s.loop(ctx); close(done) }()
	return done
}

func waitFor[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
	var zero T
	return zero
}

func TestBackupSchedulerStopsDuringStartupDelay(t *testing.T) {
	f, s := newFakeScheduler()
	ctx, cancel := context.WithCancel(context.Background())
	done := runLoop(s, ctx)

	if d := waitFor(t, f.waits, "startup delay"); d != time.Minute {
		t.Fatalf("startup delay = %s, want 1m", d)
	}
	cancel()
	waitFor(t, done, "scheduler to stop")
	if f.runs.Load() != 0 {
		t.Fatal("backup ran although the scheduler was cancelled")
	}
}

func TestBackupSchedulerRunsThenStopsWhileWaiting(t *testing.T) {
	f, s := newFakeScheduler()
	ctx, cancel := context.WithCancel(context.Background())
	done := runLoop(s, ctx)

	waitFor(t, f.waits, "startup delay")
	f.fire <- time.Time{}
	if d := waitFor(t, f.waits, "first schedule wait"); d != time.Hour {
		t.Fatalf("wait until 02:00 = %s, want 1h", d)
	}
	f.fire <- time.Time{}
	waitFor(t, f.waits, "second schedule wait")
	if got := f.runs.Load(); got != 1 {
		t.Fatalf("runs = %d, want 1", got)
	}

	// Cancelling mid-wait (the 24h sleep in production) returns promptly.
	cancel()
	waitFor(t, done, "scheduler to stop")
}

func TestBackupSchedulerSurvivesPanic(t *testing.T) {
	f, s := newFakeScheduler()
	var panics atomic.Int32
	s.runOnce = func() { f.runs.Add(1); panic("boom") }
	s.onPanic = func(any) { panics.Add(1) }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := runLoop(s, ctx)

	waitFor(t, f.waits, "startup delay")
	f.fire <- time.Time{}
	waitFor(t, f.waits, "first schedule wait")
	f.fire <- time.Time{}
	waitFor(t, f.waits, "wait after panicking run")
	if panics.Load() != 1 || f.runs.Load() != 1 {
		t.Fatalf("panics=%d runs=%d, want 1 and 1", panics.Load(), f.runs.Load())
	}
	cancel()
	waitFor(t, done, "scheduler to stop")
}
