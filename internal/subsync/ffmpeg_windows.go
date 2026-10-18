package subsync

import (
	"os/exec"
	"syscall"
)

// hideConsoleWindow sets the CREATE_NO_WINDOW creation flag in exec.Cmd.SysProcAttr
// which prevents the console window from popping up on Windows when ffmpeg is called.
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}	// CREATE_NO_WINDOW
}

