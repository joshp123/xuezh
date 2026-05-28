package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var hostConfigPath = "/etc/xuezh/config.toml"

type ConfigConflictError struct {
	HostPath string
	UserPath string
}

func (e ConfigConflictError) Error() string {
	return fmt.Sprintf("xuezh config conflict: both %s and %s exist", e.HostPath, e.UserPath)
}

func configPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "xuezh", "config.toml"), nil
}

func ConfigPath() (string, error) {
	return configPath()
}

func Load() (map[string]any, error) {
	userPath, err := configPath()
	if err != nil {
		return nil, err
	}
	hostExists, err := fileExists(hostConfigPath)
	if err != nil {
		return nil, err
	}
	userExists, err := fileExists(userPath)
	if err != nil {
		return nil, err
	}
	if hostExists && userExists {
		return nil, ConfigConflictError{HostPath: hostConfigPath, UserPath: userPath}
	}

	cfg := map[string]any{}
	if hostExists {
		cfg, err = loadFile(hostConfigPath)
	} else if userExists {
		cfg, err = loadFile(userPath)
	}
	if err != nil {
		return nil, err
	}
	if hasNonEmptyString(cfg, "client", "server_url") && hasNonEmptyString(cfg, "workspace", "dir") {
		return nil, fmt.Errorf("config cannot contain both [client] and [workspace]")
	}
	return cfg, nil
}

func GetValue(keys ...string) (any, bool, error) {
	cfg, err := Load()
	if err != nil {
		return nil, false, err
	}
	current := any(cfg)
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false, nil
		}
		value, ok := m[key]
		if !ok {
			return nil, false, nil
		}
		current = value
	}
	return current, true, nil
}

func GetString(keys ...string) (string, bool, error) {
	value, ok, err := GetValue(keys...)
	if err != nil || !ok {
		return "", false, err
	}
	text, ok := value.(string)
	if !ok {
		return "", false, nil
	}
	text = strings.TrimSpace(text)
	return text, text != "", nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func loadFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := toml.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, nil
}

func hasNonEmptyString(cfg map[string]any, keys ...string) bool {
	current := any(cfg)
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = m[key]
		if !ok {
			return false
		}
	}
	text, ok := current.(string)
	return ok && strings.TrimSpace(text) != ""
}
