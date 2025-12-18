package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/calico32/kdl-go"
)

type Config struct {
	Storage  StorageConfig  `kdl:"storage"`
	Effects  EffectsConfig  `kdl:"effects"`
	Behavior BehaviorConfig `kdl:"behavior"`
}

type StorageConfig struct {
	Sources string `kdl:"sources"`
	Cache   string `kdl:"cache"`
	Runtime string `kdl:"runtime"`
}

type EffectsConfig struct {
	Default string              `kdl:"default"`
	Effects map[string][]string `kdl:",children"`
}

type BehaviorConfig struct {
	AllowRepeat bool  `kdl:"allow-repeat"`
	Set         []Set `kdl:"set,multiple"`
}

type Set struct {
	Command []string `kdl:",arguments"`
	Effect  string   `kdl:"effect"`
	Pkill   string   `kdl:"pkill"`
}

func DefaultConfig() *Config {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		xdgDataHome = filepath.Join(home, ".local", "share")
		logger.Debugf("using default XDG_DATA_HOME: %s", xdgDataHome)
	}
	xdgCacheHome := os.Getenv("XDG_CACHE_HOME")
	if xdgCacheHome == "" {
		xdgCacheHome = filepath.Join(home, ".cache")
		logger.Debugf("using default XDG_CACHE_HOME: %s", xdgCacheHome)
	}
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntimeDir == "" {
		xdgRuntimeDir = "/tmp"
		logger.Debugf("using default XDG_RUNTIME_DIR: %s", xdgRuntimeDir)
	}

	dataDir := filepath.Join(xdgDataHome, "walls")
	cacheDir := filepath.Join(xdgCacheHome, "walls")
	runtimeDir := filepath.Join(xdgRuntimeDir, "walls")

	return &Config{
		Storage: StorageConfig{
			Sources: dataDir,
			Cache:   cacheDir,
			Runtime: runtimeDir,
		},
		Effects: EffectsConfig{
			Effects: make(map[string][]string),
		},
		Behavior: BehaviorConfig{
			AllowRepeat: false,
			Set:         []Set{},
		},
	}
}

func expandPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	path = os.ExpandEnv(path)
	path, tilde := strings.CutPrefix(path, "~"+string(filepath.Separator))
	if tilde {
		path = filepath.Join(home, path)
	}
	return path
}

func parseConfig(r io.Reader) (*Config, error) {
	var config Config
	err := kdl.Decode(r, &config)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	logger.Debugf("config loaded successfully")
	defaultConfig := DefaultConfig()
	if config.Storage.Sources == "" {
		config.Storage.Sources = defaultConfig.Storage.Sources
	}
	if config.Storage.Cache == "" {
		config.Storage.Cache = defaultConfig.Storage.Cache
	}
	if config.Storage.Runtime == "" {
		config.Storage.Runtime = defaultConfig.Storage.Runtime
	}
	if config.Effects.Default != "" {
		if _, ok := config.Effects.Effects[config.Effects.Default]; !ok {
			return nil, fmt.Errorf("effects.default: unknown effect %s", config.Effects.Default)
		}
	}
	return &config, nil
}

func loadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		logger.Warnf("without a config file, walls doesn't know how to set your wallpaper.\nConsider creating a config file at %s.\n", path)
		return DefaultConfig(), nil
	}
	logger.Debugf("using config file: %s", path)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening config file %s: %w", path, err)
	}
	defer f.Close()
	return parseConfig(f)
}
