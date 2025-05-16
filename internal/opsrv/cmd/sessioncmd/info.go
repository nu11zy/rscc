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

	cmd.Println(pprint.Info("Session info:"))
	cmd.Printf(" %s\t\t%s\n", pprint.Blue.Sprint("├─ ID:"), session.ID)
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Created:"), session.CreatedAt.Format("02.01.2006 15:04:05"))
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Username:"), session.Metadata.Username)
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Hostname:"), session.Metadata.Hostname)
	if session.Metadata.Domain != "" {
		cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Domain:"), session.Metadata.Domain)
	}
	if len(session.Metadata.IPs) > 0 {
		cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ IPs:"), strings.Join(session.Metadata.IPs, ", "))
	}
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Process:"), session.Metadata.ProcName)
	cmd.Printf(" %s\t%t\n", pprint.Blue.Sprint("├─ IsAdmin:"), session.Metadata.IsPriv)
	if session.Metadata.Extra != "" {
		cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Extra:"), session.Metadata.Extra)
	}
	cmd.Printf(" %s\t\t%s\n", pprint.Blue.Sprint("└─ OS:"), session.Metadata.OSMeta)
	cmd.Println()
	return nil
}
