// Navigator manages screen navigation with a simple stack
// It uses a factory function to create views on demand
// Each view is responsible for creating its own content and handling cleanup when navigated away from
// This allows for lazy loading of views and better resource management
// The navigator maintains a stack of views to allow for back navigation
// Each view can implement a close function to handle cleanup when navigated away from
// The navigator ensures that the close function is called appropriately on forward and backward navigation
// The navigator also provides a reset function to clear the stack and navigate to a new view
//
// WARNING: Unfortunately, fyne internally caches widgets and does not fully release resources
// when they are removed from the container. This can lead to increased memory usage over time
// if many different views are created and destroyed. To mitigate this, views should implement
// proper cleanup in their close functions, and the navigator should be used judiciously to avoid
// excessive view creation and destruction.

package navigation

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"github.com/quintans/torflix/internal/lib/ds"
)

type (
	Navigate    func(screen fyne.CanvasObject, close func(bool))
	ViewFactory interface {
		Create() (screen fyne.CanvasObject, close func(bool))
	}
)

type navigation struct {
	view  ViewFactory
	close func(bool)
}

// Navigator manages screen navigation with a simple stack
type Navigator struct {
	stack     *ds.Stack[*navigation]
	container *fyne.Container
	factory   func(any) ViewFactory
}

func New(container *fyne.Container, factory func(any) ViewFactory) *Navigator {
	return &Navigator{
		stack:     ds.NewStack[*navigation](),
		container: container,
		factory:   factory,
	}
}

func (n *Navigator) To(to any) {
	if n.factory == nil {
		slog.Error("Navigator Factory is nil")
		return
	}

	view := n.factory(to)
	if view == nil {
		slog.Error("Navigator Factory returned nil view", "to", fmt.Sprintf("%T", to))
		return
	}

	screen, close := view.Create()

	last, _ := n.stack.Peek()
	// close current screen if any
	if last != nil && last.close != nil {
		last.close(false) // Notify the previous screen to clean up on forward navigation
	}

	next := &navigation{
		view:  view,
		close: close,
	}
	n.stack.Push(next)

	n.container.Objects = []fyne.CanvasObject{screen}

	go fyne.Do(n.container.Refresh)
}

func (n *Navigator) Reset(to any) {
	n.stack = nil
	n.To(to)
}

func (n *Navigator) Back() {
	if n.stack.IsEmpty() {
		return
	}

	// Pop last screen
	last, _ := n.stack.Pop()
	if last.close != nil {
		last.close(true) // Notify the previous screen to clean up on back
	}

	last, _ = n.stack.Peek()
	if last == nil {
		slog.Warn("Navigator stack is empty after pop")
		n.container.RemoveAll()
		go fyne.Do(n.container.Refresh)
		return
	}

	view := last.view
	screen, close := view.Create()
	last.close = close // Update close function for the previous screen

	n.container.Objects = []fyne.CanvasObject{screen}

	go fyne.Do(n.container.Refresh)
}
