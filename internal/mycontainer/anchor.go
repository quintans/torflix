package mycontainer

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

var zero float32 = 0

var (
	FillConstraint   = AnchorConstraints{Top: &zero, Bottom: &zero, Left: &zero, Right: &zero}
	CenterConstraint = AnchorConstraints{}
)

type AnchorConstraints struct {
	Top, Bottom, Left, Right *float32
}

type AnchorLayout struct {
	constraints map[fyne.CanvasObject]AnchorConstraints
}

func NewAnchorLayout() *AnchorLayout {
	return &AnchorLayout{
		constraints: make(map[fyne.CanvasObject]AnchorConstraints),
	}
}

func (a *AnchorLayout) Add(obj fyne.CanvasObject, constraints AnchorConstraints) {
	a.constraints[obj] = constraints
}

func (a *AnchorLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		constraints, ok := a.constraints[obj]
		if !ok || (constraints == AnchorConstraints{}) {
			// Center the object if no constraints are provided
			obj.Resize(obj.MinSize())
			obj.Move(fyne.NewPos((size.Width-obj.Size().Width)/2, (size.Height-obj.Size().Height)/2))
			continue
		}

		var x, y, width, height float32

		if constraints.Left != nil && constraints.Right != nil {
			x = *constraints.Left
			width = size.Width - *constraints.Left - *constraints.Right
		} else if constraints.Left != nil {
			x = *constraints.Left
			width = obj.MinSize().Width
		} else if constraints.Right != nil {
			x = size.Width - obj.MinSize().Width - *constraints.Right
			width = obj.MinSize().Width
		} else {
			x = (size.Width - obj.MinSize().Width) / 2
			width = obj.MinSize().Width
		}

		if constraints.Top != nil && constraints.Bottom != nil {
			y = *constraints.Top
			height = size.Height - *constraints.Top - *constraints.Bottom
		} else if constraints.Top != nil {
			y = *constraints.Top
			height = obj.MinSize().Height
		} else if constraints.Bottom != nil {
			y = size.Height - obj.MinSize().Height - *constraints.Bottom
			height = obj.MinSize().Height
		} else {
			y = (size.Height - obj.MinSize().Height) / 2
			height = obj.MinSize().Height
		}

		obj.Resize(fyne.NewSize(width, height))
		obj.Move(fyne.NewPos(x, y))
	}
}

func (a *AnchorLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minWidth, minHeight := 0, 0
	for _, obj := range objects {
		minWidth = int(fyne.Max(float32(minWidth), obj.MinSize().Width))
		minHeight = int(fyne.Max(float32(minHeight), obj.MinSize().Height))
	}
	return fyne.NewSize(float32(minWidth), float32(minHeight))
}

type Anchor struct {
	Layout    *AnchorLayout
	Container *fyne.Container
}

func NewAnchor() *Anchor {
	layout := NewAnchorLayout()
	return &Anchor{
		Layout:    layout,
		Container: container.New(layout),
	}
}

func (a *Anchor) Add(obj fyne.CanvasObject, constraints AnchorConstraints) {
	a.Layout.Add(obj, constraints)
	a.Container.Add(obj)
}
