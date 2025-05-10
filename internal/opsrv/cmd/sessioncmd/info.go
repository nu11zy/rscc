package sessioncmd

import (
	"rscc/internal/common/pprint"
	"strings"

	"github.com/spf13/cobra"
)

func (s *SessionCmd) newCmdInfo() *cobra.Command {
	return &cobra.Command{
		Use:     "info <id>",
		Short:   "Get information about a session",
		Aliases: []string{"i"},
		Args:    cobra.ExactArgs(1),
		RunE:    s.cmdInfo,
	}
}

func (s *SessionCmd) cmdInfo(cmd *cobra.Command, args []string) error {
	id := args[0]

	session, ok := s.sm.GetSession(id)
	if !ok {
		cmd.Println(pprint.Info("No sessions found"))
		return nil
	}

	cmd.Printf("- Session ID: %s\n", session.ID)
	cmd.Printf("- Username: %s\n", session.Metadata.Username)
	cmd.Printf("- Hostname: %s\n", session.Metadata.Hostname)
	if session.Metadata.Domain != "" {
		cmd.Printf("- Domain: %s\n", session.Metadata.Domain)
	}
	if len(session.Metadata.IPs) > 0 {
		cmd.Printf("- IPs: [%s]\n", strings.Join(session.Metadata.IPs, ", "))
	}
	cmd.Printf("- OS: %s\n", session.Metadata.OSMeta)
	cmd.Printf("- ProcName: %s\n", session.Metadata.ProcName)
	cmd.Printf("- IsPriv: %t\n", session.Metadata.IsPriv)
	if session.Metadata.Extra != "" {
		cmd.Printf("- Extra: %s\n", session.Metadata.Extra)
	}

	return nil
}
