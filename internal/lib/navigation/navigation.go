package navigation

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
)

type (
	Navigate    func(screen fyne.CanvasObject, close func(bool))
	ViewFactory func() (screen fyne.CanvasObject, close func(bool))
)

type navigation struct {
	view  ViewFactory
	close func(bool)
}

// Navigator manages screen navigation with a simple stack
type Navigator struct {
	stack     []navigation
	container *fyne.Container
	Factory   func(any) ViewFactory
}

func New(container *fyne.Container) *Navigator {
	return &Navigator{container: container}
}

func (n *Navigator) To(to any) {
	if n.Factory == nil {
		slog.Error("Navigator Factory is nil")
		return
	}

	view := n.Factory(to)
	if view == nil {
		slog.Error("Navigator Factory returned nil view", "to", fmt.Sprintf("%T", to))
		return
	}

	screen, close := view()

	// Push current screen if any
	if len(n.stack) > 0 {
		last := n.stack[len(n.stack)-1]
		if last.close != nil {
			last.close(false) // Notify the previous screen to clean up on forward navigation
		}
	}

	next := navigation{
		view:  view,
		close: close,
	}
	n.stack = append(n.stack, next)

	n.container.Objects = []fyne.CanvasObject{screen}
	fyne.Do(n.container.Refresh)
}

func (n *Navigator) Reset(to any) {
	n.stack = nil
	n.To(to)
}

func (n *Navigator) Back() {
	if len(n.stack) == 0 {
		return
	}

	// Pop last screen
	last := n.stack[len(n.stack)-1]
	n.stack = n.stack[:len(n.stack)-1]

	if last.close != nil {
		last.close(true) // Notify the previous screen to clean up on back
	}

	last = n.stack[len(n.stack)-1]
	view := last.view
	screen, close := view()
	last.close = close // Update close function for the previous screen

	n.container.Objects = []fyne.CanvasObject{screen}
	n.container.Refresh()
}
