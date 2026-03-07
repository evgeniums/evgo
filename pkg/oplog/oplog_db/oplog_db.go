package oplog_db

import (
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/oplog"
	"github.com/evgeniums/evgo/pkg/utils"
)

type OplogControllerDb struct {
	Ctx op_context.Context
}

func (o *OplogControllerDb) Write(op oplog.Oplog) error {
	err := op_context.DB(o.Ctx).Create(o.Ctx, op)
	if err != nil {
		o.Ctx.Logger().Error("failed to write oplog", err, logger.Fields{"oplog": utils.ObjectTypeName(op)})
	}
	return err
}

func (o *OplogControllerDb) Read(filter *db.Filter, docs interface{}) (int64, error) {
	count, err := op_context.DB(o.Ctx).FindWithFilter(o.Ctx, filter, docs)
	if err != nil {
		o.Ctx.Logger().Error("failed to read oplog", err, logger.Fields{"oplog": utils.ObjectTypeName(docs)})
	}
	return count, err
}

func MakeOplogController(ctx op_context.Context) oplog.OplogController {
	return &OplogControllerDb{Ctx: ctx}
}
