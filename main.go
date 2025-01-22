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
			fmt.Println("Printer Service: Application path has been updated")
		} else {
			fmt.Println("Printer Service: Printer service is already registered")
		}
	} else {
		RegisterCustomProtocolHandler()
		fmt.Println("Printer Service: Application initialized and ready to print")
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

	// Create and write to temp file
	tempFile, err := os.CreateTemp("", "zpl-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	if _, err := tempFile.Write([]byte(zplCode)); err != nil {
		return "", fmt.Errorf("failed to write ZPL data: %v", err)
	}

	return tempFile.Name(), nil
}

func handlePrintRequest(urlStr string) {
	fmt.Printf("Received print request: %s\n", urlStr)

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		fmt.Printf("Failed to parse URL: %v\n", err)
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
		return
	}

	// Execute the command
	if err = cmd.Run(); err != nil {
		if strings.Contains(err.Error(), "Access is denied") {
			fmt.Println("Access denied. Please run the application with elevated privileges.")
			fmt.Println("Press Enter to continue...")
			fmt.Scanln()
		} else {
			fmt.Printf("Print failed: %v\n", err)
		}
		os.Remove(fileName) // Clean up file on error
		return
	}

	fmt.Printf("Successfully printed label for %s\n", itemName)

	// Wait a bit to ensure the print spooler has processed the file
	time.Sleep(2 * time.Second)

	// Clean up file after successful print
	if err := os.Remove(fileName); err != nil {
		fmt.Printf("Warning: Failed to clean up temporary file %s: %v\n", fileName, err)
	}
}
