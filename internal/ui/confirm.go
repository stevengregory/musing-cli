package ui

import (
	"github.com/charmbracelet/huh"
)

// ConfirmOptions configures the confirmation prompt
type ConfirmOptions struct {
	Title       string
	Description string
	Affirmative string
	Negative    string
	Inline      bool
}

// ConfirmWithBubbles shows a Bubble Tea confirmation prompt
// Uses Huh for a native Bubble Tea experience without external dependencies
func ConfirmWithBubbles(opts ConfirmOptions) bool {
	// Set defaults
	if opts.Affirmative == "" {
		opts.Affirmative = "Yes"
	}
	if opts.Negative == "" {
		opts.Negative = "No"
	}

	var confirmed bool

	confirm := huh.NewConfirm().
		Title(opts.Title).
		Affirmative(opts.Affirmative).
		Negative(opts.Negative).
		Value(&confirmed)

	// Add optional description if provided
	if opts.Description != "" {
		confirm = confirm.Description(opts.Description)
	}

	// Use inline mode for quick, non-intrusive prompts
	if opts.Inline {
		confirm = confirm.Inline(true)
	}

	// Run the confirmation
	err := confirm.Run()
	if err != nil {
		// If there's an error (like user cancelled with Ctrl+C), treat as "no"
		return false
	}

	return confirmed
}

// Confirm shows a simple confirmation prompt (keeping backwards compatibility)
// Now uses Bubble Tea instead of Gum
func Confirm(prompt string, defaultYes bool) bool {
	var confirmed bool

	// If defaultYes is true, we show it in the title
	title := prompt
	if defaultYes {
		title = prompt + " (default: yes)"
	}

	confirm := huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed)

	err := confirm.Run()
	if err != nil {
		// On error or cancellation, return the default
		return defaultYes
	}

	return confirmed
}
