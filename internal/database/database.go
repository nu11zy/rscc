package database

import (
	"context"
	"fmt"
	"rscc/internal/common/logger"
	"rscc/internal/database/ent"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

type Database struct {
	client *ent.Client
	lg     *zap.SugaredLogger
}

func NewDatabase(ctx context.Context, path string) (*Database, error) {
	lg := logger.FromContext(ctx).Named("database")

	if path == "" {
		return nil, fmt.Errorf("database path is required")
	}

	client, err := ent.Open(dialect.SQLite, fmt.Sprintf("file:%s?_fk=1&cache=shared", path))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := client.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}
	lg.Infof("Using database `%s`", path)

	return &Database{client: client, lg: lg}, nil
}

func (db *Database) CreateUser(ctx context.Context, username, publicKey string, isAdmin bool) (*ent.User, error) {
	user, err := db.client.User.Create().
		SetName(username).
		SetPublicKey(publicKey).
		SetIsAdmin(isAdmin).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

func (db *Database) GetAllUsers(ctx context.Context) ([]*ent.User, error) {
	users, err := db.client.User.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	return users, nil
}

func (db *Database) CreateListenerWithID(ctx context.Context, id, name string, privateKey []byte) (*ent.Listener, error) {
	listener, err := db.client.Listener.Create().
		SetID(id).
		SetName(name).
		SetPrivateKey(privateKey).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}
	return listener, nil
}

func (db *Database) GetListener(ctx context.Context, id string) (*ent.Listener, error) {
	listener, err := db.client.Listener.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}
	return listener, nil
}

func (db *Database) Close() error {
	return db.client.Close()
}

func (db *Database) DeleteUserByID(ctx context.Context, id string) error {
	return db.client.User.DeleteOneID(id).Exec(ctx)
}
