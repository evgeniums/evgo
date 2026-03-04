package db_gorm

import (
	"errors"

	"github.com/evgeniums/go-utils/pkg/db"
	"github.com/evgeniums/go-utils/pkg/logger"
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
		dsn := config.DB_NAME
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

func PartitionedMonthMigrator(provider string, ctx logger.WithLogger, db *gorm.DB, models ...interface{}) error {

	switch provider {
	case "postgres":
		return PostgresPartitionedMonthAutoMigrate(ctx, db, models...)
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
