package sessioncmd

import (
	"fmt"
	"rscc/internal/common/pprint"
	"rscc/internal/session"
	"strconv"
	"time"

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

	// var rows [][]string
	// for _, session := range sessions {
	// 	id := pprint.Blue.Render(session.ID)
	// 	var username string
	// 	if session.Metadata.Domain != "" {
	// 		username = fmt.Sprintf("%s/%s", session.Metadata.Username, session.Metadata.Domain)
	// 	} else {
	// 		username = session.Metadata.Username
	// 	}
	// 	if session.Metadata.IsPriv {
	// 		username = pprint.Red.Render(fmt.Sprintf("%s (*)", username))
	// 	}
	// 	osMeta := strings.Split(session.Metadata.OSMeta, ":")
	// 	rows = append(rows, []string{id, username, session.Metadata.Hostname, osMeta[0], session.CreatedAt.Format("02.01.2006 15:04:05")})
	// }

	// cmd.Println(pprint.Table([]string{"ID", "Username", "Hostname", "OS", "Created"}, rows))
	// cmd.Println()
	cmd.Print(s.renderSessionList(sessions))
	return nil
}

func (s *SessionCmd) renderSessionList(sessions []*session.Session) string {
	result := ""
	padding := len(strconv.Itoa(len(sessions)))

	for i, session := range sessions {
		id := pprint.Green.Render(session.ID)
		remoteAddr := pprint.Magenta.Render(session.RemoteAddr)

		var userHost string
		if session.Metadata.Domain != "" {
			userHost = fmt.Sprintf("%s\\%s@%s", session.Metadata.Username, session.Metadata.Domain, session.Metadata.Hostname)
		} else {
			userHost = fmt.Sprintf("%s@%s", session.Metadata.Username, session.Metadata.Hostname)
		}

		if session.Metadata.IsPriv {
			userHost = fmt.Sprintf("%s %s", userHost, pprint.Red.Render("(*)"))
		}

		duration := time.Since(session.CreatedAt)
		createdAt := pprint.Cyan.Render(duration.Round(time.Second).String())

		result += fmt.Sprintf("%*d: %s: %s [%s] <%s>\n", padding, i+1, id, userHost, remoteAddr, createdAt)
	}

	return result
}
