package agentcmd

import (
	"fmt"
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

	cmd.Printf("%s %s\n", pprint.Blue.Render("ID:"), agent.ID)
	cmd.Printf("%s %s\n", pprint.Blue.Render("Name:"), agent.Name)
	cmd.Printf("%s %s\n", pprint.Blue.Render("OS/Arch:"), fmt.Sprintf("%s/%s", agent.Os, agent.Arch))
	if agent.Comment != "" {
		cmd.Printf("%s %s\n", pprint.Blue.Render("Comment:"), agent.Comment)
	}
	cmd.Printf("%s %d\n", pprint.Blue.Render("Callbacks:"), agent.Callbacks)
	cmd.Printf("%s %s\n", pprint.Blue.Render("Created:"), agent.CreatedAt.Format("2006-01-02 15:04:05"))
	cmd.Printf("%s %s\n", pprint.Blue.Render("Servers:"), strings.Join(agent.Servers, ", "))
	if len(buildFeutures) > 0 {
		cmd.Printf("%s %s\n", pprint.Blue.Render("Features:"), strings.Join(buildFeutures, ", "))
	}
	if len(agent.Subsystems) > 0 {
		cmd.Printf("%s %s\n", pprint.Blue.Render("Subsystems:"), strings.Join(agent.Subsystems, ", "))
	}
	if agent.URL != "" {
		cmd.Printf("%s %s\n", pprint.Blue.Render("URL:"), agent.URL)
		cmd.Printf("%s %d\n", pprint.Blue.Render("Downloads:"), agent.Downloads)
	}
	cmd.Printf("%s %s\n", pprint.Blue.Render("Path:"), agent.Path)
	cmd.Printf("%s %s", pprint.Blue.Render("Public Key:"), agent.PublicKey)
	return nil
}
