package controller

import (
	"log/slog"

	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/navigator"
)

type OnExit interface {
	OnExit()
}

type OnBack interface {
	OnBack()
}

type App struct {
	view               app.AppView
	targets            map[string]app.Controller
	oldChildController app.Controller
	eventBus           app.EventBus
	repo               Repository
	secrets            app.Secrets
}

func NewApp(
	view app.AppView,
	navigations map[string]app.Controller,
	eventBus app.EventBus,
	repo Repository,
	secrets app.Secrets,
) *App {
	return &App{
		view:     view,
		targets:  navigations,
		eventBus: eventBus,
		repo:     repo,
		secrets:  secrets,
	}
}

func (a *App) OnEnter() {
	password, err := a.secrets.GetOpenSubtitles()
	if err != nil {
		a.eventBus.Publish(app.NewNotifyError("Failed to retrieve open subtitles password: %s", err))
		slog.Error("Failed to retrieve open subtitles password", "error", err)
	}
	settings, err := a.repo.LoadSettings()
	if err != nil {
		a.eventBus.Publish(app.NewNotifyError("loading settings: %s", err))
		slog.Error("Failed to load settings", "error", err)
	}

	a.view.Show(app.AppData{
		OpenSubtitles: app.OpenSubtitles{
			Username: settings.OpenSubtitles.Username,
			Password: password,
		},
	})
}

func (a *App) OnNavigation(vc navigator.To) {
	ctrl, ok := a.targets[vc.Target]
	if !ok {
		a.eventBus.Publish(app.NewNotifyError("No controller found: %s", vc.Target))
		slog.Error("No controller found", "controller", vc.Target)
		return
	}

	if a.oldChildController != nil {
		c, ok := a.oldChildController.(OnBack)
		if vc.Back && ok {
			c.OnBack()
		} else if c, ok := a.oldChildController.(OnExit); ok {
			c.OnExit()
		}
	}

	a.oldChildController = ctrl

	ctrl.OnEnter()

	a.view.EnableTabs(vc.Target == SearchNavigation)
}

func (a *App) OnExit() {
	if a.oldChildController == nil {
		return
	}

	if v, ok := a.oldChildController.(OnExit); ok {
		v.OnExit()
	}
}

func (a *App) ClearCache() {
	a.eventBus.Publish(app.ClearCache{})
}

func (v *App) ShowNotification(evt app.Notify) {
	v.view.ShowNotification(evt)
}

func (v *App) SetOpenSubtitles(username, password string) {
	err := v.secrets.SetOpenSubtitles(password)
	if err != nil {
		v.eventBus.Publish(app.NewNotifyError("Failed to set open subtitles password: %s", err))
		slog.Error("Failed to set open subtitles password", "error", err)
	}

	settings, err := v.repo.LoadSettings()
	if err != nil {
		v.eventBus.Publish(app.NewNotifyError("Failed to load settings: %s", err))
		slog.Error("Failed to load settings", "error", err)
	}

	settings.OpenSubtitles.Username = username
	err = v.repo.SaveSettings(settings)
	if err != nil {
		v.eventBus.Publish(app.NewNotifyError("Failed to save settings: %s", err))
		slog.Error("Failed to save settings", "error", err)
	}

	v.eventBus.Publish(app.NewNotifySuccess("OpenSubtitles credentials saved"))
}
