package sessioncmd

import "github.com/spf13/cobra"

func (s *SessionCmd) newCmdInfo() *cobra.Command {
	return &cobra.Command{
		Use:   "info <id>",
		Short: "Get information about a session",
		Args:  cobra.ExactArgs(1),
		RunE:  s.cmdInfo,
	}
}

func (s *SessionCmd) cmdInfo(cmd *cobra.Command, args []string) error {
	id := args[0]

	cmd.Println("Session ID:", id)
	cmd.Println("Not implemented")

	return nil
}
