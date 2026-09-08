//go:build !windows

package upgrade

import "syscall"

// reExec replaces the current process image with the new binary (POSIX exec).
func reExec(exePath string, args []string) error {
	return syscall.Exec(exePath, args, syscall.Environ())
}
