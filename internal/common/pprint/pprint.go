package pprint

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	SuccessColor = color.New(color.FgGreen)
	InfoColor    = color.New(color.FgBlue)
	WarnColor    = color.New(color.FgYellow)
	ErrorColor   = color.New(color.FgRed)

	SuccessPrefix = SuccessColor.Sprintf("[+]")
	InfoPrefix    = InfoColor.Sprintf("[i]")
	WarnPrefix    = WarnColor.Sprintf("[*]")
	ErrorPrefix   = ErrorColor.Sprintf("[-]")
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
		headerRow[i] = InfoColor.Sprint(h)
	}
	t.AppendHeader(headerRow)

	// Rows
	for _, row := range rows {
		tableRow := make(table.Row, len(row))
		for i, cell := range row {
			tableRow[i] = truncateString(cell, 32)
		}
		t.AppendRow(tableRow)
	}

	t.SetStyle(table.StyleLight)
	return t.Render()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func GetBanner() string {
	banner := fmt.Sprintf("> %s  %s\n", SuccessColor.Sprint("┳━┓┏━┓┏━┓┏━┓"), color.New(color.Bold).Sprint("RSCC - v0.1"))
	banner += fmt.Sprintf("> %s  %s\n", SuccessColor.Sprint("┣┳┛┗━┓┃  ┃  "), "Reverse SSH Command & Control")
	banner += fmt.Sprintf("> %s  %s\n", SuccessColor.Sprint("┛┗━┗━┛┗━┛┗━┛"), InfoColor.Sprint("https://github.com/nu11zy/rscc"))
	return banner
}
