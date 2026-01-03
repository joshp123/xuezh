package config

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

var (
	configOnce sync.Once
	configData map[string]any
	configErr  error
)

func configPath() (string, error) {
	if override := os.Getenv("XUEZH_CONFIG_PATH"); override != "" {
		return expandHome(override)
	}
	if override := os.Getenv("XUEZH_CONFIG_DIR"); override != "" {
		base, err := expandHome(override)
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "config.toml"), nil
	}
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

func loadConfig() (map[string]any, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
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

func getConfig() (map[string]any, error) {
	configOnce.Do(func() {
		configData, configErr = loadConfig()
	})
	if configErr != nil {
		return nil, configErr
	}
	return configData, nil
}

func GetValue(keys ...string) (any, bool, error) {
	cfg, err := getConfig()
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

func expandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	if len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
