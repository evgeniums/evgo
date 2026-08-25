package db_gorm

import (
	"context"
	"errors"
	"fmt"

	"github.com/evgeniums/evgo/pkg/db"
	"github.com/mattn/go-sqlite3"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func DbGormOpener(provider string, dsn string) (gorm.Dialector, error) {

	switch provider {
	case "postgres":
		return postgres.Open(dsn), nil
	case "sqlite":
		return sqlite.Open(dsn), nil
	}

	return nil, errors.New("unknown database provider")
}

func DbDsnBuilder(config *db.DBConfig) (string, error) {

	switch config.DB_PROVIDER {
	case "postgres":
		return PostgresDsnBuilder(config)
	case "sqlite":
		if config.DB_DSN != "" {
			return config.DB_DSN, nil
		}
		// _busy_timeout makes a writer wait for a concurrent writer to finish instead of failing
		// immediately with "database is locked" -- SQLite's default (0) fails instantly on any
		// lock conflict, which single-connection-pool call sites rarely hit but which a
		// multi-goroutine writer (a background work scheduler racing an API request's own
		// transaction, e.g.) can hit routinely. Purely additive: a lock that would have succeeded
		// immediately still does, this only changes what happens when two writers genuinely
		// overlap.
		dsn := fmt.Sprintf("%s?_txlock=immediate&_busy_timeout=5000", config.DB_NAME)
		return dsn, nil
	}

	return "", errors.New("unknown database provider")
}

func DbCreator(provider string, db *gorm.DB, dbName string) error {

	switch provider {
	case "postgres":
		return PostgresDbCreator(provider, db, dbName)
	case "sqlite":
		return nil
	}

	return errors.New("unknown database provider")
}

func CheckDuplicateKeyError(provider string, result *gorm.DB) (bool, error) {

	switch provider {
	case "postgres":
		return PostgresCheckDuplicateKeyError(provider, result)
	case "sqlite":
		if err, ok := result.Error.(sqlite3.Error); ok {
			if err.ExtendedCode == sqlite3.ErrConstraintUnique {
				return true, errors.New("record already exists")
			}
		}
	}

	return false, result.Error
}

func PartitionedMonthMigrator(provider string, sctx context.Context, db *gorm.DB, models ...interface{}) error {

	switch provider {
	case "postgres":
		return PostgresPartitionedMonthAutoMigrate(sctx, db, models...)
	case "sqlite":
		return db.AutoMigrate(models...)
	}

	return errors.New("unknown database provider")
}

func SetupGormDB() {
	NewModelStore(true)
	DefaultDbConnector = func() *DbConnector {
		c := &DbConnector{}
		c.DialectorOpener = DbGormOpener
		c.DsnBuilder = func(config *db.DBConfig) (string, error) {
			return DbDsnBuilder(config)
		}
		c.DbCreator = DbCreator
		c.PartitionedMonthMigrator = PartitionedMonthMigrator
		c.CheckDuplicateKeyError = CheckDuplicateKeyError
		return c
	}
}
