package sessioncmd

import (
	"rscc/internal/session"

	"github.com/spf13/cobra"
)

type SessionCmd struct {
	Command *cobra.Command
	sm      *session.SessionManager
}

func NewSessionCmd(sm *session.SessionManager) *SessionCmd {
	sessionCmd := &SessionCmd{
		sm: sm,
	}

	cmd := &cobra.Command{
		Use:     "session",
		Short:   "Session management",
		Aliases: []string{"s"},
		Args:    cobra.NoArgs,
	}

	sessionCmd.Command = cmd
	cmd.AddCommand(sessionCmd.newCmdList())

	return sessionCmd
}
