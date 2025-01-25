package app

import "fmt"

type Loading struct {
	// If set, it will update the loading text.
	Text string
	// If it is true, it will show the loading indicator. If false and Text is empty it will hide.
	Show bool
}

func (Loading) Kind() string {
	return "loading"
}

type ClearCache struct{}

func (ClearCache) Kind() string {
	return "clear_cache"
}

type NotifyType int

const (
	_                        = iota
	NotifySuccess NotifyType = iota + 1
	NotifyInfo
	NotifyWarn
	NotifyError
)

type Notify struct {
	Type    NotifyType
	Message string
}

func NewNotifyInfo(msg string, args ...any) Notify {
	return Notify{
		Type:    NotifyInfo,
		Message: fmt.Sprintf(msg, args...),
	}
}

func NewNotifySuccess(msg string, args ...any) Notify {
	return Notify{
		Type:    NotifySuccess,
		Message: fmt.Sprintf(msg, args...),
	}
}

func NewNotifyWarn(msg string, args ...any) Notify {
	return Notify{
		Type:    NotifyWarn,
		Message: fmt.Sprintf(msg, args...),
	}
}

func NewNotifyError(msg string, args ...any) Notify {
	return Notify{
		Type:    NotifyError,
		Message: fmt.Sprintf(msg, args...),
	}
}

func (Notify) Kind() string {
	return "notify"
}
