package operatorcmd

import (
	"fmt"
	"rscc/internal/database"
	"rscc/internal/sshd"

	"github.com/spf13/cobra"
)

type OperatorCmd struct {
	Command  *cobra.Command
	db       *database.Database
	operator *sshd.OperatorSession
}

// + operator list
// + operator add --name <name> --key <public-key> --admin
// + operator remove <name/id>
// + operator info <name/id>

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
	cmd.AddCommand(operatorCmd.newCmdRemove())
	cmd.AddCommand(operatorCmd.newCmdInfo())

	operatorCmd.Command = cmd
	return operatorCmd
}

func (o *OperatorCmd) checkAdmin(cmd *cobra.Command, args []string) error {
	_, ok := o.operator.Permissions.CriticalOptions["admin"]
	if !ok {
		return fmt.Errorf("permission denied")
	}
	return nil
}
