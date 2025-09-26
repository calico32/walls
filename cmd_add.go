package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:         "add",
		Aliases:      []string{"a"},
		Usage:        "Add a wallpaper to the store",
		HideHelp:     true,
		OnUsageError: forwardUsageError,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "no-precache",
				Usage: "Skips precaching any wallpaper effects for the new wallpaper. This is useful if you are " +
					"adding multiple wallpapers and want to precache them in parallel using the precache command.",
			},
			&cli.StringFlag{
				Name:  "id",
				Usage: "The ID of the new wallpaper. If not specified, an ID derived from the filename will be used.",
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "wallpaper",
			},
		},
		Action: addAction,
	}
}

func addAction(ctx context.Context, cmd *cli.Command) error {
	wallpaper := cmd.StringArg("wallpaper")
	if wallpaper == "" {
		return fmt.Errorf("wallpaper is required\nusage: walls add <wallpaper>")
	}

	w := getWalls(ctx)
	defer w.Sync(ctx)

	wp, err := w.AddWallpaper(ctx, wallpaper, cmd.String("id"))
	if err != nil {
		return fmt.Errorf("adding wallpaper: %w", err)
	}

	if cmd.Bool("no-precache") {
		logger.Infof("wallpaper %s added, precaching skipped", wp.Id)
		return nil
	}

	logger.Infof("wallpaper %s added, precaching %d effects...", wp.Id, len(w.Config.Effects.Effects))

	err = w.precacheWallpaper(ctx, wp, cmd.Bool("force"))
	if err != nil {
		return fmt.Errorf("precaching wallpaper: %w", err)
	}

	logger.Infof("precaching complete")

	return nil
}
