package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"rscc/internal/common/logger"
	"rscc/internal/database/ent"
	"rscc/internal/database/ent/agent"
	"strings"

	entsql "entgo.io/ent/dialect/sql"

	"entgo.io/ent/dialect"
	"go.uber.org/zap"
)

type Database struct {
	client *ent.Client
	lg     *zap.SugaredLogger
}

func NewDatabase(ctx context.Context, path string) (*Database, error) {
	lg := logger.FromContext(ctx).Named("database")

	// check if path is blank string
	if path == "" {
		return nil, errors.New("database path is required")
	}

	// create connection to database
	var d strings.Builder
	d.WriteString("file:")
	d.WriteString(path)
	d.WriteString("?cache=shared&_fk=1")
	lg.Debugf("Connection DSN: %s", d.String())
	db, err := sql.Open("sqlite3", d.String())
	if err != nil {
		return nil, err
	}
	// avoid "database is locked" errors
	db.SetMaxOpenConns(1)

	// create ent client
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))

	// performs migrations
	if err := client.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}
	lg.Infof("Use database on path %s", path)

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

// Agent
func (db *Database) CreateAgent(ctx context.Context, name, os, arch string, servers []string, shared, pie, garble bool, subsystems []string, xxhash, path string, publicKey []byte) (*ent.Agent, error) {
	agent, err := db.client.Agent.Create().
		SetName(name).
		SetOs(os).
		SetArch(arch).
		SetServers(servers).
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

func (db *Database) GetAgentByURL(ctx context.Context, url string) (*ent.Agent, error) {
	agent, err := db.client.Agent.Query().Where(agent.URL(url)).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	return agent, nil
}

func (db *Database) UpdateAgentURL(ctx context.Context, id, url string) error {
	return db.client.Agent.UpdateOneID(id).SetURL(url).Exec(ctx)
}

func (db *Database) UpdateAgentHosted(ctx context.Context, id string, hosted bool) error {
	return db.client.Agent.UpdateOneID(id).SetHosted(hosted).Exec(ctx)
}

func (db *Database) UpdateAgentComment(ctx context.Context, id, comment string) error {
	return db.client.Agent.UpdateOneID(id).SetComment(comment).Exec(ctx)
}

func (db *Database) UpdateAgentHits(ctx context.Context, id string) error {
	return db.client.Agent.UpdateOneID(id).AddCallbacks(1).Exec(ctx)
}

func (db *Database) UpdateAgentDownloads(ctx context.Context, id string) error {
	return db.client.Agent.UpdateOneID(id).AddDownloads(1).Exec(ctx)
}

func (db *Database) ResetAgentDownloads(ctx context.Context, id string) error {
	return db.client.Agent.UpdateOneID(id).SetDownloads(0).Exec(ctx)
}

func (db *Database) DeleteAgent(ctx context.Context, id string) error {
	return db.client.Agent.DeleteOneID(id).Exec(ctx)
}

// Session
func (db *Database) CreateSession(ctx context.Context, agentID, username, hostname, domain, osMeta, procName, extra string, ips []string, isPriv bool) (*ent.Session, error) {
	session, err := db.client.Session.Create().
		SetAgentID(agentID).
		SetUsername(username).
		SetHostname(hostname).
		SetDomain(domain).
		SetIsPriv(isPriv).
		SetIps(ips).
		SetOsMeta(osMeta).
		SetProcName(procName).
		SetExtra(extra).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return session, nil
}
