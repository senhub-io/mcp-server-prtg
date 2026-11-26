//go:build !windows

package main

func openElevatedConsole(_ []string) error {
	return errWindowUnsupported
}
