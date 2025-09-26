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

	doc, err := w.Store.MarshalKDLDocument()
	if err != nil {
		return fmt.Errorf("marshalling store: %w", err)
	}

	f, err := os.Create(storePath)
	if err != nil {
		return fmt.Errorf("opening store file: %w", err)
	}
	defer f.Close()
	err = kdl.NewEmitter(kdl.KdlVersion2, f).EmitDocument(doc)
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
	doc, err := kdl.NewParser(kdl.KdlVersion2, f).ParseDocument()
	if err != nil {
		return fmt.Errorf("parsing store file %s: %w", storePath, err)
	}

	w.Store = &Store{}
	err = w.Store.UnmarshalKDLDocument(doc)
	if err != nil {
		return fmt.Errorf("unmarshalling store: %w", err)
	}

	logger.Debugf("store loaded, %d wallpapers defined", len(w.Store.Wallpapers))

	return nil
}

func (r Resolution) String() string {
	return fmt.Sprintf("%dx%d", r.Width, r.Height)
}

func (r *Resolution) UnmarshalKDL(v kdl.Value) error {
	s, err := kdl.AsString(v)
	if err != nil {
		return err
	}
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return fmt.Errorf("invalid resolution %s", s)
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
		if value == "" {
			tagsNode.NewChild(tag)
		} else {
			tagsNode.NewKV(tag, kdl.NewString(value))
		}
	}
	n := kdl.NewKV("wallpaper", wp.Id).
		AddChildren(
			kdl.NewKV("path", wp.Path),
			kdl.NewKV("original", wp.OriginalFilename),
			kdl.NewKV("resolution", wp.Resolution.String()),
			kdl.NewKV("type", wp.MimeType),
			kdl.NewKV("enabled", wp.Enabled),
		)
	if len(tagsNode.Children.Nodes) > 0 {
		n.AddChild(tagsNode)
	}
	return n, nil
}

func (wp *Wallpaper) UnmarshalKDL(n *kdl.Node) error {
	var err error
	wp.Id, err = kdl.Get(n, 0, kdl.AsString)
	if err != nil {
		return err
	}
	wp.Path, err = kdl.GetKV(&n.Children, "path", kdl.AsString)
	if err != nil {
		return err
	}
	wp.OriginalFilename, err = kdl.GetKV(&n.Children, "original", kdl.AsString)
	if err != nil {
		return err
	}
	res, err := kdl.GetKV(&n.Children, "resolution", kdl.AsValue)
	if err == nil {
		err = wp.Resolution.UnmarshalKDL(res)
	}
	if err != nil {
		return err
	}
	wp.MimeType, err = kdl.GetKV(&n.Children, "type", kdl.AsString)
	if err != nil {
		return nil
	}
	wp.Enabled, err = kdl.GetKV(&n.Children, "enabled", kdl.AsBool)
	if err != nil {
		return nil
	}
	wp.Tags = make(map[string]string)
	tags := n.GetChild("tags")
	if tags != nil {
		for _, tag := range tags.Children.Nodes {
			if len(tag.Arguments) == 0 {
				wp.Tags[tag.Name] = ""
			} else if len(tag.Arguments) == 1 {
				value, err := kdl.AsString(tag.Arguments[0])
				if err != nil {
					return err
				}
				wp.Tags[tag.Name] = value
			} else {
				return fmt.Errorf("invalid tag %s", tag.Name)
			}
		}
	}

	// TODO: validate store

	return nil
}

func (s *Store) MarshalKDLDocument() (*kdl.Document, error) {
	doc := kdl.NewDocument()

	n, err := kdl.MarshalAll(s.Wallpapers)
	if err != nil {
		return nil, fmt.Errorf("marshalling wallpapers: %w", err)
	}
	doc.AddNodes(n...)

	return doc, nil
}

func (s *Store) UnmarshalKDLDocument(doc *kdl.Document) error {
	wallpapers, err := kdl.UnmarshalAll[Wallpaper](doc.Nodes)
	if err != nil {
		return fmt.Errorf("unmarshalling wallpapers: %w", err)
	}

	s.Wallpapers = wallpapers
	return nil
}
