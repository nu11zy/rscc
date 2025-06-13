package utils

import (
	"fmt"
	"rscc/internal/common/pprint"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func CobraHelp(cmd *cobra.Command) error {
	if cmd.HasExample() {
		cmd.Println("EXAMPLE:")
		cmd.Printf("  %s\n", cmd.Example)
		cmd.Println()
	}

	if cmd.HasAvailableSubCommands() {
		maxNameLen := 0
		maxAliasesLen := 0
		for _, c := range cmd.Commands() {
			if c.IsAvailableCommand() && len(c.Name()) > maxNameLen {
				maxNameLen = len(c.Name())
			}
			aliases := sort.StringSlice(c.Aliases)
			sort.Sort(sort.Reverse(aliases))
			aliasesStr := fmt.Sprintf("[%s]", pprint.Black.Render(strings.Join(aliases, ", ")))
			if len(aliasesStr) > maxAliasesLen {
				maxAliasesLen = len(aliasesStr)
			}
		}

		cmd.Println("COMMANDS:")
		for _, c := range cmd.Commands() {
			if c.IsAvailableCommand() {
				aliases := sort.StringSlice(c.Aliases)
				sort.Sort(sort.Reverse(aliases))
				aliasesStr := fmt.Sprintf("[%s]", strings.Join(aliases, ", "))
				cmd.Printf("  %-*s %-*s    %s\n", maxNameLen, c.Name(), maxAliasesLen, pprint.Black.Render(aliasesStr), c.Short)
			}
		}
		cmd.Println()
	}

	if cmd.HasAvailableFlags() {
		cmd.Println("FLAGS:")
		cmd.Println(cmd.Flags().FlagUsages())
	}

	if cmd.HasAvailableFlags() || cmd.HasAvailableSubCommands() || cmd.HasExample() {
		cmd.Println("Use '-h / --help' for more information about a command")
	}
	return nil
}
