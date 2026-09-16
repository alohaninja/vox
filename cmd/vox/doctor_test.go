package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"howett.net/plist"
)

// ledgerFixture builds a trackedApplications blob in the on-disk shape:
// a flat list where each app contributes two records — a bare
// {bundle:{_0:id}} key record followed by a {location, menuItemLocations,
// isAllowed} value record.
func ledgerFixture(t *testing.T, apps []ledgerApp) []byte {
	t.Helper()
	var entries []map[string]any
	for _, a := range apps {
		var locs []map[string]any
		for _, id := range a.menuItems {
			locs = append(locs, map[string]any{"bundle": map[string]any{"_0": id}})
		}
		entries = append(entries,
			map[string]any{"bundle": map[string]any{"_0": a.id}},
			map[string]any{
				"location":          map[string]any{"bundle": map[string]any{"_0": a.id}},
				"menuItemLocations": locs,
				"isAllowed":         a.allowed,
			},
		)
	}
	b, err := plist.Marshal(entries, plist.BinaryFormat)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return b
}

type ledgerApp struct {
	id        string
	allowed   bool
	menuItems []string
}

func TestAnalyzeLedgerCleanWhenOnlySelfReferences(t *testing.T) {
	blob := ledgerFixture(t, []ledgerApp{
		{id: "dev.vox.menubar", allowed: true, menuItems: []string{"dev.vox.menubar"}},
		{id: "com.example.other", allowed: false, menuItems: []string{"com.example.other"}},
	})

	rep, err := analyzeStatusItemLedger(blob, "dev.vox.menubar")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !rep.ownFound || !rep.ownAllowed {
		t.Errorf("own record: found=%v allowed=%v, want found+allowed", rep.ownFound, rep.ownAllowed)
	}
	if len(rep.foreignOwners) != 0 {
		t.Errorf("expected no foreign owners, got %+v", rep.foreignOwners)
	}
	if rep.blocked() {
		t.Error("clean ledger reported as blocked")
	}
}

func TestAnalyzeLedgerDetectsDisallowedForeignOwner(t *testing.T) {
	// A parent process (terminal, IDE, agent host) that is switched OFF in
	// System Settings > Menu Bar has captured Vox's bundle ID in its own
	// menuItemLocations. Vox's own record is allowed, so the Settings toggle
	// looks fine — but Control Center honours the parent's veto.
	blob := ledgerFixture(t, []ledgerApp{
		{id: "dev.vox.menubar", allowed: true, menuItems: []string{"dev.vox.menubar"}},
		{id: "com.example.parent", allowed: false, menuItems: []string{"com.example.parent", "dev.vox.menubar"}},
	})

	rep, err := analyzeStatusItemLedger(blob, "dev.vox.menubar")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !rep.ownFound || !rep.ownAllowed {
		t.Errorf("own record: found=%v allowed=%v, want found+allowed", rep.ownFound, rep.ownAllowed)
	}
	if len(rep.foreignOwners) != 1 {
		t.Fatalf("want 1 foreign owner, got %d: %+v", len(rep.foreignOwners), rep.foreignOwners)
	}
	fo := rep.foreignOwners[0]
	if fo.id != "com.example.parent" || fo.allowed {
		t.Errorf("foreign owner = %+v, want com.example.parent disallowed", fo)
	}
	if !rep.blocked() {
		t.Error("disallowed foreign owner should report blocked")
	}
}

func TestAnalyzeLedgerAllowedForeignOwnerDoesNotBlock(t *testing.T) {
	// A parent that is ON in Settings also holds a reference. That is
	// harmless — only disallowed owners veto.
	blob := ledgerFixture(t, []ledgerApp{
		{id: "dev.vox.menubar", allowed: true, menuItems: []string{"dev.vox.menubar"}},
		{id: "com.example.terminal", allowed: true, menuItems: []string{"dev.vox.menubar"}},
	})

	rep, err := analyzeStatusItemLedger(blob, "dev.vox.menubar")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(rep.foreignOwners) != 1 || !rep.foreignOwners[0].allowed {
		t.Fatalf("want 1 allowed foreign owner, got %+v", rep.foreignOwners)
	}
	if rep.blocked() {
		t.Error("allowed foreign owner must not report blocked")
	}
}

func TestAnalyzeLedgerOwnRecordDisallowed(t *testing.T) {
	blob := ledgerFixture(t, []ledgerApp{
		{id: "dev.vox.menubar", allowed: false, menuItems: []string{"dev.vox.menubar"}},
	})

	rep, err := analyzeStatusItemLedger(blob, "dev.vox.menubar")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !rep.ownFound || rep.ownAllowed {
		t.Errorf("own record: found=%v allowed=%v, want found+disallowed", rep.ownFound, rep.ownAllowed)
	}
	if !rep.blocked() {
		t.Error("own record disallowed should report blocked")
	}
}

func TestAnalyzeLedgerMissingOwnRecord(t *testing.T) {
	blob := ledgerFixture(t, []ledgerApp{
		{id: "com.example.other", allowed: true, menuItems: []string{"com.example.other"}},
	})

	rep, err := analyzeStatusItemLedger(blob, "dev.vox.menubar")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if rep.ownFound {
		t.Error("own record should be absent")
	}
	if rep.blocked() {
		t.Error("absent record is 'never launched', not blocked")
	}
}

func TestAnalyzeLedgerRejectsGarbage(t *testing.T) {
	if _, err := analyzeStatusItemLedger([]byte("not a plist"), "dev.vox.menubar"); err == nil {
		t.Error("expected error on invalid plist")
	}
}

// writeLedgerFixture wraps a trackedApplications blob in the outer plist
// shape Control Center writes to disk and returns the temp file path.
func writeLedgerFixture(t *testing.T, apps []ledgerApp) string {
	t.Helper()
	outer := map[string]any{"trackedApplications": ledgerFixture(t, apps)}
	b, err := plist.Marshal(outer, plist.BinaryFormat)
	if err != nil {
		t.Fatalf("marshal outer: %v", err)
	}
	p := filepath.Join(t.TempDir(), "group.com.apple.controlcenter.plist")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return p
}

func runDoctorWithLedger(t *testing.T, ledger string) (string, int) {
	t.Helper()
	cmd := exec.Command("go", "run", ".", "doctor")
	cmd.Env = append(os.Environ(), "VOX_LEDGER_PATH="+ledger)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run doctor: %v\n%s", err, out)
	}
	return string(out), code
}

// TestDoctorReportsBlockedForeignOwner is the negative case: a disallowed
// parent app holds Vox's ID, so doctor must say BLOCKED, name the owner, and
// exit non-zero.
func TestDoctorReportsBlockedForeignOwner(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping doctor subcommand test in short mode")
	}
	ledger := writeLedgerFixture(t, []ledgerApp{
		{id: bundleID, allowed: true, menuItems: []string{bundleID}},
		{id: "com.example.parent", allowed: false, menuItems: []string{"com.example.parent", bundleID}},
	})
	out, code := runDoctorWithLedger(t, ledger)

	for _, want := range []string{
		"Verdict:      BLOCKED",
		"Foreign owner: com.example.parent — DISALLOWED",
		"make doctor-fix",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1 for a blocked ledger", code)
	}
}

// TestDoctorReportsOKForCleanLedger is the positive case.
func TestDoctorReportsOKForCleanLedger(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping doctor subcommand test in short mode")
	}
	ledger := writeLedgerFixture(t, []ledgerApp{
		{id: bundleID, allowed: true, menuItems: []string{bundleID}},
		{id: "com.example.terminal", allowed: true, menuItems: []string{bundleID}},
	})
	out, _ := runDoctorWithLedger(t, ledger)

	if !strings.Contains(out, "Verdict:      OK") {
		t.Errorf("output missing OK verdict\n%s", out)
	}
	if !strings.Contains(out, "Foreign owner: com.example.terminal — allowed (harmless)") {
		t.Errorf("allowed foreign owner should be listed as harmless\n%s", out)
	}
	if strings.Contains(out, "BLOCKED") {
		t.Errorf("clean ledger must not say BLOCKED\n%s", out)
	}
	// Exit code is not asserted here: on a dev machine the pidfile or
	// Accessibility check may legitimately fail and drive it non-zero.
}

// TestDoctorUnreadableLedgerIsNotAPass: when the ledger can't be read the
// menu bar check is inconclusive. The summary must say so rather than
// reporting a clean bill of health.
func TestDoctorUnreadableLedgerIsNotAPass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping doctor subcommand test in short mode")
	}
	missing := filepath.Join(t.TempDir(), "does-not-exist.plist")
	out, _ := runDoctorWithLedger(t, missing)

	if strings.Contains(out, "All checks passed.") {
		t.Errorf("unreadable ledger must not produce 'All checks passed.'\n%s", out)
	}
	if !strings.Contains(out, "could not be completed") {
		t.Errorf("summary should say a check could not be completed\n%s", out)
	}
}
