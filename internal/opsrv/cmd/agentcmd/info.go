package agentcmd

import (
	"fmt"
	"path/filepath"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"rscc/internal/database/ent"
	"strings"

	"github.com/spf13/cobra"
)

func (a *AgentCmd) newCmdInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info",
		Short:   "Get agent info",
		Example: "agent info <id>",
		Aliases: []string{"i"},
		Args:    cobra.ExactArgs(1),
		RunE:    a.cmdInfo,
	}

	return cmd
}

func (a *AgentCmd) cmdInfo(cmd *cobra.Command, args []string) error {
	id := args[0]
	if len(id) != constants.IDLength {
		return fmt.Errorf("invalid agent id: %s", id)
	}

	agent, err := a.db.GetAgentByID(cmd.Context(), id)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("agent '%s' not found", id)
		}
		return fmt.Errorf("failed to get agent: %w", err)
	}

	buildFeutures := []string{}
	if agent.Shared {
		buildFeutures = append(buildFeutures, "shared")
	}
	if agent.Pie {
		buildFeutures = append(buildFeutures, "pie")
	}
	if agent.Garble {
		buildFeutures = append(buildFeutures, "garble")
	}

	fullPath, err := filepath.Abs(agent.Path)
	if err != nil {
		return fmt.Errorf("failed to get full path to agent: %w", err)
	}

	cmd.Println(pprint.Info("Agent info:"))
	cmd.Printf(" %s\t\t%s\n", pprint.Blue.Sprint("├─ ID:"), agent.ID)
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Name:"), agent.Name)
	cmd.Printf(" %s\t\t%s/%s\n", pprint.Blue.Sprint("├─ OS:"), agent.Os, agent.Arch)
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Servers:"), strings.Join(agent.Servers, ", "))

	if len(buildFeutures) > 0 {
		cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Features:"), strings.Join(buildFeutures, ", "))
	}
	if len(agent.Subsystems) > 0 {
		cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Subsystems:"), strings.Join(agent.Subsystems, ", "))
	}

	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Created:"), agent.CreatedAt.Format("2006-01-02 15:04:05"))
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Path:"), fullPath)
	cmd.Printf(" %s\t%d\n", pprint.Blue.Sprint("├─ Hits:"), agent.Hits)
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("└─ Public Key:"), agent.PublicKey)

	return nil
}
