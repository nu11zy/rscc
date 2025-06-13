package agentcmd

import (
	"fmt"
	"os"
	"rscc/internal/common/pprint"
	"rscc/internal/database/ent"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/charmbracelet/lipgloss/v2"
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

	cmd.Print(renderAgentList(agents))
	return nil
}

func renderAgentList(agents []*ent.Agent) string {
	padding := 0
	if len(agents) > 9 {
		padding = 1
	}

	result := ""
	for i, agent := range agents {
		id := pprint.Green.Render(agent.ID)
		osArch := pprint.Blue.Render(fmt.Sprintf("%s/%s", agent.Os, agent.Arch))
		callbacks := pprint.Cyan.Render(fmt.Sprintf("%d", agent.Callbacks))

		var status string
		name := agent.Name
		agentBytes, err := os.ReadFile(agent.Path)
		if err != nil {
			if os.IsNotExist(err) {
				status = pprint.Red.Render("deleted")
				name = pprint.Red.Render(agent.Name)
			} else {
				status = pprint.Red.Render("error")
				name = pprint.Red.Render(agent.Name)
			}
		} else {
			agentHash := strconv.FormatUint(xxhash.Sum64(agentBytes), 10)
			if agentHash != agent.Xxhash {
				status = pprint.Yellow.Render("modified")
				name = pprint.Yellow.Render(agent.Name)
			}
		}

		if status != "" {
			result += fmt.Sprintf("%*d: %s: %s [%s] (callbacks: %s) <%s>\n", padding+1, i+1, id, name, osArch, callbacks, status)
		} else {
			result += fmt.Sprintf("%*d: %s: %s [%s] (callbacks: %s)\n", padding+1, i+1, id, name, osArch, callbacks)
		}

		if agent.URL != "" {
			line := lipgloss.NewStyle().PaddingLeft(13 + padding).Render("url =")
			downloads := pprint.Cyan.Render(fmt.Sprintf("%d", agent.Downloads))
			if !agent.Hosted {
				result += fmt.Sprintf("%s %s (downloads: %s) <%s>\n", line, pprint.Red.Render(agent.URL), downloads, pprint.Red.Render("disabled"))
			} else {
				result += fmt.Sprintf("%s %s (downloads: %s)\n", line, pprint.Magenta.Render(agent.URL), downloads)
			}
		}

		if agent.Comment != "" {
			line := lipgloss.NewStyle().PaddingLeft(13 + padding).Render("comment =")
			result += fmt.Sprintf("%s %s\n", line, agent.Comment)
		}
	}

	return result
}
