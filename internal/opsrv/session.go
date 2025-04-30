package opsrv

import (
	"github.com/spf13/cobra"
)

func (s *OperatorServer) NewSessionCommand() *cobra.Command {
	sessionCmd := &cobra.Command{
		Use:     "session",
		Short:   "Session management",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE:    s.sessionList,
	}

	return sessionCmd
}

func (s *OperatorServer) sessionList(cmd *cobra.Command, args []string) error {
	sessions := s.sm.ListSessions()
	for _, session := range sessions {
		cmd.Printf("%s\t%s\t%s\n", session.ID, session.Metadata.Username, session.Metadata.Hostname)
	}
	return nil
}
