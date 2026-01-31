package tui

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OSOpenCmd allows mocking the open command.
var OSOpenCmd = func(url string) *exec.Cmd {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		return nil
	}
	return exec.Command(cmd, args...) //nolint:gosec
}

func openBrowser(url string) error {
	cmd := OSOpenCmd(url)
	if cmd == nil {
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
