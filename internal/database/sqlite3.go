package database

import (
	"database/sql"
	"database/sql/driver"
	"github.com/go-faster/errors"
	"modernc.org/sqlite"
)

type sqlite3Driver struct {
	*sqlite.Driver
}

// Open create connection to database
func (d sqlite3Driver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return conn, err
	}
	c := conn.(interface {
		Exec(stmt string, args []driver.Value) (driver.Result, error)
	})
	// enable PRAGMAs
	if _, err = c.Exec("PRAGMA foreign_keys = on;", nil); err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "failed to enable foreign keys")
	}
	if _, err = c.Exec("PRAGMA journal_mode = WAL;", nil); err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "failed to enable WAL mode")
	}
	if _, err = c.Exec("PRAGMA synchronous=NORMAL;", nil); err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "failed to enable normal synchronous")
	}
	return conn, nil
}

func init() {
	// register sqlite3 driver for using with ent
	sql.Register("sqlite3", sqlite3Driver{Driver: &sqlite.Driver{}})
}
