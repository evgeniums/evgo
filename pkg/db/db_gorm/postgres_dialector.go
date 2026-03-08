package db_gorm

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func PostgresOpener(provider string, dsn string) (gorm.Dialector, error) {

	if provider != "postgres" {
		return nil, errors.New("unknown database provider")
	}

	// switch provider {
	// case "postgres":
	return postgres.Open(dsn), nil
	// case "mysql":
	// 	return mysql.Open(dsn), nil
	// case "sqlite":
	// 	return sqlite.Open(dsn), nil
	// case "sqlserver":
	// 	return sqlserver.Open(dsn), nil
	// }
}

func PostgresDsnBuilder(config *db.DBConfig) (string, error) {
	dsn := fmt.Sprintf("host=%v port=%v user=%v dbname=%v password=%v sslmode=disable TimeZone=UTC", config.DB_HOST, config.DB_PORT, config.DB_USER, config.DB_NAME, config.DB_PASSWORD)
	return dsn, nil
}

func PostgresDbCreator(provider string, db *gorm.DB, dbName string) error {

	if provider != "postgres" {
		return errors.New("unknown database provider")
	}

	// check if db exists
	rs := db.Raw("SELECT * FROM pg_database WHERE datname = ?;", dbName)
	if rs.Error != nil {
		return fmt.Errorf("failed to select from pg_database: %s", rs.Error)
	}
	var rec = make(map[string]interface{})
	rs.Find(rec)

	// if not create it
	if len(rec) == 0 {
		// create database
		rs := db.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbName))
		if rs.Error != nil {
			return fmt.Errorf("failed to create postgres database: %s", rs.Error)
		}
	}

	// done
	return nil
}

func PostgresCheckDuplicateKeyError(provider string, result *gorm.DB) (bool, error) {

	if provider != "postgres" {
		return false, errors.New("unknown database provider")
	}

	if err, ok := result.Error.(*pgconn.PgError); ok {
		if err.Code == pgerrcode.UniqueViolation {
			return true, errors.New("record already exists")
		}
	}

	return false, result.Error
}

func PostgresTableExists(db *gorm.DB, tableName string) (bool, error) {

	sqlStr := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '%s' AND table_schema LIKE 'public' AND table_type LIKE 'BASE TABLE'", tableName)
	count := 0
	result := db.Raw(sqlStr).Scan(&count)
	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

func partitionTableName(table string, month utils.Month) string {
	return fmt.Sprintf("%s_%d", table, month)
}

func PostgresPartitionedMonthAutoMigrate(sctx context.Context, db *gorm.DB, models ...interface{}) error {

	if len(models) == 0 {
		return nil
	}

	err := db.Set("gorm:table_options", " PARTITION BY LIST (month)").Migrator().AutoMigrate(models...)
	if err != nil {
		ctx := logger.LoggerContext(sctx)
		return ctx.Logger().PushFatalStack("failed to migrate partitioned database models", err)
	}

	schemaCache := &sync.Map{}
	schemaNamer := &schema.NamingStrategy{}

	currentMonth := utils.CurrentMonth()
	currentMonth = currentMonth.Prev()

	for _, model := range models {

		sc, err := schema.Parse(model, schemaCache, schemaNamer)
		if err != nil {
			ctx := logger.LoggerContext(sctx)
			return ctx.Logger().PushFatalStack("failed to migrate partitioned database models", err)
		}

		fields := logger.Fields{"table": sc.Table}
		tableMonth := currentMonth
		for i := 0; i < 12; i++ {

			subfields := utils.CopyMap(fields)
			subfields["month"] = tableMonth
			partitionTableName := fmt.Sprintf("%s_%d", sc.Table, tableMonth)
			subfields["partition_table"] = partitionTableName
			partitionExists, err := PostgresTableExists(db, partitionTableName)
			if err != nil {
				ctx := logger.LoggerContext(sctx)
				return ctx.Logger().PushFatalStack("failed to check if partition exists in database", err)
			}

			if !partitionExists {
				sqlExpr := fmt.Sprintf("CREATE TABLE %s PARTITION OF %s FOR VALUES IN (%d);", partitionTableName, sc.Table, tableMonth)
				subfields["sql"] = sqlExpr
				ctx := logger.LoggerContext(sctx)
				ctx.Logger().Info("Creating partition", subfields)
				result := db.Exec(sqlExpr)
				if result.Error != nil {
					return ctx.Logger().PushFatalStack("failed to create partition for database model", err, subfields)
				}
			}

			tableMonth = tableMonth.Next()
		}
	}

	return nil
}

func PostgresPartitionedMonthMigrator(provider string, sctx context.Context, db *gorm.DB, models ...interface{}) error {
	if provider != "postgres" {
		return errors.New("unknown database provider")
	}
	return PostgresPartitionedMonthAutoMigrate(sctx, db, models...)
}

func PostgresPartitionedMonthDetach(provider string, sctx context.Context, db *gorm.DB, table string, months []utils.Month) error {
	if provider != "postgres" {
		return errors.New("unknown database provider")
	}

	for _, month := range months {

		partition := partitionTableName(table, month)
		fields := logger.Fields{"month": month, "table": table, "partition": partition}

		partitionExists, err := PostgresTableExists(db, partition)
		if err != nil {
			ctx := logger.LoggerContext(sctx)
			return ctx.Logger().PushFatalStack("failed to check if partition exists in database", err, fields)
		}

		if partitionExists {
			sqlExpr := fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s;", table, partition)
			fields["sql"] = sqlExpr
			ctx := logger.LoggerContext(sctx)
			ctx.Logger().Info("Detaching partition", fields)
			result := db.Exec(sqlExpr)
			if result.Error != nil {
				return ctx.Logger().PushFatalStack("failed to detach partition", result.Error, fields)
			}
		}

	}

	return nil
}

func PostgresPartitionedMonthDelete(provider string, sctx context.Context, db *gorm.DB, table string, months []utils.Month) error {
	if provider != "postgres" {
		return errors.New("unknown database provider")
	}

	for _, month := range months {

		partition := partitionTableName(table, month)
		fields := logger.Fields{"month": month, "table": table, "partition": partition}

		partitionExists, err := PostgresTableExists(db, partition)
		if err != nil {
			ctx := logger.LoggerContext(sctx)
			return ctx.Logger().PushFatalStack("failed to check if partition exists in database", err, fields)
		}

		if partitionExists {
			sqlExpr := fmt.Sprintf("DROP TABLE %s;", partition)
			fields["sql"] = sqlExpr
			ctx := logger.LoggerContext(sctx)
			ctx.Logger().Info("Deleting partition", fields)
			result := db.Exec(sqlExpr)
			if result.Error != nil {
				return ctx.Logger().PushFatalStack("failed to delete partition", err, fields)
			}
		}

	}

	return nil
}

func PostgresDbConnector() *DbConnector {
	c := &DbConnector{}
	c.DialectorOpener = PostgresOpener
	c.DsnBuilder = PostgresDsnBuilder
	c.CheckDuplicateKeyError = PostgresCheckDuplicateKeyError
	c.PartitionedMonthMigrator = PostgresPartitionedMonthMigrator
	c.PartitionedMonthDeleter = PostgresPartitionedMonthDelete
	c.PartitionedMonthDetacher = PostgresPartitionedMonthDetach
	c.DbCreator = PostgresDbCreator
	return c
}
