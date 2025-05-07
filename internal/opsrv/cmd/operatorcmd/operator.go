package operatorcmd

import (
	"rscc/internal/database"
	"rscc/internal/sshd"

	"github.com/spf13/cobra"
)

type OperatorCmd struct {
	Command  *cobra.Command
	db       *database.Database
	operator *sshd.OperatorSession
}

func NewOperatorCmd(db *database.Database, operator *sshd.OperatorSession) *OperatorCmd {
	operatorCmd := &OperatorCmd{
		db:       db,
		operator: operator,
	}

	cmd := &cobra.Command{
		Use:     "operator",
		Short:   "Operator management",
		Aliases: []string{"o", "op"},
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(operatorCmd.newCmdList())
	cmd.AddCommand(operatorCmd.newCmdAdd())

	operatorCmd.Command = cmd
	return operatorCmd
}
