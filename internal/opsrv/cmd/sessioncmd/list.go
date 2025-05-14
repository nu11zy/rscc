package sessioncmd

import (
	"fmt"
	"rscc/internal/common/pprint"
	"strings"

	"github.com/spf13/cobra"
)

func (s *SessionCmd) newCmdList() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List sessions",
		Aliases: []string{"l", "ls"},
		RunE:    s.cmdList,
	}
}

func (s *SessionCmd) cmdList(cmd *cobra.Command, args []string) error {
	sessions := s.sm.ListSessions()
	if len(sessions) == 0 {
		cmd.Println(pprint.Info("No sessions found"))
		return nil
	}

	var rows [][]string
	for _, session := range sessions {
		id := pprint.Blue.Sprint(session.ID)
		var username string
		if session.Metadata.Domain != "" {
			username = fmt.Sprintf("%s/%s", session.Metadata.Username, session.Metadata.Domain)
		} else {
			username = session.Metadata.Username
		}
		if session.Metadata.IsPriv {
			username = pprint.Red.Sprintf("%s (*)", username)
		}
		osMeta := strings.Split(session.Metadata.OSMeta, ":")
		rows = append(rows, []string{id, username, session.Metadata.Hostname, osMeta[0], session.CreatedAt.Format("02.01.2006 15:04:05")})
	}

	cmd.Println(pprint.Table([]string{"ID", "Username", "Hostname", "OS", "Created"}, rows))
	cmd.Println()
	return nil
}
