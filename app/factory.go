package app

import (
	"context"
	"fmt"
	"io"

	"github.com/buptczq/WinCryptSSHAgent/config"
	"github.com/lxn/walk"
)

// App is a configurable agent listener instance. It replaces the old
// Application interface: AppId becomes a per-instance string Name and
// Run receives the resolved socket config.
type App interface {
	Name() string
	Run(ctx context.Context, handler func(conn io.ReadWriteCloser)) error
	Menu(ni *walk.NotifyIcon)
}

// NewSocket builds an App instance for a config entry.
func NewSocket(sc config.Socket) (App, error) {
	switch sc.Type {
	case config.TypeNamedPipe:
		return &NamedPipe{baseApp: baseApp{sc.Name}, pipePath: socketPath(sc, NAMED_PIPE)}, nil
	case config.TypeCygwin:
		return &Cygwin{baseApp: baseApp{sc.Name}, sockFile: socketPath(sc, CYGWIN_SOCK)}, nil
	case config.TypeWSL:
		return &WSL{baseApp: baseApp{sc.Name}, sockName: socketPath(sc, WSL_SOCK)}, nil
	case config.TypePageant:
		return &Pageant{baseApp: baseApp{sc.Name}}, nil
	case config.TypeXShell:
		return &XShell{baseApp: baseApp{sc.Name}}, nil
	case config.TypeVSock:
		return &VSock{baseApp: baseApp{sc.Name}}, nil
	default:
		return nil, fmt.Errorf("socket %q: unsupported type %q", sc.Name, sc.Type)
	}
}

// socketPath returns the configured path or the built-in default when
// the config entry leaves it empty.
func socketPath(sc config.Socket, def string) string {
	if sc.Path == "" || sc.Path == "none" {
		return def
	}
	return sc.Path
}

type baseApp struct {
	name string
}

func (b baseApp) Name() string { return b.name }
