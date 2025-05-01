package pprint

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	successColor = color.New(color.FgGreen)
	infoColor    = color.New(color.FgBlue)
	warnColor    = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed)

	SuccessPrefix = successColor.Sprintf("[+]")
	InfoPrefix    = infoColor.Sprintf("[i]")
	WarnPrefix    = warnColor.Sprintf("[*]")
	ErrorPrefix   = errorColor.Sprintf("[-]")
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
		headerRow[i] = infoColor.Sprint(h)
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
