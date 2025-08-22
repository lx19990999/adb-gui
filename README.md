# adb-gui

A cross-platform GUI application for Android Debug Bridge (ADB) built with Go and Fyne.

## Prerequisites

- Go 1.16 or later
- A C compiler (like GCC or Clang for Linux/macOS, or TDM-GCC/MinGW for Windows)
- Android ADB and Fastboot tools installed and available in your system's PATH.

## Getting Started

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/lx19990999/adb-gui.git
    cd adb-gui
    ```

2.  **Install Fyne dependencies:**
    This is required for building Fyne applications.
    ```sh
    go install fyne.io/fyne/v2/cmd/fyne@latest
    ```

3.  **Tidy the Go modules:**
    ```sh
    go mod tidy
    ```

## Build and Run Instructions

Below are the platform-specific instructions for building and running the application.

### Windows

1.  **Build the application:**
    Open a Command Prompt or PowerShell window and run:
    ```cmd
    go build -o adb-gui.exe ./cmd/adb-gui
    ```
    This will create an `adb-gui.exe` executable in the root directory.

2.  **Run the application:**
    Double-click the `adb-gui.exe` file or run it from the terminal:
    ```cmd
    .\adb-gui.exe
    ```

### macOS

1.  **Build the application:**
    Open a Terminal and run:
    ```sh
    go build -o adb-gui ./cmd/adb-gui
    ```
    This will create an `adb-gui` executable in the root directory.

2.  **Run the application:**
    ```sh
    ./adb-gui
    ```

### Linux

1.  **Build the application:**
    Open a terminal and run:
    ```sh
    go build -o adb-gui ./cmd/adb-gui
    ```
    This will create an `adb-gui` executable in the root directory.

2.  **Run the application:**
    ```sh
    ./adb-gui
    ```
    If you encounter rendering issues (e.g., a blank window), you may need to use software rendering:
    ```sh
    LIBGL_ALWAYS_SOFTWARE=1 ./adb-gui