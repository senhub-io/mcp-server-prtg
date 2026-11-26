//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

func openElevatedConsole(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	cmdline := syscall.EscapeArg(exe)
	for _, a := range args {
		cmdline += " " + syscall.EscapeArg(a)
	}

	// cmd /k strips one pair of outer quotes, so the whole command line is
	// wrapped once more to keep the quoted executable path intact.
	params := `/k "` + cmdline + `"`

	ptrs := make([]*uint16, 4)
	for i, s := range []string{"runas", "cmd.exe", params, filepath.Dir(exe)} {
		if ptrs[i], err = windows.UTF16PtrFromString(s); err != nil {
			return fmt.Errorf("encode console command: %w", err)
		}
	}

	verb, file, argv, dir := ptrs[0], ptrs[1], ptrs[2], ptrs[3]

	if err := windows.ShellExecute(0, verb, file, argv, dir, windows.SW_SHOWNORMAL); err != nil {
		if errors.Is(err, windows.ERROR_CANCELLED) {
			return fmt.Errorf("elevation was cancelled: %w", err)
		}

		return fmt.Errorf("open elevated console: %w", err)
	}

	return nil
}
