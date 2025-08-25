package sessioncmd

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/nu11zy/rscc/internal/common/pprint"
	"github.com/nu11zy/rscc/internal/session"

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
	comments := s.sm.ListComments()
	if len(sessions) == 0 {
		cmd.Println(pprint.Info("No sessions found"))
		return nil
	}

	cmd.Print(s.renderSessionList(sessions, comments))
	return nil
}

func (s *SessionCmd) renderSessionList(sessions []*session.Session, comments map[string]string) string {
	result := ""
	padding := len(strconv.Itoa(len(sessions)))

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})

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

		var agentComment string
		if comment, ok := comments[session.AgentID]; ok {
			agentComment = pprint.Black.Render(fmt.Sprintf("# %s", comment))
		}

		duration := time.Since(session.CreatedAt)
		createdAt := pprint.Cyan.Render(duration.Round(time.Second).String())

		result += fmt.Sprintf("%*d: %s: %s [%s] <%s> %s\n", padding, i+1, id, userHost, remoteAddr, createdAt, agentComment)
	}

	return result
}
