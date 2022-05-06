package utils

import "github.com/lxn/walk"

const (
	IDOK     = 1
	IDCANCEL = 2
	IDABORT  = 3
	IDRETRY  = 4
	IDIGNORE = 5
	IDYES    = 6
	IDNO     = 7
)

var ni *walk.NotifyIcon

func SetNotifier(_ni *walk.NotifyIcon) {
	ni = _ni
}

func Notify(title, message string) {
	if ni != nil {
		ni.ShowInfo(title, message)
	}
}
