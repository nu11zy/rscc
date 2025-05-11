package agentcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/spf13/cobra"
)

func (a *AgentCmd) newCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List agents",
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
		id := agent.ID
		name := pprint.TruncateString(agent.Name, 32)
		osArch := fmt.Sprintf("%s/%s", agent.Os, agent.Arch)
		servers := pprint.TruncateString(strings.Join(agent.Servers, "\n"), 32)
		hits := strconv.Itoa(agent.Hits)

		// Check if agent file exists
		agentBytes, err := os.ReadFile(filepath.Join(constants.AgentDir, agent.Name))
		if err != nil {
			if os.IsNotExist(err) {
				rows[i] = []string{pprint.Blue.Sprint(id), pprint.Red.Sprint(name), osArch, servers, hits}
				continue
			} else {
				return fmt.Errorf("failed to read agent file %s: %w", agent.ID, err)
			}
		}

		// Check if agent file is modified
		agentHash := strconv.FormatUint(xxhash.Sum64(agentBytes), 10)
		if agentHash != agent.Xxhash {
			rows[i] = []string{pprint.Blue.Sprint(id), pprint.Yellow.Sprint(name), osArch, servers, hits}
		} else {
			rows[i] = []string{pprint.Blue.Sprint(id), name, osArch, servers, hits}
		}
	}

	cmd.Println(pprint.Table([]string{"ID", "NAME", "OS/ARCH", "SERVERS", "HITS"}, rows))
	cmd.Printf("[%s] - file not found; [%s] - file modified. Type 'agent info <id>' to get more info\n", pprint.Red.Sprint("*"), pprint.Yellow.Sprint("*"))
	return nil
}
