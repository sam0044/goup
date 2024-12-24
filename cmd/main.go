package cmd

import (
	"github.com/urfave/cli/v2"

	"syscall"
)

func Main(args []string) error {

	cli.VersionFlag = &cli.BoolFlag{
		Name: "version", Aliases: []string{"V"},
		Usage: "print version only",
	}
	app := &cli.App{
		Name:                 "goup",
		Usage:                "Resource monitoring go application",
		HideHelpCommand:      true,
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			cmdMonitor(),
			cmdScan(),
		},
	}

	err := app.Run(args)
	if errno, ok := err.(syscall.Errno); ok && errno == 0 {
		err = nil
	}
	return err

}
