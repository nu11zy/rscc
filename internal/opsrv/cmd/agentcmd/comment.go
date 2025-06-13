package agentcmd

import (
	"fmt"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"

	"github.com/spf13/cobra"
)

func (a *AgentCmd) newCmdComment() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "comment",
		Short:   "Comment agent",
		Example: "agent comment <id> <comment>",
		Aliases: []string{"c"},
		Args:    cobra.MinimumNArgs(1),
		RunE:    a.cmdComment,
	}
	cmd.Flags().BoolP("remove", "r", false, "remove comment")

	return cmd
}

func (a *AgentCmd) cmdComment(cmd *cobra.Command, args []string) error {
	id := args[0]
	if len(id) != constants.IDLength {
		return fmt.Errorf("invalid agent id: %s", id)
	}

	remove, err := cmd.Flags().GetBool("remove")
	if err != nil {
		return err
	}
	if remove {
		err = a.db.UpdateAgentComment(cmd.Context(), id, "")
		if err != nil {
			return fmt.Errorf("failed to remove agent comment: %w", err)
		}
		cmd.Println(pprint.Success("Agent comment removed"))
		return nil
	}

	if len(args) < 2 {
		return fmt.Errorf("comment is required")
	}

	comment := args[1]
	if comment == "" {
		return fmt.Errorf("comment is required")
	}

	err = a.db.UpdateAgentComment(cmd.Context(), id, comment)
	if err != nil {
		return fmt.Errorf("failed to update agent comment: %w", err)
	}

	cmd.Println(pprint.Success("Agent comment updated"))
	return nil
}
