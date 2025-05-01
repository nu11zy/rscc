package agentcmd

import (
	"rscc/internal/database"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

type AgentCmd struct {
	Command *cobra.Command
	db      *database.Database
}

func NewAgentCmd(publicKey ssh.PublicKey, db *database.Database) *AgentCmd {
	agentCmd := &AgentCmd{
		db: db,
	}

	cmd := &cobra.Command{
		Use:     "agent",
		Short:   "agent management",
		Aliases: []string{"a"},
		Args:    cobra.NoArgs,
	}

	agentCmd.Command = cmd
	cmd.AddCommand(agentCmd.newCmdList())
	cmd.AddCommand(agentCmd.newCmdGenerate())
	return agentCmd
}
