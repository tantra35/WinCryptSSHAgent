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
	baseApp
	pipePath string
	running  bool
}

func (s *NamedPipe) Run(ctx context.Context, handler func(conn io.ReadWriteCloser)) error {
	var cfg = &winio.PipeConfig{}
	pipe, err := winio.ListenPipe(s.pipePath, cfg)
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

func (s *NamedPipe) Menu(ni *walk.NotifyIcon) {
	laction := walk.NewAction()
	ni.ContextMenu().Actions().Add(laction)
	laction.SetText("Show " + s.Name() + " Settings")
	laction.Triggered().Attach(func() {
		s.onClick()
	})

	lsecurecrtaction := walk.NewAction()
	ni.ContextMenu().Actions().Add(lsecurecrtaction)
	lsecurecrtaction.SetText("Show " + s.Name() + " (SecureCRT) Settings")
	lsecurecrtaction.Triggered().Attach(func() {
		s.onClickSC()
	})
}

func (s *NamedPipe) onClick() {
	if s.running {
		help := fmt.Sprintf(`set SSH_AUTH_SOCK=%s`, s.pipePath)
		if walk.MsgBox(nil, s.Name()+" (OK to copy):", help, walk.MsgBoxOKCancel) == utils.IDOK {
			utils.SetClipBoard(help)
		}
	} else {
		walk.MsgBox(nil, "Error:", s.Name()+" agent doesn't work!", walk.MsgBoxIconWarning)
	}
}

func (s *NamedPipe) onClickSC() {
	if s.running {
		help := fmt.Sprintf(`setx "VANDYKE_SSH_AUTH_SOCK" "%s"`, s.pipePath)
		if walk.MsgBox(nil, s.Name()+" (OK to copy):", help, walk.MsgBoxOKCancel) == utils.IDOK {
			utils.SetClipBoard(help)
		}
	} else {
		walk.MsgBox(nil, "Error:", s.Name()+" agent doesn't work!", walk.MsgBoxIconWarning)
	}
}
