# adb-gui

A cross-platform GUI application for Android Debug Bridge (ADB) built with Go and Fyne.

## Prerequisites

- **Go:** Version 1.16 or later.
- **C Compiler:** Fyne requires a C compiler for building.
  - **Windows:** MSYS2 with MinGW 64-bit is required. Follow the setup guide on the [Fyne website](https://docs.fyne.io/started/).
  - **macOS:** Xcode Command Line Tools.
  - **Linux:** GCC or Clang. For creating `.deb` or `.rpm` packages, you will also need the appropriate packaging tools (`dpkg-deb`, `rpmbuild`).
- **Android Tools:** ADB and Fastboot must be installed and available in your system's PATH.

## Getting Started

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/lx19990999/adb-gui.git
    cd adb-gui
    ```

2.  **Install the Fyne CLI Tool:**
    ```sh
    go install fyne.io/tools/cmd/fyne@latest
    ```

3.  **Download Dependencies:**
    ```sh
    go mod tidy
    ```

## Development

To run the application for development, use the `go run` command.
```sh
go run .
```

**Linux Note:** If you experience rendering issues, you may need to enable software rendering:
```sh
LIBGL_ALWAYS_SOFTWARE=1 go run .
```

## Packaging for Release

To create a distributable application, use the `fyne release` command.

### Cross-Platform Packaging

- **For your current OS (default format):**
  ```sh
  fyne release
  ```

### Windows

- **To create a Windows executable (.exe):**
  ```sh
  fyne release -os windows
  ```

- **To create a Windows installer (.msi):**
  ```sh
  fyne release -os windows -package msi
  ```

### macOS

- **To create a macOS application bundle (.app):**
  ```sh
  fyne release -os darwin
  ```

- **To create a macOS installer (.dmg):**
  ```sh
  fyne release -os darwin -package dmg
  ```

### Linux

- **To create a Linux executable:**
  ```sh
  fyne release -os linux
  ```

- **To create a `.deb` package:**
  ```sh
  fyne release -os linux -package deb
  ```

- **To create a `.rpm` package:**
  ```sh
  fyne release -os linux -package rpm
  ```

- **To create an AppImage:**
  ```sh
  fyne release -os linux -package appimage
  ```

### Advanced Packaging Options

- **Specify application icon:**
  ```sh
  fyne release -icon Icon.png
  ```

- **Set application ID:**
  ```sh
  fyne release -id io.github.lx19990999.adb-gui
  ```

- **Create release with version:**
  ```sh
  fyne release -version 1.0.0
  ```

### Build Tags for Specific Platforms

For platform-specific builds, you can also use Go's build tags:

```sh
# Windows
GOOS=windows GOARCH=amd64 go build -o adb-gui.exe .

# macOS
GOOS=darwin GOARCH=amd64 go build -o adb-gui .

# Linux
GOOS=linux GOARCH=amd64 go build -o adb-gui .
```