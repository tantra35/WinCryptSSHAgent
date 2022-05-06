package app

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/buptczq/WinCryptSSHAgent/utils"
	"github.com/lxn/walk"
)

type Pageant struct{}

func (*Pageant) Run(ctx context.Context, handler func(conn io.ReadWriteCloser)) error {
	debug := false
	if os.Getenv("WCSA_DEBUG") == "1" {
		debug = true
	}
	win, err := utils.NewPageant(debug)
	if err != nil {
		return err
	}
	defer win.Close()

	wg := new(sync.WaitGroup)
	for {
		conn, err := win.AcceptCtx(ctx)
		if err != nil {
			if err != io.ErrClosedPipe {
				return err
			}
			return nil
		}
		wg.Add(1)
		go func() {
			handler(conn)
			wg.Done()
		}()
	}
}

func (*Pageant) AppId() AppId {
	return APP_PAGEANT
}

func (s *Pageant) Menu(ni *walk.NotifyIcon) {
}
