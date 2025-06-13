package agentcmd

import (
	"fmt"
	"os"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"rscc/internal/common/validators"
	"rscc/internal/database/ent"

	"github.com/spf13/cobra"
)

func (a *AgentCmd) newCmdRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "Remove agent",
		Example: "agent remove <id>",
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

	if !validators.ValidateFileExists(agent.Path) {
		return fmt.Errorf("agent file '%s' not found", agent.Path)
	}

	err = os.Remove(agent.Path)
	if err != nil {
		return fmt.Errorf("failed to delete agent file: %w", err)
	}

	err = a.db.DeleteAgent(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	cmd.Println(pprint.Success("Agent '%s' removed", pprint.Blue.Render(agent.Name)))
	return nil
}
