//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func (l *Logger) initPlatform() error {
	stdout := windows.Handle(os.Stdout.Fd())
	var mode uint32
	err := windows.GetConsoleMode(stdout, &mode)
	if err == nil {
		mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
		windows.SetConsoleMode(stdout, mode)
		l.stdoutHandle = uintptr(stdout)
	} else {
		l.useColors = false
	}
	return nil
}

func (l *Logger) setConsoleColor(winColor uint16) {
	if !l.useColors {
		windows.SetConsoleMode(windows.Handle(l.stdoutHandle), uint32(winColor))
	}
}

func (l *Logger) resetConsoleColor() {
	if !l.useColors {
		windows.SetConsoleMode(windows.Handle(l.stdoutHandle), uint32(winFgDefault))
	}
}
