package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/buptczq/WinCryptSSHAgent/capi"

	"github.com/Microsoft/go-winio"
	"github.com/buptczq/WinCryptSSHAgent/app"
	"github.com/buptczq/WinCryptSSHAgent/config"
	"github.com/buptczq/WinCryptSSHAgent/sshagent"
	"github.com/buptczq/WinCryptSSHAgent/utils"
	"github.com/kayrus/putty"
	"github.com/lxn/walk"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const agentTitle = "WinCrypt SSH Agent v1.1.9"

var (
	flagVerbose          int
	flagInstallHVService bool
	flagExternalAgent    []string
	flagDisableCapi      bool
	flagDisablePINCache  bool
	flagAllowMultiple    bool
	flagListSockets      bool
	flagKeys             []string
	rootCmd              *cobra.Command
)

const instanceMutexName = `Local\WinCryptSSHAgent.Mutex`

// acquireInstanceMutex tries to take ownership of the per-session instance
// mutex. allowMultiple permits running alongside the owner (the handle is
// still kept open so the process keeps a reference for its whole lifetime).
func acquireInstanceMutex(allowMultiple bool) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(instanceMutexName)
	if err != nil {
		return 0, err
	}
	h, err := windows.CreateMutex(nil, false, name)
	if h == 0 {
		return 0, err
	}
	if allowMultiple {
		return h, nil
	}
	event, waitErr := windows.WaitForSingleObject(h, 0)
	if waitErr != nil {
		windows.CloseHandle(h)
		return 0, fmt.Errorf("failed to check instance mutex: %v", waitErr)
	}
	switch event {
	case windows.WAIT_OBJECT_0:
		return h, nil
	case uint32(windows.WAIT_TIMEOUT):
		windows.CloseHandle(h)
		return 0, fmt.Errorf("another %s instance is already running (use --allow-multiple to override)", agentTitle)
	default:
		windows.CloseHandle(h)
		return 0, fmt.Errorf("failed to check instance mutex: unexpected wait event %d", event)
	}
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
	rootCmd = &cobra.Command{
		Use:   "WinCryptSSHAgent [keys.ppk...]",
		Short: agentTitle,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flagKeys = args
			return runAgent()
		},
	}
	rootCmd.Flags().CountVarP(&flagVerbose, "verbose", "v", "verbosity (-vvv enables debug log)")
	rootCmd.Flags().BoolVarP(&flagInstallHVService, "install-hv-service", "i", false,
		"Install Hyper-V Guest Communication Services")
	rootCmd.Flags().StringArrayP("agent", "a", nil,
		"External agent path (win named pipe only)")
	rootCmd.Flags().BoolVar(&flagDisableCapi, "disable-capi", false, "Disable Windows Crypto API")
	rootCmd.Flags().BoolVar(&flagDisablePINCache, "disable-pin-cache", false,
		"Clear the Smart Card PIN Cache after each operation")
	rootCmd.Flags().BoolVar(&flagAllowMultiple, "allow-multiple", false, "Allow multiple agent instances")
	rootCmd.Flags().StringArray("socket", nil, "add/override a socket: type=name|path (repeatable)")
	rootCmd.Flags().StringArray("no-socket", nil, "disable a socket by name (repeatable)")
	rootCmd.Flags().StringP("config", "c", "", "path to config.yaml (default: config.yaml next to the executable)")
	rootCmd.Flags().BoolVar(&flagListSockets, "list-sockets", false, "print resolved socket configuration and exit")

	if err := rootCmd.Execute(); err != nil {
		if err == errExitSilently {
			return
		}
		walk.MsgBox(nil, agentTitle, err.Error(), walk.MsgBoxIconError)
		log.Fatal(err)
	}
}

// errExitSilently is returned when a helper mode (--list-sockets etc.)
// already printed its output and the agent must not start.
var errExitSilently = fmt.Errorf("exit silently")

func runAgent() error {
	switch flagVerbose {
	case 3:
		os.Setenv("WCSA_DEBUG", "1")
	}

	if os.Getenv("WCSA_DEBUG") == "1" {
		initDebugLog()
	}

	if flagInstallHVService {
		installService()
		return nil
	}

	// sockets: config file + CLI overrides
	configPath, _ := rootCmd.Flags().GetString("config")
	socketFlags, _ := rootCmd.Flags().GetStringArray("socket")
	noSocketFlags, _ := rootCmd.Flags().GetStringArray("no-socket")
	sockets, err := config.Resolve(configPath, socketFlags, noSocketFlags)
	if err != nil {
		walk.MsgBox(nil, agentTitle, err.Error(), walk.MsgBoxIconError)
		return errExitSilently
	}
	if os.Getenv("WCSA_DEBUG") == "1" || flagVerbose >= 2 {
		lf, _ := os.OpenFile("wcsa-resolved.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		fmt.Fprintf(lf, "raw flags: config=%q sockets=%q no-sockets=%q\n", configPath, socketFlags, noSocketFlags)
		fmt.Fprint(lf, config.FormatSockets(sockets))
		lf.Close()
	}
	if flagListSockets {
		walk.MsgBox(nil, agentTitle, config.FormatSockets(sockets), walk.MsgBoxIconInformation)
		return errExitSilently
	}

	mutex, err := acquireInstanceMutex(flagAllowMultiple)
	if err != nil {
		walk.MsgBox(nil, agentTitle, err.Error(), walk.MsgBoxIconWarning)
		return errExitSilently
	}
	defer windows.CloseHandle(mutex)

	// hyper-v
	hvClient := false
	hvConn, err := utils.ConnectHyperV()
	if err == nil {
		hvConn.Close()
		hvClient = true
	}

	capi.SetDisablePINCache(flagDisablePINCache)

	// agent
	var ag agent.Agent
	if hvClient {
		ag = sshagent.NewHVAgent()
	} else if flagDisableCapi {
		ag = sshagent.NewKeyRingAgent()
	} else {
		cag := new(sshagent.CAPIAgent)
		defer cag.Close()
		defaultAgent := sshagent.NewKeyRingAgent()
		ag = sshagent.NewWrappedAgent(defaultAgent, []agent.Agent{agent.Agent(cag)})
	}

	for _, keyFile := range flagKeys {
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

	lexternalAgents := make([]agent.Agent, 0)
	for _, lexternalAgentPath := range flagExternalAgent {
		lpipe, lerr := os.OpenFile(lexternalAgentPath, os.O_RDWR, os.ModeNamedPipe)
		if lerr != nil {
			walk.MsgBox(nil, "Can't open pipe to external agent", err.Error(), walk.MsgBoxIconError)
			log.Fatal(lerr)
		}

		lexternalAgents = append(lexternalAgents, agent.NewClient(lpipe))
	}

	if len(lexternalAgents) > 0 {
		ag = sshagent.NewWrappedAgent(ag, lexternalAgents)
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
	apps := make([]app.App, 0, len(sockets)+1)
	apps = append(apps, new(app.PubKeyView))
	for _, sc := range sockets {
		a, err := app.NewSocket(sc)
		if err != nil {
			walk.MsgBox(nil, agentTitle, err.Error(), walk.MsgBoxIconWarning)
			continue
		}
		apps = append(apps, a)
	}

	wg := new(sync.WaitGroup)
	for _, v := range apps {
		v.Menu(ni)
		wg.Add(1)
		go func(application app.App) {
			err := application.Run(ctx, server.SSHAgentHandler)
			if err != nil {
				walk.MsgBox(nil, application.Name()+" Error:", err.Error(), walk.MsgBoxIconWarning)
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
	return nil
}
