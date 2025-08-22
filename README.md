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

- **For your current OS (default format):**
  ```sh
  fyne release
  ```

### Linux Package Formats

- **To create a `.deb` package:**
  ```sh
  fyne release deb
  ```

- **To create a `.rpm` package:**
  ```sh
  fyne release rpm