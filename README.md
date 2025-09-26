# walls

walls is a simple wallpaper and effect manager for Linux systems. It stores
wallpapers, applies user-defined effects to them, and controls other wallpaper
programs (swaybg, swww, feh, etc.) to set the wallpaper.

## Installation

Clone the repository and run `go install .` to install the binary. Make sure `go
env GOBIN` is in your `$PATH`.

## Quick Start

1. Create a config file at `~/.config/walls/config.kdl`, using the example
   [here](config.kdl).
2. Add wallpapers with `walls add <path>`.
3. Set the wallpaper with `walls set [id]` (omit id for a random wallpaper).
4. Enjoy your desktop!

## Configuration

walls reads its configuration from:

- a file specified by the `-c` flag, or
- at `$WALLS_CONFIG`, or
- `$XDG_CONFIG_HOME/walls/config.kdl`, or
- `~/.config/walls/config.kdl`;

whichever is found first.

The configuration is written in [KDL](https://kdl.dev). See the example
documented config file [here](config.kdl).

## Usage

See `walls help` for a list of commands and their usage. Generate shell
completions with `walls completion <shell>`.
