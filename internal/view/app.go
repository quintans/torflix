package view

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/quintans/torflix/internal/components"
	"github.com/quintans/torflix/internal/viewmodel"
)

func App(vm *viewmodel.App) (fyne.CanvasObject, func(bool)) {
	search := buildSearch(vm)
	cache := buildCache(vm)

	settings := container.NewVBox()

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Search", theme.SearchIcon(), container.NewBorder(nil, nil, nil, nil, search)),
		container.NewTabItemWithIcon("Cache", theme.StorageIcon(), container.NewBorder(nil, nil, nil, nil, cache)),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settings),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	tabs.OnSelected = func(*container.TabItem) {
		vm.SelectedTab = tabs.SelectedIndex()
	}
	appAddSubtitlesSection(settings, vm)

	selectedTab := vm.SelectedTab
	enableTabs := func(u, p string) {
		if u != "" && p != "" {
			appEnableAllTabs(tabs, selectedTab)
			return
		}
		appDisableAllTabsButSettings(tabs)
	}

	vm.OSUsername.BindInMain(func(s string) {
		enableTabs(s, vm.OSPassword.Get())
	})
	vm.OSPassword.BindInMain(func(s string) {
		enableTabs(vm.OSUsername.Get(), s)
	})

	return tabs, func(bool) {
		vm.Unmount()
	}
}

func appAddSubtitlesSection(sections *fyne.Container, vm *viewmodel.App) {
	sections.Add(widget.NewLabel("OpenSubtitles.com"))
	sections.Add(canvas.NewLine(color.Gray{128}))

	usernameEntry := widget.NewEntry()
	usernameEntry.SetText(vm.OSUsername.Get())

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetText(vm.OSPassword.Get())

	sections.Add(container.NewHBox(
		widget.NewForm(
			widget.NewFormItem("Username", components.NewMinSizeWrapper(usernameEntry, fyne.NewSize(200, 40))),
			widget.NewFormItem("Password", components.NewMinSizeWrapper(passwordEntry, fyne.NewSize(200, 40))),
		),
		layout.NewSpacer(),
	),
	)
	bt := widget.NewButton("CHANGE", func() {
		vm.SetOpenSubtitles(usernameEntry.Text, passwordEntry.Text)
	})
	bt.Importance = widget.HighImportance
	sections.Add(container.NewHBox(bt, layout.NewSpacer()))
	sections.Add(widget.NewSeparator())
}

func appDisableAllTabsButSettings(tabs *container.AppTabs) {
	settingsIdx := len(tabs.Items) - 1
	tabs.SelectIndex(settingsIdx)
	for k := range len(tabs.Items) {
		if k == settingsIdx {
			continue
		}

		tabs.DisableIndex(k)
	}
}

func appEnableAllTabs(tabs *container.AppTabs, index int) {
	for k := range len(tabs.Items) {
		tabs.EnableIndex(k)
	}
	tabs.SelectIndex(index)
}
