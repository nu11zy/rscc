package sessioncmd

import (
	"rscc/internal/common/pprint"
	"strings"

	"github.com/spf13/cobra"
)

func (s *SessionCmd) newCmdInfo() *cobra.Command {
	return &cobra.Command{
		Use:     "info",
		Short:   "Get information about a session",
		Example: "session info <id>",
		Aliases: []string{"i"},
		Args:    cobra.ExactArgs(1),
		RunE:    s.cmdInfo,
	}
}

func (s *SessionCmd) cmdInfo(cmd *cobra.Command, args []string) error {
	id := args[0]

	session := s.sm.GetSession(id)
	if session == nil {
		cmd.Println(pprint.Info("No sessions found"))
		return nil
	}

	cmd.Printf("%s %s\n", pprint.Blue.Render("ID:"), session.ID)
	cmd.Printf("%s %s\n", pprint.Blue.Render("Created:"), session.CreatedAt.Format("02.01.2006 15:04:05"))
	cmd.Printf("%s %s\n", pprint.Blue.Render("Remote Address:"), session.RemoteAddr)
	cmd.Printf("%s %s\n", pprint.Blue.Render("Username:"), session.Metadata.Username)
	cmd.Printf("%s %s\n", pprint.Blue.Render("Hostname:"), session.Metadata.Hostname)
	if session.Metadata.Domain != "" {
		cmd.Printf("%s %s\n", pprint.Blue.Render("Domain:"), session.Metadata.Domain)
	}
	cmd.Printf("%s %s\n", pprint.Blue.Render("Process:"), session.Metadata.ProcName)
	cmd.Printf("%s %t\n", pprint.Blue.Render("Privileged:"), session.Metadata.IsPriv)
	if len(session.Metadata.IPs) > 0 {
		cmd.Printf("%s [%s]\n", pprint.Blue.Render("IPs:"), strings.Join(session.Metadata.IPs, ", "))
	}
	if session.Metadata.Extra != "" {
		cmd.Printf("%s %s\n", pprint.Blue.Render("Extra:"), session.Metadata.Extra)
	}
	cmd.Printf("%s %s\n", pprint.Blue.Render("OS:"), session.Metadata.OSMeta)
	return nil
}
