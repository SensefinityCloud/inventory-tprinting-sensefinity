package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
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
	client := resty.New()
	client.SetTimeout(5 * time.Second)

	resp, err := client.R().
		SetHeader("User-Agent", "InventoryPrinter/1.0").
		Get("https://inventory.sensefinity.com/apptest")

	if err != nil {
		log.Printf("Error making request: %v", err)
		return fmt.Errorf("connection failed: %v", err)
	}

	log.Printf("Response: Status=%v, Body=%v", resp.Status(), string(resp.Body()))
	return nil
}

func handlePrintRequest(urlStr string) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		showNotification("Error", fmt.Sprintf("Failed to parse URL: %v", err))
		return
	}

	log.Printf("Received URL: %s, Path: %s, Host: %s", urlStr, parsedURL.Path, parsedURL.Host)

	// Normalize the path by trimming slashes and comparing lowercase
	normalizedPath := strings.Trim(parsedURL.Path, "/")
	normalizedHost := strings.ToLower(parsedURL.Host)

	// Check for test command in both path and host
	isTest := normalizedPath == "test" || normalizedHost == "test"

	if isTest {
		err := handleTestCommand()
		if err == nil {
			log.Printf("Test successful")
			showNotification("Test Success", "Connection test successful")
			return
		}
		return
	}

	log.Printf("Processing print request: %s", urlStr)
	// Parse the URL to get itemId and itemName
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
