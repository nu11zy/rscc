package agentcmd

import (
	"fmt"
	"rscc/internal/common/constants"
	"rscc/internal/database/ent"

	"github.com/spf13/cobra"
)

func (a *AgentCmd) newCmdInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info",
		Short:   "Get agent info",
		Aliases: []string{"i"},
		Args:    cobra.ExactArgs(1),
		RunE:    a.cmdInfo,
	}

	return cmd
}

func (a *AgentCmd) cmdInfo(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	var agent *ent.Agent
	if len(idOrName) == constants.IDLength {
		var err error
		agent, err = a.db.GetAgentByID(cmd.Context(), idOrName)
		if err != nil && !ent.IsNotFound(err) {
			return fmt.Errorf("failed to get agent: %w", err)
		}
	}

	if agent == nil {
		var err error
		agent, err = a.db.GetAgentByName(cmd.Context(), idOrName)
		if err != nil && !ent.IsNotFound(err) {
			return fmt.Errorf("failed to get agent: %w", err)
		} else {
			return fmt.Errorf("agent '%s' not found", idOrName)
		}
	}

	// TODO: Pretty print
	cmd.Printf("Agent ID: %s\n", agent.ID)
	cmd.Printf("Agent Name: %s\n", agent.Name)
	cmd.Printf("Agent OS: %s\n", agent.Os)
	cmd.Printf("Agent Arch: %s\n", agent.Arch)
	cmd.Printf("Agent Server: %s\n", agent.Server)
	cmd.Printf("Agent Shared: %t\n", agent.Shared)
	cmd.Printf("Agent PIE: %t\n", agent.Pie)
	cmd.Printf("Agent Garble: %t\n", agent.Garble)
	cmd.Printf("Agent Subsystems: %v\n", agent.Subsystems)
	cmd.Printf("Agent Public Key: %s\n", agent.PublicKey)
	cmd.Printf("Agent XXHash: %s\n", agent.Xxhash)

	return nil
}
