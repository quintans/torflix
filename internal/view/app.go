package view

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/components"
	"github.com/quintans/torflix/internal/lib/navigation"
	"github.com/quintans/torflix/internal/mycontainer"
	"github.com/quintans/torflix/internal/viewmodel"
)

func App(vm *viewmodel.ViewModel, _ *navigation.Navigator[*viewmodel.ViewModel]) (fyne.CanvasObject, func(bool)) {
	search := container.NewBorder(nil, nil, nil, nil)
	cache := container.NewBorder(nil, nil, nil, nil)
	settings := container.NewVBox()

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Search", theme.SearchIcon(), search),
		container.NewTabItemWithIcon("Cache", theme.StorageIcon(), cache),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settings),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	unbindSubtitles := appAddSubtitlesSection(settings, vm)

	enableTabs := func(u, p string) {
		if u != "" && p != "" {
			appEnableAllTabs(tabs)
			return
		}
		appDisableAllTabsButSettings(tabs)
	}
	unbindUsername := vm.App.OSUsername.Bind(func(s string) {
		enableTabs(s, vm.App.OSPassword.Get())
	})
	unbindPassword := vm.App.OSPassword.Bind(func(s string) {
		enableTabs(vm.App.OSUsername.Get(), s)
	})

	notification := mycontainer.NewNotification()
	unbindNotificantions := vm.App.ShowNotification.Bind(showNotification(notification))

	anchor := mycontainer.NewAnchor()
	anchor.Add(tabs, mycontainer.FillConstraint)
	margin := float32(10)
	anchor.Add(notification.Container(), mycontainer.AnchorConstraints{Top: &margin, Right: &margin})

	vm.App.Mount()

	// loads search view into its tab
	navigation.New[*viewmodel.ViewModel](search).To(vm, Search)
	// loads cache view into its tab
	navigation.New[*viewmodel.ViewModel](cache).To(vm, Cache)

	return anchor.Container, func(bool) {
		// this will never be called. It is here for completeness.
		unbindNotificantions()
		unbindSubtitles()

		unbindUsername()
		unbindPassword()
	}
}

func appAddSubtitlesSection(sections *fyne.Container, vm *viewmodel.ViewModel) func() {
	sections.Add(widget.NewLabel("OpenSubtitles.com"))
	sections.Add(canvas.NewLine(color.Gray{128}))

	usernameEntry := widget.NewEntry()
	unbindUsername := vm.App.OSUsername.Bind(usernameEntry.SetText)

	passwordEntry := widget.NewPasswordEntry()
	unbindPassword := vm.App.OSPassword.Bind(passwordEntry.SetText)

	sections.Add(container.NewHBox(
		widget.NewForm(
			widget.NewFormItem("Username", components.NewMinSizeWrapper(usernameEntry, fyne.NewSize(200, 40))),
			widget.NewFormItem("Password", components.NewMinSizeWrapper(passwordEntry, fyne.NewSize(200, 40))),
		),
		layout.NewSpacer(),
	),
	)
	bt := widget.NewButton("Change", func() {
		vm.App.SetOpenSubtitles(usernameEntry.Text, passwordEntry.Text)
	})
	bt.Importance = widget.HighImportance
	sections.Add(container.NewHBox(bt, layout.NewSpacer()))
	sections.Add(widget.NewSeparator())

	return func() {
		unbindUsername()
		unbindPassword()
	}
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

func appEnableAllTabs(tabs *container.AppTabs) {
	for k := range len(tabs.Items) {
		tabs.EnableIndex(k)
	}
	tabs.SelectIndex(0)
}

func showNotification(notification *mycontainer.NotificationContainer) func(evt app.Notify) {
	return func(evt app.Notify) {
		switch evt.Type {
		case app.NotifyError:
			notification.ShowError(evt.Message)
		case app.NotifyWarn:
			notification.ShowWarning(evt.Message)
		case app.NotifyInfo:
			notification.ShowInfo(evt.Message)
		case app.NotifySuccess:
			notification.ShowSuccess(evt.Message)
		}
	}
}

func navigate(vm *viewmodel.ViewModel, navigator *navigation.Navigator[*viewmodel.ViewModel], destination viewmodel.DownloadType) bool {
	switch destination {
	case viewmodel.DownloadSingle:
		navigator.To(vm, Download)
	case viewmodel.DownloadMultiple:
		navigator.To(vm, DownloadList)
	default:
		return false
	}
	return true
}
