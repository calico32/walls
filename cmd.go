package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

func main() {
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	defaultConfigPath := filepath.Join(xdgConfigHome, "walls", "config.kdl")

	cmd := &cli.Command{
		Name:  "walls",
		Usage: "Manage wallpapers and effects",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose logging",
				Value:   false,
				Sources: cli.EnvVars("WALLS_VERBOSE"),
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to config file",
				Value:   defaultConfigPath,
				Sources: cli.EnvVars("WALLS_CONFIG"),
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			logger.Level = LogLevelInfo
			if cmd.Bool("verbose") {
				logger.Level = LogLevelDebug
			}

			if cmd.NArg() == 0 {
				return ctx, nil
			}
			subcommand := cmd.Args().First()
			if subcommand == "completion" || subcommand == "help" {
				// skip loading config for completion
				return ctx, nil
			}

			return loadWalls(ctx, cmd.String("config"))
		},
		Commands: []*cli.Command{
			addCommand(),
			precacheCommand(),
			listCommand(),
			deleteCommand(),
			setCommand(),
		},
		EnableShellCompletion: true,
		ConfigureShellCompletionCommand: func(cmd *cli.Command) {
			cmd.Hidden = false
			cmd.HideHelp = true
		},
		Suggest: true,
		CommandNotFound: func(ctx context.Context, cmd *cli.Command, command string) {
			suggestion := cli.SuggestCommand(cmd.Root().Commands, command)
			if suggestion != "" {
				logger.Fatalf("command %s not found (did you mean %s?)", command, suggestion)
			}

			logger.Fatalf("command %s not found (see `walls help` for a list of commands)", command)
		},
		OnUsageError: forwardUsageError,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		logger.Fatal(err)
	}
}

func forwardUsageError(ctx context.Context, cmd *cli.Command, err error, isSubcommand bool) error {
	return err
}
