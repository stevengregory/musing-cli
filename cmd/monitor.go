package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/stevengregory/musing-cli/internal/config"
	"github.com/stevengregory/musing-cli/internal/docker"
	"github.com/stevengregory/musing-cli/internal/health"
	"github.com/urfave/cli/v2"
)

// Styles using Lip Gloss
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF00FF")). // Magenta/purple
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF00FF")).
			Padding(0, 2).
			MarginBottom(1)

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			MarginBottom(1)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF00FF")).
				MarginTop(1).
				MarginBottom(1)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			MarginTop(1)

	statusRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00")). // Green
				Bold(true)

	statusDownStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")). // Red
			Bold(true)
)

// Messages
type tickMsg time.Time
type healthCheckMsg struct {
	services []ServiceHealth
}

type ServiceHealth struct {
	Name   string
	Port   int
	Status string
}

// Model holds the dashboard state
type monitorModel struct {
	table      table.Model
	spinner    spinner.Model
	lastUpdate time.Time
	services   []ServiceHealth
	isChecking bool
	width      int
	height     int
}

func MonitorCommand() *cli.Command {
	return &cli.Command{
		Name:  "monitor",
		Usage: "Live monitoring dashboard for development stack",
		Action: func(c *cli.Context) error {
			return runMonitor()
		},
	}
}

func runMonitor() error {
	// Check Docker is running (don't auto-start for monitor - just inform user)
	if err := docker.CheckRunning(); err != nil {
		fmt.Println("\n⚠️  Docker is not running.")
		fmt.Println("Please start Docker Desktop and try again, or run: ./musing dev")
		return err
	}

	// Create Bubble Tea program with alternate screen
	p := tea.NewProgram(
		initialMonitorModel(),
		tea.WithAltScreen(),       // Use alternate screen buffer (no flicker!)
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func initialMonitorModel() monitorModel {
	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))

	// Create initial table
	columns := []table.Column{
		table.NewColumn("status", "", 3),
		table.NewColumn("name", "Service", 25),
		table.NewColumn("port", "Port", 8),
		table.NewColumn("latency", "Latency", 12),
	}

	t := table.New(columns).
		WithRows([]table.Row{}).
		Focused(false).
		WithPageSize(20).
		WithBaseStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			BorderForeground(lipgloss.Color("#FF00FF"))).
		HeaderStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF00FF")).
			Bold(true)).
		WithStaticFooter("")

	return monitorModel{
		table:      t,
		spinner:    s,
		lastUpdate: time.Now(),
		services:   []ServiceHealth{},
		isChecking: false,
	}
}

func (m monitorModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tickCmd(),
		checkHealthCmd(), // Initial health check
	)
}

func (m monitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Update every 3 seconds
		m.lastUpdate = time.Time(msg)
		if !m.isChecking {
			m.isChecking = true
			return m, tea.Batch(tickCmd(), checkHealthCmd())
		}
		return m, tickCmd()

	case healthCheckMsg:
		m.services = msg.services
		m.isChecking = false
		m.table = m.table.WithRows(m.buildTableRows())
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m monitorModel) View() string {
	var s string

	// Header
	s += headerStyle.Render("Development Stack - Live Monitor")
	s += "\n"

	// Timestamp at top
	s += timeStyle.Render(fmt.Sprintf("Updated: %s", time.Now().Format("15:04:05 MST")))
	s += "\n"

	// Docker Section
	dockerServices := m.getDockerServices()
	if len(dockerServices) > 0 {
		s += sectionHeaderStyle.Render("━━━ Docker ━━━")
		s += "\n"
		s += m.renderServiceList(dockerServices)
		s += "\n"
	}

	// Database Section
	databaseServices := m.getDatabaseServices()
	if len(databaseServices) > 0 {
		s += sectionHeaderStyle.Render("━━━ Database ━━━")
		s += "\n"
		s += m.renderServiceList(databaseServices)
		s += "\n"
	}

	// API Services Section
	apiServices := m.getAPIServices()
	if len(apiServices) > 0 {
		s += sectionHeaderStyle.Render(fmt.Sprintf("━━━ API Services (%d) ━━━", len(apiServices)))
		s += "\n"
		s += m.renderServiceList(apiServices)
		s += "\n"
	}

	// Frontend Section
	angularServices := m.getFrontendServices()
	if len(angularServices) > 0 {
		s += sectionHeaderStyle.Render("━━━ Frontend ━━━")
		s += "\n"
		s += m.renderServiceList(angularServices)
		s += "\n"
	}

	// SSH Tunnels Section
	sshServices := m.getSSHTunnelServices()
	if len(sshServices) > 0 {
		s += sectionHeaderStyle.Render("━━━ SSH Tunnels ━━━")
		s += "\n"
		s += m.renderServiceList(sshServices)
		s += "\n"
	}

	// Footer
	s += footerStyle.Render("Press q or Ctrl+C to exit • Updates every 3 seconds")

	return s
}

func (m monitorModel) getDockerServices() []ServiceHealth {
	var dockerSvcs []ServiceHealth
	for _, svc := range m.services {
		if svc.Name == "Docker Desktop" {
			dockerSvcs = append(dockerSvcs, svc)
		}
	}
	return dockerSvcs
}

func (m monitorModel) getSSHTunnelServices() []ServiceHealth {
	var sshSvcs []ServiceHealth
	for _, svc := range m.services {
		if svc.Name == "DigitalOcean" {
			sshSvcs = append(sshSvcs, svc)
		}
	}
	return sshSvcs
}

func (m monitorModel) getFrontendServices() []ServiceHealth {
	var frontend []ServiceHealth
	for _, svc := range m.services {
		if svc.Name == "Angular" {
			frontend = append(frontend, svc)
		}
	}
	return frontend
}

func (m monitorModel) getDatabaseServices() []ServiceHealth {
	var database []ServiceHealth
	for _, svc := range m.services {
		if svc.Name == "MongoDB" {
			database = append(database, svc)
		}
	}
	return database
}

func (m monitorModel) getAPIServices() []ServiceHealth {
	var apis []ServiceHealth
	for _, svc := range m.services {
		if svc.Name != "MongoDB" && svc.Name != "Angular" && svc.Name != "Docker Desktop" && svc.Name != "DigitalOcean" {
			apis = append(apis, svc)
		}
	}
	return apis
}

func (m monitorModel) renderServiceList(services []ServiceHealth) string {
	var s string
	for _, svc := range services {
		// Status indicator
		var statusIcon string
		if svc.Status == "running" {
			statusIcon = statusRunningStyle.Render("●")
		} else {
			statusIcon = statusDownStyle.Render("●")
		}

		// Service line: ● Service Name       :8080
		var line string
		if svc.Port == 0 {
			// Docker Desktop doesn't have a port
			line = fmt.Sprintf("%s %-25s",
				statusIcon,
				svc.Name,
			)
		} else {
			line = fmt.Sprintf("%s %-25s :%-6d",
				statusIcon,
				svc.Name,
				svc.Port,
			)
		}

		s += "  " + line + "\n"
	}
	return s
}

func (m monitorModel) buildTableRows() []table.Row {
	rows := []table.Row{}
	for _, svc := range m.services {
		var statusIcon string
		if svc.Status == "running" {
			statusIcon = "●"
		} else {
			statusIcon = "✗"
		}

		rows = append(rows, table.NewRow(table.RowData{
			"status":  statusIcon,
			"name":    svc.Name,
			"port":    fmt.Sprintf(":%d", svc.Port),
			"latency": svc.Status,
		}))
	}
	return rows
}

// Commands
func tickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func checkHealthCmd() tea.Cmd {
	return func() tea.Msg {
		var services []ServiceHealth

		// Check Docker Desktop
		dockerRunning := docker.CheckRunning() == nil
		services = append(services, ServiceHealth{
			Name:   "Docker Desktop",
			Port:   0, // Docker Desktop doesn't have a specific port
			Status: getStatus(dockerRunning),
		})

		// Check DigitalOcean SSH Tunnel (to production MongoDB)
		prodTunnelStatus := health.CheckPort(config.MongoProdPort)
		services = append(services, ServiceHealth{
			Name:   "DigitalOcean",
			Port:   config.MongoProdPort,
			Status: getStatus(prodTunnelStatus.Open),
		})

		// Check MongoDB
		mongoStatus := health.CheckPort(config.MongoDevPort)
		services = append(services, ServiceHealth{
			Name:   "MongoDB",
			Port:   config.MongoDevPort,
			Status: getStatus(mongoStatus.Open),
		})

		// Check Angular
		angularStatus := health.CheckPort(config.AngularPort)
		services = append(services, ServiceHealth{
			Name:   "Angular",
			Port:   config.AngularPort,
			Status: getStatus(angularStatus.Open),
		})

		// Check API services
		for _, svc := range config.APIServices {
			status := health.CheckPort(svc.Port)
			services = append(services, ServiceHealth{
				Name:   svc.Name,
				Port:   svc.Port,
				Status: getStatus(status.Open),
			})
		}

		return healthCheckMsg{services: services}
	}
}

func getStatus(open bool) string {
	if open {
		return "running"
	}
	return "down"
}

func getLatency(status health.PortStatus) string {
	if !status.Open {
		return "timeout"
	}
	return health.FormatLatency(status.Latency)
}
