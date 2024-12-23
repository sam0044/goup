package cmd

import (
	"runtime"
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/urfave/cli/v2"
	"sort"
	"time"
)

func cmdMonitor() *cli.Command {
	return &cli.Command{
		Name:   "profile",
		Action: profile,
		Usage:  "Show reousrce usage of system",
		Description: `
Resource monitoring tools

Examples:
# Monitor real time operations
$ goup profile 

`,
	}
}
type model struct {
	processTable table.Model
	cpuPercent   float64
	memStats     *mem.VirtualMemoryStat
	err          error
}

type tickMsg struct{}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tickMsg:
		var err error
		cpuPercent, err := cpu.Percent(0, false)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.cpuPercent = cpuPercent[0]

		m.memStats, err = mem.VirtualMemory()
		if err != nil {
			m.err = err
			return m, nil
		}

		// Update process table
		procs, _ := process.Processes()
		
		// Pre-filter to only get processes with non-zero CPU usage
		type procInfo struct {
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
				procInfos = append(procInfos, procInfo{
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

		// Convert to table rows
		rows := make([]table.Row, len(procInfos))
		for i, info := range procInfos {
			rows[i] = table.Row{
				info.name,
				fmt.Sprintf("%.2f%%", info.cpu),
				fmt.Sprintf("%d", info.io),
			}
		}

		m.processTable.SetRows(rows)
		return m, tickCmd()
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	var memTotal float64
	var memUsed float64
	if m.memStats != nil {
		memTotal = float64(m.memStats.Total) / 1e9
		memUsed = m.memStats.UsedPercent
	}

	// Create styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginLeft(2)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		MarginLeft(2)

	docStyle := lipgloss.NewStyle().
		Padding(1, 2, 1, 2)

	// Create the header with stats
	title := titleStyle.Render("System Monitor (press q to quit)")
	stats := statsStyle.Render(fmt.Sprintf(
		"CPU Usage: %.2f%% | Memory Usage: %.2f%% of %.2fGB",
		m.cpuPercent,
		memUsed,
		memTotal,
	))

	// Create a border around the table
	tableStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	// Combine all elements
	return docStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			stats,
			"",  // Empty line for spacing
			tableStyle.Render(m.processTable.View()),
		),
	)
}

func profile(ctx *cli.Context) error {
	// Set GOMAXPROCS to limit CPU usage
	runtime.GOMAXPROCS(2)
	columns := []table.Column{
		{Title: "Process", Width: 30},
		{Title: "CPU%", Width: 10},
		{Title: "IO", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(30),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	t.SetStyles(s)

	m := model{
		processTable: t,
	}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)
	if err := p.Start(); err != nil {
		return fmt.Errorf("error running program: %v", err)
	}

	return nil
}
