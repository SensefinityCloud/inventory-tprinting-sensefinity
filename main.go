package main

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

func main() {
	// Load configuration first
	if err := LoadConfig(); err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		fmt.Println("Press Enter to exit...")
		fmt.Scanln()
		return
	}

	// Configure logger using values from config
	config := LoggerConfig{
		EnableFileLogging: appConfig.EnableFileLogging,
		LogFilePath:       appConfig.LogFilePath,
	}

	if err := InitLoggerWithConfig(config); err != nil {
		fmt.Printf("Failed to initialize logging: %v\n", err)
		fmt.Println("Press Enter to exit...")
		fmt.Scanln()
		return
	}
	defer CloseLogger()

	Info("=== InventoryT Printer Service ===")
	Info("Waiting for print requests...")
	Info("Service started")

	if len(os.Args) > 1 {
		url := os.Args[1]
		if strings.HasPrefix(url, "inventoryt-printer://") {
			handlePrintRequest(url)
			return
		}
	}

	// Check if protocol exists and if the path needs updating
	if ProtocolExists() {
		if NeedsPathUpdate() {
			RegisterCustomProtocolHandler()
			showNotification("Printer Service", "Application path has been updated")
		} else {
			showNotification("Printer Service", "Printer service is already registered")
		}
	} else {
		RegisterCustomProtocolHandler()
		showNotification("Printer Service", "Application initialized and ready to print")
	}
}

func handleTestCommand() error {
	Info("Running connection test")

	client := resty.New()
	client.SetTimeout(5 * time.Second)
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	// Use configured endpoint
	endpoint := appConfig.TestEndpoint
	Info(fmt.Sprintf("Testing connection to: %s", endpoint))

	resp, err := client.R().
		SetHeader("User-Agent", "InventoryPrinter/1.0").
		Get(endpoint)

	if err != nil {
		if strings.HasPrefix(endpoint, "https://") {
			Warning("HTTPS connection failed, trying HTTP...")
			// Try HTTP as fallback
			httpEndpoint := "http://" + strings.TrimPrefix(endpoint, "https://")
			resp, err = client.R().
				SetHeader("User-Agent", "InventoryPrinter/1.0").
				Get(httpEndpoint)
		}

		if err != nil {
			Error("Connection test failed")
			Warning("Please check:")
			Warning("1. Is the server running?")
			Warning("2. Is the correct endpoint configured?")
			Warning("3. Is the firewall blocking the connection?")
			Warning(fmt.Sprintf("Current endpoint: %s", endpoint))
			Warning("Press Enter to continue...")
			fmt.Scanln()
			return fmt.Errorf("connection failed: %v", err)
		}
	}

	Success("Connection test successful")
	Info(fmt.Sprintf("Response: Status=%v, Body=%v", resp.Status(), string(resp.Body())))
	return nil
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
	Info(fmt.Sprintf("Notification: %s - %s", title, message))

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
			$notify.ShowBalloonTip(5000)
			$notify.Dispose()
		`, title, message))
		if err := cmd.Run(); err != nil {
			Error(fmt.Sprintf("Failed to show Windows notification: %v", err))
		}
	case "linux":
		cmd := exec.Command("notify-send", title, message)
		if err := cmd.Run(); err != nil {
			Error(fmt.Sprintf("Failed to show Linux notification: %v", err))
		}
	default:
		Info(fmt.Sprintf("%s: %s", title, message))
	}
}

func handlePrintRequest(urlStr string) {
	Info(fmt.Sprintf("Received print request: %s", urlStr))

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		Error(fmt.Sprintf("Failed to parse URL: %v", err))
		showNotification("Error", fmt.Sprintf("Failed to parse URL: %v", err))
		return
	}

	Info(fmt.Sprintf("Received URL: %s, Path: %s, Host: %s", urlStr, parsedURL.Path, parsedURL.Host))

	// Normalize the path by trimming slashes and comparing lowercase
	normalizedPath := strings.Trim(parsedURL.Path, "/")
	normalizedHost := strings.ToLower(parsedURL.Host)

	// Check for test command in both path and host
	isTest := normalizedPath == "test" || normalizedHost == "test"

	if isTest {
		err := handleTestCommand()
		if err == nil {
			Info("Test successful")
			showNotification("Test Success", "Connection test successful")
			return
		}
		return
	}

	// Add configuration handling
	if normalizedPath == "config" || normalizedHost == "config" {
		handleConfigCommand(os.Args)
		return
	}

	Info(fmt.Sprintf("Processing print request: %s", urlStr))
	// Parse the URL to get itemId and itemName
	queryParams := parsedURL.Query()
	itemId := queryParams.Get("id")
	itemName := queryParams.Get("name")

	// Create a dummy file to simulate the print job
	fileName, err := createDummyFile(itemId, itemName)
	if err != nil {
		Error(fmt.Sprintf("Failed to create file: %v", err))
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
		Error("OS Not supported")
		showNotification("Print Error", "OS Not supported")
		return
	}

	// Execute the command
	if err = cmd.Run(); err != nil {
		if strings.Contains(err.Error(), "Access is denied") {
			Error("Access denied. Please run the application with elevated privileges.")
			Warning("Press Enter to continue...")
			fmt.Scanln()
			showNotification("Print Error", "Access denied. Please run the application with elevated privileges.")
		} else {
			Error(fmt.Sprintf("Print failed: %v", err))
			showNotification("Print Error", fmt.Sprintf("Print failed: %v", err))
		}
		return
	}

	Success(fmt.Sprintf("Successfully printed label for %s", itemName))
	showNotification("Print Success", fmt.Sprintf("Successfully printed label for %s", itemName))
}

// Add this new function to handle configuration commands
func handleConfigCommand(args []string) {
	if len(args) < 3 {
		Error("Usage: inventoryt-printer://config/endpoint?url=<new_endpoint>")
		return
	}

	queryParams, err := url.ParseQuery(args[2])
	if err != nil {
		Error(fmt.Sprintf("Failed to parse query parameters: %v", err))
		return
	}

	newEndpoint := queryParams.Get("url")
	if newEndpoint == "" {
		Error("No endpoint URL provided")
		return
	}

	if err := SetTestEndpoint(newEndpoint); err != nil {
		Error(fmt.Sprintf("Failed to save configuration: %v", err))
		return
	}

	Success(fmt.Sprintf("Test endpoint updated to: %s", newEndpoint))
}
