package operatorcmd

import (
	"fmt"
	"rscc/internal/common/pprint"
	"rscc/internal/database/ent"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func (o *OperatorCmd) newCmdAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "Add an operator",
		Aliases: []string{"a"},
		Args:    cobra.NoArgs,
		PreRunE: o.checkAdmin,
		RunE:    o.cmdAdd,
	}
	cmd.Flags().StringP("name", "n", "", "operator name")
	cmd.Flags().StringP("key", "k", "", "operator public key")
	cmd.Flags().BoolP("admin", "a", false, "create admin operator")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("public-key")

	return cmd
}

func (o *OperatorCmd) cmdAdd(cmd *cobra.Command, args []string) error {
	// Get flags
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return fmt.Errorf("failed to get operator name: %w", err)
	}
	publicKey, err := cmd.Flags().GetString("key")
	if err != nil {
		return fmt.Errorf("failed to get operator public key: %w", err)
	}
	admin, err := cmd.Flags().GetBool("admin")
	if err != nil {
		return fmt.Errorf("failed to get admin flag: %w", err)
	}

	// Validate flags
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return fmt.Errorf("operator name cannot be empty")
	}
	publicKey = strings.TrimSpace(publicKey)
	_, _, _, _, err = ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	user, err := o.db.CreateOperator(cmd.Context(), name, publicKey, admin)
	if err != nil {
		if ent.IsConstraintError(err) {
			return fmt.Errorf("operator '%s' already exists", name)
		}
		return fmt.Errorf("failed to create operator: %w", err)
	}

	cmd.Println(pprint.Success("Added operator '%s' [id: %s] [admin: %t]", name, user.ID, user.IsAdmin))
	return nil
}
