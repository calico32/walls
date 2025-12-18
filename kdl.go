package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/calico32/kdl-go"
)

func (w *Walls) WriteStore(ctx context.Context) error {
	storePath := filepath.Join(w.Config.Storage.Sources, "store.kdl")
	logger.Debugf("writing store with %d wallpapers to %s", len(w.Store.Wallpapers), storePath)

	doc, err := w.Store.MarshalKDL()
	if err != nil {
		return fmt.Errorf("marshalling store: %w", err)
	}

	f, err := os.Create(storePath)
	if err != nil {
		return fmt.Errorf("opening store file: %w", err)
	}
	defer f.Close()
	err = kdl.Emit(doc, f)
	if err != nil {
		return fmt.Errorf("emitting store file %s: %w", storePath, err)
	}

	return nil
}

func (w *Walls) LoadStore(ctx context.Context) error {
	storePath := filepath.Join(w.Config.Storage.Sources, "store.kdl")
	logger.Debugf("loading store from %s", storePath)
	f, err := os.Open(storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// no store file, that's okay - nothing to load
			w.Store = &Store{}
			return nil
		}
		return fmt.Errorf("opening store file: %w", err)
	}
	defer f.Close()

	var store Store
	err = kdl.Decode(f, &store)
	if err != nil {
		return fmt.Errorf("parsing store file %s: %w", storePath, err)
	}
	w.Store = &store

	logger.Debugf("store loaded, %d wallpapers defined", len(w.Store.Wallpapers))

	return nil
}

func (r Resolution) String() string {
	return fmt.Sprintf("%dx%d", r.Width, r.Height)
}

var _ kdl.ValueUnmarshaler = (*Resolution)(nil)

func (r *Resolution) UnmarshalKDL(v kdl.Value) error {
	if v.Kind() != kdl.String {
		return fmt.Errorf("invalid resolution %s", v)
	}
	parts := strings.Split(v.String(), "x")
	if len(parts) != 2 {
		return fmt.Errorf("invalid resolution %s", v)
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}
	r.Width = width
	r.Height = height
	return nil
}

func (wp *Wallpaper) MarshalKDL() (*kdl.Node, error) {
	tagsNode := kdl.NewNode("tags")
	for tag, value := range wp.Tags {
		tagsNode.NewKV(tag, kdl.NewString(value))
	}
	n := kdl.NewKV("wallpaper", wp.Id).
		AddChildren(
			kdl.NewKV("path", wp.Path),
			kdl.NewKV("original", wp.OriginalFilename),
			kdl.NewKV("resolution", wp.Resolution.String()),
			kdl.NewKV("type", wp.MimeType),
			kdl.NewKV("enabled", wp.Enabled),
			tagsNode,
		)
	return n, nil
}

func (s *Store) MarshalKDL() (*kdl.Document, error) {
	doc := kdl.NewDocument()

	n, err := kdl.MarshalAll(s.Wallpapers)
	if err != nil {
		return nil, fmt.Errorf("marshalling wallpapers: %w", err)
	}
	doc.AddNodes(n...)

	return doc, nil
}
