package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Style renders styled text using Gum
func Style(text string, opts ...string) {
	args := append([]string{"style"}, opts...)
	args = append(args, text)
	cmd := exec.Command("gum", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// Header renders a styled header with border
func Header(text string) {
	Style(text,
		"--border", "double",
		"--border-foreground", "212",
		"--padding", "1 2",
		"--align", "center",
		"--width", "60",
	)
}

// Success renders a success message
func Success(text string) {
	Style(fmt.Sprintf("✓ %s", text),
		"--foreground", "212",
		"--bold",
	)
}

// Error renders an error message
func Error(text string) {
	Style(fmt.Sprintf("✗ %s", text),
		"--foreground", "196",
		"--bold",
	)
}

// Info renders an info message
func Info(text string) {
	Style(fmt.Sprintf("ℹ %s", text),
		"--foreground", "214",
	)
}

// Warning renders a warning message
func Warning(text string) {
	Style(fmt.Sprintf("⚠ %s", text),
		"--foreground", "214",
		"--bold",
	)
}

// Spin runs a command with a spinner
func Spin(title string, command string, args ...string) error {
	spinArgs := []string{"spin", "--spinner", "dot", "--title", title, "--"}
	spinArgs = append(spinArgs, command)
	spinArgs = append(spinArgs, args...)

	cmd := exec.Command("gum", spinArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Confirm shows a confirmation prompt
func Confirm(prompt string, defaultYes bool) bool {
	args := []string{"confirm", prompt}
	if defaultYes {
		args = append(args, "--default")
	}

	cmd := exec.Command("gum", args...)
	err := cmd.Run()
	return err == nil // gum confirm returns exit code 0 for yes, 1 for no
}

// ServiceStatus renders a service status line
func ServiceStatus(name string, status string, port int, latency string) {
	var symbol, color string

	switch status {
	case "running":
		symbol = "✓"
		color = "212"
	case "down":
		symbol = "✗"
		color = "196"
	case "checking":
		symbol = "⠿"
		color = "214"
	default:
		symbol = "○"
		color = "246"
	}

	var text string
	if latency == "" {
		text = fmt.Sprintf("%s %-20s :%-5d", symbol, name, port)
	} else {
		text = fmt.Sprintf("%s %-20s :%-5d [%s]", symbol, name, port, latency)
	}

	Style(text, "--foreground", color)
}

// Box renders text in a bordered box
func Box(title string, content string) {
	fullText := title
	if content != "" {
		fullText = fmt.Sprintf("%s\n\n%s", title, content)
	}

	Style(fullText,
		"--border", "rounded",
		"--border-foreground", "212",
		"--padding", "1 2",
		"--margin", "1",
	)
}

// List renders a list of items
func List(items []string, header string) {
	if header != "" {
		Style(header,
			"--foreground", "212",
			"--bold",
			"--margin", "1 0",
		)
	}

	for _, item := range items {
		fmt.Printf("  • %s\n", item)
	}
}

// Table renders data in a simple table format using Gum
func Table(headers []string, rows [][]string) {
	// Build CSV format for gum table
	var lines []string
	lines = append(lines, strings.Join(headers, ","))

	for _, row := range rows {
		lines = append(lines, strings.Join(row, ","))
	}

	csvData := strings.Join(lines, "\n")

	cmd := exec.Command("gum", "table")
	cmd.Stdin = strings.NewReader(csvData)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// ClearScreen clears the terminal screen without flickering
// Uses ANSI escape codes to move cursor to top instead of full clear
func ClearScreen() {
	// Move cursor to home position (top-left) and clear from cursor to end of screen
	fmt.Print("\033[H\033[2J")
}
