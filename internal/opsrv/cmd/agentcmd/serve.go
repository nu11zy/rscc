package agentcmd

import (
	"fmt"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"rscc/internal/database/ent"
	"strings"

	"github.com/spf13/cobra"
)

func (a *AgentCmd) newCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "serve",
		Short:   "Serve agent",
		Example: "agent serve <id> <url>",
		Aliases: []string{"s"},
		Args:    cobra.ExactArgs(2),
		RunE:    a.cmdServe,
	}

	return cmd
}

func (a *AgentCmd) cmdServe(cmd *cobra.Command, args []string) error {
	id := args[0]
	if len(id) != constants.IDLength {
		return fmt.Errorf("invalid agent id: %s", id)
	}

	url := args[1]
	if url == "" {
		return fmt.Errorf("url is required")
	}
	if url == "/" {
		return fmt.Errorf("url cannot be '/'")
	}
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	agent, err := a.db.GetAgentByID(cmd.Context(), id)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("agent '%s' not found", id)
		}
		return fmt.Errorf("failed to get agent: %w", err)
	}

	err = a.db.UpdateAgentURL(cmd.Context(), id, url)
	if err != nil {
		return fmt.Errorf("failed to update agent url: %w", err)
	}

	cmd.Println(
		pprint.Success(
			"Agent '%s' served at %s\n\nCurl: %s\n",
			pprint.Blue.Sprint(agent.Name),
			pprint.Green.Sprint(a.addr+url),
			pprint.Yellow.Sprint("curl -skOL "+a.addr+url),
		),
	)
	return nil
}
