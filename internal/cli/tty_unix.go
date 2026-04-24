//go:build !windows

package cli

import (
	"os/exec"
)

// setRawEcho toggles terminal echo via stty. This is best-effort; if
// stdin is not a TTY the exec call fails harmlessly and we silently
// fall back to echoing input.
func setRawEcho(on bool) error {
	arg := "-echo"
	if on {
		arg = "echo"
	}
	cmd := exec.Command("stty", arg)
	cmd.Stdin = nil
	// We don't wire stdin because stty probes /dev/tty; skipping keeps
	// the call cheap.
	return cmd.Run()
}
