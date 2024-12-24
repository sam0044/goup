package cmd

import (
	"fmt"
	// "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	// "github.com/shirou/gopsutil/v4/cpu"
	// "github.com/shirou/gopsutil/v4/mem"
	"github.com/urfave/cli/v2"
	// "time"
)

func cmdScan() *cli.Command {
	return &cli.Command{
		Name:   "scan",
		Action: scanResource,
		Usage:  "Show resource usage of system",
		Description: `
Resource monitoring tools

Examples:
# Monitor real time operations with tview
$ goup scan 
`,
	}
}

func scanResource(ctx *cli.Context) error {
	box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")
	if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
		return fmt.Errorf("application error: %w", err)
	}
	return nil
}
