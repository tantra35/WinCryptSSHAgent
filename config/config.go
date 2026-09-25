package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Socket describes a single agent listener from config.yaml / CLI.
type Socket struct {
	Type   string   `mapstructure:"type"`
	Name   string   `mapstructure:"name"`
	Path   string   `mapstructure:"path"`
	Keys   []string `mapstructure:"keys"` // reserved, not used yet
	Notify *bool    `mapstructure:"notify"`
}

// NotifyOn reports whether tray notifications are enabled for this socket.
// Missing field means enabled.
func (s Socket) NotifyOn() bool {
	return s.Notify == nil || *s.Notify
}

type Config struct {
	Sockets []Socket `mapstructure:"sockets"`
}

// known socket transport types
const (
	TypeNamedPipe = "named-pipe"
	TypeCygwin    = "cygwin"
	TypeWSL       = "wsl"
	TypePageant   = "pageant"
	TypeXShell    = "xshell"
	TypeVSock     = "vsock"
)

var knownTypes = map[string]bool{
	TypeNamedPipe: true,
	TypeCygwin:    true,
	TypeWSL:       true,
	TypePageant:   true,
	TypeXShell:    true,
	TypeVSock:     true,
}

// DefaultSockets mirrors the hard-wired listener set of the original app,
// so a missing config file keeps the behavior of previous releases.
func DefaultSockets() []Socket {
	return []Socket{
		{Type: TypeWSL, Name: "wsl"},
		{Type: TypeVSock, Name: "vsock"},
		{Type: TypeCygwin, Name: "cygwin"},
		{Type: TypeNamedPipe, Name: "win32openssh"},
		{Type: TypePageant, Name: "pageant"},
		{Type: TypeXShell, Name: "xshell"},
	}
}

// LoadFile reads and parses the config file.
func LoadFile(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ParseSocketFlag parses a --socket CLI entry.
//
// Preferred form: type=name|path (the pipe is an explicit, unambiguous
// separator — Windows paths are full of colons: C:\, \\.\pipe\).
//
// Colon form type=name:path is best-effort: a colon followed by \ or /
// (UNC continuation or drive-letter colon) is never treated as the
// separator, so only relative unix-style paths parse reliably with it.
func ParseSocketFlag(s string) (Socket, error) {
	typePart, rest, found := strings.Cut(s, "=")
	if !found {
		return Socket{}, fmt.Errorf("--socket %q: expected type=name|path or type=name:path", s)
	}
	name := rest
	path := ""
	if strings.Contains(rest, "|") {
		name, path, _ = strings.Cut(rest, "|")
	} else {
		for i := 0; i+1 < len(rest); i++ {
			if rest[i] != ':' {
				continue
			}
			next := rest[i+1]
			isDriveColon := i > 0 && isLetter(rest[i-1]) && (next == '\\' || next == '/')
			isUNCContinuation := next == '\\' || next == '/'
			if isDriveColon || isUNCContinuation {
				continue
			}
			name = rest[:i]
			path = rest[i+1:]
			break
		}
	}
	sc := Socket{Type: typePart, Name: name, Path: path}
	if err := validate(sc, "cli"); err != nil {
		return Socket{}, err
	}
	return sc, nil
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// Merge applies CLI additions (--socket) and removals (--no-socket)
// on top of the file/default socket list.
func Merge(base []Socket, add []Socket, remove []string) []Socket {
	out := make([]Socket, 0, len(base)+len(add))
	for _, sc := range base {
		dropped := false
		for _, name := range remove {
			if sc.Name == name {
				dropped = true
				break
			}
		}
		if !dropped {
			out = append(out, sc)
		}
	}
	for _, add := range add {
		// a CLI entry replaces a file entry with the same name
		replaced := false
		for i := range out {
			if out[i].Name == add.Name {
				out[i] = add
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, add)
		}
	}
	return out
}

// Validate checks a resolved socket list for unknown types, duplicate
// names and paths that are required but missing.
func Validate(sockets []Socket) error {
	seenName := map[string]bool{}
	seenPath := map[string]bool{}
	for _, sc := range sockets {
		if err := validate(sc, "config"); err != nil {
			return err
		}
		if seenName[sc.Name] {
			return fmt.Errorf("duplicate socket name %q", sc.Name)
		}
		seenName[sc.Name] = true
		if sc.Path != "" && sc.Path != "none" {
			if seenPath[sc.Path] {
				return fmt.Errorf("socket %q: path %q is used by another socket", sc.Name, sc.Path)
			}
			seenPath[sc.Path] = true
		}
	}
	return nil
}

func validate(sc Socket, src string) error {
	if !knownTypes[sc.Type] {
		return fmt.Errorf("%s: socket %q: unknown type %q (known: %s)",
			src, sc.Name, sc.Type, strings.Join(knownTypesSlice(), ", "))
	}
	if sc.Name == "" {
		return fmt.Errorf("%s: socket of type %q has empty name", src, sc.Type)
	}
	switch sc.Type {
	case TypeNamedPipe, TypeCygwin, TypeWSL:
		// path may be empty -> app falls back to the built-in default
	case TypePageant, TypeXShell, TypeVSock:
		if sc.Path != "" && sc.Path != "none" {
			return fmt.Errorf("%s: socket %q: type %q does not use path", src, sc.Name, sc.Type)
		}
	}
	return nil
}

func knownTypesSlice() []string {
	return []string{TypeNamedPipe, TypeCygwin, TypeWSL, TypePageant, TypeXShell, TypeVSock}
}
