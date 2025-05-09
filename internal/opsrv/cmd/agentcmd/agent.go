package agentcmd

import (
	"rscc/internal/database"

	"github.com/spf13/cobra"
)

type AgentCmd struct {
	Command *cobra.Command
	db      *database.Database
}

// + agent list
// + agent generate --name <name> --os <os> --arch <arch> --server <server> --shared --pie --garble --ss <subsystem-list>
// - agent add --name <name> --key <public-key> --os <os> --arch <arch> --server <server> --shared --pie --garble --ss <subsystem-list> <path>
// - agent remove <id>
// + agent info <id>

func NewAgentCmd(db *database.Database) *AgentCmd {
	agentCmd := &AgentCmd{
		db: db,
	}

	cmd := &cobra.Command{
		Use:     "agent",
		Short:   "Agent management",
		Aliases: []string{"a"},
		Args:    cobra.NoArgs,
	}

	agentCmd.Command = cmd
	cmd.AddCommand(agentCmd.newCmdList())
	cmd.AddCommand(agentCmd.newCmdGenerate())
	cmd.AddCommand(agentCmd.newCmdInfo())
	cmd.AddCommand(agentCmd.newCmdRemove())
	return agentCmd
}
