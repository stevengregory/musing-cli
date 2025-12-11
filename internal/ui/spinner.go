package ui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	messageStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
)

type spinnerModel struct {
	spinner  spinner.Model
	message  string
	command  string
	args     []string
	done     bool
	success  bool
	err      error
	quitting bool
}

type commandDoneMsg struct {
	success bool
	err     error
}

func initialSpinnerModel(message string, command string, args ...string) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return spinnerModel{
		spinner: s,
		message: message,
		command: command,
		args:    args,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		runCommand(m.command, m.args...),
	)
}

func runCommand(command string, args ...string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(command, args...)

		// Inherit environment variables and working directory from parent process
		cmd.Env = os.Environ()
		if wd, err := os.Getwd(); err == nil {
			cmd.Dir = wd
		}

		// Suppress output during spinner - keeps output clean
		// Open /dev/null to properly discard output
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err == nil {
			defer devNull.Close()
			cmd.Stdout = devNull
			cmd.Stderr = devNull
		}

		err = cmd.Run()
		return commandDoneMsg{
			success: err == nil,
			err:     err,
		}
	}
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case commandDoneMsg:
		m.done = true
		m.success = msg.success
		m.err = msg.err
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m spinnerModel) View() string {
	if m.quitting {
		return ""
	}

	if m.done {
		if m.success {
			return successStyle.Render("✓ " + m.message)
		}
		return errorStyle.Render("✗ " + m.message)
	}

	return fmt.Sprintf("%s %s", m.spinner.View(), messageStyle.Render(m.message))
}

// SpinWithBubbles runs a command with a beautiful Bubbles spinner
// Falls back to Gum if no TTY is available
func SpinWithBubbles(message string, command string, args ...string) error {
	// Check if we have a TTY - if not, fall back to Gum
	if !isTTY() {
		return Spin(message, command, args...)
	}

	p := tea.NewProgram(initialSpinnerModel(message, command, args...))

	model, err := p.Run()
	if err != nil {
		// Fall back to Gum on error
		return Spin(message, command, args...)
	}

	// Get the final model state
	finalModel := model.(spinnerModel)
	if !finalModel.success && finalModel.err != nil {
		return finalModel.err
	}

	return nil
}

// isTTY checks if stdout is a terminal
func isTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
