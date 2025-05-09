package agentcmd

import (
	"fmt"
	"path/filepath"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"rscc/internal/database/ent"

	"github.com/spf13/cobra"
)

func (a *AgentCmd) newCmdInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info <id/name>",
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

	cmd.Println(pprint.Info("Agent extra info:"))
	cmd.Printf("- ID: %s\n", agent.ID)
	cmd.Printf("- Name: %s\n", agent.Name)
	cmd.Printf("- OS: %s (%s)\n", agent.Os, agent.Arch)
	cmd.Printf("- Server: %s\n", agent.Server)

	if len(buildFeutures) > 0 {
		cmd.Printf("- Features: %v\n", buildFeutures)
	}
	if len(agent.Subsystems) > 0 {
		cmd.Printf("- Subsystems: %v\n", agent.Subsystems)
	}

	cmd.Printf("- Path: %s\n", fullPath)
	cmd.Printf("- Public Key: %s", agent.PublicKey)

	return nil
}
