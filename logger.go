package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type LoggerConfig struct {
	EnableFileLogging bool
	LogFilePath       string
}

type Logger struct {
	file         *os.File
	useColors    bool
	stdoutHandle uintptr
	config       LoggerConfig
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorWhite  = "\033[37m" // Changed from blue to white for Info
	colorPurple = "\033[35m"
)

// Windows console colors
const (
	winFgBlue    = 0x0001
	winFgGreen   = 0x0002
	winFgRed     = 0x0004
	winFgWhite   = 0x0007 // Added white color
	winFgYellow  = winFgRed | winFgGreen
	winFgDefault = winFgWhite
)

var logger *Logger
var DefaultConfig = LoggerConfig{
	EnableFileLogging: true,
	LogFilePath:       filepath.Join(os.TempDir(), "inventoryt-printer.log"),
}

func InitLoggerWithConfig(config LoggerConfig) error {
	logger = &Logger{
		useColors: true,
		config:    config,
	}

	if config.EnableFileLogging {
		f, err := os.OpenFile(config.LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %v", err)
		}
		logger.file = f
		log.SetOutput(f)
	}

	return logger.initPlatform()
}

func InitLogger() error {
	return InitLoggerWithConfig(DefaultConfig)
}

func CloseLogger() {
	if logger != nil && logger.file != nil {
		logger.file.Close()
	}
}

func (l *Logger) log(level, color string, winColor uint16, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, message)

	// Write to log file if enabled
	if l.config.EnableFileLogging && l.file != nil {
		log.Print(logEntry)
	}

	// Write to console with color
	if runtime.GOOS == "windows" && !l.useColors {
		l.setConsoleColor(winColor)
		fmt.Print(logEntry)
		l.resetConsoleColor()
	} else if l.useColors {
		fmt.Printf("%s%s%s", color, logEntry, colorReset)
	} else {
		fmt.Print(logEntry)
	}
}

func Info(message string) {
	logger.log("INFO", colorWhite, winFgWhite, message)
}

func Success(message string) {
	logger.log("SUCCESS", colorGreen, winFgGreen, message)
}

func Error(message string) {
	logger.log("ERROR", colorRed, winFgRed, message)
}

func Warning(message string) {
	logger.log("WARNING", colorYellow, winFgYellow, message)
}

// Add Debug level logging
func Debug(message string) {
	logger.log("DEBUG", colorPurple, winFgBlue|winFgRed, message)
}

// Add Fatal level logging
func Fatal(message string) {
	logger.log("FATAL", colorRed, winFgRed, message)
	os.Exit(1)
}

// Add this new function after the existing logging functions
func FatalWithWait(message string) {
	logger.log("FATAL", colorRed, winFgRed, message)
	logger.log("INFO", colorWhite, winFgWhite, "Press Enter to exit...")
	fmt.Scanln() // Wait for user input
	os.Exit(1)
}
