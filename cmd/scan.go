package cmd

import (
	"fmt"
	"github.com/gdamore/tcell/v2"	
	"github.com/rivo/tview"
	"github.com/shirou/gopsutil/v4/cpu"
	//"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	//"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/urfave/cli/v2"
	"sort"
	"time"
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

func updateProcess(table *tview.Table){

	procs, _ := process.Processes()
	// Pre-filter to only get processes with non-zero CPU usage
	type procInfo struct {
		pid int32
		name string
		cpu  float64
		io   int32
	}
	procInfos := make([]procInfo, 0, 30)
	for _, p := range procs {
		cpu, _ := p.CPUPercent()
		if cpu > 0.01 { // Only include processes with notable CPU usage
			name, _ := p.Name()
			io, _ := p.IOnice()
			pid := p.Pid
			procInfos = append(procInfos, procInfo{
				pid: pid,
				name: name,
				cpu:  cpu,
				io:   io,
			})
		}
	}
	
	// Sort only the filtered processes
	sort.Slice(procInfos, func(i, j int) bool {
		return procInfos[i].cpu > procInfos[j].cpu
	})

	// Take only top 30
	if len(procInfos) > 30 {
		procInfos = procInfos[:30]
	}
	for i, info := range procInfos {
        table.SetCell(i+1, 0, tview.NewTableCell(fmt.Sprintf("%d", info.pid)))
        table.SetCell(i+1, 1, tview.NewTableCell(info.name))
        table.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprintf("%.2f%%", info.cpu)))
        table.SetCell(i+1, 3, tview.NewTableCell(fmt.Sprintf("%d", info.io)))
    }
	
}

func updateSystem(view *tview.TextView){
	memStats, _:= mem.VirtualMemory()
	cpuPercent, _ := cpu.Percent(0, false)
	var memTotal float64
    var memUsed float64
    if memStats != nil {
        memTotal = float64(memStats.Total) / 1e9  // Convert to GB
        memUsed = memStats.UsedPercent
    }
	view.Clear()
	fmt.Fprintf(view, "\nSystem Information (Press 'q' to quit)\n")
    fmt.Fprintf(view, "[green]CPU Usage: %.2f%% | Memory Usage: %.1f%% of %.1f GB", cpuPercent[0],memUsed, memTotal)
}

func scanResource(ctx *cli.Context) error {
	app := tview.NewApplication()
	systemView := tview.NewTextView().
				SetDynamicColors(true).
				SetTextColor(tcell.ColorPink)
		
		
	processTable := tview.NewTable().SetBorders(false)

	
	// Define and set headers
    headers := []string{"PID", "Program", "CPU%", "IO"}
    for col, header := range headers {
        processTable.SetCell(0, col,
            tview.NewTableCell(header).
                SetTextColor(tcell.ColorPink).
                SetAttributes(tcell.AttrBold))
    }
	flex := tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(systemView, 5, 2, false).
				AddItem(processTable, 0, 1, false), 0, 3, false)


	//event loop to update the values of the process table
	go func(){
		ticker := time.NewTicker(2 * time.Second)
			for range ticker.C{
				app.QueueUpdateDraw(func(){
					updateProcess(processTable)
					updateSystem(systemView)
				})
			}
	}()

	// quit the app with 'q'
	// 'Ctrl+c' is a global event that stops the application by default
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey{
		if event.Rune()=='q'{
			app.Stop()
			return nil
		}
		return event
	})

	if err := app.SetRoot(flex, true).Run(); err != nil {
		return fmt.Errorf("application error: %w", err)
	}
	return nil
}
