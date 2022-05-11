package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/buptczq/WinCryptSSHAgent/app"
	"github.com/buptczq/WinCryptSSHAgent/sshagent"
	"github.com/buptczq/WinCryptSSHAgent/utils"
	flags "github.com/jessevdk/go-flags"
	"github.com/kayrus/putty"
	"github.com/lxn/walk"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const agentTitle = "WinCrypt SSH Agent v1.1.9"

var applications = []app.Application{
	new(app.PubKeyView),
	new(app.WSL),
	new(app.VSock),
	new(app.Cygwin),
	new(app.NamedPipe),
	new(app.Pageant),
	new(app.XShell),
}

type Opts struct {
	Verbose          []bool `short:"v" long:"verbose" description:"Verbosity"`
	InstallHVService bool   `short:"i" description:"Install Hyper-V Guest Communication Services"`
	DisableCapi      bool   `long:"disable-capi" description:"Disable Windows Crypto API"`
}

func installService() {
	if !utils.IsAdmin() {
		err := utils.RunMeElevated()
		if err != nil {
			walk.MsgBox(nil, "Install Service Error:", err.Error(), walk.MsgBoxIconError)
		}
		return
	}

	err := winio.RunWithPrivilege(winio.SeRestorePrivilege, func() error {
		gcs, err := registry.OpenKey(registry.LOCAL_MACHINE, utils.HyperVServiceRegPath, registry.ALL_ACCESS)
		if err != nil {
			return err
		}
		defer gcs.Close()
		agentSrv, _, err := registry.CreateKey(gcs, utils.HyperVServiceGUID.String(), registry.ALL_ACCESS)
		if err != nil {
			return err
		}
		err = agentSrv.SetStringValue("ElementName", "WinCryptSSHAgent")
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		walk.MsgBox(nil, "Install Service Error:", err.Error(), walk.MsgBoxIconError)
	} else {
		walk.MsgBox(nil, "Install Service Success:", "Please reboot your computer to take effect!", walk.MsgBoxIconInformation)
	}
}

func initDebugLog() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(home, "WCSA_DEBUG.log"), os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0664)
	if err != nil {
		return
	}
	err = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(f.Fd()))
	if err != nil {
		return
	}
	err = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd()))
	if err != nil {
		return
	}
	os.Stdout = f
	os.Stderr = f
}

func main() {
	var opts Opts
	keysFiles, lerr := flags.Parse(&opts)
	if lerr != nil {
		if lperr, ok := lerr.(*flags.Error); ok {
			switch lperr.Type {
			case flags.ErrHelp:
				return
			case flags.ErrUnknown:
				log.Fatal(lerr)
			case flags.ErrTag:
				log.Fatal(lerr)
			}

			return
		} else {
			log.Fatal(lerr)
		}
	}

	switch len(opts.Verbose) {
	case 3:
		os.Setenv("WCSA_DEBUG", "1")
	}

	if os.Getenv("WCSA_DEBUG") == "1" {
		initDebugLog()
	}

	if opts.InstallHVService {
		installService()
		return
	}
	// hyper-v
	hvClient := false
	hvConn, err := utils.ConnectHyperV()
	if err == nil {
		hvConn.Close()
		hvClient = true
	}

	// agent
	var ag agent.Agent
	if hvClient {
		ag = sshagent.NewHVAgent()
	} else if opts.DisableCapi {
		ag = sshagent.NewKeyRingAgent()
	} else {
		cag := new(sshagent.CAPIAgent)
		defer cag.Close()
		defaultAgent := sshagent.NewKeyRingAgent()
		ag = sshagent.NewWrappedAgent(defaultAgent, []agent.Agent{agent.Agent(cag)})
	}

	for _, keyFile := range keysFiles {
		puttyKey, err := putty.NewFromFile(keyFile)
		if err != nil {
			continue
		}

		if puttyKey.Encryption != "none" {
			continue
		}

		privkey, err := puttyKey.ParseRawPrivateKey(nil)
		if err != nil {
			continue
		}

		ag.Add(agent.AddedKey{PrivateKey: privkey, Comment: puttyKey.Comment})
	}

	// systray
	ico, _ := walk.NewIconFromResourceId(2)
	mw, err := walk.NewMainWindow()
	if err != nil {
		log.Fatal(err)
	}

	// Create the notify icon and make sure we clean it up on exit.
	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		log.Fatal(err)
	}
	defer ni.Dispose()
	utils.SetNotifier(ni)

	// Set the icon and a tool tip text.
	if err := ni.SetIcon(ico); err != nil {
		log.Fatal(err)
	}

	title := agentTitle
	if hvClient {
		title += " (Hyper-V)"
	}

	ni.SetToolTip(title)

	ctx, cancel := context.WithCancel(context.Background()) // context
	ctx = context.WithValue(ctx, "agent", ag)
	ctx = context.WithValue(ctx, "hv", hvClient)
	server := &sshagent.Server{Agent: ag}

	// application
	wg := new(sync.WaitGroup)
	for _, v := range applications {
		v.Menu(ni)
		wg.Add(1)
		go func(application app.Application) {
			err := application.Run(ctx, server.SSHAgentHandler)
			if err != nil {
				walk.MsgBox(nil, application.AppId().String()+" Error:", err.Error(), walk.MsgBoxIconWarning)
			}
			wg.Done()
		}(v)
	}

	// show systray
	// We put an exit action into the context menu.
	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())
	exitAction := walk.NewAction()
	ni.ContextMenu().Actions().Add(exitAction)
	exitAction.SetText("E&xit")
	exitAction.Triggered().Attach(func() {
		walk.App().Exit(0)
	})

	ni.SetVisible(true)
	mw.Run() // Run the message loop.

	cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()
	select {
	case <-time.After(time.Second * 5):
	case <-done:
	}
}
