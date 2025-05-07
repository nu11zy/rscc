package operatorcmd

import (
	"fmt"
	"rscc/internal/common/pprint"
	"strconv"

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
		rows = append(rows, []string{operator.ID, operator.Name, strconv.FormatBool(operator.IsAdmin)})
	}

	cmd.Println(pprint.Table([]string{"ID", "Name", "Is Admin"}, rows))
	return nil
}
