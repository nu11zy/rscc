package agentcmd

import (
	"rscc/internal/database"

	"github.com/spf13/cobra"
)

type AgentCmd struct {
	Command *cobra.Command
	db      *database.Database
	// used to control base directory instead of CWD
	baseDir string
}

func NewAgentCmd(db *database.Database, dir string) *AgentCmd {
	agentCmd := &AgentCmd{
		db:      db,
		baseDir: dir,
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
