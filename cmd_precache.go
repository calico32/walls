package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func precacheCommand() *cli.Command {
	return &cli.Command{
		Name:         "precache",
		Usage:        "Precache wallpaper effects",
		HideHelp:     true,
		OnUsageError: forwardUsageError,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Force precaching of effects even if they have already been precached.",
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:      "wallpapers",
				UsageText: "IDs of wallpapers to precache. If not specified, all wallpapers will be precached.",
				Min:       0,
				Max:       -1,
			},
		},
		Action: precacheAction,
		ShellComplete: func(ctx context.Context, cmd *cli.Command) {
			ctx, err := loadWalls(ctx, cmd.String("config"))
			if err != nil {
				return
			}

			wps := getWalls(ctx).Store.Wallpapers
			for _, wp := range wps {
				fmt.Println(wp.Id)
			}
		},
	}
}

func precacheAction(ctx context.Context, cmd *cli.Command) error {
	w := getWalls(ctx)

	wpIds := cmd.StringArgs("wallpapers")
	var err error

	if len(wpIds) == 0 {
		err := w.PrecacheAll(ctx, cmd.Bool("force"))
		if err != nil {
			return fmt.Errorf("precaching wallpapers: %w", err)
		}
		return nil
	}

	for _, wp := range wpIds {
		err = w.PrecacheWallpaper(ctx, wp, cmd.Bool("force"))
		if err != nil {
			return fmt.Errorf("precaching wallpaper %s: %w", wp, err)
		}
	}

	return nil
}
