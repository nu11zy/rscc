package operatorcmd

import (
	"fmt"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"rscc/internal/database/ent"

	"github.com/spf13/cobra"
)

func (o *OperatorCmd) newCmdRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "Remove operator",
		Example: "operator remove <id / name>",
		Aliases: []string{"r", "rm"},
		Args:    cobra.ExactArgs(1),
		PreRunE: o.checkAdmin,
		RunE:    o.cmdRemove,
	}
	return cmd
}

func (o *OperatorCmd) cmdRemove(cmd *cobra.Command, args []string) error {
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

	err := o.db.DeleteOperatorByID(cmd.Context(), operator.ID)
	if err != nil {
		return fmt.Errorf("failed to delete operator: %w", err)
	}

	cmd.Println(pprint.Success("Operator '%s' deleted", operator.Name))
	return nil
}
