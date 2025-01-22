package main

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// Change the test endpoint to a reliable test service
const testEndpoint = "https://inventory.sensefinity.com/apptest"

func main() {
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

	endpoint := testEndpoint
	fmt.Printf("Testing connection to: %s\n", endpoint)

	resp, err := client.R().
		SetHeader("User-Agent", "InventoryPrinter/1.0").
		Get(endpoint)

	if err != nil {
		fmt.Printf("Network error: %v\n", err)
		if strings.Contains(err.Error(), "no such host") {
			fmt.Println("Internet connection might be down or DNS resolution failed")
		} else if strings.Contains(err.Error(), "connection refused") {
			fmt.Println("Server is not accepting connections")
		} else if strings.Contains(err.Error(), "timeout") {
			fmt.Println("Connection timed out - server might be slow or unreachable")
		}
		fmt.Println("Press Enter to continue...")
		fmt.Scanln()
		return fmt.Errorf("connection failed: %v", err)
	}

	if resp.StatusCode() != 200 {
		fmt.Printf("Server returned error status: %d\n", resp.StatusCode())
		fmt.Println("Press Enter to continue...")
		fmt.Scanln()
		return fmt.Errorf("server error: %d", resp.StatusCode())
	}

	fmt.Println("Connection test successful!")
	fmt.Printf("Server responded with status: %d\n", resp.StatusCode())
	return nil
}

func createZPLFile(itemId, itemName string) (string, error) {
	// Create labels directory if it doesn't exist
	labelsDir := "./labels"
	if err := os.MkdirAll(labelsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create labels directory: %v", err)
	}

	// Generate ZPL code
	labelHeight := 73
	centerY := labelHeight / 2
	textX := 155
	textY := centerY - 10
	barcodeX := 0
	barcodeY := centerY - 7

	// Truncate item name if longer than 28 characters
	if len(itemName) > 28 {
		itemName = itemName[:28]
	}

	// Build ZPL code
	zplInit := "^XA"
	zplName := fmt.Sprintf("^FT%d,%d^A0N,18,18^FD%s^FS", textX, textY, itemName)
	zplBarcode := fmt.Sprintf("^FT%d,%d^BY1^BCN,30,Y,N,N^FD%s^FS", barcodeX, barcodeY, itemId)
	zplEnd := "^XZ"
	zplCode := zplInit + zplBarcode + zplName + zplEnd

	// Create and write to file
	fileName := filepath.Join(labelsDir, itemId+".txt")
	err := os.WriteFile(fileName, []byte(zplCode), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write ZPL file: %v", err)
	}

	return fileName, nil
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
		}
		return
	}

	fmt.Printf("Processing print request: %s\n", urlStr)
	// Parse the URL to get itemId and itemName
	queryParams := parsedURL.Query()
	itemId := queryParams.Get("id")
	itemName := queryParams.Get("name")

	// Create a ZPL file to simulate the print job
	fileName, err := createZPLFile(itemId, itemName)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		showNotification("Print Error", fmt.Sprintf("Failed to create file: %v", err))
		return
	}

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
		os.Remove(fileName) // Clean up file on error
		return
	}

	fmt.Printf("Successfully printed label for %s\n", itemName)
	showNotification("Print Success", fmt.Sprintf("Successfully printed label for %s", itemName))

	// Wait a bit to ensure the print spooler has processed the file
	time.Sleep(2 * time.Second)

	// Clean up file after successful print
	if err := os.Remove(fileName); err != nil {
		fmt.Printf("Warning: Failed to clean up temporary file %s: %v\n", fileName, err)
	}
}
