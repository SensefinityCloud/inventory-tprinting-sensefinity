//go:build !windows
// +build !windows

package main

func windowsProtocolExists() bool {
	return false
}

func windowsNeedsPathUpdate() bool {
	return false
}

func registerWindowsProtocol(exePath string) {
	// Do nothing on non-Windows platforms
}
