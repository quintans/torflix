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
	animations := make([]func(float32), 0)
	titleLabel, ani := newText(title, fgColor, fyne.TextStyle{Bold: true})
	animations = append(animations, ani)
	labelX, ani := newText("x", fgColor, fyne.TextStyle{Bold: true})
	animations = append(animations, ani)
	messageLabel, ani := newText(message, fgColor, fyne.TextStyle{})
	animations = append(animations, ani)

	tap := NewTappable(labelX)
	var cardBody fyne.CanvasObject = container.NewBorder(
		container.NewHBox(titleLabel, layout.NewSpacer(), tap),
		nil,
		nil,
		nil,
		messageLabel,
	)

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 5
	animations = append(animations, rectFadeAnimation(bg))

	card := container.NewStack(
		bg,
		cardBody,
	)
	nc.container.Add(card)
	nc.container.Refresh()

	tap.OnTapped = func() {
		nc.container.Remove(card)
		nc.container.Refresh()
	}

	if opts.timeout > 0 {
		go func() {
			time.Sleep(opts.timeout - 300*time.Millisecond)
			done := make(chan struct{})
			fyne.NewAnimation(300*time.Millisecond, func(p float32) {
				for _, a := range animations {
					a(p)
				}
				if p == 1 {
					close(done)
				}
			}).Start()
			<-done
			nc.container.Remove(card)
			nc.container.Refresh()
		}()
	}
}

func (nc *NotificationContainer) Container() *fyne.Container {
	return nc.container
}

func newText(txt string, color color.Color, style fyne.TextStyle) (*fyne.Container, func(done float32)) {
	label := components.NewCustomLabel(txt, color)
	label.WrapWidth = 300
	label.TextStyle = style
	label.Alignment = fyne.TextAlignLeading

	return container.NewPadded(label), labelFadeAnimation(label)
}

func rectFadeAnimation(rect *canvas.Rectangle) func(done float32) {
	r, g, b, a := rect.FillColor.RGBA()
	return func(done float32) {
		alpha := uint8(float32(uint8(a)) - float32(uint8(a))*done)
		rect.FillColor = color.RGBA{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
			A: alpha,
		}
	}
}

func labelFadeAnimation(label *components.CustomLabel) func(done float32) {
	r, g, b, a := label.Color.RGBA()
	return func(done float32) {
		alpha := uint8(float32(uint8(a)) - float32(uint8(a))*done)
		label.Color = color.RGBA{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
			A: alpha,
		}
	}
}
