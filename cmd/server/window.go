package main

import "errors"

var errWindowUnsupported = errors.New("--window is only supported on Windows")

func clientArgsWithoutWindow(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a != "--window" {
			out = append(out, a)
		}
	}
	return out
}
