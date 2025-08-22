//go:build windows

package adb

import (
	"os/exec"
	"syscall"
)

// hideWindowsWindow 在Windows下隐藏CMD窗口
func hideWindowsWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW 常量值
	}
}
