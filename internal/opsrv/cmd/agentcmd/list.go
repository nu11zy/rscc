package agentcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/spf13/cobra"
)

func (a *AgentCmd) newCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "list agents",
		Aliases: []string{"l", "ls"},
		Args:    cobra.NoArgs,
		RunE:    a.cmdList,
	}

	return cmd
}

func (a *AgentCmd) cmdList(cmd *cobra.Command, args []string) error {
	agents, err := a.db.GetAllAgents(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get agents: %w", err)
	}
	if len(agents) == 0 {
		cmd.Println(pprint.Info("No agents found"))
		return nil
	}

	rows := make([][]string, len(agents))
	for i, agent := range agents {
		// Check if agent file exists
		agentBytes, err := os.ReadFile(filepath.Join(constants.AgentDir, agent.Name))
		if err != nil {
			if os.IsNotExist(err) {
				rows[i] = []string{agent.ID, pprint.ErrorColor.Sprint(agent.Name), agent.Os, agent.Arch, agent.Server}
				continue
			} else {
				return fmt.Errorf("failed to read agent file %s: %w", agent.ID, err)
			}
		}

		// Check if agent file is modified
		agentHash := strconv.FormatUint(xxhash.Sum64(agentBytes), 10)
		if agentHash != agent.Xxhash {
			rows[i] = []string{agent.ID, pprint.WarnColor.Sprint(agent.Name), agent.Os, agent.Arch, agent.Server}
		} else {
			rows[i] = []string{agent.ID, agent.Name, agent.Os, agent.Arch, agent.Server}
		}
	}

	cmd.Println(pprint.Table([]string{"ID", "Name", "OS", "Arch", "Server"}, rows))
	cmd.Printf("[%s] - file not found; [%s] - file modified\n", pprint.ErrorColor.Sprint("*"), pprint.WarnColor.Sprint("*"))
	return nil
}
