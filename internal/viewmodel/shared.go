package viewmodel

import (
	"fmt"
	"log/slog"

	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/lib/bus"
	"github.com/quintans/torflix/internal/lib/navigation"
	"github.com/quintans/torflix/internal/lib/values"
)

type Shared struct {
	Navigate         *navigation.Navigator
	ShowNotification bind.Notifier[app.Notify]
	Publish          func(msg bus.Message)
}

func (s *Shared) Error(err error, msg string, args ...any) {
	if err == nil {
		return
	}

	mm := msg
	if len(args) > 0 {
		m := values.ToMap(args)
		mm = fmt.Sprintf("%s %s", msg, values.ToStr(m))
	}

	s.ShowNotification.Notify(app.NewNotifyError("%s: %s", mm, err))
	slog.Error(msg, append(args, "error", err)...)
}

func (s *Shared) Warn(msg string, args ...any) {
	s.ShowNotification.Notify(app.NewNotifyWarn(fmt.Sprintf(msg, args...)))
}

func (s *Shared) Info(msg string, args ...any) {
	s.ShowNotification.Notify(app.NewNotifyInfo(fmt.Sprintf(msg, args...)))
}

func (s *Shared) Success(msg string, args ...any) {
	s.ShowNotification.Notify(app.NewNotifySuccess(fmt.Sprintf(msg, args...)))
}
