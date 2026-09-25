package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/Microsoft/go-winio"
	"github.com/buptczq/WinCryptSSHAgent/utils"
	"github.com/lxn/walk"
)

type NamedPipe struct {
	running bool
}

func (s *NamedPipe) Run(ctx context.Context, handler func(conn io.ReadWriteCloser)) error {
	var cfg = &winio.PipeConfig{}
	pipe, err := winio.ListenPipe(NAMED_PIPE, cfg)
	if err != nil {
		return err
	}

	s.running = true
	defer pipe.Close()

	wg := new(sync.WaitGroup)
	// close the listener on context cancellation to unblock Accept
	go func() {
		<-ctx.Done()
		pipe.Close()
	}()
	// loop
	for {
		conn, err := pipe.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || err == winio.ErrPipeListenerClosed {
				wg.Wait()
				return nil
			}
			return err
		}
		wg.Add(1)
		go func() {
			handler(conn)
			wg.Done()
		}()
	}
}

func (*NamedPipe) AppId() AppId {
	return APP_WINSSH
}

func (s *NamedPipe) Menu(ni *walk.NotifyIcon) {
	laction := walk.NewAction()
	ni.ContextMenu().Actions().Add(laction)
	laction.SetText("Show " + s.AppId().String() + " Settings")
	laction.Triggered().Attach(func() {
		s.onClick()
	})

	app := AppId(APP_SECURECRT)
	lsecurecrtaction := walk.NewAction()
	ni.ContextMenu().Actions().Add(lsecurecrtaction)
	lsecurecrtaction.SetText("Show " + app.String() + " Settings")
	lsecurecrtaction.Triggered().Attach(func() {
		s.onClickSC()
	})
}

func (s *NamedPipe) onClick() {
	if s.running {
		help := fmt.Sprintf(`set SSH_AUTH_SOCK=%s`, NAMED_PIPE)
		if walk.MsgBox(nil, s.AppId().FullName()+" (OK to copy):", help, walk.MsgBoxOKCancel) == utils.IDOK {
			utils.SetClipBoard(help)
		}
	} else {
		walk.MsgBox(nil, "Error:", s.AppId().String()+" agent doesn't work!", walk.MsgBoxIconWarning)
	}
}

func (s *NamedPipe) onClickSC() {
	if s.running {
		help := fmt.Sprintf(`setx "VANDYKE_SSH_AUTH_SOCK" "%s"`, NAMED_PIPE)
		if walk.MsgBox(nil, s.AppId().FullName()+" (OK to copy):", help, walk.MsgBoxOKCancel) == utils.IDOK {
			utils.SetClipBoard(help)
		}
	} else {
		walk.MsgBox(nil, "Error:", s.AppId().String()+" agent doesn't work!", walk.MsgBoxIconWarning)
	}
}
