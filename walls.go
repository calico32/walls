package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/gen2brain/avif"
	"github.com/prometheus/procfs"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type Walls struct {
	Config *Config
	Store  *Store
}

type Store struct {
	Wallpapers []*Wallpaper
}

type Wallpaper struct {
	// Unique identifier for the wallpaper
	Id string `kdl:"id" json:"id"`
	// Path (relative to the data directory) to the wallpaper file
	Path string `kdl:"path" json:"source_path"`
	// Original filename of the wallpaper when it was added
	OriginalFilename string `kdl:"original" json:"original_filename"`
	// The resolution of the wallpaper
	Resolution Resolution `kdl:"resolution" json:"resolution"`
	// The mime type of the wallpaper (e.g. image/jpeg)
	MimeType string `kdl:"type" json:"mime_type"`
	// Whether or not the wallpaper can be automatically selected
	Enabled bool `kdl:"enabled" json:"enabled"`
	// Tags associated with the wallpaper
	Tags map[string]string `kdl:"tags" json:"tags"`
}

type Resolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func loadWalls(ctx context.Context, configPath string) (context.Context, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return ctx, err
	}
	w := &Walls{Config: config}
	err = w.Init(ctx)
	if err != nil {
		return ctx, err
	}
	ctx = setWalls(ctx, w)
	return ctx, nil
}

func (w *Walls) Init(ctx context.Context) error {
	logger.Debugf("walls initializing")
	err := w.CreateDirs(ctx)
	if err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}
	err = w.LoadStore(ctx)
	if err != nil {
		return fmt.Errorf("loading store: %w", err)
	}

	return nil
}

func (w *Walls) Sync(ctx context.Context) {
	logger.Debugf("walls syncing")
	err := w.WriteStore(ctx)
	if err != nil {
		logger.Errorf("writing store: %w", err)
	}
}

func (w *Walls) CreateDirs(ctx context.Context) error {
	dirs := []string{
		w.Config.Storage.Sources,
		filepath.Join(w.Config.Storage.Sources, "sources"),
		w.Config.Storage.Cache,
		w.Config.Storage.Runtime,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	return nil
}

func (w *Walls) AddWallpaper(ctx context.Context, path string, id string) (*Wallpaper, error) {
	logger.Debugf("adding wallpaper %s", path)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening wallpaper file %s: %w", path, err)
	}
	defer f.Close()

	if id == "" {
		base := filepath.Base(path)
		ext := filepath.Ext(base)
		id = strings.TrimSuffix(base, ext)
		id = strings.ReplaceAll(id, " ", "_")
		logger.Debugf("using id %s from filename", id)
	}

	// read and decode image
	img, format, err := image.DecodeConfig(f)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	pathInStore := filepath.Join(w.Config.Storage.Sources, "sources", id+"."+format)

	wallpaper := &Wallpaper{
		Id:               id,
		Path:             pathInStore,
		OriginalFilename: filepath.Base(path),
		MimeType:         "image/" + format,
		Resolution: Resolution{
			Width:  img.Width,
			Height: img.Height,
		},
		Enabled: true,
	}

	exists := false
	for _, wp := range w.Store.Wallpapers {
		if wp.Id == wallpaper.Id {
			exists = true
			break
		}
	}
	if exists {
		return nil, fmt.Errorf("a wallpaper with the id %s already exists, aborting\n(specify a different id with --id or change the filename)", wallpaper.Id)
	}

	// write image to store
	f.Seek(0, io.SeekStart)
	destination, err := os.Create(pathInStore)
	logger.Debugf("writing wallpaper to store at %s", pathInStore)
	if err != nil {
		return nil, fmt.Errorf("creating store file %s: %w", pathInStore, err)
	}
	defer destination.Close()
	if _, err := io.Copy(destination, f); err != nil {
		return nil, fmt.Errorf("copying image to store: %w", err)
	}

	w.Store.Wallpapers = append(w.Store.Wallpapers, wallpaper)

	return wallpaper, nil
}

func (w *Walls) DeleteWallpaper(ctx context.Context, id string) error {
	for i, wp := range w.Store.Wallpapers {
		if wp.Id == id {
			w.Store.Wallpapers = append(w.Store.Wallpapers[:i], w.Store.Wallpapers[i+1:]...)
			return w.deleteFromDisk(ctx, wp)
		}
	}

	return fmt.Errorf("wallpaper with id %s not found", id)
}

func (w *Walls) deleteFromDisk(ctx context.Context, wp *Wallpaper) error {
	logger.Debugf("deleting wallpaper %s: deleting %s", wp.Id, wp.Path)
	if err := os.Remove(wp.Path); err != nil {
		return fmt.Errorf("removing wallpaper file %s: %w", wp.Path, err)
	}
	for effect, _ := range w.Config.Effects.Effects {
		logger.Debugf("deleting effect %s for wallpaper %s: deleting %s", effect, wp.Id, wp.PathWithEffect(ctx, effect))
		if err := os.Remove(wp.PathWithEffect(ctx, effect)); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("removing effect file %s: %w", wp.PathWithEffect(ctx, effect), err)
			}
		}
	}
	return nil
}

func (w *Walls) PrecacheAll(ctx context.Context, force bool) error {
	var wg sync.WaitGroup
	wg.Add(len(w.Store.Wallpapers))
	for _, wp := range w.Store.Wallpapers {
		go func(wp *Wallpaper) {
			defer wg.Done()
			if err := w.precacheWallpaper(ctx, wp, force); err != nil {
				logger.Errorf("precaching wallpaper %s: %w", wp.Id, err)
			}
		}(wp)
	}

	logger.Infof("precaching %d wallpapers with %d effects...", len(w.Store.Wallpapers), len(w.Config.Effects.Effects))
	wg.Wait()
	logger.Infof("precaching complete")

	return nil
}

func (w *Walls) PrecacheWallpaper(ctx context.Context, id string, force bool) error {
	for _, wp := range w.Store.Wallpapers {
		if wp.Id == id {
			return w.precacheWallpaper(ctx, wp, force)
		}
	}
	return fmt.Errorf("wallpaper with id %s not found", id)
}

func (w *Walls) precacheWallpaper(ctx context.Context, wp *Wallpaper, force bool) error {
	var wg sync.WaitGroup
	wg.Add(len(w.Config.Effects.Effects))
	errors := false
	effects := w.Config.Effects.Effects
	for effect, command := range effects {
		go func(effect string, command []string) {
			defer wg.Done()
			if err := w.applyEffect(ctx, wp, effect, command, force); err != nil {
				logger.Errorf("applying effect %s: %w", effect, err)
				errors = true
			}
		}(effect, command)
	}
	wg.Wait()
	if errors {
		return fmt.Errorf("applying effects failed, see above for details")
	} else {
		return nil
	}
}

func (wp *Wallpaper) PathWithEffect(ctx context.Context, effect string) string {
	return filepath.Join(getWalls(ctx).Config.Storage.Cache, effect, filepath.Base(wp.Path))
}

func (w *Walls) applyEffect(ctx context.Context, wp *Wallpaper, effect string, command []string, force bool) error {
	outputPath := wp.PathWithEffect(ctx, effect)
	effectDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(effectDir, 0755); err != nil {
		return fmt.Errorf("creating effect directory %s: %w", effectDir, err)
	}

	if _, err := os.Stat(outputPath); err == nil && !force {
		logger.Debugf("effect %s already applied to %s, skipping", effect, wp.Id)
		return nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking if effect %s has already been applied: %w", effect, err)
	}

	if force {
		logger.Debugf("overwriting effect %s for %s", effect, wp.Id)
	} else {
		logger.Debugf("applying effect %s for %s", effect, wp.Id)
	}

	commandCopy := make([]string, len(command))
	copy(commandCopy, command)
	command = commandCopy

	// replace %i and %o with the input and output paths
	for i, arg := range command {
		if arg == "%i" {
			command[i] = wp.Path
		} else if arg == "%o" {
			command[i] = outputPath
		}
	}

	commandStr := strings.Join(command, " ")
	logger.Debugf("exec effect: %s", commandStr)
	cmd := exec.CommandContext(ctx, "sh", "-c", commandStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running effect %s: %w", effect, err)
	}

	return nil
}

func (w *Walls) RandomWallpaper(ctx context.Context) *Wallpaper {
	enabled := make([]*Wallpaper, 0, len(w.Store.Wallpapers))
	for _, wp := range w.Store.Wallpapers {
		if wp.Enabled {
			enabled = append(enabled, wp)
		}
	}
	if len(enabled) == 0 {
		return nil
	}
	n := rand.IntN(len(enabled))
	return enabled[n]

}

func (w *Walls) SetWallpaper(ctx context.Context, id string) error {
	var wp *Wallpaper
	for _, w := range w.Store.Wallpapers {
		if w.Id == id {
			wp = w
			break
		}
	}

	if wp == nil {
		return fmt.Errorf("wallpaper with id %s not found", id)
	}

	if len(w.Config.Behavior.Set) == 0 {
		return fmt.Errorf("no wallpaper set behaviors configured")
	}

	for _, set := range w.Config.Behavior.Set {
		path := wp.Path
		effect := w.Config.Effects.Default
		if set.Effect != "" {
			effect = set.Effect
		}
		if effect != "" {
			path = wp.PathWithEffect(ctx, effect)
			if _, err := os.Stat(path); err != nil {
				logger.Debugf("effect %s not precached for wallpaper %s, precaching...", effect, id)
				err = w.precacheWallpaper(ctx, wp, false)
				if err != nil {
					return fmt.Errorf("precaching wallpaper: %w", err)
				}
			}
		}

		command := make([]string, len(set.Command))
		copy(command, set.Command)

		// replace %w with the wallpaper path
		for i, arg := range command {
			if arg == "%w" {
				command[i] = path
			}
		}
		commandStr := strings.Join(command, " ")
		logger.Debugf("exec set: %s", commandStr)
		cmd := exec.CommandContext(ctx, "sh", "-c", commandStr)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("running set command: %w", err)
		}
		logger.Debugf("started process %d", cmd.Process.Pid)

		if set.Pkill != "" {
			slept := false
			ownPid := cmd.Process.Pid
			logger.Debugf("killing processes named %s except for %d", set.Pkill, ownPid)
			procs, err := procfs.AllProcs()
			if err != nil {
				return fmt.Errorf("listing processes: %w", err)
			}
			logger.Debugf("examining %d processes", len(procs))
			for _, p := range procs {
				execPath, err := p.Executable()
				if err != nil {
					continue
				}
				exe := filepath.Base(execPath)
				if exe == set.Pkill {
					if p.PID == ownPid {
						// don't kill ourselves
						continue
					}
					// kill the process after delay
					if !slept {
						time.Sleep(time.Millisecond * 500)
						slept = true
					}
					proc, err := os.FindProcess(p.PID)
					if err != nil {
						return fmt.Errorf("finding process: %w", err)
					}
					err = proc.Signal(syscall.SIGTERM)
					if err != nil {
						return fmt.Errorf("sending signal: %w", err)
					}
					logger.Debugf("killed process %d, exe %s", p.PID, exe)
				}
			}
		}

	}

	return nil

}
