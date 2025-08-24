package viewmodel

import (
	"fmt"
	"log/slog"

	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/lib/values"
	"github.com/quintans/torflix/internal/model"
)

type AppService interface {
	LoadData() (model.AppData, error)
	SetOpenSubtitles(username, password string) error
}

type App struct {
	root             *ViewModel
	service          AppService
	SelectedTab      int
	OSUsername       *bind.Bind[string]
	OSPassword       *bind.Bind[string]
	ShowNotification bind.Notifier[app.Notify]
	EscapeKey        bind.Notifier[func()]
}

func NewApp(service AppService) *App {
	return &App{
		service:          service,
		OSUsername:       bind.New[string](),
		OSPassword:       bind.New[string](),
		ShowNotification: bind.NewNotifier[app.Notify](),
	}
}

func (a *App) Mount() {
	data, err := a.service.LoadData()
	if err != nil {
		a.logAndPub(err, "Failed to load app data")
		return
	}

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
