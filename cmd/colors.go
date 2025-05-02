package cmd

import (
	"github.com/fatih/color"
)

// Color definitions for CLI output
var (
	// ErrorColor is used for error messages
	ErrorColor = color.New(color.FgRed).Add(color.Bold)
	// SuccessColor is used for success messages
	SuccessColor = color.New(color.FgGreen).Add(color.Bold)
	// WarnColor is used for warning messages
	WarnColor = color.New(color.FgYellow).Add(color.Bold)
	// InfoColor is used for informational messages
	InfoColor = color.New(color.FgCyan)
)
