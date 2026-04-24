//go:build windows

package cli

// Windows stub — input is echoed. Users should prefer the --token flag
// or an env var (BT_TOKEN) to avoid leaking secrets into scrollback.
func setRawEcho(on bool) error { return nil }
