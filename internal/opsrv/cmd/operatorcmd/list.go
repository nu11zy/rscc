package operatorcmd

import (
	"fmt"
	"rscc/internal/common/pprint"

	"github.com/spf13/cobra"
)

func (o *OperatorCmd) newCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List operators",
		Aliases: []string{"l", "ls"},
		Args:    cobra.NoArgs,
		RunE:    o.cmdList,
	}

	return cmd
}

func (o *OperatorCmd) cmdList(cmd *cobra.Command, args []string) error {
	operators, err := o.db.GetAllOperators(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get operators: %w", err)
	}

	var rows [][]string
	for _, operator := range operators {
		id := pprint.Blue.Sprint(operator.ID)
		name := operator.Name
		if name == o.operator.Username {
			name = pprint.Green.Sprint(name)
		}

		var role = "operator"
		if operator.IsAdmin {
			role = pprint.Yellow.Sprint(pprint.Bold.Sprint("admin"))
		}

		var lastLogin = "never"
		if operator.LastLogin != nil {
			lastLogin = operator.LastLogin.Format("02.01.2006 15:04:05")
		}

		rows = append(rows, []string{id, name, role, lastLogin})
	}

	cmd.Println(pprint.Table([]string{"ID", "Name", "Role", "Last Login"}, rows))
	cmd.Println()
	return nil
}
