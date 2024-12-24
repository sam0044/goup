package cmd

import (
	"fmt"
	// "github.com/gdamore/tcell/v2"
	"time"
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

	// Create a new TextView
	textView := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true) // Enable color tags if needed

	// Create a new application
	app := tview.NewApplication()

	// Update the TextView in a goroutine
	go func() {
		number := 0
		for {
			time.Sleep(1 * time.Second) // Wait for 1 second
			number++

			// Update the TextView's text
			app.QueueUpdateDraw(func() {
				textView.SetText(fmt.Sprintf("Incrementing Number: %d", number))
			})
		}
	}()

	// Set the root and run the application
	if err := app.SetRoot(textView, true).Run(); err != nil {
			fmt.Println("ooga");
	}
	return nil
}
