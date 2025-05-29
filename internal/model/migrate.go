package model

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

const datasourcePostgres = "postgres"

func AutoMigrate(c config.Database) error {
	if !c.AutoMigrate.Enable {
		fmt.Println("AutoMigrate is disabled.")
		return nil
	}
	fmt.Println("===start to migrate===")
	db, err := sql.Open(datasourcePostgres, c.DataSource)
	if err != nil {
		return err
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(
		c.AutoMigrate.Scripts,
		datasourcePostgres, driver)
	if err != nil {
		return err
	}
	err = m.Up()
	if err != nil {
		return err
	}
	fmt.Println("===auto migrate successfully===")
	return nil
}
