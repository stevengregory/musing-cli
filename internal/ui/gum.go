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

// HeaderGradient renders a header with multi-color gradient effect
func HeaderGradient(text string) {
	// Split text into words for gradient effect
	words := strings.Fields(text)
	colors := []string{"212", "213", "177", "141"} // Magenta to purple gradient

	styledWords := make([]string, len(words))
	for i, word := range words {
		color := colors[i%len(colors)]
		styledWords[i] = colorText(word, color)
	}

	fullText := strings.Join(styledWords, " ")

	// Render with border but no color (colors are in the text itself)
	Style(fullText,
		"--border", "double",
		"--border-foreground", "212",
		"--padding", "1 2",
		"--align", "center",
		"--width", "60",
	)
}

// HeaderMinimal renders a clean header with top/bottom borders only
func HeaderMinimal(text string) {
	// Simple divider line
	divider := strings.Repeat("─", 60)

	fmt.Println()
	Style(divider, "--foreground", "212")
	Style(text,
		"--foreground", "212",
		"--bold",
		"--align", "center",
		"--width", "60",
	)
	Style(divider, "--foreground", "212")
	fmt.Println()
}

// HeaderBold renders a bold header with underline
func HeaderBold(text string) {
	Style(text,
		"--foreground", "212",
		"--bold",
		"--underline",
		"--align", "center",
		"--width", "60",
		"--margin", "1 0",
	)
}

// colorText wraps text with gum style inline color
func colorText(text string, color string) string {
	cmd := exec.Command("gum", "style", "--foreground", color, text)
	output, err := cmd.Output()
	if err != nil {
		return text
	}
	return strings.TrimSpace(string(output))
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

// Confirm has been moved to confirm.go and now uses Bubble Tea (Huh)
// This placeholder is kept for reference - the actual implementation is in confirm.go

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
