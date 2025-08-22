package adb

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Manager encapsulates ADB operations and path configuration.
type Manager struct {
	Path string
}

func NewManager(path string) *Manager {
	if path == "" {
		path = AutoDetect()
	}
	return &Manager{Path: path}
}

func (m *Manager) IsAvailable() bool {
	if m.Path != "" {
		if _, err := os.Stat(m.Path); err == nil {
			return true
		}
	}
	if _, err := exec.LookPath("adb"); err == nil {
		return true
	}
	return false
}

// Exec runs adb with provided args and returns combined output.
func (m *Manager) Exec(args ...string) (string, error) {
	bin := m.Path
	if bin == "" {
		bin = "adb"
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()

	// 在Windows下隐藏CMD窗口
	if runtime.GOOS == "windows" {
		hideWindowsWindow(cmd)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ExecSerial runs adb for a specific serial by injecting "-s <serial>".
func (m *Manager) ExecSerial(serial string, args ...string) (string, error) {
	if strings.TrimSpace(serial) != "" {
		args = append([]string{"-s", serial}, args...)
	}
	return m.Exec(args...)
}

// ExecFastboot runs a fastboot command.
func (m *Manager) ExecFastboot(serial string, args ...string) (string, error) {
	// Fastboot may not be in the same directory as adb, so we look for it in the path.
	bin, err := exec.LookPath("fastboot")
	if err != nil {
		return "", errors.New("fastboot executable not found in PATH")
	}
	if strings.TrimSpace(serial) != "" {
		args = append([]string{"-s", serial}, args...)
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()

	// 在Windows下隐藏CMD窗口
	if runtime.GOOS == "windows" {
		hideWindowsWindow(cmd)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (m *Manager) Version() (string, error) {
	return m.Exec("version")
}

// EnsureServer starts the adb server if it's not already running.
func (m *Manager) EnsureServer() {
	_, _ = m.Exec("start-server")
}

type Device struct {
	Serial      string
	State       string
	Product     string
	Model       string
	Device      string
	TransportID string
}

func (m *Manager) Devices() ([]Device, string, error) {
	m.EnsureServer()
	// Get ADB devices
	adbOut, adbErr := m.Exec("devices", "-l")
	adbDevices := parseDevices(adbOut)

	// Get Fastboot devices
	fbOut, _ := m.ExecFastboot("", "devices")
	fbDevices := parseFastbootDevices(fbOut)

	// Merge lists, giving precedence to ADB info if a device is in both
	merged := make(map[string]Device)
	for _, d := range adbDevices {
		merged[d.Serial] = d
	}
	for _, d := range fbDevices {
		if _, exists := merged[d.Serial]; !exists {
			merged[d.Serial] = d
		}
	}

	var result []Device
	for _, d := range merged {
		result = append(result, d)
	}

	return result, adbOut + "\n" + fbOut, adbErr
}

// Sideload puts the device into sideload mode and installs a package.
func (m *Manager) Sideload(serial, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("sideload path cannot be empty")
	}
	return m.ExecSerial(serial, "sideload", path)
}

// StartShizuku executes the Shizuku start script.
func (m *Manager) StartShizuku(serial string) (string, error) {
	return m.ExecSerial(serial, "shell", "sh", "/sdcard/Android/data/moe.shizuku.privileged.api/start.sh")
}

func parseFastbootDevices(output string) []Device {
	var devices []Device
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "fastboot" {
			devices = append(devices, Device{
				Serial: fields[0],
				State:  "fastboot",
			})
		}
	}
	return devices
}

func parseDevices(output string) []Device {
	var res []Device
	lines := strings.Split(output, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		// Skip headers and noise from server startup
		if strings.HasPrefix(ln, "List of devices") ||
			strings.HasPrefix(ln, "*") ||
			strings.Contains(ln, "daemon") ||
			strings.Contains(ln, "adb server") {
			continue
		}
		f := strings.Fields(ln)
		if len(f) < 2 {
			continue
		}
		d := Device{}
		d.Serial = f[0]
		rest := f[1:]
		// If second token is state (no colon), record it
		if len(rest) > 0 && !strings.Contains(rest[0], ":") {
			d.State = rest[0]
			rest = rest[1:]
		}
		for _, tok := range rest {
			kv := strings.SplitN(tok, ":", 2)
			if len(kv) != 2 {
				continue
			}
			key := kv[0]
			val := kv[1]
			switch key {
			case "product":
				d.Product = val
			case "model":
				d.Model = val
			case "device":
				d.Device = val
			case "transport_id":
				d.TransportID = val
			}
		}
		// Only append if we have at least a serial and a state or some details
		if d.Serial != "" {
			res = append(res, d)
		}
	}
	return res
}

// InstalledPackages returns a list of installed package names for the device.
func (m *Manager) InstalledPackages(serial string) ([]string, string, error) {
	m.EnsureServer()
	_, _ = m.ExecSerial(serial, "wait-for-device")
	out, err := m.ExecSerial(serial, "shell", "pm", "list", "packages")
	if err != nil {
		return nil, out, err
	}
	var pkgs []string
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		// "package:com.example.app"
		if strings.HasPrefix(ln, "package:") {
			ln = strings.TrimPrefix(ln, "package:")
		}
		pkgs = append(pkgs, ln)
	}
	return pkgs, out, nil
}

// User represents a device user.
type User struct {
	ID    int
	Name  string
	State string
}

// Users lists users on the device using "cmd user list" or "pm list users" fallback.
func (m *Manager) Users(serial string) ([]User, string, error) {
	// Ensure server is up to avoid empty results on some environments
	m.EnsureServer()
	out, err := m.ExecSerial(serial, "shell", "cmd", "user", "list")
	if err != nil || !strings.Contains(out, "UserInfo{") {
		out, err = m.ExecSerial(serial, "shell", "pm", "list", "users")
		// continue parsing whatever we got
	}
	var users []User
	re := regexp.MustCompile(`UserInfo\{(\d+):([^:}]+).*?\}\s*([a-zA-Z]+)?`)
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		mm := re.FindStringSubmatch(ln)
		if len(mm) >= 3 {
			id, _ := strconv.Atoi(mm[1])
			name := strings.TrimSpace(mm[2])
			state := ""
			if len(mm) >= 4 {
				state = strings.TrimSpace(mm[3])
			}
			users = append(users, User{ID: id, Name: name, State: state})
		}
	}
	if len(users) == 0 {
		// default single user 0
		users = append(users, User{ID: 0, Name: "Owner", State: ""})
	}
	return users, out, nil
}

// FileEntry is a directory listing entry.
type FileEntry struct {
	Name    string
	IsDir   bool
	Size    int64  // bytes (best-effort from ls -l)
	Mode    string // permission string from ls -l (e.g. drwxr-xr-x)
	ModTime string // best-effort last modified time text from ls -l (may vary by ROM)
}

// ListDir lists a path on device. Prefers "ls -1p" to mark directories with '/'.
func (m *Manager) ListDir(serial, path string) ([]FileEntry, string, error) {
	if strings.TrimSpace(path) == "" {
		path = "/"
	}
	// Try a detailed listing first to obtain metadata (toybox/busybox compatible).
	// Fall back to simpler formats if flags are unsupported.
	out, err := m.ExecSerial(serial, "shell", "ls", "-lAp", "--", path)
	if err != nil || strings.Contains(out, "Unknown option") || strings.Contains(out, "bad -") {
		out, err = m.ExecSerial(serial, "shell", "ls", "-lA", "--", path)
	}
	if err != nil {
		// Final fallback: names only
		out, err2 := m.ExecSerial(serial, "shell", "ls", "-1p", "--", path)
		if err2 != nil {
			return nil, out, err
		}
		var list []FileEntry
		for _, ln := range strings.Split(out, "\n") {
			name := strings.TrimSpace(ln)
			if name == "" || name == "." || name == ".." {
				continue
			}
			isDir := strings.HasSuffix(name, "/")
			name = strings.TrimSuffix(name, "/")
			list = append(list, FileEntry{Name: name, IsDir: isDir})
		}
		return list, out, nil
	}

	// Parse ls -l style lines:
	// perms links owner group size date/time name
	var list []FileEntry
	for _, ln := range strings.Split(out, "\n") {
		line := strings.TrimSpace(ln)
		if line == "" || line == "." || line == ".." {
			continue
		}
		// Skip "total N" lines
		if strings.HasPrefix(line, "total ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			// Could not parse metadata, fallback to name-only detection
			name := strings.TrimSuffix(line, "/")
			if name == "" || name == "." || name == ".." {
				continue
			}
			list = append(list, FileEntry{
				Name:  name,
				IsDir: strings.HasSuffix(line, "/"),
			})
			continue
		}
		mode := fields[0]
		isDir := strings.HasPrefix(mode, "d")

		// Heuristic: pick the largest numeric index as size (toybox layout makes size at index 4).
		sizeIdx := -1
		for i := 1; i < len(fields); i++ {
			if _, e := strconv.ParseInt(fields[i], 10, 64); e == nil {
				sizeIdx = i
			}
		}
		var size int64
		if sizeIdx >= 0 {
			size, _ = strconv.ParseInt(fields[sizeIdx], 10, 64)
		}

		// Name is usually the last field
		nameField := fields[len(fields)-1]
		name := strings.TrimSuffix(nameField, "/")
		if name == "" || name == "." || name == ".." {
			continue
		}

		// Best-effort ModTime: join last two tokens before name if present
		modTime := ""
		if len(fields) >= 8 {
			modTime = fields[len(fields)-3] + " " + fields[len(fields)-2]
		} else if len(fields) >= 7 {
			modTime = fields[len(fields)-2]
		}

		list = append(list, FileEntry{
			Name:    name,
			IsDir:   isDir,
			Size:    size,
			Mode:    mode,
			ModTime: modTime,
		})
	}
	return list, out, nil
}

// GetProps returns system properties (getprop) as a map.
func (m *Manager) GetProps(serial string) (map[string]string, string, error) {
	out, err := m.ExecSerial(serial, "shell", "getprop")
	if err != nil {
		return nil, out, err
	}
	props := make(map[string]string)
	re := regexp.MustCompile(`^\[([^\]]+)\]: \[([^\]]*)\]$`)
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		mm := re.FindStringSubmatch(ln)
		if len(mm) == 3 {
			props[mm[1]] = mm[2]
		}
	}
	return props, out, nil
}

// GetVarAll returns all fastboot variables as a map.
func (m *Manager) GetVarAll(serial string) (map[string]string, string, error) {
	out, err := m.ExecFastboot(serial, "getvar", "all")
	if err != nil {
		return nil, out, err
	}
	vars := make(map[string]string)
	// Output is line-by-line, but multiline values are indented.
	// Example:
	// (bootloader) C:\fakepath\NON-HLOS.bin:raw data download...
	// (bootloader) partition-type:modem:raw
	// (bootloader) partition-size:modem: 0x0...
	re := regexp.MustCompile(`^\(bootloader\) (.+?):(.*)`)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		match := re.FindStringSubmatch(line)
		if len(match) == 3 {
			key := strings.TrimSpace(match[1])
			val := strings.TrimSpace(match[2])
			vars[key] = val
		}
	}
	return vars, out, nil
}

// InstalledPackagesForUser returns installed package names for a specific user.
func (m *Manager) InstalledPackagesForUser(serial string, userID int) ([]string, string, error) {
	m.EnsureServer()
	args := []string{"shell", "cmd", "package", "list", "packages", "--user", strconv.Itoa(userID)}
	out, err := m.ExecSerial(serial, args...)
	if err != nil || !strings.Contains(out, "package:") || strings.TrimSpace(out) == "" {
		// Fallback to pm
		out, err = m.ExecSerial(serial, "shell", "pm", "list", "packages", "--user", strconv.Itoa(userID))
	}
	// Final fallback: some devices may not support per-user filtering; return all packages
	if (err == nil && (!strings.Contains(out, "package:") || strings.TrimSpace(out) == "")) || err != nil {
		pkgs, out2, err2 := m.InstalledPackages(serial)
		return pkgs, out2, err2
	}
	var pkgs []string
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if strings.HasPrefix(ln, "package:") {
			ln = strings.TrimPrefix(ln, "package:")
		}
		pkgs = append(pkgs, ln)
	}
	return pkgs, out, nil
}

// InstalledPackagesForUserTyped lists packages for a user filtered by type.
// typ accepts "user" for third-party apps (-3) and "system" for system apps (-s).
// Falls back gracefully if flags are unsupported on the device.
func (m *Manager) InstalledPackagesForUserTyped(serial string, userID int, typ string) ([]string, string, error) {
	m.EnsureServer()
	flag := ""
	switch typ {
	case "user":
		flag = "-3"
	case "system":
		flag = "-s"
	default:
		flag = ""
	}
	// Try cmd package first
	var out string
	var err error
	if flag != "" {
		out, err = m.ExecSerial(serial, "shell", "cmd", "package", "list", "packages", "--user", strconv.Itoa(userID), flag)
		if err == nil && strings.Contains(out, "package:") && strings.TrimSpace(out) != "" {
			goto PARSE
		}
	}
	// Fallback to pm with flags
	if flag != "" {
		out, err = m.ExecSerial(serial, "shell", "pm", "list", "packages", "--user", strconv.Itoa(userID), flag)
		if err == nil && strings.Contains(out, "package:") && strings.TrimSpace(out) != "" {
			goto PARSE
		}
	}
	// Final fallback: without type filter (returns all). Caller can still see results.
	out, err = m.ExecSerial(serial, "shell", "pm", "list", "packages", "--user", strconv.Itoa(userID))
PARSE:
	if err != nil {
		return nil, out, err
	}
	var pkgs []string
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if strings.HasPrefix(ln, "package:") {
			ln = strings.TrimPrefix(ln, "package:")
		}
		// Normalize possible "path=package"
		if eq := strings.LastIndex(ln, "="); eq >= 0 && eq+1 < len(ln) {
			ln = ln[eq+1:]
		}
		pkgs = append(pkgs, ln)
	}
	return pkgs, out, nil
}

// Uninstall removes an app for a given user (or all profiles if userID < 0 is passed; here we always pass a userID).
func (m *Manager) Uninstall(serial string, userID int, pkg string) (string, error) {
	if strings.TrimSpace(pkg) == "" {
		return "", errors.New("empty package")
	}
	// Prefer cmd package
	out, err := m.ExecSerial(serial, "shell", "cmd", "package", "uninstall", "--user", strconv.Itoa(userID), pkg)
	if err != nil || (!strings.Contains(out, "Success") && !strings.Contains(out, "success")) {
		// Fallback to pm
		out, err = m.ExecSerial(serial, "shell", "pm", "uninstall", "--user", strconv.Itoa(userID), pkg)
	}
	return out, err
}

// ClearData clears app data using package manager.
func (m *Manager) ClearData(serial, pkg string) (string, error) {
	if strings.TrimSpace(pkg) == "" {
		return "", errors.New("empty package")
	}
	return m.ExecSerial(serial, "shell", "pm", "clear", pkg)
}

// ForceStop calls ActivityManager to force stop an app.
func (m *Manager) ForceStop(serial, pkg string) (string, error) {
	if strings.TrimSpace(pkg) == "" {
		return "", errors.New("empty package")
	}
	return m.ExecSerial(serial, "shell", "am", "force-stop", pkg)
}

// ExtractApk pulls APK(s) of the given package into destDir.
// It uses "pm path <pkg>" which may return multiple split APK lines (package:/...apk).
func (m *Manager) ExtractApk(serial, pkg, destDir string) (string, error) {
	if strings.TrimSpace(pkg) == "" {
		return "", errors.New("empty package")
	}
	pathsOut, err := m.ExecSerial(serial, "shell", "pm", "path", pkg)
	if err != nil {
		return pathsOut, err
	}
	var remoteAPKs []string
	for _, ln := range strings.Split(pathsOut, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if strings.HasPrefix(ln, "package:") {
			ln = strings.TrimPrefix(ln, "package:")
		}
		if ln != "" {
			remoteAPKs = append(remoteAPKs, ln)
		}
	}
	if len(remoteAPKs) == 0 {
		return pathsOut, errors.New("no APK paths found for package")
	}
	// Ensure destination directory exists (unless ".")
	if destDir != "" && destDir != "." {
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return "", err
		}
	}
	var allOut []string
	for _, r := range remoteAPKs {
		local := filepath.Join(destDir, filepath.Base(r))
		o, e := m.ExecSerial(serial, "pull", r, local)
		allOut = append(allOut, o)
		if e != nil {
			// continue collecting output but return the first error encountered
			if err == nil {
				err = e
			}
		}
	}
	return strings.Join(allOut, "\n"), err
}

// Reboot reboots the device (mode can be "", "recovery", "bootloader").
func (m *Manager) Reboot(serial, mode string) (string, error) {
	args := []string{}
	if mode == "" {
		args = []string{"reboot"}
	} else {
		args = []string{"reboot", mode}
	}
	return m.ExecSerial(serial, args...)
}

// AppLabel attempts to get the human-readable application label (name) for a package.
// To reduce device load it first tries grepped one-line outputs; if unavailable it falls back to a single full dump.
// Returns: label, rawOutput, error (if the shell command failed). On parse failure it returns pkg as label.
func (m *Manager) AppLabel(serial, pkg string) (string, string, error) {
	if strings.TrimSpace(pkg) == "" {
		return "", "", errors.New("empty package")
	}
	// 1) Newer Android: cmd package dump | grep first matching label line (quiet and fast)
	out, err := m.ExecSerial(serial, "shell", "sh", "-lc", "cmd package dump "+pkg+" 2>/dev/null | grep -m 1 -E 'application-label|nonLocalizedLabel'")
	content := out
	// 2) Older Android: dumpsys package | grep first matching label line
	if err != nil || strings.TrimSpace(out) == "" {
		out, err = m.ExecSerial(serial, "shell", "sh", "-lc", "dumpsys package "+pkg+" 2>/dev/null | grep -m 1 -E 'application-label|nonLocalizedLabel'")
		content = out
	}
	// 3) If grep not available, fall back to one full dump (single attempt)
	if strings.TrimSpace(content) == "" {
		out, err = m.ExecSerial(serial, "shell", "cmd", "package", "dump", pkg)
		if err != nil || strings.TrimSpace(out) == "" {
			out, err = m.ExecSerial(serial, "shell", "dumpsys", "package", pkg)
		}
		content = out
	}

	label := ""
	// application-label: AppName
	re1 := regexp.MustCompile(`application-label(?:-[\w-]+)?\s*:\s*'?([^']*)'?`)
	// nonLocalizedLabel=AppName or nonLocalizedLabel='App Name'
	re2 := regexp.MustCompile(`nonLocalizedLabel=?'?(.*?)'?(\s|$)`)

	for _, ln := range strings.Split(content, "\n") {
		s := strings.TrimSpace(ln)
		if label == "" {
			if mm := re1.FindStringSubmatch(s); len(mm) >= 2 {
				label = strings.TrimSpace(mm[1])
			}
		}
		if label == "" {
			if mm := re2.FindStringSubmatch(s); len(mm) >= 2 {
				label = strings.TrimSpace(mm[1])
			}
		}
		if label != "" {
			break
		}
	}
	if label == "" {
		label = pkg
	}
	return label, content, err
}

// AutoDetect tries to find adb in PATH or common install locations
func AutoDetect() string {
	if p, err := exec.LookPath(adbExecutableName()); err == nil {
		return p
	}
	// Check env SDK roots
	sdkRoots := []string{
		os.Getenv("ANDROID_SDK_ROOT"),
		os.Getenv("ANDROID_HOME"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		if runtime.GOOS == "darwin" {
			sdkRoots = append(sdkRoots, filepath.Join(home, "Library", "Android", "sdk"))
		} else if runtime.GOOS == "windows" {
			sdkRoots = append(sdkRoots, filepath.Join(home, "AppData", "Local", "Android", "Sdk"))
		} else {
			sdkRoots = append(sdkRoots, filepath.Join(home, "Android", "Sdk"))
			sdkRoots = append(sdkRoots, filepath.Join(home, "Android", "sdk"))
		}
	}
	exe := adbExecutableName()
	for _, root := range sdkRoots {
		if root == "" {
			continue
		}
		cand := filepath.Join(root, "platform-tools", exe)
		if fileExists(cand) {
			return cand
		}
	}
	// OS-specific common locations
	candidates := []string{}
	switch runtime.GOOS {
	case "darwin":
		candidates = append(candidates,
			"/usr/local/bin/"+exe,
			"/opt/homebrew/bin/"+exe,
		)
	case "linux":
		candidates = append(candidates,
			"/usr/bin/"+exe,
			"/usr/local/bin/"+exe,
		)
	case "windows":
		// Fall back locations
		candidates = append(candidates,
			filepath.Join("C:\\", "Android", "platform-tools", exe),
		)
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

func adbExecutableName() string {
	if runtime.GOOS == "windows" {
		return "adb.exe"
	}
	return "adb"
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	if err != nil {
		return false
	}
	if st.IsDir() {
		return false
	}
	return true
}

// ValidatePath returns the normalized adb path if valid, else an error.
func ValidatePath(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if !fileExists(p) {
		// Allow PATH lookup if just 'adb' or 'adb.exe'
		if base := filepath.Base(p); base == "adb" || base == "adb.exe" {
			if q, err := exec.LookPath(base); err == nil {
				return q, nil
			}
		}
		return "", errors.New("adb not found at path")
	}
	return p, nil
}

// --- Added raw exec helpers, push/pull utilities, and app-data extraction ---

// ExecRaw runs adb and returns raw bytes (suitable for binary streams like exec-out tar).
func (m *Manager) ExecRaw(args ...string) ([]byte, error) {
	bin := m.Path
	if bin == "" {
		bin = "adb"
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()

	// 在Windows下隐藏CMD窗口
	if runtime.GOOS == "windows" {
		hideWindowsWindow(cmd)
	}

	return cmd.CombinedOutput()
}

// ExecSerialRaw runs adb with -s <serial> and returns raw bytes.
func (m *Manager) ExecSerialRaw(serial string, args ...string) ([]byte, error) {
	if strings.TrimSpace(serial) != "" {
		args = append([]string{"-s", serial}, args...)
	}
	return m.ExecRaw(args...)
}

// Push uploads one local file to a remote directory on the device.
func (m *Manager) Push(serial, localPath, remoteDir string) (string, error) {
	if strings.TrimSpace(localPath) == "" || strings.TrimSpace(remoteDir) == "" {
		return "", errors.New("invalid push arguments")
	}
	// Ensure destination is treated as a directory by adb
	if !strings.HasSuffix(remoteDir, "/") {
		remoteDir += "/"
	}
	return m.ExecSerial(serial, "push", localPath, remoteDir)
}

// PushMultiple uploads multiple local files to a remote directory.
func (m *Manager) PushMultiple(serial string, localPaths []string, remoteDir string) (string, error) {
	if len(localPaths) == 0 {
		return "", errors.New("no files to push")
	}
	var outs []string
	var firstErr error
	for _, lp := range localPaths {
		out, err := m.Push(serial, lp, remoteDir)
		outs = append(outs, out)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return strings.Join(outs, "\n"), firstErr
}

// Pull downloads one remote path (file or directory) to a local directory.
// If preserve is true, tries "adb pull -a" first, falling back to plain pull.
func (m *Manager) Pull(serial, remotePath, localDir string, preserve bool) (string, error) {
	if strings.TrimSpace(remotePath) == "" || strings.TrimSpace(localDir) == "" {
		return "", errors.New("invalid pull arguments")
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return "", err
	}
	args := []string{"pull"}
	if preserve {
		args = append(args, "-a")
	}
	args = append(args, remotePath, localDir)
	out, err := m.ExecSerial(serial, args...)
	if err != nil && preserve {
		// Fallback without -a for compatibility
		out, err = m.ExecSerial(serial, "pull", remotePath, localDir)
	}
	return out, err
}

// PullMultiple downloads multiple remote paths to a local directory.
func (m *Manager) PullMultiple(serial string, remotePaths []string, localDir string, preserve bool) (string, error) {
	if len(remotePaths) == 0 {
		return "", errors.New("no files to pull")
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return "", err
	}
	var outs []string
	var firstErr error
	for _, rp := range remotePaths {
		out, err := m.Pull(serial, rp, localDir, preserve)
		outs = append(outs, out)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return strings.Join(outs, "\n"), firstErr
}

// ExtractAppData attempts to archive app-internal data (best-effort).
// It tries "exec-out run-as <pkg> sh -c 'cd /data/data/<pkg> && tar cf - .'" (debuggable apps)
// and falls back to "exec-out su -c 'tar cf - -C /data/user/0/<pkg> .'" (rooted devices).
// On success, writes a data.tar inside destDir and returns a message.
func (m *Manager) ExtractAppData(serial, pkg, destDir string) (string, error) {
	if strings.TrimSpace(pkg) == "" {
		return "", errors.New("empty package")
	}
	if destDir == "" {
		destDir = pkg
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}

	// Try run-as (requires debuggable app signed with same cert as 'run-as' uid).
	out, err := m.ExecSerialRaw(serial, "exec-out", "run-as", pkg, "sh", "-c", "cd /data/data/"+pkg+" && tar cf - .")
	if err != nil || len(out) == 0 {
		// Fallback: require root (su)
		out, err = m.ExecSerialRaw(serial, "exec-out", "su", "-c", "tar cf - -C /data/user/0/"+pkg+" .")
	}
	if err != nil || len(out) == 0 {
		return "app data tar not available (requires debuggable app or root)", err
	}

	tarPath := filepath.Join(destDir, "data.tar")
	if writeErr := os.WriteFile(tarPath, out, 0o644); writeErr != nil {
		return "", writeErr
	}
	return "app data archived to " + tarPath, nil
}

// hideWindowsWindow 在Windows下隐藏CMD窗口
// 在非Windows系统下是空实现，在Windows下由adb_windows.go提供实现
func hideWindowsWindow(cmd *exec.Cmd) {
	// 空实现，Windows版本会覆盖此函数
}
