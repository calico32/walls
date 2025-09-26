package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func setCommand() *cli.Command {
	return &cli.Command{
		Name:         "set",
		Aliases:      []string{"s"},
		Usage:        "Set the wallpaper",
		HideHelp:     true,
		OnUsageError: forwardUsageError,
		Flags:        []cli.Flag{},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "wallpaper",
			},
		},
		Action: setAction,
	}
}

func setAction(ctx context.Context, cmd *cli.Command) error {
	w := getWalls(ctx)
	// defer w.Sync(ctx)

	wallpaperId := cmd.StringArg("wallpaper")
	if wallpaperId == "" {
		wp := w.RandomWallpaper(ctx)
		if wp == nil {
			return fmt.Errorf("no wallpapers enabled/found")
		}
		wallpaperId = wp.Id
	}

	err := w.SetWallpaper(ctx, wallpaperId)
	if err != nil {
		return fmt.Errorf("setting wallpaper: %w", err)
	}

	return nil
}
