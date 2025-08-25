package ui

import (
	"fmt"
	"image/color"
	"log"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"adb-gui/internal/adb"
	"adb-gui/internal/config"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// coloredLabel is a custom widget for displaying colored text
type coloredLabel struct {
	widget.BaseWidget
	text  string
	color color.Color
}

func newColoredLabel(text string, color color.Color) *coloredLabel {
	l := &coloredLabel{text: text, color: color}
	l.ExtendBaseWidget(l)
	return l
}

func (l *coloredLabel) CreateRenderer() fyne.WidgetRenderer {
	text := canvas.NewText(l.text, l.color)
	return &coloredLabelRenderer{
		label: l,
		text:  text,
	}
}

type coloredLabelRenderer struct {
	label *coloredLabel
	text  *canvas.Text
}

func (r *coloredLabelRenderer) Layout(size fyne.Size) {
	r.text.Resize(size)
}

func (r *coloredLabelRenderer) MinSize() fyne.Size {
	return r.text.MinSize()
}

func (r *coloredLabelRenderer) Refresh() {
	r.text.Text = r.label.text
	r.text.Color = r.label.color
	r.text.Refresh()
}

func (r *coloredLabelRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.text}
}

func (r *coloredLabelRenderer) Destroy() {}

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
//  1. Applications
//  2. Storage
//  3. Parameters
//  4. Commands
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
	appsTab := buildApplicationsTab(w, mgr, selectedSerialBind, &devices)
	storageTab := buildStorageTab(w, mgr, selectedSerialBind)
	paramsTab := buildParametersTab(w, mgr, selectedSerialBind)
	getVarTab := buildGetVarTab(w, mgr, selectedSerialBind)
	cmdsTab := buildCommandsTab(w, mgr, selectedSerialBind)
	fastbootTab := buildFastbootTab(w, mgr, selectedSerialBind)

	rightTabs := container.NewAppTabs(
		container.NewTabItem(T("applications"), appsTab),
		container.NewTabItem(T("storage"), storageTab),
		container.NewTabItem(T("parameters"), paramsTab),
		container.NewTabItem(T("getvar"), getVarTab),
		container.NewTabItem(T("commands"), cmdsTab),
		container.NewTabItem(T("fastboot"), fastbootTab),
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
		dialog.ShowInformation(T("adb_not_found"), T("adb_not_detected"), w)
	}
	refreshDevices()

	go func() {
		ver, _ := mgr.Version()
		fyne.Do(func() {
			_ = statusBind.Set(fmt.Sprintf("%s | %s: %d", firstLine(ver), T("devices"), mustGetInt(devCountBind)))
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
	// 创建设置菜单项
	settings := fyne.NewMenuItem(T("settings"), func() {
		openSettingsDialog(w, mgr, cfg)
	})

	// 创建关于菜单项
	about := fyne.NewMenuItem(T("about"), func() {
		dialog.ShowInformation(T("about"), T("about_description"), w)
	})

	// 构建文件菜单，只包含设置选项
	fileMenu := fyne.NewMenu(T("file"), settings)

	// 构建帮助菜单
	helpMenu := fyne.NewMenu(T("help"), about)

	// 返回主菜单
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

	header := widget.NewLabel(T("devices"))

	// Refresh button
	refreshBtn := widget.NewButton(T("refresh"), nil)

	// Devices list
	list := widget.NewList(
		func() int {
			return len(*devices)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("serial")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(*devices) {
				return
			}
			d := (*devices)[i]
			lbl := o.(*widget.Label)
			lbl.Truncation = fyne.TextTruncateEllipsis
			lbl.SetText(d.Serial)
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
func buildApplicationsTab(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String, devices *[]adb.Device) fyne.CanvasObject {
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
	title := widget.NewLabel(T("applications"))
	userSelect := widget.NewSelect([]string{"0 (" + T("owner_user") + ")"}, func(string) {})
	userSelect.PlaceHolder = T("select_user")
	// App type filter: 用户应用(第三方) / 系统应用
	appTypeSelect := widget.NewSelect([]string{T("user_apps"), T("system_apps")}, nil)
	appTypeSelect.PlaceHolder = T("app_category")
	appTypeSelect.SetSelected(T("user_apps"))
	pkgCount := widget.NewLabel(T("packages_count") + ": 0")
	refreshBtn := widget.NewButton(T("refresh"), nil)

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
			// 应用自适应颜色
			if app := fyne.CurrentApp(); app != nil {
				if settings := app.Settings(); settings != nil {
					if currentTheme := settings.Theme(); currentTheme != nil {
						// 尝试检测当前主题变体
						bgColorLight := currentTheme.Color(theme.ColorNameBackground, theme.VariantLight)
						bgColorDark := currentTheme.Color(theme.ColorNameBackground, theme.VariantDark)
						if bgColorLight != bgColorDark {
							// 如果深色和浅色主题的背景色不同，尝试检测当前主题
							// 使用颜色亮度来判断
							if isDarkColor(bgColorLight) {
								// 当前是深色主题
							}
						}
						// 注意：这里我们不能直接设置Label的颜色，但可以通过主题系统来影响
					}
				}
			}
			btnUninstall := widget.NewButton(T("uninstall"), nil)
			btnClear := widget.NewButton(T("clear_data"), nil)
			btnForceStop := widget.NewButton(T("force_stop"), nil)
			btnExtractApk := widget.NewButton(T("extract_apk"), nil)
			btnExtractAll := widget.NewButton(T("extract_apk_data"), nil)
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
					dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
					return
				}
				go func() {
					out, err := mgr.Uninstall(serial, selectedUserID, pkg)
					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(fmt.Errorf("%s: %v\n%s", T("uninstall_failed"), err, out), w)
						} else {
							dialog.ShowInformation(T("uninstall"), T("uninstalled")+" "+pkg, w)
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
					dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
					return
				}
				go func() {
					out, err := mgr.ClearData(serial, pkg)
					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(fmt.Errorf("%s: %v\n%s", T("clear_data_failed"), err, out), w)
						} else {
							dialog.ShowInformation(T("clear_data"), T("cleared_data")+" for "+pkg, w)
						}
					})
				}()
			}
			btnForce.OnTapped = func() {
				serial, _ := selectedSerialBind.Get()
				if strings.TrimSpace(serial) == "" {
					dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
					return
				}
				go func() {
					out, err := mgr.ForceStop(serial, pkg)
					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(fmt.Errorf("%s: %v\n%s", T("force_stop_failed"), err, out), w)
						} else {
							dialog.ShowInformation(T("force_stop"), T("forced_stop")+" for "+pkg, w)
						}
					})
				}()
			}
			btnApk.OnTapped = func() {
				serial, _ := selectedSerialBind.Get()
				if strings.TrimSpace(serial) == "" {
					dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
					return
				}
				go func() {
					out, err := mgr.ExtractApk(serial, pkg, pkg)
					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(fmt.Errorf("%s: %v\n%s", T("extract_apk_failed"), err, out), w)
						} else {
							dialog.ShowInformation(T("extract_apk"), T("apk_extracted")+" for "+pkg+" into current directory.", w)
						}
					})
				}()
			}
			btnAll.OnTapped = func() {
				serial, _ := selectedSerialBind.Get()
				if strings.TrimSpace(serial) == "" {
					dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
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
							dialog.ShowError(fmt.Errorf("%s:\nAPK: %v\nData: %v\n\n%s", T("extract_issues"), err1, err2, msg), w)
						} else {
							if msg == "" {
								msg = T("extracted_apk_data") + " ./" + dest
							}
							dialog.ShowInformation(T("extract_apk_data"), msg, w)
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
					pkgs = []string{T("error") + ": " + err.Error()}
					pkgCount.SetText(T("packages_count") + ": 0")
					list.Refresh()
					return
				}
				log.Printf("[apps] packages loaded: %d", len(plist))
				pkgs = plist
				// reset labels for new list and start async label fetching
				labels = map[string]string{}
				list.Refresh()
				pkgCount.SetText(fmt.Sprintf(T("packages_count")+": %d", len(plist)))
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
		serial, _ := selectedSerialBind.Get()
		for _, d := range *devices {
			if d.Serial == serial && d.State != "fastboot" {
				refreshUsers()
				if refreshPackages != nil {
					refreshPackages()
				}
				return
			}
		}
	}))

	// Initial tip
	pkgs = []string{T("select_device_to_list_packages")}
	pkgCount.SetText(T("packages_count") + ": 0")

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
			dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
			return
		}
		targets := getSelected()
		if len(targets) == 0 {
			dialog.ShowInformation(title, T("please_select_at_least_one_app"), w)
			return
		}
		go func() {
			var okN, failN int
			var msgs []string
			for _, p := range targets {
				out, err := op(p)
				if err != nil {
					failN++
					msgs = append(msgs, fmt.Sprintf("[%s] %s: %v\n%s", p, T("error"), err, out))
				} else {
					okN++
					if strings.TrimSpace(out) != "" {
						msgs = append(msgs, fmt.Sprintf("[%s] %s", p, out))
					} else {
						msgs = append(msgs, fmt.Sprintf("[%s] %s", p, T("ok")))
					}
				}
				time.Sleep(100 * time.Millisecond)
			}
			summary := fmt.Sprintf("%s %s: %s %d, %s %d\n\n%s", title, T("complete"), T("success"), okN, T("failed"), failN, strings.Join(msgs, "\n"))
			fyne.Do(func() {
				dialog.ShowInformation(title, summary, w)
				if refreshAfter && refreshPackages != nil {
					refreshPackages()
				}
			})
		}()
	}

	// Batch buttons
	btnSelAll := widget.NewButton(T("select_all"), func() {
		for _, p := range pkgs {
			selectedPkgs[p] = true
		}
		list.Refresh()
	})
	btnSelNone := widget.NewButton(T("clear_selection"), func() {
		selectedPkgs = map[string]bool{}
		list.Refresh()
	})
	btnBatchUninst := widget.NewButton(T("batch_uninstall"), func() {
		doBatch(T("batch_uninstall"), func(p string) (string, error) {
			return mgr.Uninstall(mustGet(selectedSerialBind), selectedUserID, p)
		}, true)
	})
	btnBatchClear := widget.NewButton(T("batch_clear_data"), func() {
		doBatch(T("batch_clear_data"), func(p string) (string, error) {
			return mgr.ClearData(mustGet(selectedSerialBind), p)
		}, false)
	})
	btnBatchForce := widget.NewButton(T("batch_force_stop"), func() {
		doBatch(T("batch_force_stop"), func(p string) (string, error) {
			return mgr.ForceStop(mustGet(selectedSerialBind), p)
		}, false)
	})
	btnBatchExtractApk := widget.NewButton(T("batch_extract_apk"), func() {
		doBatch(T("batch_extract_apk"), func(p string) (string, error) {
			return mgr.ExtractApk(mustGet(selectedSerialBind), p, p)
		}, false)
	})
	btnBatchExtractAll := widget.NewButton(T("batch_extract_apk_data"), func() {
		doBatch(T("batch_extract_apk_data"), func(p string) (string, error) {
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
		widget.NewLabel(" "+T("user")),
		userSelect,
		widget.NewLabel("  "+T("type")+":"),
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
		dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
		return
	}
	go func() {
		pkgs, _, err := mgr.InstalledPackages(serial)
		fyne.Do(func() {
			if err != nil {
				_ = outBind.Set([]string{T("error") + ": " + err.Error()})
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
	sortMode := T("sort_alphabetical")
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
		case T("sort_alphabetical"):
			sort.SliceStable(files, func(i, j int) bool {
				return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
			})
		case T("sort_alphabetical_reverse"):
			sort.SliceStable(files, func(i, j int) bool {
				return strings.ToLower(files[i].Name) > strings.ToLower(files[j].Name)
			})
		case T("sort_size_small_to_large"):
			sort.SliceStable(files, func(i, j int) bool {
				return files[i].Size < files[j].Size
			})
		case T("sort_size_large_to_small"):
			sort.SliceStable(files, func(i, j int) bool {
				return files[i].Size > files[j].Size
			})
		case T("sort_time_old_to_new"):
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
		case T("sort_time_new_to_old"):
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
			// 使用自适应颜色，不再硬编码白色
			label := newColoredLabel("file", color.Black) // 默认颜色，会在更新时动态调整
			return container.NewBorder(nil, nil, chk, nil, label)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(files) {
				return
			}
			f := files[i]
			box := o.(*fyne.Container)

			var chk *widget.Check
			var label *coloredLabel
			for _, obj := range box.Objects {
				if c, ok := obj.(*widget.Check); ok {
					chk = c
				}
				if l, ok := obj.(*coloredLabel); ok {
					label = l
				}
			}

			if label != nil {
				label.text = f.Name
				// 使用改进的自适应颜色系统
				if app := fyne.CurrentApp(); app != nil {
					if settings := app.Settings(); settings != nil {
						if currentTheme := settings.Theme(); currentTheme != nil {
							// 简化的主题检测：直接使用Fyne的内置检测
							variant := app.Settings().ThemeVariant()

							// 应用自适应颜色
							label.color = getFileItemColor(f.IsDir, variant)
						} else {
							// 回退到传统颜色
							if f.IsDir {
								label.color = color.NRGBA{B: 255, A: 255}
							} else {
								label.color = color.Black
							}
						}
					} else {
						// 回退到传统颜色
						if f.IsDir {
							label.color = color.NRGBA{B: 255, A: 255}
						} else {
							label.color = color.Black
						}
					}
				} else {
					// 回退到传统颜色
					if f.IsDir {
						label.color = color.NRGBA{B: 255, A: 255}
					} else {
						label.color = color.Black
					}
				}
				label.Refresh()
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

	// 共享的路径导航函数
	navigateToPath := func(targetPath string) {
		// 如果没有选中目录项，则处理路径输入框中的自定义路径
		if targetPath == "" {
			targetPath = strings.TrimSpace(pathEntry.Text)
		}

		if targetPath == "" {
			dialog.ShowInformation(T("error"), T("please_enter_path"), w)
			return
		}

		// 尝试切换到指定路径
		go func() {
			// 验证路径是否存在
			serial, _ := selectedSerialBind.Get()
			if serial == "" {
				fyne.Do(func() {
					dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
				})
				return
			}

			// 尝试列出目录内容来验证路径是否存在
			_, _, err := mgr.ListDir(serial, targetPath)
			fyne.Do(func() {
				if err != nil {
					// 路径不存在或切换失败
					dialog.ShowError(fmt.Errorf("%s: %v", T("path_not_found"), err), w)
				} else {
					// 路径存在，切换到该路径
					loadDir(targetPath)
				}
			})
		}()
	}

	// Make path entry editable for custom path input
	pathEntry.OnSubmitted = func(path string) {
		// Navigate to custom path when user presses Enter
		navigateToPath(path)
	}

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
					a := []string{T("error") + ": " + err.Error()}
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
			dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
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

	btnUp := widget.NewButton(T("up"), func() {
		p, _ := curPathBind.Get()
		parent := path.Dir(p)
		if parent == "." || parent == "/" {
			parent = "/"
		}
		loadDir(parent)
	})
	btnRefresh := widget.NewButton(T("refresh"), func() {
		p, _ := curPathBind.Get()
		loadDir(p)
	})
	btnOpen := widget.NewButton(T("open"), func() {
		// 优先处理选中的目录项
		if selectedIndex >= 0 && selectedIndex < len(files) {
			entry := files[selectedIndex]
			if entry.IsDir {
				p, _ := curPathBind.Get()
				loadDir(path.Join(p, entry.Name))
				return
			}
		}

		// 如果没有选中目录项，则使用共享的路径导航函数
		navigateToPath("")
	})

	// React to device selection changes: reload users and default path
	selectedSerialBind.AddListener(binding.NewDataListener(func() {
		refreshUsers()
	}))

	// Sorting and transfer controls
	sortSelect := widget.NewSelect([]string{
		T("sort_alphabetical"),
		T("sort_alphabetical_reverse"),
		T("sort_size_small_to_large"),
		T("sort_size_large_to_small"),
		T("sort_time_old_to_new"),
		T("sort_time_new_to_old"),
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
	sortSelect.SetSelected(T("sort_alphabetical"))

	btnUpload := widget.NewButton(T("upload"), func() {
		serial, _ := selectedSerialBind.Get()
		if serial == "" {
			dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
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
				dialog.ShowInformation(T("upload"), T("invalid_local_path"), w)
				return
			}
			cur, _ := curPathBind.Get()
			go func() {
				out, e := mgr.Push(serial, lp, cur)
				fyne.Do(func() {
					if e != nil {
						dialog.ShowError(fmt.Errorf("%s: %v\n%s", T("upload_failed"), e, out), w)
					} else {
						dialog.ShowInformation(T("upload"), T("upload_complete"), w)
						loadDir(cur)
					}
				})
			}()
		}, w)
		fd.Show()
	})

	btnDownload := widget.NewButton(T("download"), func() {
		serial, _ := selectedSerialBind.Get()
		if serial == "" {
			dialog.ShowInformation(T("no_device"), T("please_select_device"), w)
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
			dialog.ShowInformation(T("download"), T("please_select_files"), w)
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
						dialog.ShowError(fmt.Errorf("%s: %v\n%s", T("download_failed"), e, out), w)
					} else {
						dialog.ShowInformation(T("download"), T("download_complete")+"\n"+out, w)
					}
				})
			}()
		}, w)
		dd.Show()
	})

	// Top controls
	btnSelAllFiles := widget.NewButton(T("select_all"), func() {
		for _, f := range files {
			selectedNames[f.Name] = true
		}
		filesList.Refresh()
	})
	btnSelNoneFiles := widget.NewButton(T("clear_selection"), func() {
		selectedNames = map[string]bool{}
		filesList.Refresh()
	})
	controls := container.NewHBox(userSelect, btnUp, btnRefresh, sortSelect, btnSelAllFiles, btnSelNoneFiles, btnUpload, btnDownload)
	// Make path entry expand to full width; keep label at left and "Open" at right
	pathRow := container.NewBorder(nil, nil, widget.NewLabel(T("path")), btnOpen, pathEntry)
	top := container.NewVBox(
		container.NewHBox(widget.NewLabel(T("user")), controls),
		pathRow,
	)

	return container.NewBorder(top, nil, nil, nil, filesList)
}

// Parameters tab: show getprop key/value
func buildParametersTab(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String) fyne.CanvasObject {
	return buildKeyValueTab(w, T("parameters"), selectedSerialBind, func(serial string) (map[string]string, error) {
		props, _, err := mgr.GetProps(serial)
		return props, err
	})
}

func buildGetVarTab(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String) fyne.CanvasObject {
	return buildKeyValueTab(w, T("getvar"), selectedSerialBind, func(serial string) (map[string]string, error) {
		vars, _, err := mgr.GetVarAll(serial)
		return vars, err
	})
}

// buildKeyValueTab creates a generic tab for displaying key-value pairs with multi-select and copy.
func buildKeyValueTab(w fyne.Window, title string, selectedSerialBind binding.String, fetcher func(string) (map[string]string, error)) fyne.CanvasObject {
	var items []string
	selectedItems := make(map[string]bool)

	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewCheck("", nil), widget.NewLabel("key = value"))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(items) {
				return
			}
			item := items[i]
			box := o.(*fyne.Container)
			check := box.Objects[0].(*widget.Check)
			label := box.Objects[1].(*widget.Label)
			label.SetText(item)
			check.SetChecked(selectedItems[item])
			check.OnChanged = func(checked bool) {
				selectedItems[item] = checked
			}
		},
	)

	doRefresh := func() {
		serial, _ := selectedSerialBind.Get()
		if serial == "" {
			items = []string{T("select_device")}
			list.Refresh()
			return
		}
		go func() {
			data, err := fetcher(serial)
			fyne.Do(func() {
				if err != nil {
					items = []string{T("error") + ": " + err.Error()}
					list.Refresh()
					return
				}
				var lines []string
				for k, v := range data {
					lines = append(lines, fmt.Sprintf("%s = %s", k, v))
				}
				sort.Strings(lines)
				items = lines
				if len(items) == 0 {
					items = []string{T("no_data")}
				}
				selectedItems = make(map[string]bool)
				list.Refresh()
			})
		}()
	}

	refreshBtn := widget.NewButton(T("refresh"), doRefresh)
	copyBtn := widget.NewButton(T("copy_selected"), func() {
		var toCopy []string
		for _, item := range items {
			if selectedItems[item] {
				toCopy = append(toCopy, item)
			}
		}
		if len(toCopy) > 0 {
			w.Clipboard().SetContent(strings.Join(toCopy, "\n"))
		}
	})
	selectAllBtn := widget.NewButton(T("select_all"), func() {
		for _, item := range items {
			selectedItems[item] = true
		}
		list.Refresh()
	})
	deselectAllBtn := widget.NewButton(T("deselect_all"), func() {
		selectedItems = make(map[string]bool)
		list.Refresh()
	})

	selectedSerialBind.AddListener(binding.NewDataListener(doRefresh))

	return container.NewBorder(
		container.NewHBox(widget.NewLabel(title), refreshBtn, copyBtn, selectAllBtn, deselectAllBtn),
		nil, nil, nil, list,
	)
}

// Commands tab: basic device actions (reboot etc.)
func buildCommandsTab(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String) fyne.CanvasObject {
	// ADB Commands
	btnReboot := widget.NewButton(T("reboot"), func() {
		go func() {
			out, err := mgr.Reboot(mustGet(selectedSerialBind), "")
			showCmdResult(T("reboot"), out, err, w)
		}()
	})
	btnRebootBootloader := widget.NewButton(T("reboot_bootloader"), func() {
		go func() {
			out, err := mgr.Reboot(mustGet(selectedSerialBind), "bootloader")
			showCmdResult(T("reboot_bootloader"), out, err, w)
		}()
	})
	btnRebootRecovery := widget.NewButton(T("reboot_recovery"), func() {
		go func() {
			out, err := mgr.Reboot(mustGet(selectedSerialBind), "recovery")
			showCmdResult(T("reboot_recovery"), out, err, w)
		}()
	})
	fileSideload := widget.NewLabel("")
	btnSideload := widget.NewButton(T("sideload_zip"), func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			path := reader.URI().Path()
			fileSideload.SetText(path)
			go func() {
				out, err := mgr.Sideload(mustGet(selectedSerialBind), path)
				showCmdResult(T("sideload"), out, err, w)
			}()
		}, w)
	})
	btnStartShizuku := widget.NewButton(T("start_shizuku"), func() {
		go func() {
			out, err := mgr.StartShizuku(mustGet(selectedSerialBind))
			showCmdResult(T("start_shizuku"), out, err, w)
		}()
	})

	return container.NewVBox(
		widget.NewLabelWithStyle(T("adb_commands"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(btnReboot, btnRebootBootloader, btnRebootRecovery),
		container.NewHBox(btnSideload, fileSideload),
		btnStartShizuku,
	)
}

func buildFastbootTab(w fyne.Window, mgr *adb.Manager, selectedSerialBind binding.String) fyne.CanvasObject {
	// Fastboot Commands
	btnFbReboot := widget.NewButton(T("reboot"), func() {
		go func() {
			out, err := mgr.ExecFastboot(mustGet(selectedSerialBind), "reboot")
			showCmdResult(T("fastboot_reboot"), out, err, w)
		}()
	})
	btnFbRebootBootloader := widget.NewButton(T("reboot_bootloader"), func() {
		go func() {
			out, err := mgr.ExecFastboot(mustGet(selectedSerialBind), "reboot-bootloader")
			showCmdResult(T("fastboot_reboot_bootloader"), out, err, w)
		}()
	})
	btnFbContinue := widget.NewButton(T("continue"), func() {
		go func() {
			out, err := mgr.ExecFastboot(mustGet(selectedSerialBind), "continue")
			showCmdResult(T("fastboot_continue"), out, err, w)
		}()
	})
	btnFbUnlock := widget.NewButton(T("oem_unlock"), func() {
		go func() {
			out, err := mgr.ExecFastboot(mustGet(selectedSerialBind), "oem", "unlock")
			showCmdResult(T("fastboot_oem_unlock"), out, err, w)
		}()
	})
	btnFbFlashingUnlock := widget.NewButton(T("flashing_unlock"), func() {
		go func() {
			out, err := mgr.ExecFastboot(mustGet(selectedSerialBind), "flashing", "unlock")
			showCmdResult(T("fastboot_flashing_unlock"), out, err, w)
		}()
	})
	fileFlash := widget.NewLabel("")
	btnFbFlash := widget.NewButton(T("flash_partition"), func() {
		partitionEntry := widget.NewEntry()
		partitionEntry.PlaceHolder = T("partition_placeholder")
		dialog.ShowForm(T("flash_partition"), T("flash"), T("cancel"), []*widget.FormItem{
			widget.NewFormItem(T("partition_name"), partitionEntry),
		}, func(ok bool) {
			if !ok {
				return
			}
			partition := partitionEntry.Text
			if partition == "" {
				return
			}
			dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil || reader == nil {
					return
				}
				path := reader.URI().Path()
				fileFlash.SetText(fmt.Sprintf("%s: %s", partition, path))
				go func() {
					out, err := mgr.ExecFastboot(mustGet(selectedSerialBind), "flash", partition, path)
					showCmdResult(T("fastboot_flash"), out, err, w)
				}()
			}, w)
		}, w)
	})
	fileUpdate := widget.NewLabel("")
	btnFbUpdate := widget.NewButton(T("update_from_zip"), func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			path := reader.URI().Path()
			fileUpdate.SetText(path)
			go func() {
				out, err := mgr.ExecFastboot(mustGet(selectedSerialBind), "update", path)
				showCmdResult(T("fastboot_update"), out, err, w)
			}()
		}, w)
	})
	btnFbOemDeviceInfo := widget.NewButton(T("oem_device_info"), func() {
		go func() {
			out, err := mgr.ExecFastboot(mustGet(selectedSerialBind), "oem", "device-info")
			showCmdResult(T("fastboot_oem_device_info"), out, err, w)
		}()
	})
	btnFbOemEdl := widget.NewButton(T("oem_edl"), func() {
		go func() {
			out, err := mgr.ExecFastboot(mustGet(selectedSerialBind), "oem", "edl")
			showCmdResult(T("fastboot_oem_edl"), out, err, w)
		}()
	})

	return container.NewVBox(
		widget.NewLabelWithStyle(T("fastboot_commands"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(btnFbReboot, btnFbRebootBootloader, btnFbContinue),
		container.NewHBox(btnFbUnlock, btnFbFlashingUnlock),
		container.NewHBox(btnFbFlash, fileFlash),
		container.NewHBox(btnFbUpdate, fileUpdate),
		container.NewHBox(btnFbOemDeviceInfo, btnFbOemEdl),
	)
}

// showCmdResult displays the output of a command in a dialog.
func showCmdResult(title, out string, err error, w fyne.Window) {
	fyne.Do(func() {
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		d := dialog.NewCustom(title, T("ok"), widget.NewLabel(out), w)
		d.Resize(fyne.NewSize(600, 400))
		d.Show()
	})
}

func updateStatusDevices(statusBind binding.String, mgr *adb.Manager, count int) {
	go func() {
		ver, _ := mgr.Version()
		fyne.Do(func() {
			_ = statusBind.Set(fmt.Sprintf("%s | %s: %d", firstLine(ver), T("devices"), count))
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
	pathEntry.SetPlaceHolder(T("adb_path_placeholder"))

	// Theme mode select (System/Light/Dark), default to System when empty
	themeSelect := widget.NewSelect([]string{T("system"), T("light"), T("dark")}, nil)
	curMode := strings.ToLower(strings.TrimSpace(cfg.ThemeMode))
	if curMode == "" {
		curMode = "system"
	}
	switch curMode {
	case "light":
		themeSelect.SetSelected(T("light"))
	case "dark":
		themeSelect.SetSelected(T("dark"))
	default:
		themeSelect.SetSelected(T("system"))
	}

	detectBtn := widget.NewButton(T("detect"), func() {
		p := adb.AutoDetect()
		if p == "" {
			dialog.ShowInformation(T("detect"), T("could_not_auto_detect_adb"), w)
			return
		}
		pathEntry.SetText(p)
	})
	browseBtn := widget.NewButton(T("browse"), func() {
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
				dialog.ShowInformation(T("select_adb"), T("invalid_selection"), w)
				return
			}
			pathEntry.SetText(p)
		}, w)
		fd.Show()
	})
	saveBtn := widget.NewButton(T("save"), func() {
		p := strings.TrimSpace(pathEntry.Text)
		valid, err := adb.ValidatePath(p)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		// Map theme selection to config value
		mode := "system"
		switch themeSelect.Selected {
		case T("light"):
			mode = "light"
		case T("dark"):
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

		// Theme change requires restart
		dialog.ShowInformation(T("saved"), T("settings_saved"), w)
	})

	form := widget.NewForm(
		widget.NewFormItem(T("adb_path"), pathEntry),
		widget.NewFormItem(T("theme_mode"), themeSelect),
	)
	actions := container.NewHBox(detectBtn, browseBtn, saveBtn)
	content := container.NewVBox(form, actions)
	dialog.NewCustom(T("settings"), T("close"), content, w).Show()
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
