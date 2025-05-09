package database

import (
	"context"
	"fmt"
	"rscc/internal/common/logger"
	"rscc/internal/database/ent"
	"rscc/internal/database/ent/agent"
	"rscc/internal/database/ent/operator"

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

func (db *Database) Close() error {
	return db.client.Close()
}

// Listener
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

// Operator
func (db *Database) CreateOperator(ctx context.Context, username, publicKey string, isAdmin bool) (*ent.Operator, error) {
	operator, err := db.client.Operator.Create().
		SetName(username).
		SetPublicKey(publicKey).
		SetIsAdmin(isAdmin).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create operator: %w", err)
	}
	return operator, nil
}

func (db *Database) GetAllOperators(ctx context.Context) ([]*ent.Operator, error) {
	operators, err := db.client.Operator.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all operators: %w", err)
	}
	return operators, nil
}

func (db *Database) GetOperatorByName(ctx context.Context, username string) (*ent.Operator, error) {
	operator, err := db.client.Operator.Query().Where(operator.Name(username)).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get operator: %w", err)
	}
	return operator, nil
}

func (db *Database) GetOperatorByID(ctx context.Context, id string) (*ent.Operator, error) {
	operator, err := db.client.Operator.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get operator: %w", err)
	}
	return operator, nil
}

func (db *Database) DeleteOperatorByID(ctx context.Context, id string) error {
	return db.client.Operator.DeleteOneID(id).Exec(ctx)
}

// Agent
func (db *Database) CreateAgent(ctx context.Context, name, os, arch, server string, shared, pie, garble bool, subsystems []string, xxhash, path string, publicKey []byte) (*ent.Agent, error) {
	agent, err := db.client.Agent.Create().
		SetName(name).
		SetOs(os).
		SetArch(arch).
		SetServer(server).
		SetShared(shared).
		SetPie(pie).
		SetGarble(garble).
		SetSubsystems(subsystems).
		SetXxhash(xxhash).
		SetPath(path).
		SetPublicKey(publicKey).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	return agent, nil
}

func (db *Database) GetAllAgents(ctx context.Context) ([]*ent.Agent, error) {
	agents, err := db.client.Agent.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all agents: %w", err)
	}
	return agents, nil
}

func (db *Database) GetAgentByName(ctx context.Context, name string) (*ent.Agent, error) {
	agent, err := db.client.Agent.Query().Where(agent.Name(name)).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	return agent, nil
}

func (db *Database) GetAgentByID(ctx context.Context, id string) (*ent.Agent, error) {
	agent, err := db.client.Agent.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	return agent, nil
}

func (db *Database) DeleteAgent(ctx context.Context, id string) error {
	return db.client.Agent.DeleteOneID(id).Exec(ctx)
}
