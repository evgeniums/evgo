package db_gorm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/evgeniums/evgo/pkg/logger"
	"gorm.io/gorm"
)

type GormCursor struct {
	rows   gorm.Rows
	gormDB *GormDB

	sql *sql.Rows
}

func (c *GormCursor) Close(sctx context.Context) error {
	err := c.rows.Close()
	if err != nil {
		err = fmt.Errorf("failed to close rows")
		ctx := logger.LoggerContext(sctx)
		ctx.Logger().Error("GormDB.Cursor", err)
	}
	return err
}

func (c *GormCursor) Scan(sctx context.Context, obj interface{}) error {
	err := c.gormDB.db.ScanRows(c.sql, obj)
	if err != nil {
		err = fmt.Errorf("failed to scan rows to object %v", ObjectTypeName(obj))
		ctx := logger.LoggerContext(sctx)
		ctx.Logger().Error("GormDB.Cursor", err)
	}
	return err
}

func (c *GormCursor) Next(sctx context.Context) (bool, error) {
	next := c.rows.Next()
	err := c.rows.Err()
	if err != nil {
		err = fmt.Errorf("failed to read next rows")
		ctx := logger.LoggerContext(sctx)
		ctx.Logger().Error("GormDB.Cursor", err)
	}
	return next, err
}
