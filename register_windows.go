//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows"
)

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
