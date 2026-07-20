// SPDX-License-Identifier: GPL-3.0-or-later
// SPDX-FileCopyrightText: 2026 The Forkinator authors

package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

func GetTerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 100 // Fallback
	}
	// Ensure a minimum width for readability
	if w < 80 {
		return 80
	}
	return w
}

var (
	Subtle  = lipgloss.AdaptiveColor{Light: "#D9D9D9", Dark: "#383838"}
	Magenta = lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}
	Cyan    = lipgloss.AdaptiveColor{Light: "#02BAEF", Dark: "#02BAEF"}
	Green   = lipgloss.AdaptiveColor{Light: "#02BA59", Dark: "#02BA59"}
	Yellow  = lipgloss.AdaptiveColor{Light: "#EAD027", Dark: "#EAD027"}
	Red     = lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#FF4672"}
	Gray    = lipgloss.AdaptiveColor{Light: "#828282", Dark: "#828282"}

	BoldStyle  = lipgloss.NewStyle().Bold(true)
	TitleStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFFFFF")).
			Bold(true)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Magenta).
			Bold(true).
			Underline(true)

	LabelStyle = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)

	BorderStyle = lipgloss.NewStyle().
			Foreground(Subtle).
			Border(lipgloss.NormalBorder(), true, false, true, false)

	UsageStyle       = lipgloss.NewStyle().Padding(1, 2)
	CommandStyle     = lipgloss.NewStyle().Foreground(Magenta).Bold(true).Width(10)
	DescriptionStyle = lipgloss.NewStyle().Foreground(Gray)

	LogoStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	HelpTitleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true).MarginBottom(1)
	HelpDescStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#AFB1B6")).Italic(true).MarginBottom(1)
	HelpSectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).MarginTop(1)
	HelpFlagStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#AFB1B6"))
)

// SuccessMsg returns a formatted success message
func SuccessMsg(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return BoldStyle.Foreground(Green).Render("✔ " + msg)
}

// ErrorMsg returns a formatted error message
func ErrorMsg(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return BoldStyle.Foreground(Red).Render("✘ Error: " + msg)
}
