# Inventory T-Printing

A cross-platform printer service that handles custom protocol URLs for label printing.

## Features

- Custom protocol handler (`inventoryt-printer://`)
- Automatic protocol registration
- Support for Windows and Linux
- System tray notifications
- Automatic privilege elevation when needed

## Building from Source

### Prerequisites

- Go 1.20 or higher
- Git

### Build Instructions

1. Clone the repository:
```bash
git clone https://github.com/yourusername/InventoryTPrinting.git
cd InventoryTPrinting
```

2. Build for your platform:

**Windows:**
```bash
go build -o inventory-printer.exe
```

**Linux:**
```bash
go build -o inventory-printer
```

### Cross-compilation

**For Windows (from Linux):**
```bash
GOOS=windows GOARCH=amd64 go build -o inventory-printer.exe
```

**For Linux (from Windows):**
```bash
set GOOS=linux
set GOARCH=amd64
go build -o inventory-printer
```

## Installation

1. Build the application for your platform
2. Run the executable once to register the protocol handler
3. The application will request elevated privileges if needed

## Usage

The application can be triggered using URLs in the following format:
```
inventoryt-printer://?id=123&name=ItemName
```

### Parameters:
- `id`: Item identifier
- `name`: Item name

## Development

The project structure:
- `main.go`: Core application logic and printing functionality
- `register.go`: Protocol registration handling
- `.github/workflows/release.yml`: CI/CD pipeline for automated builds

## CI/CD

The project uses GitHub Actions to automatically:
- Build for Windows and Linux
- Create releases with binaries
- Tag versions automatically

Builds are triggered on pushes to the main branch.
