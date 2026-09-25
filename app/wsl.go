package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/buptczq/WinCryptSSHAgent/utils"
	"github.com/lxn/walk"
)

type WSL struct {
	baseApp
	sockName string // file name or absolute path of the unix socket
	running  bool
	help     string
}

func listenUnixSock(filename string) (string, net.Listener, error) {
	var path string
	if filepath.IsAbs(filename) {
		path = filename
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", nil, err
		}
		path = filepath.Join(home, filename)
	}
	os.Remove(path)
	l, err := net.Listen("unix", path)
	return path, l, err
}

func winPath2Unix(path string) string {
	volumeName := filepath.VolumeName(path)
	vnl := len(volumeName)
	fileName := path[vnl:]
	if vnl == 2 {
		return "/mnt/" + strings.ToLower(string(volumeName[0])) + filepath.ToSlash(fileName)
	} else {
		return filepath.ToSlash(path)
	}
}

func (s *WSL) Run(ctx context.Context, handler func(conn io.ReadWriteCloser)) error {
	fallback := false
	// try to listen unix sock (Win10 1803)
	path, l, err := listenUnixSock(s.sockName)
	if err != nil {
		// fallback to raw tcp
		l, err = net.Listen("tcp", "localhost:0")
		fallback = true
		if err != nil {
			return err
		}
	}
	defer l.Close()

	s.running = true
	if !fallback {
		s.help = fmt.Sprintf("export SSH_AUTH_SOCK=" + winPath2Unix(path))
	} else {
		s.help = fmt.Sprintf("socat UNIX-LISTEN:/tmp/ssh-capi-agent.sock,reuseaddr,fork TCP:localhost:%d &\n", l.Addr().(*net.TCPAddr).Port)
		s.help += "export SSH_AUTH_SOCK=/tmp/ssh-capi-agent.sock"
	}
	// close the listener on context cancellation to unblock Accept
	go func() {
		<-ctx.Done()
		l.Close()
	}()
	// loop
	wg := new(sync.WaitGroup)
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
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

func (s *WSL) Menu(ni *walk.NotifyIcon) {
	laction := walk.NewAction()
	ni.ContextMenu().Actions().Add(laction)
	laction.SetText("Show " + s.Name() + " Settings")
	laction.Triggered().Attach(func() {
		s.onClick()
	})
}

func (s *WSL) onClick() {
	if s.running {
		if walk.MsgBox(nil, s.Name()+" (OK to copy):", s.help, walk.MsgBoxOKCancel) == utils.IDOK {
			utils.SetClipBoard(s.help)
		}
	} else {
		walk.MsgBox(nil, "Error:", s.Name()+" agent doesn't work!", walk.MsgBoxIconWarning)
	}
}
