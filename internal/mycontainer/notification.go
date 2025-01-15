package mycontainer

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"github.com/quintans/torflix/internal/components"
)

type options struct {
	timeout time.Duration
}

type Option func(*options)

func Timeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

type NotificationContainer struct {
	container *fyne.Container
}

func NewNotification() *NotificationContainer {
	return &NotificationContainer{
		container: container.NewVBox(),
	}
}

var (
	bgcSuccess = color.RGBA{0, 255, 255, 192} // Cyan
	fgcSuccess = color.Black
	bgcInfo    = color.RGBA{255, 255, 0, 192} // Yellow
	fgcInfo    = color.Black
	bgcWarning = color.RGBA{255, 165, 0, 192} // Orange
	fgcWarning = color.Black
	bgcError   = color.RGBA{255, 0, 0, 192} // Red
	fgcError   = color.White
)

func (nc *NotificationContainer) ShowSuccess(message string, opts ...Option) {
	o := options{timeout: 3 * time.Second}
	for _, opt := range opts {
		opt(&o)
	}
	nc.show("Success", message, bgcSuccess, fgcSuccess, o)
}

func (nc *NotificationContainer) ShowInfo(message string, opts ...Option) {
	o := options{timeout: 3 * time.Second}
	for _, opt := range opts {
		opt(&o)
	}
	nc.show("Information", message, bgcInfo, fgcInfo, o)
}

func (nc *NotificationContainer) ShowWarning(message string, opts ...Option) {
	o := options{timeout: 3 * time.Second}
	for _, opt := range opts {
		opt(&o)
	}
	nc.show("Warning", message, bgcWarning, fgcWarning, o)
}

func (nc *NotificationContainer) ShowError(message string, opts ...Option) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	nc.show("Error", message, bgcError, fgcError, o)
}

func (nc *NotificationContainer) show(title, message string, bgColor, fgColor color.Color, opts options) {
	titleLabel := newText(title, fgColor, fyne.TextStyle{Bold: true})
	labelX := newText("x", fgColor, fyne.TextStyle{Bold: true})
	messageLabel := newText(message, fgColor, fyne.TextStyle{})

	tap := NewTappable(labelX)
	var notification fyne.CanvasObject = container.NewBorder(
		container.NewHBox(titleLabel, layout.NewSpacer(), tap),
		nil,
		nil,
		nil,
		messageLabel,
	)

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 5

	notificationContainer := container.NewStack(
		bg,
		notification,
	)
	nc.container.Add(notificationContainer)
	nc.container.Refresh()

	tap.OnTapped = func() {
		nc.container.Remove(notificationContainer)
		nc.container.Refresh()
	}

	if opts.timeout > 0 {
		go func() {
			time.Sleep(opts.timeout)
			nc.container.Remove(notificationContainer)
			nc.container.Refresh()
		}()
	}
}

func (nc *NotificationContainer) Container() *fyne.Container {
	return nc.container
}

func newText(txt string, color color.Color, style fyne.TextStyle) *fyne.Container {
	label := components.NewCustomLabel(txt, color)
	label.WrapWidth = 300
	label.TextStyle = style
	label.Alignment = fyne.TextAlignLeading

	return container.NewPadded(label)
}
