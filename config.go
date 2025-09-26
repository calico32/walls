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
	Default string `kdl:"default"`
	Effects map[string][]string
}

type BehaviorConfig struct {
	AllowRepeat bool  `kdl:"allow-repeat"`
	Set         []Set `kdl:"set"`
}

type Set struct {
	Command []string `kdl:"command"`
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
	doc, err := kdl.NewParser(kdl.KdlVersion2, r).ParseDocument()
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	var parseErrors []error
	parseError := func(err error) {
		parseErrors = append(parseErrors, err)
	}

	config := DefaultConfig()

	if n := doc.GetNode("storage"); n != nil {
		sourcePath, err := kdl.GetKV(&n.Children, "sources", kdl.AsString)
		if err == nil {
			config.Storage.Sources = expandPath(sourcePath)
			if _, err := os.Stat(sourcePath); errors.Is(err, os.ErrNotExist) {
				parseError(fmt.Errorf("storage.sources: directory does not exist"))
			} else {
				logger.Debugf("using storage.sources: %s", config.Storage.Sources)
			}
		} else if !errors.Is(err, kdl.ErrNotFound) {
			parseError(fmt.Errorf("parsing storage.sources: %w", err))
		} else {
			logger.Debugf("using default storage.sources: %s", config.Storage.Sources)
			os.MkdirAll(config.Storage.Sources, 0755)
		}

		cachePath, err := kdl.GetKV(&n.Children, "cache", kdl.AsString)
		if err == nil {
			config.Storage.Cache = expandPath(cachePath)
			if _, err := os.Stat(cachePath); errors.Is(err, os.ErrNotExist) {
				parseError(fmt.Errorf("storage.cache: directory does not exist"))
			} else {
				logger.Debugf("using storage.cache: %s", config.Storage.Cache)
			}
		} else if !errors.Is(err, kdl.ErrNotFound) {
			parseError(fmt.Errorf("parsing storage.cache: %w", err))
		} else {
			logger.Debugf("using default storage.cache: %s", config.Storage.Cache)
			os.MkdirAll(config.Storage.Cache, 0755)
		}

		runtimePath, err := kdl.GetKV(&n.Children, "runtime", kdl.AsString)
		if err == nil {
			config.Storage.Runtime = expandPath(runtimePath)
			if _, err := os.Stat(runtimePath); errors.Is(err, os.ErrNotExist) {
				parseError(fmt.Errorf("storage.runtime: directory does not exist"))
			} else {
				logger.Debugf("using storage.runtime: %s", config.Storage.Runtime)
			}
		} else if !errors.Is(err, kdl.ErrNotFound) {
			parseError(fmt.Errorf("parsing storage.runtime: %w", err))
		} else {
			logger.Debugf("using default storage.runtime: %s", config.Storage.Runtime)
			os.MkdirAll(config.Storage.Runtime, 0755)
		}

	}

	if n := doc.GetNode("effects"); n != nil {
		defaultEffect, err := kdl.Get(n, "default", kdl.AsString)
		if err == nil {
			config.Effects.Default = defaultEffect
		} else if !errors.Is(err, kdl.ErrNotFound) {
			parseError(fmt.Errorf("parsing effects.default: %w", err))
		}

		for _, effect := range n.Children.Nodes {
			effectName := effect.Name
			args, err := kdl.CastAll(effect.Arguments, kdl.AsString)
			if err != nil {
				parseError(fmt.Errorf("parsing effects.%s: %w", effectName, err))
			}
			hasInput := false
			hasOutput := false
			for _, arg := range args {
				if arg == "%i" {
					hasInput = true
				} else if arg == "%o" {
					hasOutput = true
				}
			}
			if !hasInput {
				parseError(fmt.Errorf("effects.%s: missing input file placeholder %%i", effectName))
			}
			if !hasOutput {
				parseError(fmt.Errorf("effects.%s: missing output file placeholder %%o", effectName))
			}
			config.Effects.Effects[effectName] = args
		}

		if config.Effects.Default != "" {
			if _, ok := config.Effects.Effects[config.Effects.Default]; !ok {
				parseError(fmt.Errorf("effects.default: unknown effect %s", config.Effects.Default))
			}
		}
	}

	if n := doc.GetNode("behavior"); n != nil {
		d := &n.Children

		allowRepeat, err := kdl.GetKV(d, "allow-repeat", kdl.AsBool)
		if err == nil {
			config.Behavior.AllowRepeat = allowRepeat
		} else if !errors.Is(err, kdl.ErrNotFound) {
			parseError(fmt.Errorf("parsing behavior.allow-repeat: %w", err))
		}

		sets := d.GetNodes("set")
		for _, n := range sets {
			var set Set
			set.Command, err = kdl.CastAll(n.Arguments, kdl.AsString)
			if err != nil {
				parseError(fmt.Errorf("parsing behavior.set: %w", err))
			}
			set.Effect, err = kdl.Get(n, "effect", kdl.AsString)
			if err != nil && !errors.Is(err, kdl.ErrNotFound) {
				parseError(fmt.Errorf("parsing behavior.set#effect: %w", err))
			}
			set.Pkill, err = kdl.Get(n, "pkill", kdl.AsString)
			if err != nil && !errors.Is(err, kdl.ErrNotFound) {
				parseError(fmt.Errorf("parsing behavior.set#pkill: %w", err))
			}

			if len(set.Command) == 0 {
				parseError(fmt.Errorf("behavior.set: missing command"))
			}
			hasWallpaper := false
			for _, arg := range set.Command {
				if arg == "%w" {
					hasWallpaper = true
				}
			}
			if !hasWallpaper {
				parseError(fmt.Errorf("behavior.set: missing wallpaper placeholder %%w"))
			}
			if set.Effect != "" {
				if _, ok := config.Effects.Effects[set.Effect]; !ok {
					parseError(fmt.Errorf("behavior.set: unknown effect %s", set.Effect))
				}
			}

			config.Behavior.Set = append(config.Behavior.Set, set)
		}
	}

	if len(parseErrors) > 0 {
		errorText := ""
		for _, err := range parseErrors {
			errorText += "  " + err.Error() + "\n"
		}
		errorPlural := ""
		if len(parseErrors) > 1 {
			errorPlural = "s"
		}
		return nil, fmt.Errorf("parsing config: encountered %d error%s\n%s", len(parseErrors), errorPlural, errorText)
	}

	logger.Debugf("config loaded successfully")
	return config, nil
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
