package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"howett.net/plist"

	"vox/internal/hotkey"
)

// bundleID must match CFBundleIdentifier in packaging/Info.plist and
// APP_BUNDLE_ID in the Makefile.
const bundleID = "dev.vox.menubar"

// statusItemLedgerPath is where Control Center (macOS 15+) persists which
// apps may show a menu bar item. It is a sandboxed group container: reading
// it requires the *calling terminal* to have Full Disk Access.
//
// VOX_LEDGER_PATH overrides it so the doctor can be exercised against a
// fixture in tests.
func statusItemLedgerPath() string {
	if p := os.Getenv("VOX_LEDGER_PATH"); p != "" {
		return p
	}
	return filepath.Join(
		os.Getenv("HOME"),
		"Library", "Group Containers", "group.com.apple.controlcenter",
		"Library", "Preferences", "group.com.apple.controlcenter.plist",
	)
}

// checkResult is the outcome of one doctor section. Distinguishing
// "couldn't check" from "passed" matters: an unreadable ledger is not a
// clean bill of health.
type checkResult int

const (
	checkPass checkResult = iota
	checkFail
	checkInconclusive
)

// runDoctor prints a read-only health report. It never modifies anything
// and never triggers a permission prompt.
func runDoctor() {
	fmt.Print(banner)
	fmt.Println("Doctor")
	fmt.Println("======")
	fmt.Println()

	results := []checkResult{
		doctorProcess(),
		doctorPermissions(),
		doctorMenuBar(),
	}

	failed, inconclusive := false, false
	for _, r := range results {
		switch r {
		case checkFail:
			failed = true
		case checkInconclusive:
			inconclusive = true
		}
	}

	fmt.Println()
	switch {
	case failed:
		fmt.Println("Some checks need attention (see above).")
		os.Exit(1)
	case inconclusive:
		fmt.Println("No problems found, but one or more checks could not be completed (see above).")
	default:
		fmt.Println("All checks passed.")
	}
}

// doctorProcess reports whether Vox is running and whether the pidfile that
// `make start` writes agrees with reality. A stale pidfile is the usual
// reason `make start` says "running" when nothing is.
func doctorProcess() checkResult {
	fmt.Println("[1/3] Process")
	pids := runningVoxPIDs()
	pidfile := filepath.Join("logs", "vox.pid")
	filePID, fileErr := readPIDFile(pidfile)

	switch {
	case len(pids) == 0:
		fmt.Println("  Vox:      not running")
	case len(pids) == 1:
		fmt.Printf("  Vox:      running (pid %d)\n", pids[0])
	default:
		fmt.Printf("  Vox:      %d instances running (%v) — only one can hold the lock\n", len(pids), pids)
	}

	if fileErr != nil {
		fmt.Printf("  PID file: %s (none)\n", pidfile)
		return checkPass
	}
	alive := false
	for _, p := range pids {
		if p == filePID {
			alive = true
		}
	}
	if alive {
		fmt.Printf("  PID file: %s -> %d (matches)\n", pidfile, filePID)
		return checkPass
	}
	fmt.Printf("  PID file: %s -> %d (STALE — process gone; run `make stop` to clear)\n", pidfile, filePID)
	return checkFail
}

// doctorPermissions checks the two TCC grants Vox needs. It only *checks*;
// it does not trigger prompts.
func doctorPermissions() checkResult {
	fmt.Println()
	fmt.Println("[2/3] Permissions (for this build's signature)")
	result := checkPass

	fmt.Print("  Accessibility: ")
	if hotkey.AccessibilityGranted() {
		fmt.Println("granted")
	} else {
		fmt.Println("MISSING — System Settings > Privacy & Security > Accessibility")
		fmt.Println("                 (a rebuild changes the signature; re-toggle Vox off/on)")
		result = checkFail
	}

	fmt.Print("  Microphone:    ")
	if hotkey.MicrophoneAuthorized() {
		fmt.Println("granted")
	} else {
		fmt.Println("not yet granted — Vox will prompt on first launch")
	}
	return result
}

// doctorMenuBar inspects Control Center's status-item ledger for the one
// failure mode that System Settings cannot show: Vox's own toggle is ON, but
// a *different* app that is switched OFF has Vox's bundle ID recorded under
// its menuItemLocations. Control Center honours that veto and hides the icon.
//
// This happens when Vox is launched as a child of another app (a terminal,
// an IDE, an agent host): Control Center attributes the status item to the
// parent as well as to Vox.
func doctorMenuBar() checkResult {
	fmt.Println()
	fmt.Println("[3/3] Menu bar (Control Center ledger)")

	raw, err := os.ReadFile(statusItemLedgerPath())
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Println("  Ledger:   unreadable (this terminal lacks Full Disk Access)")
			fmt.Println("            Grant it: System Settings > Privacy & Security > Full Disk Access")
			fmt.Println("            then re-run `make doctor` from that terminal.")
		} else {
			fmt.Printf("  Ledger:   unreadable: %v\n", err)
		}
		fmt.Println("  Verdict:      INCONCLUSIVE")
		return checkInconclusive
	}

	var outer map[string]any
	if _, err := plist.Unmarshal(raw, &outer); err != nil {
		fmt.Printf("  Ledger:   cannot parse: %v\n", err)
		return checkInconclusive
	}
	blob, ok := outer["trackedApplications"].([]byte)
	if !ok {
		fmt.Println("  Ledger:   no trackedApplications key (Control Center has not tracked any app yet)")
		return checkPass
	}

	rep, err := analyzeStatusItemLedger(blob, bundleID)
	if err != nil {
		fmt.Printf("  Ledger:   cannot decode trackedApplications: %v\n", err)
		return checkInconclusive
	}

	switch {
	case !rep.ownFound:
		fmt.Printf("  Own record:   %s — not present (never launched, or ledger was reset)\n", bundleID)
	case rep.ownAllowed:
		fmt.Printf("  Own record:   %s — allowed\n", bundleID)
	default:
		fmt.Printf("  Own record:   %s — DISALLOWED (toggle it on in System Settings > Menu Bar)\n", bundleID)
	}

	for _, fo := range rep.foreignOwners {
		state := "allowed (harmless)"
		if !fo.allowed {
			state = "DISALLOWED — this is blocking Vox"
		}
		fmt.Printf("  Foreign owner: %s — %s\n", fo.id, state)
	}

	if !rep.blocked() {
		fmt.Println("  Verdict:      OK")
		return checkPass
	}

	fmt.Println("  Verdict:      BLOCKED")
	fmt.Println()
	fmt.Println("  Vox's own toggle in System Settings > Menu Bar cannot fix this: another app")
	fmt.Println("  that is switched OFF there has claimed Vox's status item. Either switch that")
	fmt.Println("  app ON in System Settings > Menu Bar, or remove the stale reference:")
	fmt.Println("      make doctor-fix")
	fmt.Println("  Then launch Vox from Finder or a plain terminal — not from inside an IDE or")
	fmt.Println("  agent — so Control Center attributes the item to Vox alone.")
	return checkFail
}

// ledgerReport is the result of inspecting trackedApplications for one app.
type ledgerReport struct {
	ownFound      bool
	ownAllowed    bool
	foreignOwners []ledgerOwner
}

// ledgerOwner is another app whose menuItemLocations reference our bundle ID.
type ledgerOwner struct {
	id      string
	allowed bool
}

// blocked is true when Control Center will hide the item: either our own
// record is disallowed, or some other app that references us is disallowed.
func (r ledgerReport) blocked() bool {
	if r.ownFound && !r.ownAllowed {
		return true
	}
	for _, fo := range r.foreignOwners {
		if !fo.allowed {
			return true
		}
	}
	return false
}

// analyzeStatusItemLedger decodes the nested trackedApplications plist and
// reports every record that references target. The blob is a flat array of
// dicts; the ones we care about have the shape
//
//	{ location: {bundle:{_0: ownerID}},
//	  menuItemLocations: [ {bundle:{_0: id}}, ... ],
//	  isAllowed: bool }
func analyzeStatusItemLedger(blob []byte, target string) (ledgerReport, error) {
	var entries []any
	if _, err := plist.Unmarshal(blob, &entries); err != nil {
		return ledgerReport{}, err
	}

	var rep ledgerReport
	for _, e := range entries {
		rec, ok := e.(map[string]any)
		if !ok {
			continue
		}
		locs, ok := rec["menuItemLocations"].([]any)
		if !ok {
			continue
		}
		owner := bundleIDOf(rec["location"])
		allowed, _ := rec["isAllowed"].(bool)

		if owner == target {
			rep.ownFound = true
			rep.ownAllowed = allowed
			continue
		}
		for _, l := range locs {
			if bundleIDOf(l) == target {
				rep.foreignOwners = append(rep.foreignOwners, ledgerOwner{id: owner, allowed: allowed})
				break
			}
		}
	}
	return rep, nil
}

// bundleIDOf extracts the _0 string from a {bundle:{_0: id}} node.
func bundleIDOf(node any) string {
	m, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	b, ok := m["bundle"].(map[string]any)
	if !ok {
		return ""
	}
	s, _ := b["_0"].(string)
	return s
}

// runningVoxPIDs finds live vox processes by executable path, independent of
// any pidfile.
func runningVoxPIDs() []int {
	out, err := exec.Command("pgrep", "-f", "Vox.app/Contents/MacOS/vox").Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, line := range strings.Fields(string(out)) {
		if p, err := strconv.Atoi(line); err == nil && p != os.Getpid() {
			pids = append(pids, p)
		}
	}
	return pids
}

func readPIDFile(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}
