package operatorcmd

import (
	"fmt"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
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

	cmd.Println(pprint.Info("Operator info:"))
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Name:"), operator.Name)
	cmd.Printf(" %s\t\t%s\n", pprint.Blue.Sprint("├─ ID:"), operator.ID)

	var role = "operator"
	if operator.IsAdmin {
		role = "admin"
	}
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Role:"), role)

	var lastLogin = "never"
	if operator.LastLogin != nil {
		lastLogin = operator.LastLogin.Format("2006-01-02 15:04:05")
	}
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("├─ Last Login:"), lastLogin)
	cmd.Printf(" %s\t%s\n", pprint.Blue.Sprint("└─ Public Key:"), operator.PublicKey)

	cmd.Println()
	return nil
}
