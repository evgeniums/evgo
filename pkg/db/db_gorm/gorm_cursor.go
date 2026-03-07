package db_gorm

import (
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

func (c *GormCursor) Close(ctx logger.WithLogger) error {
	err := c.rows.Close()
	if err != nil {
		err = fmt.Errorf("failed to close rows")
		ctx.Logger().Error("GormDB.Cursor", err)
	}
	return err
}

func (c *GormCursor) Scan(ctx logger.WithLogger, obj interface{}) error {
	err := c.gormDB.db.ScanRows(c.sql, obj)
	if err != nil {
		err = fmt.Errorf("failed to scan rows to object %v", ObjectTypeName(obj))
		ctx.Logger().Error("GormDB.Cursor", err)
	}
	return err
}

func (c *GormCursor) Next(ctx logger.WithLogger) (bool, error) {
	next := c.rows.Next()
	err := c.rows.Err()
	if err != nil {
		err = fmt.Errorf("failed to read next rows")
		ctx.Logger().Error("GormDB.Cursor", err)
	}
	return next, err
}
