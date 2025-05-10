package operatorcmd

import (
	"fmt"
	"rscc/internal/common/constants"
	"rscc/internal/database/ent"

	"github.com/spf13/cobra"
)

func (o *OperatorCmd) newCmdInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <id / name>",
		Short: "Get information about an operator",
		Args:  cobra.ExactArgs(1),
		RunE:  o.cmdInfo,
	}

	return cmd
}

func (o *OperatorCmd) cmdInfo(cmd *cobra.Command, args []string) error {
	idOrName := args[0]

	var operator *ent.Operator
	if len(idOrName) == constants.IDLength {
		var err error
		operator, err = o.db.GetOperatorByID(cmd.Context(), idOrName)
		if err != nil {
			if ent.IsNotFound(err) {
				return fmt.Errorf("operator '%s' not found", idOrName)
			}
			return fmt.Errorf("failed to get operator: %w", err)
		}
	}

	if operator == nil {
		var err error
		operator, err = o.db.GetOperatorByName(cmd.Context(), idOrName)
		if err != nil {
			if ent.IsNotFound(err) {
				return fmt.Errorf("operator '%s' not found", idOrName)
			}
			return fmt.Errorf("failed to get operator: %w", err)
		}
	}

	cmd.Println("Name:", operator.Name)
	cmd.Println("ID:", operator.ID)

	var role = "operator"
	if operator.IsAdmin {
		role = "admin"
	}
	cmd.Println("Role:", role)

	var lastLogin = "never"
	if operator.LastLogin != nil {
		lastLogin = operator.LastLogin.Format("2006-01-02 15:04:05")
	}
	cmd.Println("Last Login:", lastLogin)

	cmd.Println("Public Key:", operator.PublicKey)

	return nil
}
