package pprint

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	Bold   = color.New(color.Bold)
	Blue   = color.New(color.FgBlue)
	Green  = color.New(color.FgGreen)
	Cyan   = color.New(color.FgCyan)
	Red    = color.New(color.FgRed)
	Purple = color.New(color.FgMagenta)
	Yellow = color.New(color.FgYellow)
	Gray   = color.New(color.FgHiBlack)
	Reset  = color.New(color.Reset)

	SuccessPrefix = Green.Sprintf("[+]")
	InfoPrefix    = Blue.Sprintf("[i]")
	WarnPrefix    = Yellow.Sprintf("[*]")
	ErrorPrefix   = Red.Sprintf("[-]")
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
		headerRow[i] = Bold.Sprint(h)
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
	banner := fmt.Sprintf("> %s  %s\n", Green.Sprint("┳━┓┏━┓┏━┓┏━┓"), Bold.Sprint("RSCC - v0.1"))
	banner += fmt.Sprintf("> %s  %s\n", Green.Sprint("┣┳┛┗━┓┃  ┃  "), "Reverse SSH Command & Control")
	banner += fmt.Sprintf("> %s  %s\n", Green.Sprint("┛┗━┗━┛┗━┛┗━┛"), Blue.Sprint("https://github.com/nu11zy/rscc"))
	banner += "\n"
	return banner
}
