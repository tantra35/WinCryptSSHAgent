package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Resolve builds the final socket list. Priority: CLI additions/removals
// override entries from config.yaml; a missing config file falls back to
// the built-in defaults. configPath is the --config flag value ("" means
// config.yaml next to the executable) and must exist when non-empty.
func Resolve(configPath string, add []string, remove []string) ([]Socket, error) {
	base := DefaultSockets()
	explicit := configPath != ""
	if !explicit {
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("can't locate executable dir: %w", err)
		}
		configPath = filepath.Join(filepath.Dir(exe), "config.yaml")
	}
	if _, err := os.Stat(configPath); err == nil {
		cfg, err := LoadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("config %s: %w", configPath, err)
		}
		if len(cfg.Sockets) > 0 {
			base = cfg.Sockets
		}
	} else if explicit {
		// explicit --config path must exist
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	addSockets := make([]Socket, 0, len(add))
	for _, raw := range add {
		sc, err := ParseSocketFlag(raw)
		if err != nil {
			return nil, err
		}
		addSockets = append(addSockets, sc)
	}
	sockets := Merge(base, addSockets, remove)
	if err := Validate(sockets); err != nil {
		return nil, err
	}
	return sockets, nil
}

// FormatSockets renders a resolved socket list as TSV text.
func FormatSockets(sockets []Socket) string {
	out := "type\tname\tpath\tnotify\n"
	for _, sc := range sockets {
		out += fmt.Sprintf("%s\t%s\t%s\t%v\n", sc.Type, sc.Name, sc.Path, sc.NotifyOn())
	}
	return out
}
