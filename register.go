package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/windows"
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
		fmt.Printf("Failed to get executable path: %v\n", err)
		return
	}

	osType := runtime.GOOS
	switch osType {
	case "windows":
		registerWindowsProtocol(exePath)
	case "linux":
		registerLinuxProtocol(exePath)
	default:
		fmt.Println("OS not supported for protocol registration")
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

// Windows Implementation
// ----------------------------------------

func windowsProtocolExists() bool {
	cmd := exec.Command("powershell", "-Command", `
		Test-Path -Path "HKLM:\SOFTWARE\Classes\inventoryt-printer"
	`)
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) == "True"
}

func windowsNeedsPathUpdate() bool {
	cmd := exec.Command("powershell", "-Command", `
		(Get-ItemProperty -Path "HKLM:\SOFTWARE\Classes\inventoryt-printer\shell\open\command").'(Default)'
	`)
	output, err := cmd.Output()
	if err != nil {
		return true
	}

	currentExePath, err := os.Executable()
	if err != nil {
		return true
	}

	registryPath := strings.TrimSpace(string(output))
	registryPath = strings.Trim(registryPath, `"`)
	registryPath = strings.Split(registryPath, `" "`)[0]

	return !strings.EqualFold(registryPath, currentExePath)
}

func registerWindowsProtocol(exePath string) {
	if !isAdmin() {
		elevatePrivileges()
		return
	}

	psCmd := fmt.Sprintf(`
		New-Item -Path "HKLM:\SOFTWARE\Classes\inventoryt-printer" -Force
		Set-ItemProperty -Path "HKLM:\SOFTWARE\Classes\inventoryt-printer" -Name "(Default)" -Value "URL:Inventory Printer Protocol"
		Set-ItemProperty -Path "HKLM:\SOFTWARE\Classes\inventoryt-printer" -Name "URL Protocol" -Value ""
		New-Item -Path "HKLM:\SOFTWARE\Classes\inventoryt-printer\shell\open\command" -Force
		Set-ItemProperty -Path "HKLM:\SOFTWARE\Classes\inventoryt-printer\shell\open\command" -Name "(Default)" -Value '"%s" "%%1"'
	`, strings.ReplaceAll(exePath, `\`, `\\`))

	cmd := exec.Command("powershell", "-Command", psCmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to register Windows protocol: %v\nOutput: %s\n", err, string(out))
		return
	}

	fmt.Println("Windows protocol handler registered successfully")
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
		fmt.Printf("Failed to create directory: %v\n", err)
		return
	}

	if err := os.WriteFile(desktopPath, []byte(desktopEntry), 0755); err != nil {
		fmt.Printf("Failed to create .desktop file: %v\n", err)
		return
	}

	cmd := exec.Command("xdg-mime", "default", "inventoryt-printer.desktop", "x-scheme-handler/inventoryt-printer")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to register Linux protocol: %v\n", err)
		return
	}

	fmt.Println("Linux protocol handler registered successfully")
}

// Utility Functions
// ----------------------------------------

func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func elevatePrivileges() {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()

	verbPtr, _ := windows.UTF16PtrFromString(verb)
	exePtr, _ := windows.UTF16PtrFromString(exe)
	cwdPtr, _ := windows.UTF16PtrFromString(cwd)
	argPtr, _ := windows.UTF16PtrFromString("")

	var showCmd int32 = 1 //SW_NORMAL

	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		fmt.Printf("Failed to elevate privileges: %v\n", err)
	}
	os.Exit(0)
}
