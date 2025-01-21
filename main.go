package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/windows"
)

func main() {
	if len(os.Args) > 1 {
		url := os.Args[1]
		if strings.HasPrefix(url, "inventoryt-printer://") {
			handlePrintRequest(url)
			return
		}
	}

	// Check if protocol exists and if the path needs updating
	if protocolExists() {
		if needsPathUpdate() {
			registerCustomProtocolHandler()
			showNotification("Printer Service", "Application path has been updated")
		} else {
			showNotification("Printer Service", "Printer service is already registered")
		}
	} else {
		registerCustomProtocolHandler()
		showNotification("Printer Service", "Application initialized and ready to print")
	}
}

func protocolExists() bool {
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

func windowsProtocolExists() bool {
	cmd := exec.Command("powershell", "-Command", `
		Test-Path -Path "HKLM:\SOFTWARE\Classes\inventoryt-printer"
	`)
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) == "True"
}

func linuxProtocolExists() bool {
	desktopPath := fmt.Sprintf("%s/.local/share/applications/inventoryt-printer.desktop", os.Getenv("HOME"))
	_, err := os.Stat(desktopPath)
	return !errors.Is(err, os.ErrNotExist)
}

func registerCustomProtocolHandler() {
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

func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func registerWindowsProtocol(exePath string) {
	if !isAdmin() {
		// Re-run the application with elevated privileges
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
		return
	}

	// Use PowerShell to set registry entries with proper escaping
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

func registerLinuxProtocol(exePath string) {
	desktopEntry := fmt.Sprintf(`[Desktop Entry]
Name=Inventory Printer
Exec=%s %%u
Type=Application
Terminal=false
Categories=Application;
MimeType=x-scheme-handler/inventoryt-printer;
`, exePath)

	// Create .desktop file
	desktopPath := fmt.Sprintf("%s/.local/share/applications/inventoryt-printer.desktop", os.Getenv("HOME"))
	if err := os.MkdirAll(filepath.Dir(desktopPath), 0755); err != nil {
		fmt.Printf("Failed to create directory: %v\n", err)
		return
	}

	if err := os.WriteFile(desktopPath, []byte(desktopEntry), 0755); err != nil {
		fmt.Printf("Failed to create .desktop file: %v\n", err)
		return
	}

	// Register protocol handler
	cmd := exec.Command("xdg-mime", "default", "inventoryt-printer.desktop", "x-scheme-handler/inventoryt-printer")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to register Linux protocol: %v\n", err)
		return
	}
	fmt.Println("Linux protocol handler registered successfully")
}

func handlePrintRequest(urlStr string) {
	// Parse the URL to get itemId and itemName
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		showNotification("Print Error", fmt.Sprintf("Failed to parse URL: %v", err))
		return
	}

	queryParams := parsedURL.Query()
	itemId := queryParams.Get("id")
	itemName := queryParams.Get("name")

	// Create a dummy file to simulate the print job
	fileName, err := createDummyFile(itemId, itemName)
	if err != nil {
		showNotification("Print Error", fmt.Sprintf("Failed to create file: %v", err))
		return
	}
	defer os.Remove(fileName) // Ensure the file is removed after use

	// Determine the OS and execute the appropriate print command
	osType := runtime.GOOS
	var cmd *exec.Cmd

	switch osType {
	case "windows":
		computerName := os.Getenv("COMPUTERNAME")
		cmd = exec.Command("cmd", "/C", "copy", fileName, fmt.Sprintf("\\\\%s\\ZD420", computerName))
	case "linux":
		cmd = exec.Command("lpr", "-l", fileName)
	default:
		showNotification("Print Error", "OS Not supported")
		return
	}

	// Execute the command
	if err = cmd.Run(); err != nil {
		if strings.Contains(err.Error(), "Access is denied") {
			showNotification("Print Error", "Access denied. Please run the application with elevated privileges.")
		} else {
			showNotification("Print Error", fmt.Sprintf("Print failed: %v", err))
		}
		return
	}

	showNotification("Print Success", fmt.Sprintf("Successfully printed label for %s", itemName))
}

func createDummyFile(itemId, itemName string) (string, error) {
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s_%s_*.txt", itemId, itemName))
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString("This is a dummy print file.")
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func showNotification(title, message string) {
	osType := runtime.GOOS
	switch osType {
	case "windows":
		cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms
			$notify = New-Object System.Windows.Forms.NotifyIcon
			$notify.Icon = [System.Drawing.SystemIcons]::Information
			$notify.BalloonTipIcon = "Info"
			$notify.BalloonTipTitle = "%s"
			$notify.BalloonTipText = "%s"
			$notify.Visible = $True
			$notify.ShowBalloonTip(5000)
			Start-Sleep -Seconds 5
			$notify.Dispose()
		`, title, message))
		cmd.Run()
	case "linux":
		cmd := exec.Command("notify-send", title, message)
		cmd.Run()
	default:
		fmt.Printf("%s: %s\n", title, message)
	}
}

func needsPathUpdate() bool {
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

	// Clean up the registry value for comparison
	registryPath := strings.TrimSpace(string(output))
	registryPath = strings.Trim(registryPath, `"`)
	registryPath = strings.Split(registryPath, `" "`)[0] // Remove the "%1" part

	return !strings.EqualFold(registryPath, currentExePath)
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
			registeredPath = strings.Split(registeredPath, " ")[0] // Remove the %u part
			return registeredPath != currentExePath
		}
	}
	return true
}
