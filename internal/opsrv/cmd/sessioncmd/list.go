package sessioncmd

import (
	"rscc/internal/common/pprint"

	"github.com/spf13/cobra"
)

func (s *SessionCmd) newCmdList() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "list sessions",
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
		rows = append(rows, []string{session.ID, "mock", session.Metadata.Username, session.Metadata.Hostname})
	}

	cmd.Println(pprint.Table([]string{"ID", "OS", "Username", "Hostname"}, rows))
	return nil
}
