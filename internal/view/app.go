package view

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/components"
	"github.com/quintans/torflix/internal/mycontainer"
)

type ShowViewer interface {
	ShowView(v fyne.CanvasObject)
}

type AppController interface {
	ClearCache()
	SetOpenSubtitles(username, password string)
}

type App struct {
	window       fyne.Window
	container    *fyne.Container
	controller   AppController
	clear        *widget.Button
	notification *mycontainer.NotificationContainer
	tabs         *container.AppTabs
	loading      *dialog.CustomDialog
	loadingText  *widget.Label
}

func NewApp(w fyne.Window) *App {
	inifiniteProgress := widget.NewProgressBarInfinite()
	inifiniteProgress.Start()
	// Custom content for the dialog
	loadingText := widget.NewLabel("")
	customContent := container.NewVBox(
		loadingText,
		inifiniteProgress,
	)

	// Create the dialog
	dialog := dialog.NewCustomWithoutButtons("Working...", customContent, w)

	return &App{
		window:       w,
		container:    container.NewBorder(nil, nil, nil, nil),
		notification: mycontainer.NewNotification(),
		loading:      dialog,
		loadingText:  loadingText,
	}
}

func (v *App) SetController(controller AppController) {
	v.controller = controller
}

func (v *App) Show(data app.AppData) {
	sections := container.NewVBox()

	v.addCacheSection(sections, data)
	v.addSubtitlesSection(sections, data)

	v.tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Search", theme.SearchIcon(), v.container),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), sections),
	)
	v.tabs.SetTabLocation(container.TabLocationLeading)

	anchor := mycontainer.NewAnchor()
	anchor.Add(v.tabs, mycontainer.FillConstraint)
	margin := float32(10)
	anchor.Add(v.notification.Container(), mycontainer.AnchorConstraints{Top: &margin, Right: &margin})

	v.window.SetContent(anchor.Container)
}

func (v *App) addCacheSection(sections *fyne.Container, data app.AppData) {
	v.clear = widget.NewButton("Clear Cache", func() {
		v.controller.ClearCache()
	})
	v.clear.Importance = widget.WarningImportance

	sections.Add(widget.NewLabel("Cache"))
	sections.Add(canvas.NewLine(color.Gray{128}))
	sections.Add(container.NewHBox(
		widget.NewForm(
			widget.NewFormItem("Directory", widget.NewLabel(data.CacheDir)),
		),
		layout.NewSpacer(),
	))
	sections.Add(container.NewHBox(v.clear, layout.NewSpacer()))
	sections.Add(widget.NewSeparator())
}

func (v *App) addSubtitlesSection(sections *fyne.Container, data app.AppData) {
	sections.Add(widget.NewLabel("OpenSubtitles.com"))
	sections.Add(canvas.NewLine(color.Gray{128}))

	usernameEntry := widget.NewEntry()
	usernameEntry.SetText(data.OpenSubtitles.Username)
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetText(data.OpenSubtitles.Password)

	sections.Add(container.NewHBox(
		widget.NewForm(
			widget.NewFormItem("Username", components.NewMinSizeWrapper(usernameEntry, fyne.NewSize(200, 40))),
			widget.NewFormItem("Password", components.NewMinSizeWrapper(passwordEntry, fyne.NewSize(200, 40))),
		),
		layout.NewSpacer(),
	),
	)
	bt := widget.NewButton("Change", func() {
		v.controller.SetOpenSubtitles(usernameEntry.Text, passwordEntry.Text)
	})
	bt.Importance = widget.HighImportance
	sections.Add(container.NewHBox(bt, layout.NewSpacer()))
	sections.Add(widget.NewSeparator())
}

func (v *App) DisableAllTabsButSettings() {
	v.tabs.SelectIndex(len(v.tabs.Items) - 1)
	v.EnableTabs(false)
}

func (v *App) EnableTabs(enable bool) {
	cur := v.tabs.SelectedIndex()

	for k := range len(v.tabs.Items) {
		if k == cur {
			continue
		}

		if enable {
			v.tabs.EnableIndex(k)
		} else {
			v.tabs.DisableIndex(k)
		}
	}
}

func (v *App) ShowView(c fyne.CanvasObject) {
	v.container.RemoveAll()
	v.container.Add(c)
	v.container.Refresh()
}

func (v *App) ShowNotification(evt app.Notify) {
	switch evt.Type {
	case app.NotifyError:
		v.notification.ShowError(evt.Message)
	case app.NotifyWarn:
		v.notification.ShowWarning(evt.Message)
	case app.NotifyInfo:
		v.notification.ShowInfo(evt.Message)
	case app.NotifySuccess:
		v.notification.ShowSuccess(evt.Message)
	}
}

func (v *App) Loading(msg app.Loading) {
	if msg.Text != "" {
		v.loadingText.SetText(msg.Text)
	}

	if msg.Show {
		v.loading.Show()
		return
	}

	if msg.Text == "" && !msg.Show {
		v.loading.Hide()
		v.loadingText.SetText("")
	}
}
