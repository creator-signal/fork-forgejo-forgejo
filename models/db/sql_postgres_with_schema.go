// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db

import (
	"database/sql"
	"database/sql/driver"

	"forgejo.org/modules/setting"

	"github.com/jackc/pgx/v5/stdlib"
	"xorm.io/xorm/dialects"
)

func init() {
	// Register pgx-based driver as "postgresschema" for PostgreSQL with schema support
	// This wraps pgx/v5/stdlib and injects search_path configuration on connection
	// For PostgreSQL without schema, engine.go uses "pgx" directly (registered by pgx/v5/stdlib)
	driver := &postgresSchemaDriver{}
	sql.Register("postgresschema", driver)
	dialects.RegisterDriver("postgresschema", dialects.QueryDriver("postgres"))
}

type postgresSchemaDriver struct {
	stdlib.Driver
}

// Open opens a new connection to the database. name is a connection string.
// This function opens the postgres connection in the default manner but immediately
// runs set_config to set the search_path appropriately
func (d *postgresSchemaDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return conn, err
	}
	schemaValue, _ := driver.String.ConvertValue(setting.Database.Schema)

	// golangci lint is incorrect here - there is no benefit to using driver.ExecerContext here
	// and in any case pq does not implement it
	if execer, ok := conn.(driver.Execer); ok { //nolint
		_, err := execer.Exec(`SELECT set_config(
			'search_path',
			$1 || ',' || current_setting('search_path'),
			false)`, []driver.Value{schemaValue})
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, nil
	}

	stmt, err := conn.Prepare(`SELECT set_config(
		'search_path',
		$1 || ',' || current_setting('search_path'),
		false)`)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	defer stmt.Close()

	// driver.String.ConvertValue will never return err for string

	// golangci lint is incorrect here - there is no benefit to using stmt.ExecWithContext here
	_, err = stmt.Exec([]driver.Value{schemaValue}) //nolint
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}
