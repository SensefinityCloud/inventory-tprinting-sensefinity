//go:build !windows

package main

func (l *Logger) initPlatform() error {
	l.useColors = true
	return nil
}

func (l *Logger) setConsoleColor(color string, _ uint16) {
	// Do nothing for non-Windows platforms
}

func (l *Logger) resetConsoleColor() {
	// Do nothing for non-Windows platforms
}
