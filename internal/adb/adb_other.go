//go:build !windows

package adb

import "os/exec"

// hideWindowsWindow 在非Windows系统下的空实现
func hideWindowsWindow(cmd *exec.Cmd) {
	// 在非Windows系统下，这个函数什么都不做
}
