package navigation

import (
	"fyne.io/fyne/v2"
)

type (
	Navigate     func(screen fyne.CanvasObject, close func(bool))
	View[VM any] func(vm VM, navigator *Navigator[VM]) (screen fyne.CanvasObject, close func(bool))
)

type navigation[VM any] struct {
	view  View[VM]
	close func(bool)
}

// Navigator manages screen navigation with a simple stack
type Navigator[VM any] struct {
	stack     []navigation[VM]
	container *fyne.Container
}

func New[VM any](container *fyne.Container) *Navigator[VM] {
	return &Navigator[VM]{container: container}
}

func (n *Navigator[VM]) To(vm VM, view View[VM]) {
	screen, close := view(vm, n)

	// Push current screen if any
	if len(n.stack) > 0 {
		last := n.stack[len(n.stack)-1]
		last.close(false) // Notify the previous screen to clean up on forward navigation
	}

	next := navigation[VM]{
		view:  view,
		close: close,
	}
	n.stack = append(n.stack, next)

	n.container.Objects = []fyne.CanvasObject{screen}
	n.container.Refresh()
}

func (n *Navigator[VM]) Reset(vm VM, view View[VM]) {
	n.stack = nil
	n.To(vm, view)
}

func (n *Navigator[VM]) Back(vm VM) {
	if len(n.stack) == 0 {
		return
	}

	// Pop last screen
	last := n.stack[len(n.stack)-1]
	n.stack = n.stack[:len(n.stack)-1]

	last.close(true) // Notify the previous screen to clean up on back
	screen, close := last.view(vm, n)

	last = n.stack[len(n.stack)-1]
	last.close = close // Update close function for the previous screen

	n.container.Objects = []fyne.CanvasObject{screen}
	n.container.Refresh()
}
