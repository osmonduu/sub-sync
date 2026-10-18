//go:build !windows

package subsync

import "os/exec"

// hideConsoleWindow will not set any operating system-specific attributes
// for any other OS other than Windows.
func hideConsoleWindow(cmd *exec.Cmd) {
	// DO NOTHING
}
