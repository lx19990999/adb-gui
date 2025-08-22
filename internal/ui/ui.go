package ui

import (
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"time"
	"sort"

	"adb-gui/internal/adb"
	"adb-gui/internal/config"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)
// helpers to safely locate widgets inside compound row containers
func findFirstLabel(obj fyne.CanvasObject) *widget.Label {
	switch t := obj.(type) {
	case *widget.Label:
		return t
	case *fyne.Container:
		for _, c := range t.Objects {
			if lbl := findFirstLabel(c); lbl != nil {
				return lbl
			}
		}
	}
	return nil
}

func findButtons(obj fyne.CanvasObject, n int) []*widget.Button {
	var res []*widget.Button
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch t := o.(type) {
		case *widget.Button:
			res = append(res, t)
		case *fyne.Container:
			for _, c := range t.Objects {
				walk(c)
			}
		}
	}
	walk(obj)
	if len(res) >= n {
		return res[:n]
	}
	return res
}
// findFirstCheck walks the object tree to find the first checkbox widget.
func findFirstCheck(obj fyne.CanvasObject) *widget.Check {
	switch t := obj.(type) {
	case *widget.Check:
		return t
	case *fyne.Container:
		for _, c := range t.Objects {
			if cb := findFirstCheck(c); cb != nil {
				return cb
			}
		}
	}
	return nil
}

// DefaultWindowSize returns the preferred initial window size.
func DefaultWindowSize() fyne.Size {
	return fyne.NewSize(1200, 760)
}
 

// BuildUI constructs the main application UI:
// - top main menu
// - below a horizontal split: left devices list, right AppTabs:
//   1) Applications
//   2) Storage
//   3) Parameters
//   4) Commands
func BuildUI(w fyne.Window, a fyne.App, mgr *adb.Manager, cfg *config.Config) {
	// Menu bar
	menu := buildMainMenu(a, w, mgr, cfg)
	w.SetMainMenu(menu)

	// Data state
	var devices []adb.Device
	devCountBind := binding.NewInt()
	selectedSerialBind := binding.NewString() // currently selected device serial
	statusBind := binding.NewString()         // status bar text

	// Left: devices list panel
	leftPanel, refreshDevices := buildDevicesPanel(w, mgr, &devices, selectedSerialBind, devCountBind, statusBind)

	// Right: tabs dependent on selected device
	appsTab := buildApplicationsTab(w, mgr, selectedSerialBind)
	storageTab := buildStorageTab(w, mgr, selectedSerialBind)
	paramsTab := buildParametersTab(w, mgr, selectedSerialBind)
	cmdsTab := buildCommandsTab(w, mgr, selectedSerialBind)

	rightTabs := container.NewAppTabs(
		container.NewTabItem("Applications", appsTab),
		container.NewTabItem("Storage", storageTab),
		container.NewTabItem("Parameters", paramsTab),
		container.NewTabItem("Commands", cmdsTab),
	)
	rightTabs.SetTabLocation(container.TabLocationTop)

	// Split layout (left devices, right tabs)
	split := container.NewHSplit(leftPanel, rightTabs)
	split.Offset = 0.25

	// Status bar at bottom
	statusBar := widget.NewLabelWithData(statusBind)

	root := container.NewBorder(nil, statusBar, nil, nil, split)
	w.SetContent(root)

	// Initial load & status
	if !mgr.IsAvailable() {
		dialog.ShowInformation("ADB Not Found", "ADB was not detected. Set the ADB path in Settings or ensure it is in PATH.", w)
	}
	refreshDevices()

	go func() {
		ver, _ := mgr.Version()
		fyne.Do(func() {
			_ = statusBind.Set(fmt.Sprintf("%s | Devices: %d", firstLine(ver), mustGetInt(devCountBind)))
		})
	}()

	// Periodic device refresh (optional quality-of-life)
	go func() {
		t := time.NewTicker(8 * time.Second)
		defer t.Stop()
		for range t.C {
			refreshDevices()
		}
	}()
}

func buildMainMenu(a fyne.App, w fyne.Window, mgr *adb.Manager, cfg *config.Config) *fyne.MainMenu {
	settings := fyne.NewMenuItem("Settings…", func() {
		openSettingsDialog(w, mgr, cfg)
	})
	quit := fyne.NewMenuItem("Quit", func() { a.Quit() })
	about := fyne.NewMenuItem("About", func() {
		dialog.ShowInformation("About", "ADB GUI\nCross-platform UI for Android Debug Bridge", w)
	})
	fileMenu := fyne.NewMenu("File", settings, quit)
	helpMenu := fyne.NewMenu("Help", about)
	return fyne.NewMainMenu(fileMenu, helpMenu)
}

func buildDevicesPanel(
	w fyne.Window,
	mgr *adb.Manager,
	devices *[]adb.Device,
	selectedSerialBind binding.String,
	devCountBind binding.Int,
	statusBind binding.String,
) (fyne.CanvasObject, func()) {

	header := widget.NewLabel("Devices")

	// Refresh button
	refreshBtn := widget.NewButton("Refresh", nil)

	// Devices list
	list := widget.NewList(
		func() int {
			return len(*devices)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("serial | state | model | device")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(*devices) {
				return
			}
			d := (*devices)[i]
			lbl := o.(*widget.Label)
			lbl.Truncation = fyne.TextTruncateEllipsis
			lbl.Text = fmt.Sprintf("%s   | %s | %s | %s", d.Serial, d.State, d.Model, d.Device)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(*devices) {
			_ = selectedSerialBind.Set((*devices)[id].Serial)
			updateStatusDevices(statusBind, mgr, mustGetInt(devCountBind))
		}
	}

	refresh := func() {
		go func() {
			devs, _, err := mgr.Devices()
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				*devices = devs
				list.Refresh()
				_ = devCountBind.Set(len(devs))
				updateStatusDevices(statusBind, mgr, len(devs))

				// Auto-select first if none selected
				cur, _ := selectedSerialBind.Get()
				if cur == "" && len(devs) > 0 {
					_ = selectedSerialBind.Set(devs[0].Serial)
				}
			})
		}()
	}
	refreshBtn.OnTapped = refresh

	return container.NewBorder(
		container.NewVBox(header, refreshBtn),
		nil, nil, nil,
		list,
	), refresh
}

// Applications tab: list installed packages for selected device
func buildApplicationsTab(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String) fyne.CanvasObject {
	// State
	var pkgs []string
	selectedUserID := 0
	labels := map[string]string{}
	labelFetchRunning := false
	fetchGen := 0
	selectedPkgs := map[string]bool{}
	var refreshPackages func()
	var startLabelFetch func()

	// UI elements
	title := widget.NewLabel("Applications")
	userSelect := widget.NewSelect([]string{"0 (Owner)"}, func(string) {})
	userSelect.PlaceHolder = "Select user"
	// App type filter: 用户应用(第三方) / 系统应用
	appTypeSelect := widget.NewSelect([]string{"用户应用", "系统应用"}, nil)
	appTypeSelect.PlaceHolder = "应用类别"
	appTypeSelect.SetSelected("用户应用")
	pkgCount := widget.NewLabel("Packages: 0")
	refreshBtn := widget.NewButton("Refresh", nil)

	// List with action buttons per row (HBox: [checkbox][label][spacer][buttons...])
	list := widget.NewList(
		func() int { return len(pkgs) },
		func() fyne.CanvasObject {
			chk := widget.NewCheck("", nil)
			name := widget.NewLabel("package.name")
			// Ensure the label shows plain text (no forced ellipsis) and expands in center
			name.Truncation = fyne.TextTruncateOff
			name.Wrapping = fyne.TextWrapOff
			name.Alignment = fyne.TextAlignLeading
			btnUninstall := widget.NewButton("Uninstall", nil)
			btnClear := widget.NewButton("Clear Data", nil)
			btnForceStop := widget.NewButton("Force Stop", nil)
			btnExtractApk := widget.NewButton("Extract APK", nil)
			btnExtractAll := widget.NewButton("Extract APK+Data", nil)
			btnBar := container.NewHBox(btnUninstall, btnClear, btnForceStop, btnExtractApk, btnExtractAll)
			// Put label in center so it expands, checkbox on the left, buttons on the right
			row := container.NewBorder(nil, nil, chk, btnBar, name)
			return row
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(pkgs) {
				return
			}
			pkg := pkgs[i]

			box := o.(*fyne.Container)

			// Robustly find the center label and the buttons bar (right)
			var lbl *widget.Label
			var btnBar *fyne.Container
			for _, child := range box.Objects {
				switch t := child.(type) {
				case *widget.Label:
					// Prefer direct label child as the row title
					if lbl == nil {
						lbl = t
					}
				case *fyne.Container:
					// Consider this as the buttons bar if all its children are buttons (>=5)
					allBtns := true
					if len(t.Objects) >= 5 {
						for _, b := range t.Objects {
							if _, ok := b.(*widget.Button); !ok {
								allBtns = false
								break
							}
						}
						if allBtns {
							btnBar = t
						}
					}
				}
			}
			// Fallback for older/newer layouts
			if lbl == nil {
				lbl = findFirstLabel(box)
			}
			if btnBar == nil {
				if c, ok := box.Objects[len(box.Objects)-1].(*fyne.Container); ok {
					btnBar = c
				}
			}
			if lbl == nil || btnBar == nil || len(btnBar.Objects) < 5 {
				return
			}
			// checkbox reflect selection state
			if chk := findFirstCheck(box); chk != nil {
				cur := false
				if v, ok := selectedPkgs[pkg]; ok {
					cur = v
				}
				chk.SetChecked(cur)
				chk.Refresh()
				nameCopy := pkg
				chk.OnChanged = func(v bool) {
					selectedPkgs[nameCopy] = v
				}
			}
	
			btnUninstall := btnBar.Objects[0].(*widget.Button)
			btnClear := btnBar.Objects[1].(*widget.Button)
			btnForce := btnBar.Objects[2].(*widget.Button)
			btnApk := btnBar.Objects[3].(*widget.Button)
			btnAll := btnBar.Objects[4].(*widget.Button)

			// Show: "package<TAB>AppName" when name is known and valid; otherwise just "package"
			appName := strings.TrimSpace(labels[pkg])
			if appName != "" && strings.ToLower(appName) != "null" && appName != pkg {
				lbl.SetText(fmt.Sprintf("%s\t%s", pkg, appName))
			} else {
				lbl.SetText(pkg)
			}
			lbl.Refresh()

			// Icons and tooltips
			btnUninstall.SetText("")
			btnUninstall.SetIcon(theme.DeleteIcon())

			btnClear.SetText("")
			btnClear.SetIcon(theme.CancelIcon())

			btnForce.SetText("")
			btnForce.SetIcon(theme.MediaStopIcon())

			btnApk.SetText("")
			btnApk.SetIcon(theme.ContentCopyIcon())

			btnAll.SetText("")
			btnAll.SetIcon(theme.FolderOpenIcon())

			btnUninstall.OnTapped = func() {
				serial, _ := selectedSerialBind.Get()
				if strings.TrimSpace(serial) == "" {
					dialog.ShowInformation("No device", "Please select a device.", w)
					return
				}
				go func() {
					out, err := mgr.Uninstall(serial, selectedUserID, pkg)
					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(fmt.Errorf("uninstall failed: %v\n%s", err, out), w)
						} else {
							dialog.ShowInformation("Uninstall", "Uninstalled "+pkg, w)
							if refreshPackages != nil {
								refreshPackages()
							}
						}
					})
				}()
			}
			btnClear.OnTapped = func() {
				serial, _ := selectedSerialBind.Get()
				if strings.TrimSpace(serial) == "" {
					dialog.ShowInformation("No device", "Please select a device.", w)
					return
				}
				go func() {
					out, err := mgr.ClearData(serial, pkg)
					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(fmt.Errorf("clear data failed: %v\n%s", err, out), w)
						} else {
							dialog.ShowInformation("Clear Data", "Cleared data for "+pkg, w)
						}
					})
				}()
			}
			btnForce.OnTapped = func() {
				serial, _ := selectedSerialBind.Get()
				if strings.TrimSpace(serial) == "" {
					dialog.ShowInformation("No device", "Please select a device.", w)
					return
				}
				go func() {
					out, err := mgr.ForceStop(serial, pkg)
					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(fmt.Errorf("force stop failed: %v\n%s", err, out), w)
						} else {
							dialog.ShowInformation("Force Stop", "Forced stop for "+pkg, w)
						}
					})
				}()
			}
			btnApk.OnTapped = func() {
				serial, _ := selectedSerialBind.Get()
				if strings.TrimSpace(serial) == "" {
					dialog.ShowInformation("No device", "Please select a device.", w)
					return
				}
				go func() {
					out, err := mgr.ExtractApk(serial, pkg, pkg)
					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(fmt.Errorf("extract apk failed: %v\n%s", err, out), w)
						} else {
							dialog.ShowInformation("Extract APK", "APK(s) extracted for "+pkg+" into current directory.", w)
						}
					})
				}()
			}
			btnAll.OnTapped = func() {
				serial, _ := selectedSerialBind.Get()
				if strings.TrimSpace(serial) == "" {
					dialog.ShowInformation("No device", "Please select a device.", w)
					return
				}
				// Destination directory: ./<package>
				dest := pkg
				go func() {
					// Attempt APK extraction and best-effort app data archive (data.tar)
					out1, err1 := mgr.ExtractApk(serial, pkg, dest)
					out2, err2 := mgr.ExtractAppData(serial, pkg, dest)
					msg := strings.TrimSpace(out1 + "\n" + out2)
					fyne.Do(func() {
						if err1 != nil || err2 != nil {
							// Show combined message with any errors
							dialog.ShowError(fmt.Errorf("extract issues:\nAPK: %v\nData: %v\n\n%s", err1, err2, msg), w)
						} else {
							if msg == "" {
								msg = "Extracted APK and app data (data.tar) to ./" + dest
							}
							dialog.ShowInformation("Extract APK + Data", msg, w)
						}
					})
				}()
			}
		},
	)

	// Helpers
	refreshUsers := func() {
		serial, _ := selectedSerialBind.Get()
		if strings.TrimSpace(serial) == "" {
			fyne.Do(func() {
				userSelect.Options = []string{}
				userSelect.SetSelected("")
				userSelect.Refresh()
			})
			return
		}
		go func() {
			users, _, err := mgr.Users(serial)
			fyne.Do(func() {
				opts := []string{}
				if err == nil && len(users) > 0 {
					ownerIdx := -1
					for i, u := range users {
						opts = append(opts, fmt.Sprintf("%d (%s)", u.ID, u.Name))
						if u.ID == 0 {
							ownerIdx = i
						}
					}
					// IMPORTANT: set Options before SetSelected so selection is applied
					userSelect.Options = opts
					if ownerIdx >= 0 {
						selectedUserID = users[ownerIdx].ID
						userSelect.SetSelected(opts[ownerIdx])
					} else {
						selectedUserID = users[0].ID
						userSelect.SetSelected(opts[0])
					}
				} else {
					opts = []string{"0 (Owner)"}
					userSelect.Options = opts
					selectedUserID = 0
					userSelect.SetSelected(opts[0])
				}
				userSelect.Refresh()
				// Trigger package list refresh after user selection is applied
				if refreshPackages != nil {
					refreshPackages()
				}
			})
		}()
	}

	refreshPackages = func() {
		serial, _ := selectedSerialBind.Get()
		if strings.TrimSpace(serial) == "" {
			return
		}
		go func() {
			// map UI selection to adb flag type
			at := appTypeSelect.Selected
			typ := "user"
			if at == "系统应用" {
				typ = "system"
			}
			plist, _, err := mgr.InstalledPackagesForUserTyped(serial, selectedUserID, typ)
			fyne.Do(func() {
				if err != nil {
					log.Printf("[apps] load error: %v", err)
					pkgs = []string{"Error: " + err.Error()}
					pkgCount.SetText("Packages: 0")
					list.Refresh()
					return
				}
				log.Printf("[apps] packages loaded: %d", len(plist))
				pkgs = plist
				// reset labels for new list and start async label fetching
				labels = map[string]string{}
				list.Refresh()
				pkgCount.SetText(fmt.Sprintf("Packages: %d", len(plist)))
				// Start sequential, rate-limited fetching of app labels for current list.
				// This avoids overloading the device and cancels when a new generation starts.
				startLabelFetch = func() {
					if labelFetchRunning {
						return
					}
					labelFetchRunning = true
					localGen := fetchGen

					serial, _ := selectedSerialBind.Get()
					pkgsSnapshot := make([]string, len(pkgs))
					copy(pkgsSnapshot, pkgs)

					go func(serial string, listSnapshot []string, gen int) {
						defer func() {
							labelFetchRunning = false
						}()

						maxCount := 40
						count := 0
						for _, p := range listSnapshot {
							// Cancel if a newer fetch cycle started
							if gen != fetchGen {
								return
							}
							// Skip non-package placeholders
							if !strings.Contains(p, ".") {
								continue
							}
							// Fetch label with defensive checks to avoid hammering a device that is offline/restarting
							name, out, err := mgr.AppLabel(serial, p)
							if err != nil {
								lo := strings.ToLower(out + " " + err.Error())
								// Stop fetching if device is offline/unauthorized/disconnected to avoid cascading issues
								if strings.Contains(lo, "offline") || strings.Contains(lo, "unauthorized") || strings.Contains(lo, "no devices") {
									return
								}
							}
							nm := strings.TrimSpace(name)
							if nm != "" && strings.ToLower(nm) != "null" && nm != p {
								fyne.Do(func() {
									labels[p] = nm
									list.Refresh()
								})
							}
							count++
							if count >= maxCount {
								// Stop after a safe number to avoid heavy load; user can refresh again for more
								return
							}
							// Gentle delay between queries (reduce system_server load)
							time.Sleep(200 * time.Millisecond)
						}
					}(serial, pkgsSnapshot, localGen)
				}
				if startLabelFetch != nil {
					startLabelFetch()
				}
			})
		}()
	}

	// Wire controls
	userSelect.OnChanged = func(sel string) {
		if sel == "" {
			return
		}
		if sp := strings.SplitN(sel, " ", 2); len(sp) > 0 {
			if v, err := strconv.Atoi(sp[0]); err == nil {
				selectedUserID = v
			}
		}
		if refreshPackages != nil {
			refreshPackages()
		}
	}
	refreshBtn.OnTapped = func() {
		refreshUsers()
		if refreshPackages != nil {
			refreshPackages()
		}
	}
	// Trigger refresh when app type changes
	appTypeSelect.OnChanged = func(string) {
		if refreshPackages != nil {
			refreshPackages()
		}
	}

	// Auto refresh when device selection changes
	selectedSerialBind.AddListener(binding.NewDataListener(func() {
		refreshUsers()
		if refreshPackages != nil {
			refreshPackages()
		}
	}))

	// Initial tip
	pkgs = []string{"Select a device to list installed packages."}
	pkgCount.SetText("Packages: 0")

	// Batch helpers
	getSelected := func() []string {
		var sel []string
		for _, p := range pkgs {
			if selectedPkgs[p] {
				sel = append(sel, p)
			}
		}
		return sel
	}
	doBatch := func(title string, op func(string) (string, error), refreshAfter bool) {
		serial, _ := selectedSerialBind.Get()
		if strings.TrimSpace(serial) == "" {
			dialog.ShowInformation("No device", "Please select a device.", w)
			return
		}
		targets := getSelected()
		if len(targets) == 0 {
			dialog.ShowInformation(title, "请选择至少一个应用。", w)
			return
		}
		go func() {
			var okN, failN int
			var msgs []string
			for _, p := range targets {
				out, err := op(p)
				if err != nil {
					failN++
					msgs = append(msgs, fmt.Sprintf("[%s] ERROR: %v\n%s", p, err, out))
				} else {
					okN++
					if strings.TrimSpace(out) != "" {
						msgs = append(msgs, fmt.Sprintf("[%s] %s", p, out))
					} else {
						msgs = append(msgs, fmt.Sprintf("[%s] OK", p))
					}
				}
				time.Sleep(100 * time.Millisecond)
			}
			summary := fmt.Sprintf("%s 完成: 成功 %d, 失败 %d\n\n%s", title, okN, failN, strings.Join(msgs, "\n"))
			fyne.Do(func() {
				dialog.ShowInformation(title, summary, w)
				if refreshAfter && refreshPackages != nil {
					refreshPackages()
				}
			})
		}()
	}

	// Batch buttons
	btnSelAll := widget.NewButton("全选", func() {
		for _, p := range pkgs {
			selectedPkgs[p] = true
		}
		list.Refresh()
	})
	btnSelNone := widget.NewButton("清空选择", func() {
		selectedPkgs = map[string]bool{}
		list.Refresh()
	})
	btnBatchUninst := widget.NewButton("批量卸载", func() {
		doBatch("批量卸载", func(p string) (string, error) {
			return mgr.Uninstall(mustGet(selectedSerialBind), selectedUserID, p)
		}, true)
	})
	btnBatchClear := widget.NewButton("批量清除数据", func() {
		doBatch("批量清除数据", func(p string) (string, error) {
			return mgr.ClearData(mustGet(selectedSerialBind), p)
		}, false)
	})
	btnBatchForce := widget.NewButton("批量强行停止", func() {
		doBatch("批量强行停止", func(p string) (string, error) {
			return mgr.ForceStop(mustGet(selectedSerialBind), p)
		}, false)
	})
	btnBatchExtractApk := widget.NewButton("批量提取APK", func() {
		doBatch("批量提取APK", func(p string) (string, error) {
			return mgr.ExtractApk(mustGet(selectedSerialBind), p, p)
		}, false)
	})
	btnBatchExtractAll := widget.NewButton("批量提取APK+数据", func() {
		doBatch("批量提取APK+数据", func(p string) (string, error) {
			out1, err1 := mgr.ExtractApk(mustGet(selectedSerialBind), p, p)
			out2, err2 := mgr.ExtractAppData(mustGet(selectedSerialBind), p, p)
			out := strings.TrimSpace(out1 + "\n" + out2)
			if err1 != nil {
				return out, err1
			}
			return out, err2
		}, false)
	})

	topRow := container.NewHBox(
		title,
		widget.NewLabel(" User:"),
		userSelect,
		widget.NewLabel("  Type:"),
		appTypeSelect,
		widget.NewLabel("  "),
		pkgCount,
		refreshBtn,
	)
	batchRow := container.NewHBox(
		btnSelAll, btnSelNone,
		btnBatchUninst, btnBatchClear, btnBatchForce, btnBatchExtractApk, btnBatchExtractAll,
	)
	top := container.NewVBox(topRow, batchRow)
	return container.NewBorder(top, nil, nil, nil, list)
}

func refreshApps(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String, outBind binding.StringList) {
	serial, _ := selectedSerialBind.Get()
	if strings.TrimSpace(serial) == "" {
		dialog.ShowInformation("No device", "Please select a device.", w)
		return
	}
	go func() {
		pkgs, _, err := mgr.InstalledPackages(serial)
		fyne.Do(func() {
			if err != nil {
				_ = outBind.Set([]string{"Error: " + err.Error()})
				return
			}
			if len(pkgs) == 0 {
				_ = outBind.Set([]string{"No packages found."})
				return
			}
			_ = outBind.Set(pkgs)
		})
	}()
}

// Storage tab: list users and their default storage, browse directories
func buildStorageTab(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String) fyne.CanvasObject {
	usersBind := binding.NewStringList()
	files := []adb.FileEntry{}
	selectedIndex := -1
	selectedNames := map[string]bool{}
	sortMode := "按照字母顺序排序"
	// For double-click detection
	lastClickIdx := -1
	lastClickAt := time.Time{}
	// Forward declarations used in callbacks before assignment
	var curPathBind binding.String
	var loadDir func(string)
	var applySort func()
// Apply sorting according to current sortMode
applySort = func() {
	mode := sortMode

	// Best-effort parse of ModTime captured from ls -l (varies by ROM)
	parseTime := func(s string) time.Time {
		layouts := []string{
			"2006-01-02 15:04",
			"2006-01-02 15:04:05",
			"2006-01-02",
			"Jan _2 15:04",
			"Jan _2 2006",
			time.RFC3339,
			time.ANSIC,
		}
		for _, l := range layouts {
			if t, err := time.Parse(l, strings.TrimSpace(s)); err == nil {
				return t
			}
		}
		return time.Time{}
	}

	switch mode {
	case "按照字母顺序排序":
		sort.SliceStable(files, func(i, j int) bool {
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		})
	case "按照字母倒序排序":
		sort.SliceStable(files, func(i, j int) bool {
			return strings.ToLower(files[i].Name) > strings.ToLower(files[j].Name)
		})
	case "按照文件大小从小到大排序":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].Size < files[j].Size
		})
	case "按照文件大小从大到小排序":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].Size > files[j].Size
		})
	case "按照修改时间从远到近排序":
		sort.SliceStable(files, func(i, j int) bool {
			ti := parseTime(files[i].ModTime)
			tj := parseTime(files[j].ModTime)
			if ti.IsZero() && tj.IsZero() {
				return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
			}
			if ti.IsZero() {
				return true
			}
			if tj.IsZero() {
				return false
			}
			return ti.Before(tj)
		})
	case "按照修改时间从近到远排序":
		sort.SliceStable(files, func(i, j int) bool {
			ti := parseTime(files[i].ModTime)
			tj := parseTime(files[j].ModTime)
			if ti.IsZero() && tj.IsZero() {
				return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
			}
			if ti.IsZero() {
				return false
			}
			if tj.IsZero() {
				return true
			}
			return ti.After(tj)
		})
	default:
		sort.SliceStable(files, func(i, j int) bool {
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		})
	}
}
	filesList := widget.NewList(
		func() int { return len(files) },
		func() fyne.CanvasObject {
			chk := widget.NewCheck("", nil)
			lbl := widget.NewLabel("")
			lbl.Truncation = fyne.TextTruncateOff
			lbl.Wrapping = fyne.TextWrapOff
			return container.NewBorder(nil, nil, chk, nil, lbl)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(files) {
				return
			}
			f := files[i]
			box := o.(*fyne.Container)
			lbl := findFirstLabel(box)
			chk := findFirstCheck(box)
			if lbl != nil {
				prefix := ""
				if f.IsDir {
					prefix = "[DIR] "
				}
				lbl.SetText(prefix + f.Name)
			}
			if chk != nil {
				nameCopy := f.Name
				cur := selectedNames[nameCopy]
				chk.SetChecked(cur)
				chk.Refresh()
				chk.OnChanged = func(v bool) {
					if v {
						selectedNames[nameCopy] = true
					} else {
						delete(selectedNames, nameCopy)
					}
				}
			}
		},
	)
	filesList.OnSelected = func(id widget.ListItemID) {
		// Detect double-click within 500ms on same item to open directory.
		now := time.Now()
		if int(id) == lastClickIdx && now.Sub(lastClickAt) <= 500*time.Millisecond {
			if int(id) >= 0 && int(id) < len(files) && files[int(id)].IsDir && loadDir != nil && curPathBind != nil {
				p, _ := curPathBind.Get()
				loadDir(path.Join(p, files[int(id)].Name))
			}
		}
		// Update click tracking and always unselect to allow repeated selection events
		selectedIndex = int(id)
		lastClickIdx = int(id)
		lastClickAt = now
		filesList.Unselect(id)
	}

	// UI controls
	userSelect := widget.NewSelect([]string{}, nil)
	curPathBind = binding.NewString()
	curPath := "/"
	_ = curPathBind.Set(curPath)
	pathEntry := widget.NewEntryWithData(curPathBind)
	pathEntry.Disable()

	refreshUsers := func() {
		serial, _ := selectedSerialBind.Get()
		if serial == "" {
			fyne.Do(func() {
				_ = usersBind.Set([]string{})
				userSelect.Options = []string{}
				userSelect.Refresh()
			})
			return
		}
		go func() {
			usrs, _, err := mgr.Users(serial)
			fyne.Do(func() {
				if err != nil {
					a := []string{"Error: " + err.Error()}
					_ = usersBind.Set(a)
					userSelect.Options = a
					userSelect.Refresh()
					return
				}
				opts := make([]string, 0, len(usrs))
				for _, u := range usrs {
					opts = append(opts, fmt.Sprintf("%d (%s)", u.ID, u.Name))
				}
				_ = usersBind.Set(opts)
				userSelect.Options = opts
				if len(opts) > 0 {
					userSelect.SetSelected(opts[0])
				}
				userSelect.Refresh()
			})
		}()
	}

	loadDir = func(p string) {
		serial, _ := selectedSerialBind.Get()
		if serial == "" {
			dialog.ShowInformation("No device", "Please select a device.", w)
			return
		}
		go func() {
			list, _, err := mgr.ListDir(serial, p)
			fyne.Do(func() {
				if err != nil {
					files = []adb.FileEntry{}
				} else {
					files = list
				}
				// Apply current sorting if configured
				if applySort != nil {
					applySort()
				}
				// Reset selection on directory load
				selectedIndex = -1
				selectedNames = map[string]bool{}
				// Update UI safely on main thread
				_ = curPathBind.Set(p)
				filesList.Refresh()
			})
		}()
	}

	onUserChanged := func(sel string) {
		// Extract user id
		if sel == "" {
			return
		}
		uid := 0
		if sp := strings.SplitN(sel, " ", 2); len(sp) > 0 {
			if v, err := strconv.Atoi(sp[0]); err == nil {
				uid = v
			}
		}
		start := "/storage/emulated/" + strconv.Itoa(uid)
		loadDir(start)
	}

	userSelect.OnChanged = onUserChanged

	btnUp := widget.NewButton("Up", func() {
		p, _ := curPathBind.Get()
		parent := path.Dir(p)
		if parent == "." || parent == "/" {
			parent = "/"
		}
		loadDir(parent)
	})
	btnRefresh := widget.NewButton("Refresh", func() {
		p, _ := curPathBind.Get()
		loadDir(p)
	})
	btnOpen := widget.NewButton("Open", func() {
		if selectedIndex < 0 || selectedIndex >= len(files) {
			return
		}
		entry := files[selectedIndex]
		if entry.IsDir {
			p, _ := curPathBind.Get()
			loadDir(path.Join(p, entry.Name))
		}
	})

	// React to device selection changes: reload users and default path
	selectedSerialBind.AddListener(binding.NewDataListener(func() {
		refreshUsers()
	}))

	// Sorting and transfer controls
	sortSelect := widget.NewSelect([]string{
		"按照字母顺序排序",
		"按照字母倒序排序",
		"按照文件大小从小到大排序",
		"按照文件大小从大到小排序",
		"按照修改时间从远到近排序",
		"按照修改时间从近到远排序",
	}, func(s string) {
		if s == "" {
			return
		}
		sortMode = s
		if applySort != nil {
			applySort()
		}
		filesList.Refresh()
	})
	sortSelect.SetSelected("按照字母顺序排序")

	btnUpload := widget.NewButton("Upload…", func() {
		serial, _ := selectedSerialBind.Get()
		if serial == "" {
			dialog.ShowInformation("No device", "Please select a device.", w)
			return
		}
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if rc == nil {
				return
			}
			defer rc.Close()
			lp := rc.URI().Path()
			if strings.TrimSpace(lp) == "" {
				dialog.ShowInformation("Upload", "无效的本地文件路径。", w)
				return
			}
			cur, _ := curPathBind.Get()
			go func() {
				out, e := mgr.Push(serial, lp, cur)
				fyne.Do(func() {
					if e != nil {
						dialog.ShowError(fmt.Errorf("upload failed: %v\n%s", e, out), w)
					} else {
						dialog.ShowInformation("Upload", "上传完成。", w)
						loadDir(cur)
					}
				})
			}()
		}, w)
		fd.Show()
	})

	btnDownload := widget.NewButton("Download", func() {
		serial, _ := selectedSerialBind.Get()
		if serial == "" {
			dialog.ShowInformation("No device", "Please select a device.", w)
			return
		}
		// collect selected names
		var names []string
		for _, f := range files {
			if selectedNames[f.Name] {
				names = append(names, f.Name)
			}
		}
		if len(names) == 0 {
			dialog.ShowInformation("Download", "请在文件列表勾选要下载的文件/目录。", w)
			return
		}
		dd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			localDir := uri.Path()
			cur, _ := curPathBind.Get()
			// build remote paths
			var remote []string
			for _, n := range names {
				remote = append(remote, path.Join(cur, n))
			}
			go func() {
				out, e := mgr.PullMultiple(serial, remote, localDir, true)
				fyne.Do(func() {
					if e != nil {
						dialog.ShowError(fmt.Errorf("download failed: %v\n%s", e, out), w)
					} else {
						dialog.ShowInformation("Download", "下载完成。\n"+out, w)
					}
				})
			}()
		}, w)
		dd.Show()
	})

		// Top controls
		btnSelAllFiles := widget.NewButton("全选", func() {
			for _, f := range files {
				selectedNames[f.Name] = true
			}
			filesList.Refresh()
		})
		btnSelNoneFiles := widget.NewButton("清空选择", func() {
			selectedNames = map[string]bool{}
			filesList.Refresh()
		})
		controls := container.NewHBox(userSelect, btnUp, btnRefresh, sortSelect, btnSelAllFiles, btnSelNoneFiles, btnUpload, btnDownload)
		// Make path entry expand to full width; keep label at left and "Open" at right
		pathRow := container.NewBorder(nil, nil, widget.NewLabel("Path:"), btnOpen, pathEntry)
		top := container.NewVBox(
			container.NewHBox(widget.NewLabel("User:"), controls),
			pathRow,
		)
	
		return container.NewBorder(top, nil, nil, nil, filesList)
}

// Parameters tab: show getprop key/value
func buildParametersTab(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String) fyne.CanvasObject {
	propsBind := binding.NewStringList()
	list := widget.NewListWithData(
		propsBind,
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(di binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(di.(binding.String))
		},
	)
	doRefresh := func() {
		serial, _ := selectedSerialBind.Get()
		if serial == "" {
			fyne.Do(func() { _ = propsBind.Set([]string{"Select a device."}) })
			return
		}
		go func() {
			props, _, err := mgr.GetProps(serial)
			fyne.Do(func() {
				if err != nil {
					_ = propsBind.Set([]string{"Error: " + err.Error()})
					return
				}
				lines := make([]string, 0, len(props))
				for k, v := range props {
					lines = append(lines, fmt.Sprintf("%s = %s", k, v))
				}
				if len(lines) == 0 {
					lines = []string{"No properties returned."}
				}
				_ = propsBind.Set(lines)
			})
		}()
	}
	refreshBtn := widget.NewButton("Refresh", doRefresh)

	// Auto refresh on device change
	selectedSerialBind.AddListener(binding.NewDataListener(func() { doRefresh() }))

	return container.NewBorder(
		container.NewHBox(widget.NewLabel("Parameters"), refreshBtn),
		nil, nil, nil, list,
	)
}

// Commands tab: basic device actions (reboot etc.)
func buildCommandsTab(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String) fyne.CanvasObject {
	modeSelect := widget.NewSelect([]string{"", "recovery", "bootloader"}, func(string) {})
	modeSelect.PlaceHolder = "normal"
	rebootBtn := widget.NewButton("Reboot", func() {
		serial, _ := selectedSerialBind.Get()
		if serial == "" {
			dialog.ShowInformation("No device", "Please select a device.", w)
			return
		}
		mode := modeSelect.Selected
		go func() {
			out, err := mgr.Reboot(serial, mode)
			msg := "Reboot requested."
			if err != nil {
				msg = "Error: " + err.Error() + "\n" + out
			}
			fyne.Do(func() {
				dialog.ShowInformation("Reboot", msg, w)
			})
		}()
	})
	form := widget.NewForm(
		widget.NewFormItem("Reboot mode", modeSelect),
	)
	return container.NewBorder(
		nil, nil, nil, nil,
		container.NewVBox(form, rebootBtn),
	)
}

func updateStatusDevices(statusBind binding.String, mgr *adb.Manager, count int) {
	go func() {
		ver, _ := mgr.Version()
		fyne.Do(func() {
			_ = statusBind.Set(fmt.Sprintf("%s | Devices: %d", firstLine(ver), count))
		})
	}()
}

// openSettingsDialog shows a dialog to configure ADB path with Detect/Browse/Save.
func openSettingsDialog(w fyne.Window, mgr *adb.Manager, cfg *config.Config) {
	pathEntry := widget.NewEntry()
	initial := cfg.ADBPath
	if strings.TrimSpace(initial) == "" {
		initial = mgr.Path
	}
	pathEntry.SetText(initial)
	pathEntry.SetPlaceHolder("/path/to/adb or adb")

	// Theme mode select (System/Light/Dark), default to System when empty
	themeSelect := widget.NewSelect([]string{"System", "Light", "Dark"}, nil)
	curMode := strings.ToLower(strings.TrimSpace(cfg.ThemeMode))
	if curMode == "" {
		curMode = "system"
	}
	switch curMode {
	case "light":
		themeSelect.SetSelected("Light")
	case "dark":
		themeSelect.SetSelected("Dark")
	default:
		themeSelect.SetSelected("System")
	}

	detectBtn := widget.NewButton("Detect", func() {
		p := adb.AutoDetect()
		if p == "" {
			dialog.ShowInformation("Detect", "Could not auto-detect ADB. Please browse and select manually.", w)
			return
		}
		pathEntry.SetText(p)
	})
	browseBtn := widget.NewButton("Browse…", func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if rc == nil {
				return
			}
			defer rc.Close()
			p := rc.URI().Path()
			if strings.TrimSpace(p) == "" {
				dialog.ShowInformation("Select ADB", "Invalid selection.", w)
				return
			}
			pathEntry.SetText(p)
		}, w)
		fd.Show()
	})
	saveBtn := widget.NewButton("Save", func() {
		p := strings.TrimSpace(pathEntry.Text)
		valid, err := adb.ValidatePath(p)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		// Map theme selection to config value
		mode := "system"
		switch themeSelect.Selected {
		case "Light":
			mode = "light"
		case "Dark":
			mode = "dark"
		default:
			mode = "system"
		}

		cfg.ADBPath = valid
		cfg.ThemeMode = mode
		if err := config.Save(cfg); err != nil {
			dialog.ShowError(err, w)
			return
		}
		mgr.Path = valid

		// Apply theme immediately
		var th fyne.Theme = theme.DefaultTheme()
		switch mode {
		case "light":
			th = theme.LightTheme()
		case "dark":
			th = theme.DarkTheme()
		default:
			th = theme.DefaultTheme()
		}
		fyne.CurrentApp().Settings().SetTheme(th)

		dialog.ShowInformation("Saved", "Settings saved successfully.", w)
	})

	form := widget.NewForm(
		widget.NewFormItem("ADB Path", pathEntry),
		widget.NewFormItem("Theme Mode", themeSelect),
	)
	actions := container.NewHBox(detectBtn, browseBtn, saveBtn)
	content := container.NewVBox(form, actions)
	dialog.NewCustom("Settings", "Close", content, w).Show()
}

func mustGetInt(b binding.Int) int {
	v, _ := b.Get()
	return v
}

// mustGet returns the current value of a binding.String (empty on error).
func mustGet(b binding.String) string {
	v, _ := b.Get()
	return v
}

func firstLine(s string) string {
	i := strings.IndexByte(s, '\n')
	if i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}