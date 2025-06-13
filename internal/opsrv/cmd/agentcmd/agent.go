package agentcmd

import (
	"rscc/internal/database"

	"github.com/spf13/cobra"
)

type AgentCmd struct {
	Command  *cobra.Command
	db       *database.Database
	addr     string
	dataPath string
}

func NewAgentCmd(db *database.Database, dataPath, addr string) *AgentCmd {
	agentCmd := &AgentCmd{
		db:       db,
		dataPath: dataPath,
		addr:     addr,
	}

	agentCmd.Command = &cobra.Command{
		Use:     "agent",
		Short:   "Agent management",
		Aliases: []string{"a"},
		Args:    cobra.NoArgs,
	}

	agentCmd.Command.AddCommand(agentCmd.newCmdList())
	agentCmd.Command.AddCommand(agentCmd.newCmdGenerate())
	agentCmd.Command.AddCommand(agentCmd.newCmdInfo())
	agentCmd.Command.AddCommand(agentCmd.newCmdRemove())
	agentCmd.Command.AddCommand(agentCmd.newCmdHost())
	agentCmd.Command.AddCommand(agentCmd.newCmdComment())
	return agentCmd
}
