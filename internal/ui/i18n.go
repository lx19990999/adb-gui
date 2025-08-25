package ui

import (
	"os"
	"strings"
)

// Language represents the supported languages
type Language string

const (
	Chinese Language = "zh"
	English Language = "en"
)

// I18n provides internationalization support
type I18n struct {
	language Language
	strings  map[Language]map[string]string
}

// NewI18n creates a new I18n instance with system language detection
func NewI18n() *I18n {
	i18n := &I18n{
		strings: make(map[Language]map[string]string),
	}

	// Initialize translations
	i18n.initTranslations()

	// Detect system language
	i18n.language = i18n.detectSystemLanguage()

	return i18n
}

// detectSystemLanguage detects the system language
func (i *I18n) detectSystemLanguage() Language {
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = os.Getenv("LANGUAGE")
	}
	if lang == "" {
		lang = os.Getenv("LC_ALL")
	}

	lang = strings.ToLower(lang)
	if strings.Contains(lang, "zh") || strings.Contains(lang, "cn") {
		return Chinese
	}
	return English
}

// SetLanguage sets the current language
func (i *I18n) SetLanguage(lang Language) {
	i.language = lang
}

// GetLanguage returns the current language
func (i *I18n) GetLanguage() Language {
	return i.language
}

// T returns the translated string for the given key
func (i *I18n) T(key string) string {
	if strings, ok := i.strings[i.language]; ok {
		if str, ok := strings[key]; ok {
			return str
		}
	}
	// Fallback to English if translation not found
	if strings, ok := i.strings[English]; ok {
		if str, ok := strings[key]; ok {
			return str
		}
	}
	return key
}

// initTranslations initializes all translations
func (i *I18n) initTranslations() {
	// Chinese translations
	i.strings[Chinese] = map[string]string{
		// Common
		"path":                           "路径:",
		"user":                           "用户:",
		"type":                           "类型",
		"refresh":                        "刷新",
		"open":                           "打开",
		"up":                             "上级",
		"upload":                         "上传…",
		"download":                       "下载",
		"select_all":                     "全选",
		"deselect_all":                   "取消全选",
		"clear_selection":                "清空选择",
		"no_device":                      "无设备",
		"please_select_device":           "请选择设备。",
		"upload_complete":                "上传完成。",
		"download_complete":              "下载完成。",
		"invalid_local_path":             "无效的本地文件路径。",
		"please_select_files":            "请在文件列表勾选要下载的文件/目录。",
		"please_select_at_least_one_app": "请选择至少一个应用。",
		"please_enter_path":              "请输入路径。",
		"path_not_found":                 "路径不存在或无法访问。",
		"select_device":                  "选择设备。",
		"select_device_to_list_packages": "选择设备以列出已安装的包。",
		"owner_user":                     "所有者",
		"packages_count":                 "包数量",
		"app_category":                   "应用类别",
		"user_apps":                      "用户应用",
		"system_apps":                    "系统应用",
		"select_user":                    "选择用户",
		"applications":                   "应用程序",
		"storage":                        "存储",
		"commands":                       "命令",
		"fastboot":                       "Fastboot",
		"adb_commands":                   "ADB命令",
		"fastboot_commands":              "Fastboot命令",
		"reboot":                         "重启",
		"reboot_bootloader":              "重启到Bootloader",
		"reboot_recovery":                "重启到Recovery",
		"sideload_zip":                   "侧载ZIP...",
		"sideload":                       "侧载",
		"start_shizuku":                  "启动Shizuku",
		"fastboot_reboot":                "Fastboot重启",
		"fastboot_reboot_bootloader":     "Fastboot重启到Bootloader",
		"fastboot_continue":              "Fastboot继续",
		"fastboot_oem_unlock":            "Fastboot OEM解锁",
		"fastboot_flashing_unlock":       "Fastboot刷机解锁",
		"fastboot_flash":                 "Fastboot刷写",
		"fastboot_update":                "Fastboot更新",
		"fastboot_oem_device_info":       "Fastboot OEM设备信息",
		"fastboot_oem_edl":               "Fastboot OEM EDL",
		"detect":                         "检测",
		"browse":                         "浏览…",
		"save":                           "保存",
		"system":                         "系统",
		"light":                          "浅色",
		"dark":                           "深色",
		"adb_not_found":                  "ADB未找到",
		"adb_not_detected":               "ADB未检测到。请在设置中设置ADB路径或确保其在PATH中。",
		"could_not_auto_detect_adb":      "无法自动检测ADB。请手动浏览并选择。",
		"select_adb":                     "选择ADB",
		"invalid_selection":              "无效选择",
		"settings":                       "设置…",
		"file":                           "文件",
		"saved":                          "已保存",
		"settings_saved":                 "设置已保存。请重启应用程序以使主题更改生效。",
		"help":                           "帮助",
		"about":                          "关于",
		"about_description":              "ADB GUI\nAndroid调试桥的跨平台界面",
		"continue":                       "继续",
		"oem_unlock":                     "OEM解锁",
		"flashing_unlock":                "刷机解锁",
		"flash_partition":                "刷写分区...",
		"flash":                          "刷写",
		"update_from_zip":                "从ZIP更新...",
		"oem_device_info":                "OEM设备信息",
		"oem_edl":                        "OEM EDL",
		"devices":                        "设备",
		"error":                          "错误",
		"no_data":                        "无数据返回。",
		"copy_selected":                  "复制选中项",

		// Device operations
		"uninstall":          "卸载",
		"clear_data":         "清除数据",
		"force_stop":         "强制停止",
		"extract_apk":        "提取APK",
		"extract_apk_data":   "提取APK+数据",
		"uninstalled":        "已卸载",
		"cleared_data":       "已清除数据",
		"forced_stop":        "已强制停止",
		"apk_extracted":      "APK已提取",
		"extract_issues":     "提取问题",
		"extracted_apk_data": "已提取APK和应用数据到",
		"uninstall_failed":   "卸载失败",
		"clear_data_failed":  "清除数据失败",
		"force_stop_failed":  "强制停止失败",
		"extract_apk_failed": "提取APK失败",

		// Sorting
		"sort_alphabetical":         "按照字母顺序排序",
		"sort_alphabetical_reverse": "按照字母倒序排序",
		"sort_size_small_to_large":  "按照文件大小从小到大排序",
		"sort_size_large_to_small":  "按照文件大小从大到小排序",
		"sort_time_old_to_new":      "按照修改时间从远到近排序",
		"sort_time_new_to_old":      "按照修改时间从近到远排序",

		// Tabs
		"parameters":            "参数",
		"getvar":                "获取变量",
		"file_manager":          "文件管理",
		"device_info":           "设备信息",
		"adb_path":              "ADB 路径",
		"theme_mode":            "主题模式",
		"partition_name":        "分区名称",
		"partition_placeholder": "例如：boot, recovery",
		"adb_path_placeholder":  "/path/to/adb 或 adb",

		// Messages
		"upload_failed":          "上传失败",
		"download_failed":        "下载失败",
		"operation_failed":       "操作失败",
		"complete":               "完成",
		"failed":                 "失败",
		"batch_uninstall":        "批量卸载",
		"batch_clear_data":       "批量清除数据",
		"batch_force_stop":       "批量强行停止",
		"batch_extract_apk":      "批量提取APK",
		"batch_extract_apk_data": "批量提取APK+数据",
		"success":                "成功",
		"cancel":                 "取消",
		"ok":                     "确定",
		"close":                  "关闭",
	}

	// English translations
	i.strings[English] = map[string]string{
		// Common
		"path":                           "Path:",
		"user":                           "User:",
		"type":                           "Type",
		"refresh":                        "Refresh",
		"open":                           "Open",
		"up":                             "Up",
		"upload":                         "Upload…",
		"download":                       "Download",
		"select_all":                     "Select All",
		"deselect_all":                   "Deselect All",
		"clear_selection":                "Clear Selection",
		"no_device":                      "No device",
		"please_select_device":           "Please select a device.",
		"upload_complete":                "Upload complete.",
		"download_complete":              "Download complete.",
		"invalid_local_path":             "Invalid local file path.",
		"please_select_files":            "Please select files/directories to download in the file list.",
		"please_select_at_least_one_app": "Please select at least one app.",
		"please_enter_path":              "Please enter a path.",
		"path_not_found":                 "Path not found or cannot be accessed.",
		"select_device":                  "Select a device.",
		"select_device_to_list_packages": "Select a device to list installed packages.",
		"owner_user":                     "Owner",
		"packages_count":                 "Packages Count",
		"app_category":                   "App Category",
		"user_apps":                      "User Apps",
		"system_apps":                    "System Apps",
		"select_user":                    "Select User",
		"applications":                   "Applications",
		"storage":                        "Storage",
		"commands":                       "Commands",
		"fastboot":                       "Fastboot",
		"adb_commands":                   "ADB Commands",
		"fastboot_commands":              "Fastboot Commands",
		"reboot":                         "Reboot",
		"reboot_bootloader":              "Reboot to Bootloader",
		"reboot_recovery":                "Reboot to Recovery",
		"sideload_zip":                   "Sideload ZIP...",
		"sideload":                       "Sideload",
		"start_shizuku":                  "Start Shizuku",
		"fastboot_reboot":                "Fastboot Reboot",
		"fastboot_reboot_bootloader":     "Fastboot Reboot to Bootloader",
		"fastboot_continue":              "Fastboot Continue",
		"fastboot_oem_unlock":            "Fastboot OEM Unlock",
		"fastboot_flashing_unlock":       "Fastboot Flashing Unlock",
		"fastboot_flash":                 "Fastboot Flash",
		"fastboot_update":                "Fastboot Update",
		"fastboot_oem_device_info":       "Fastboot OEM Device Info",
		"fastboot_oem_edl":               "Fastboot OEM EDL",
		"detect":                         "Detect",
		"browse":                         "Browse…",
		"save":                           "Save",
		"system":                         "System",
		"light":                          "Light",
		"dark":                           "Dark",
		"adb_not_found":                  "ADB Not Found",
		"adb_not_detected":               "ADB Not Detected. Please set the ADB path in settings or ensure it's in your PATH.",
		"could_not_auto_detect_adb":      "ADB could not be automatically detected. Please browse manually and select.",
		"select_adb":                     "Select ADB",
		"settings":                       "Settings…",
		"file":                           "File",
		"saved":                          "Saved",
		"settings_saved":                 "Settings saved. Please restart the application to apply theme changes.",
		"help":                           "Help",
		"about":                          "About",
		"about_description":              "ADB GUI\nCross-platform UI for Android Debug Bridge",
		"continue":                       "Continue",
		"oem_unlock":                     "OEM Unlock",
		"flashing_unlock":                "Flashing Unlock",
		"flash_partition":                "Flash Partition...",
		"flash":                          "Flash",
		"update_from_zip":                "Update from ZIP...",
		"oem_device_info":                "OEM Device Info",
		"oem_edl":                        "OEM EDL",
		"devices":                        "Devices",
		"error":                          "Error",
		"no_data":                        "No data returned.",
		"copy_selected":                  "Copy Selected",

		// Device operations
		"uninstall":          "Uninstall",
		"clear_data":         "Clear Data",
		"force_stop":         "Force Stop",
		"extract_apk":        "Extract APK",
		"extract_apk_data":   "Extract APK+Data",
		"uninstalled":        "Uninstalled",
		"cleared_data":       "Cleared Data",
		"forced_stop":        "Forced Stop",
		"apk_extracted":      "APK Extracted",
		"extract_issues":     "Extraction Issues",
		"extracted_apk_data": "APK and App Data Extracted to",
		"uninstall_failed":   "Uninstall failed",
		"clear_data_failed":  "Clear data failed",
		"force_stop_failed":  "Force Stop failed",
		"extract_apk_failed": "Extract APK failed",

		// Sorting
		"sort_alphabetical":         "Sort alphabetically",
		"sort_alphabetical_reverse": "Sort alphabetically (reverse)",
		"sort_size_small_to_large":  "Sort by size (small to large)",
		"sort_size_large_to_small":  "Sort by size (large to small)",
		"sort_time_old_to_new":      "Sort by time (old to new)",
		"sort_time_new_to_old":      "Sort by time (new to old)",

		// Tabs
		"parameters":            "Parameters",
		"getvar":                "GetVar",
		"file_manager":          "File Manager",
		"device_info":           "Device Info",
		"adb_path":              "ADB Path",
		"theme_mode":            "Theme Mode",
		"partition_name":        "Partition Name",
		"partition_placeholder": "e.g., boot, recovery",
		"adb_path_placeholder":  "/path/to/adb or adb",

		// Messages
		"upload_failed":          "Upload failed",
		"download_failed":        "Download failed",
		"operation_failed":       "Operation failed",
		"complete":               "Complete",
		"failed":                 "Failed",
		"batch_uninstall":        "Batch Uninstall",
		"batch_clear_data":       "Batch Clear Data",
		"batch_force_stop":       "Batch Force Stop",
		"batch_extract_apk":      "Batch Extract APK",
		"batch_extract_apk_data": "Batch Extract APK+Data",
		"success":                "Success",
		"cancel":                 "Cancel",
		"ok":                     "OK",
		"close":                  "Close",
	}
}

// Global I18n instance
var GlobalI18n *I18n

// InitI18n initializes the global I18n instance
func InitI18n() {
	GlobalI18n = NewI18n()
}

// T is a convenience function to get translated strings
func T(key string) string {
	if GlobalI18n == nil {
		GlobalI18n = NewI18n()
	}
	return GlobalI18n.T(key)
}

// SetLanguage sets the global language
func SetLanguage(lang Language) {
	if GlobalI18n == nil {
		GlobalI18n = NewI18n()
	}
	GlobalI18n.SetLanguage(lang)
}

// GetLanguage returns the global language
func GetLanguage() Language {
	if GlobalI18n == nil {
		GlobalI18n = NewI18n()
	}
	return GlobalI18n.GetLanguage()
}
