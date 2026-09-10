package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	osExecutable = os.Executable
	evalSymlinks = filepath.EvalSymlinks
	reExecFunc   = reExec
	osExit       = os.Exit
)

// Apply atomically replaces the running executable with newBinaryPath, then
// re-executes the new binary with the same arguments.
//
// Strategy (cross-platform):
//  1. Rename current exe → <exe>.old (works on Windows for running binaries)
//  2. Rename newBinaryPath → <exe> (atomic on POSIX; same-partition move on Windows)
//  3. syscall.Exec (POSIX) or os.StartProcess + exit (Windows) the new binary
//
// The caller must ensure the newBinaryPath is verified before calling Apply.
func Apply(newBinaryPath string) error {
	exePath, err := osExecutable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	// Resolve symlinks so we operate on the real file.
	exePath, err = evalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("eval symlinks: %w", err)
	}

	oldPath := exePath + ".old"

	// Step 1 — Move the current executable out of the way.
	// On Windows a running exe cannot be overwritten but CAN be renamed.
	if err := os.Rename(exePath, oldPath); err != nil {
		return fmt.Errorf("rename current exe to .old: %w", err)
	}

	// Step 2 — Move new binary into place.
	if err := os.Rename(newBinaryPath, exePath); err != nil {
		// Rollback: restore original.
		_ = os.Rename(oldPath, exePath)
		return fmt.Errorf("move a new binary into place: %w", err)
	}

	// Step 3 — Re-exec the new binary.
	args := os.Args
	if err := reExecFunc(exePath, args); err != nil {
		// Rollback.
		_ = os.Rename(exePath, newBinaryPath)
		_ = os.Rename(oldPath, exePath)
		return fmt.Errorf("re-exec failed: %w", err)
	}

	// reExec does not return on success (POSIX). On Windows we exit here.
	osExit(0)
	return nil
}

// cleanOldBinary removes any leftover "<exe>.old" from a previous upgrade.
func cleanOldBinary() {
	exePath, err := osExecutable()
	if err != nil {
		return
	}
	old := exePath + ".old"
	if _, err := os.Stat(old); err == nil {
		_ = os.Remove(old)
	}
}

