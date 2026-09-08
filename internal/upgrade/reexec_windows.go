//go:build windows

package upgrade

import (
	"os"
	"os/exec"
)

// reExec on Windows spawns a new process and lets the current one exit.
// syscall.Exec is not available on Windows; os/exec is used instead.
func reExec(exePath string, args []string) error {
	cmd := exec.Command(exePath, args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return err
	}
	// Detach — the new process runs independently.
	// Returning nil signals Apply to call os.Exit(0).
	return nil
}
