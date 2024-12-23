package cmd

import (
	"runtime"
	"fmt"
	"path/filepath"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
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
	processTable   table.Model
	diskTable     table.Model
	networkTable  table.Model
	cpuPercent    float64
	memStats      *mem.VirtualMemoryStat
	diskStats     map[string]disk.UsageStat
	networkStats  map[string]net.IOCountersStat
	lastNetStats  map[string]net.IOCountersStat
	lastCheckTime time.Time
	err           error
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
		
		// CPU Usage
		cpuPercent, err := cpu.Percent(0, false)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.cpuPercent = cpuPercent[0]

		// Memory Stats
		m.memStats, err = mem.VirtualMemory()
		if err != nil {
			m.err = err
			return m, nil
		}

		// Disk Stats
		partitions, err := disk.Partitions(false)
		if err != nil {
			m.err = err
			return m, nil
		}

		m.diskStats = make(map[string]disk.UsageStat)
		for _, partition := range partitions {
			usage, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				continue
			}
			m.diskStats[partition.Mountpoint] = *usage
		}

		// Network Stats
		m.lastNetStats = m.networkStats
		m.lastCheckTime = time.Now()
		
		netStats, err := net.IOCounters(true)
		if err != nil {
			m.err = err
			return m, nil
		}
		
		m.networkStats = make(map[string]net.IOCountersStat)
		for _, stat := range netStats {
			if stat.Name != "lo" { // Skip loopback interface
				m.networkStats[stat.Name] = stat
			}
		}

		// Update disk table
		var diskRows []table.Row
		for path, usage := range m.diskStats {
			diskName := filepath.Base(path)
			usedGB := float64(usage.Used) / 1e9
			totalGB := float64(usage.Total) / 1e9
			diskRows = append(diskRows, table.Row{
				diskName,
				fmt.Sprintf("%.1f GB", usedGB),
				fmt.Sprintf("%.1f GB", totalGB),
				fmt.Sprintf("%.1f%%", usage.UsedPercent),
			})
		}
		m.diskTable.SetRows(diskRows)

		// Update network table
		var networkRows []table.Row
		for iface, stat := range m.networkStats {
			if m.lastNetStats != nil {
				if lastStat, ok := m.lastNetStats[iface]; ok {
					duration := time.Since(m.lastCheckTime).Seconds()
					if duration > 0 {
						bytesInPerSec := float64(stat.BytesRecv-lastStat.BytesRecv) / duration
						bytesOutPerSec := float64(stat.BytesSent-lastStat.BytesSent) / duration
						
						networkRows = append(networkRows, table.Row{
							iface,
							fmt.Sprintf("%.1f MB/s", bytesInPerSec/1e6),
							fmt.Sprintf("%.1f MB/s", bytesOutPerSec/1e6),
						})
					}
				}
			}
		}
		m.networkTable.SetRows(networkRows)

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

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginLeft(2)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		MarginLeft(2)

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginRight(2)

	sectionTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))

	// Header section
	title := titleStyle.Render("System Monitor (press q to quit)")
	stats := statsStyle.Render(fmt.Sprintf(
		"CPU Usage: %.2f%% | Memory: %.1f%% of %.1fGB used",
		m.cpuPercent,
		memUsed,
		memTotal,
	))

	// Create flexbox columns for tables
	leftColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		sectionTitleStyle.Render("Processes"),
		boxStyle.Render(m.processTable.View()),
	)

	rightColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		sectionTitleStyle.Render("Disk Usage"),
		boxStyle.Render(m.diskTable.View()),
		"",
		sectionTitleStyle.Render("Network Activity"),
		boxStyle.Render(m.networkTable.View()),
	)

	// Join columns side by side
	tables := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		rightColumn,
	)

	// Final layout
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		stats,
		"",
		tables,
	)
}

func profile(ctx *cli.Context) error {
	// Set GOMAXPROCS to limit CPU usage
	runtime.GOMAXPROCS(2)
	
		// Process table
		processColumns := []table.Column{
			{Title: "Process", Width: 30},
			{Title: "CPU%", Width: 10},
			{Title: "IO", Width: 10},
		}
	
		processTable := table.New(
			table.WithColumns(processColumns),
			table.WithFocused(true),
			table.WithHeight(20),
		)
	
		// Disk table
		diskColumns := []table.Column{
			{Title: "Mount", Width: 15},
			{Title: "Used", Width: 10},
			{Title: "Total", Width: 10},
			{Title: "Usage%", Width: 10},
		}
	
		diskTable := table.New(
			table.WithColumns(diskColumns),
			table.WithHeight(10),
		)
	
		// Network table
		networkColumns := []table.Column{
			{Title: "Interface", Width: 15},
			{Title: "Download", Width: 15},
			{Title: "Upload", Width: 15},
		}
	
		networkTable := table.New(
			table.WithColumns(networkColumns),
			table.WithHeight(8),
		)
	
		// Common table styles
		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(true)
	
		processTable.SetStyles(s)
		diskTable.SetStyles(s)
		networkTable.SetStyles(s)

	m := model{
		processTable:   processTable,
		diskTable:     diskTable,
		networkTable:  networkTable,
		diskStats:     make(map[string]disk.UsageStat),
		networkStats:  make(map[string]net.IOCountersStat),
		lastNetStats:  make(map[string]net.IOCountersStat),
		lastCheckTime: time.Now(),
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
