package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Public API
// ----------------------------------------

func ProtocolExists() bool {
	osType := runtime.GOOS
	switch osType {
	case "windows":
		return windowsProtocolExists()
	case "linux":
		return linuxProtocolExists()
	default:
		return false
	}
}

func RegisterCustomProtocolHandler() {
	exePath, err := os.Executable()
	if err != nil {
		Error(fmt.Sprintf("Failed to get executable path: %v", err))
		return
	}

	osType := runtime.GOOS
	switch osType {
	case "windows":
		registerWindowsProtocol(exePath)
	case "linux":
		registerLinuxProtocol(exePath)
	default:
		Warning("OS not supported for protocol registration")
	}
}

func NeedsPathUpdate() bool {
	osType := runtime.GOOS
	switch osType {
	case "windows":
		return windowsNeedsPathUpdate()
	case "linux":
		return linuxNeedsPathUpdate()
	default:
		return false
	}
}

// Linux Implementation
// ----------------------------------------

func linuxProtocolExists() bool {
	desktopPath := fmt.Sprintf("%s/.local/share/applications/inventoryt-printer.desktop", os.Getenv("HOME"))
	_, err := os.Stat(desktopPath)
	return !errors.Is(err, os.ErrNotExist)
}

func linuxNeedsPathUpdate() bool {
	currentExePath, err := os.Executable()
	if err != nil {
		return true
	}

	desktopPath := fmt.Sprintf("%s/.local/share/applications/inventoryt-printer.desktop", os.Getenv("HOME"))
	content, err := os.ReadFile(desktopPath)
	if err != nil {
		return true
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Exec=") {
			registeredPath := strings.TrimPrefix(line, "Exec=")
			registeredPath = strings.Split(registeredPath, " ")[0]
			return registeredPath != currentExePath
		}
	}
	return true
}

func registerLinuxProtocol(exePath string) {
	desktopEntry := fmt.Sprintf(`[Desktop Entry]
Name=Inventory Printer
Exec=%s %%u
Type=Application
Terminal=false
Categories=Application;
MimeType=x-scheme-handler/inventoryt-printer;
`, exePath)

	desktopPath := fmt.Sprintf("%s/.local/share/applications/inventoryt-printer.desktop", os.Getenv("HOME"))
	if err := os.MkdirAll(filepath.Dir(desktopPath), 0755); err != nil {
		Error(fmt.Sprintf("Failed to create directory: %v", err))
		return
	}

	if err := os.WriteFile(desktopPath, []byte(desktopEntry), 0755); err != nil {
		Error(fmt.Sprintf("Failed to create .desktop file: %v", err))
		return
	}

	cmd := exec.Command("xdg-mime", "default", "inventoryt-printer.desktop", "x-scheme-handler/inventoryt-printer")
	if err := cmd.Run(); err != nil {
		Error(fmt.Sprintf("Failed to register Linux protocol: %v", err))
		return
	}

	Success("Linux protocol handler registered successfully")
}
