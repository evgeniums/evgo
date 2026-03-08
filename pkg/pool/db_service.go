package pool

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

func ParseDbService(service *PoolServiceBaseData) (*db.DBConfig, error) {

	if service.TypeName() != TypeDatabase {
		return nil, errors.New("invalid service type")
	}

	d := &db.DBConfig{}
	d.DB_DSN = service.PrivateUrl()
	d.DB_HOST = service.PrivateHost()
	d.DB_PORT = service.PrivatePort()
	d.DB_PROVIDER = service.Provider()
	d.DB_USER = service.User()
	d.DB_PASSWORD = service.Secret1()
	d.DB_NAME = service.DbName()
	d.DB_EXTRA_CONFIG = service.Parameter1()
	return d, nil
}

func ConnectDatabaseService(sctx context.Context, pool Pool, role string, dbName string, newDb ...bool) (db.DB, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("pool.ConnectDatabaseService", logger.Fields{"role": role})
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find service for database role
	dbService, err := pool.Service(role)
	if err != nil {
		genErr := generic_error.NewFromOriginal(ErrorCodeNoServiceWithRole, "Pool does not include service for database role", err)
		genErr.SetDetails(role)
		ctx.SetGenericError(genErr)
		err = genErr
		return nil, err
	}
	if !dbService.IsActive() {
		genErr := generic_error.New(ErrorCodeServiceNotActive, "Service for database in the pool is not active")
		ctx.SetGenericError(genErr)
		err = genErr
		return nil, err
	}

	// parse db config
	dbConfig, err := ParseDbService(&dbService.PoolServiceBaseData)
	if err != nil {
		genErr := generic_error.NewFromOriginal(ErrorCodeInvalidServiceConfiguration, "Invalid configuration of service for database", err)
		genErr.SetDetails(dbService.ServiceName)
		ctx.SetGenericError(genErr)
		err = genErr
		return nil, err
	}
	name := utils.OptionalString(dbService.DB_NAME, dbName)

	// create database
	createDb := utils.OptionalArg(false, newDb...) && dbConfig.DB_NAME != ""
	if createDb {

		// connect to master database
		database := ctx.App().Db().Clone()
		err = database.InitWithConfig(sctx, ctx.App().Validator(), dbConfig)
		if err != nil {
			genErr := generic_error.NewFromOriginal(ErrorCodeServiceInitializationFailed, "Failed to connect to master database", err)
			genErr.SetDetails(dbService.ServiceName)
			ctx.SetGenericError(genErr)
			err = genErr
			return nil, err
		}

		// create new database
		err = database.CreateDatabase(sctx, name)
		database.Close()
		if err != nil {
			genErr := generic_error.NewFromOriginal(ErrorCodeCreateServiceDatabaseFailed, "Failed to create database", err)
			genErr.SetDetails(name)
			ctx.SetGenericError(genErr)
			err = genErr
			return nil, err
		}
	}

	// create and init database connection
	dbConfig.DB_NAME = name
	database := ctx.App().Db().Clone()
	err = database.InitWithConfig(sctx, ctx.App().Validator(), dbConfig)
	if err != nil {
		genErr := generic_error.NewFromOriginal(ErrorCodeServiceInitializationFailed, "Failed to connect to database", err)
		genErr.SetDetails(dbService.ServiceName)
		ctx.SetGenericError(genErr)
		err = genErr
		return nil, err
	}

	// done
	return database, nil
}
