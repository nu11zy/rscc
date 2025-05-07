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
		RunE:    o.cmdAdd,
	}
	cmd.Flags().StringP("user", "u", "", "operator username")
	cmd.Flags().StringP("key", "k", "", "operator public key")
	cmd.Flags().BoolP("admin", "a", false, "create admin operator")
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("public-key")

	return cmd
}

func (o *OperatorCmd) cmdAdd(cmd *cobra.Command, args []string) error {
	username, err := cmd.Flags().GetString("user")
	if err != nil {
		return fmt.Errorf("failed to get username: %w", err)
	}
	publicKey, err := cmd.Flags().GetString("key")
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}
	admin, err := cmd.Flags().GetBool("admin")
	if err != nil {
		return fmt.Errorf("failed to get admin flag: %w", err)
	}

	// Validate flags
	username = strings.TrimSpace(username)
	if len(username) == 0 {
		return fmt.Errorf("username cannot be empty")
	}
	publicKey = strings.TrimSpace(publicKey)
	_, _, _, _, err = ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	user, err := o.db.CreateUser(cmd.Context(), username, publicKey, admin)
	if err != nil {
		if ent.IsConstraintError(err) {
			return fmt.Errorf("operator '%s' already exists", username)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	cmd.Println(pprint.Success("Added operator '%s' [id: %s] [admin: %t]", username, user.ID, user.IsAdmin))
	return nil
}
