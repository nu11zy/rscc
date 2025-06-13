package pprint

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	Bold    = lipgloss.NewStyle().Bold(true)
	Blue    = lipgloss.NewStyle().Foreground(lipgloss.Blue)
	Green   = lipgloss.NewStyle().Foreground(lipgloss.Green)
	Cyan    = lipgloss.NewStyle().Foreground(lipgloss.Cyan)
	Red     = lipgloss.NewStyle().Foreground(lipgloss.Red)
	Magenta = lipgloss.NewStyle().Foreground(lipgloss.Magenta)
	Yellow  = lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	Black   = lipgloss.NewStyle().Foreground(lipgloss.Black)

	SuccessPrefix = Green.Render("[+]")
	InfoPrefix    = Blue.Render("[i]")
	WarnPrefix    = Yellow.Render("[*]")
	ErrorPrefix   = Red.Render("[-]")
)

func Error(format string, a ...any) string {
	return ErrorPrefix + " " + fmt.Sprintf(format, a...)
}

func Warn(format string, a ...any) string {
	return WarnPrefix + " " + fmt.Sprintf(format, a...)
}

func Info(format string, a ...any) string {
	return InfoPrefix + " " + fmt.Sprintf(format, a...)
}

func Success(format string, a ...any) string {
	return SuccessPrefix + " " + fmt.Sprintf(format, a...)
}

func Table(headers []string, rows [][]string) string {
	t := table.NewWriter()

	// Headers
	headerRow := make(table.Row, len(headers))
	for i, h := range headers {
		headerRow[i] = Bold.Render(h)
	}
	t.AppendHeader(headerRow)

	// Rows
	for _, row := range rows {
		tableRow := make(table.Row, len(row))
		for i, cell := range row {
			tableRow[i] = cell
		}
		t.AppendRow(tableRow)
		t.AppendSeparator()
	}

	t.SetStyle(table.StyleLight)
	return t.Render()
}

func TruncateString(s string, maxLen int) string {
	if strings.Contains(s, "\n") {
		lines := strings.Split(s, "\n")
		for i, line := range lines {
			if len(line) > maxLen {
				lines[i] = line[:maxLen-3] + "..."
			}
		}
		return strings.Join(lines, "\n")
	}

	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}

	return s
}

func GetBanner() string {
	banner := fmt.Sprintf("> %s  %s\n", Green.Render("┳━┓┏━┓┏━┓┏━┓"), Bold.Render("RSCC - v0.1"))
	banner += fmt.Sprintf("> %s  %s\n", Green.Render("┣┳┛┗━┓┃  ┃  "), "Reverse SSH Command & Control")
	banner += fmt.Sprintf("> %s  %s\n", Green.Render("┛┗━┗━┛┗━┛┗━┛"), Blue.Render("https://github.com/nu11zy/rscc"))
	return banner
}
