package viewmodel

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2/data/binding"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/lib/values"
)

type AppService interface {
	LoadData() (AppData, error)
	SetOpenSubtitles(username, password string) error
	ClearCache() error
}

type AppData struct {
	CacheDir      string
	OpenSubtitles OpenSubtitles
}

type OpenSubtitles struct {
	Username string
	Password string
}

type App struct {
	root             *ViewModel
	service          AppService
	CacheDir         binding.String
	OSUsername       *bind.Bind[string]
	OSPassword       *bind.Bind[string]
	ShowNotification bind.Notifier[app.Notify]
	CacheCleared     bind.Notifier[bool]
	EscapeKey        bind.Notifier[func()]
}

func NewApp(service AppService) *App {
	return &App{
		service:          service,
		CacheDir:         binding.NewString(),
		OSUsername:       bind.New[string](),
		OSPassword:       bind.New[string](),
		ShowNotification: bind.NewNotifier[app.Notify](),
		CacheCleared:     bind.NewNotifier[bool](),
	}
}

func (a *App) Init() {
	data, err := a.service.LoadData()
	if err != nil {
		a.logAndPub(err, "Failed to load app data")
		return
	}

	a.CacheDir.Set(data.CacheDir)
	a.OSUsername.Set(data.OpenSubtitles.Username)
	a.OSPassword.Set(data.OpenSubtitles.Password)
}

func (a *App) SetOpenSubtitles(username, password string) {
	err := a.service.SetOpenSubtitles(username, password)
	if err != nil {
		a.logAndPub(err, "Failed to set open subtitles")
		return
	}

	a.ShowNotification.Notify(app.Notify{
		Type:    app.NotifySuccess,
		Message: "OpenSubtitles credentials saved",
	})

	a.OSUsername.Set(username)
	a.OSPassword.Set(password)
}

func (a *App) logAndPub(err error, msg string, args ...any) {
	s := msg
	if len(args) > 0 {
		m := values.ToMap(args)
		s = fmt.Sprintf("%s %s", msg, values.ToStr(m))
	}

	if err != nil {
		a.ShowNotification.Notify(app.NewNotifyError("%s: %s", s, err))
		slog.Error(msg, append(args, "error", err)...)
		return
	}

	a.ShowNotification.Notify(app.NewNotifyInfo(s))
	slog.Error(msg, args...)
}

func (a *App) ClearCache() {
	err := a.service.ClearCache()
	if err != nil {
		a.root.App.logAndPub(err, "Failed to clear cache")
		return
	}

	a.CacheCleared.Notify(true)

	a.ShowNotification.Notify(app.NewNotifyInfo("Cache cleared"))
}
