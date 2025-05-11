package agentcmd

import (
	"fmt"
	"os"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"rscc/internal/database/ent"

	"github.com/spf13/cobra"
)

func (a *AgentCmd) newCmdRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <id>",
		Short:   "Remove agent",
		Long:    "Remove agent\nWARNING: all active agents will no longer be able to connect to the server",
		Aliases: []string{"r", "rm"},
		Args:    cobra.ExactArgs(1),
		RunE:    a.cmdRemove,
	}

	return cmd
}

func (a *AgentCmd) cmdRemove(cmd *cobra.Command, args []string) error {
	id := args[0]
	if len(id) != constants.IDLength {
		return fmt.Errorf("invalid agent id: %s", id)
	}

	// TODO: Ask user to confirm

	agent, err := a.db.GetAgentByID(cmd.Context(), id)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("agent '%s' not found", id)
		}
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(agent.Path); os.IsNotExist(err) {
		cmd.Println(pprint.Warn("Agent file '%s' not found", agent.Path))
	} else {
		err = os.Remove(agent.Path)
		if err != nil {
			cmd.Println(pprint.Warn("Failed to delete agent file: %v", err))
		}
	}

	err = a.db.DeleteAgent(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	cmd.Println(pprint.Success("Agent '%s' removed\n", pprint.Blue.Sprint(agent.Name)))
	return nil
}
