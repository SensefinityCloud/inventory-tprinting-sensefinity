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

	fmt.Println("=== InventoryT Printer Service ===")
	fmt.Println("Waiting for print requests...")
	fmt.Println("Service started")

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
	fmt.Println("Running connection test")

	client := resty.New()
	client.SetTimeout(5 * time.Second)
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	// Use configured endpoint
	endpoint := appConfig.TestEndpoint
	fmt.Printf("Testing connection to: %s\n", endpoint)

	resp, err := client.R().
		SetHeader("User-Agent", "InventoryPrinter/1.0").
		Get(endpoint)

	if err != nil {
		if strings.HasPrefix(endpoint, "https://") {
			fmt.Println("HTTPS connection failed, trying HTTP...")
			// Try HTTP as fallback
			httpEndpoint := "http://" + strings.TrimPrefix(endpoint, "https://")
			resp, err = client.R().
				SetHeader("User-Agent", "InventoryPrinter/1.0").
				Get(httpEndpoint)
		}

		if err != nil {
			fmt.Println("Connection test failed")
			fmt.Println("Please check:")
			fmt.Println("1. Is the server running?")
			fmt.Println("2. Is the correct endpoint configured?")
			fmt.Println("3. Is the firewall blocking the connection?")
			fmt.Printf("Current endpoint: %s\n", endpoint)
			fmt.Println("Press Enter to continue...")
			fmt.Scanln()
			return fmt.Errorf("connection failed: %v", err)
		}
	}

	fmt.Println("Connection test successful")
	fmt.Printf("Response: Status=%v, Body=%v\n", resp.Status(), string(resp.Body()))
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
	fmt.Printf("Notification: %s - %s\n", title, message)

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
			fmt.Printf("Failed to show Windows notification: %v\n", err)
		}
	case "linux":
		cmd := exec.Command("notify-send", title, message)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to show Linux notification: %v\n", err)
		}
	default:
		fmt.Printf("%s: %s\n", title, message)
	}
}

func handlePrintRequest(urlStr string) {
	fmt.Printf("Received print request: %s\n", urlStr)

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		fmt.Printf("Failed to parse URL: %v\n", err)
		showNotification("Error", fmt.Sprintf("Failed to parse URL: %v", err))
		return
	}

	fmt.Printf("Received URL: %s, Path: %s, Host: %s\n", urlStr, parsedURL.Path, parsedURL.Host)

	// Normalize the path by trimming slashes and comparing lowercase
	normalizedPath := strings.Trim(parsedURL.Path, "/")
	normalizedHost := strings.ToLower(parsedURL.Host)

	// Check for test command in both path and host
	isTest := normalizedPath == "test" || normalizedHost == "test"

	if isTest {
		err := handleTestCommand()
		if err == nil {
			fmt.Println("Test successful")
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

	fmt.Printf("Processing print request: %s\n", urlStr)
	// Parse the URL to get itemId and itemName
	queryParams := parsedURL.Query()
	itemId := queryParams.Get("id")
	itemName := queryParams.Get("name")

	// Create a dummy file to simulate the print job
	fileName, err := createDummyFile(itemId, itemName)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
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
		fmt.Println("OS Not supported")
		showNotification("Print Error", "OS Not supported")
		return
	}

	// Execute the command
	if err = cmd.Run(); err != nil {
		if strings.Contains(err.Error(), "Access is denied") {
			fmt.Println("Access denied. Please run the application with elevated privileges.")
			fmt.Println("Press Enter to continue...")
			fmt.Scanln()
			showNotification("Print Error", "Access denied. Please run the application with elevated privileges.")
		} else {
			fmt.Printf("Print failed: %v\n", err)
			showNotification("Print Error", fmt.Sprintf("Print failed: %v", err))
		}
		return
	}

	fmt.Printf("Successfully printed label for %s\n", itemName)
	showNotification("Print Success", fmt.Sprintf("Successfully printed label for %s", itemName))
}

// Add this new function to handle configuration commands
func handleConfigCommand(args []string) {
	if len(args) < 3 {
		fmt.Println("Usage: inventoryt-printer://config/endpoint?url=<new_endpoint>")
		return
	}

	queryParams, err := url.ParseQuery(args[2])
	if err != nil {
		fmt.Printf("Failed to parse query parameters: %v\n", err)
		return
	}

	newEndpoint := queryParams.Get("url")
	if newEndpoint == "" {
		fmt.Println("No endpoint URL provided")
		return
	}

	if err := SetTestEndpoint(newEndpoint); err != nil {
		fmt.Printf("Failed to save configuration: %v\n", err)
		return
	}

	fmt.Printf("Test endpoint updated to: %s\n", newEndpoint)
}
