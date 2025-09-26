package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func deleteCommand() *cli.Command {
	return &cli.Command{
		Name:         "delete",
		Aliases:      []string{"rm", "del"},
		Usage:        "Delete a wallpaper from the store",
		HideHelp:     true,
		OnUsageError: forwardUsageError,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Ignore errors when deleting wallpapers that don't exist or when errors occur while deleting files from disk.",
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "wallpaper",
			},
		},
		Action: deleteAction,
	}
}

func deleteAction(ctx context.Context, cmd *cli.Command) error {
	wallpaper := cmd.StringArg("wallpaper")
	if wallpaper == "" {
		return fmt.Errorf("wallpaper is required\nusage: walls delete <wallpaper>")
	}

	w := getWalls(ctx)
	defer w.Sync(ctx)

	err := w.DeleteWallpaper(ctx, wallpaper)
	if err != nil {
		if cmd.Bool("force") {
			logger.Warnf("error deleting wallpaper %s: %w", wallpaper, err)
			return nil
		}
		return fmt.Errorf("deleting wallpaper %s: %w", wallpaper, err)
	}

	logger.Infof("wallpaper %s deleted", wallpaper)

	return nil
}
