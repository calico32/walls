package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:         "list",
		Aliases:      []string{"ls", "l"},
		Usage:        "List wallpapers in the store",
		HideHelp:     true,
		OnUsageError: forwardUsageError,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "long",
				Aliases: []string{"l"},
				Usage:   "Show extended information about each wallpaper.",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "Output wallpapers in JSON format.",
			},
		},
		Action: listAction,
	}
}

func listAction(ctx context.Context, cmd *cli.Command) error {
	w := getWalls(ctx)

	wps := w.Store.Wallpapers

	if cmd.Bool("json") {
		if !cmd.Bool("long") {
			// ids in array
			ids := make([]string, len(wps))
			for i, wp := range wps {
				ids[i] = wp.Id
			}
			out, err := json.Marshal(ids)
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		}

		wps := w.Store.Wallpapers
		out, err := json.Marshal(wps)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	if !cmd.Bool("long") {
		for _, wp := range wps {
			fmt.Println(wp.Id)
		}
		return nil
	}

	for _, wp := range wps {
		fmt.Printf("Wallpaper %s:\n", wp.Id)
		fmt.Printf("  Source path: %s\n", wp.Path)
		fmt.Printf("  Original filename: %s\n", wp.OriginalFilename)
		fmt.Printf("  Resolution: %s\n", wp.Resolution.String())
		fmt.Printf("  Mime type: %s\n", wp.MimeType)
		fmt.Printf("  Enabled: %t\n", wp.Enabled)
		if len(w.Config.Effects.Effects) > 0 {
			fmt.Printf("  Effects:\n")
			for e, _ := range w.Config.Effects.Effects {
				path := wp.PathWithEffect(ctx, e)
				fmt.Printf("    %s: %s", e, path)
				if _, err := os.Stat(path); err == nil {
					fmt.Printf(" (precached)\n")
				} else if errors.Is(err, os.ErrNotExist) {
					fmt.Printf(" (not precached)\n")
				} else {
					fmt.Printf(" (error: %s)\n", err)
				}
			}
		}
		fmt.Println()
	}

	return nil
}
