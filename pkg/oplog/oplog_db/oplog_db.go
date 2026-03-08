package oplog_db

import (
	"context"

	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/oplog"
	"github.com/evgeniums/evgo/pkg/utils"
)

type OplogControllerDb struct {
}

func (o *OplogControllerDb) Write(sctx context.Context, op oplog.Oplog) error {
	ctx := op_context.OpContext[op_context.Context](sctx)
	err := op_context.DB(ctx).Create(sctx, op)
	if err != nil {
		ctx.Logger().Error("failed to write oplog", err, logger.Fields{"oplog": utils.ObjectTypeName(op)})
	}
	return err
}

func (o *OplogControllerDb) Read(sctx context.Context, filter *db.Filter, docs interface{}) (int64, error) {
	ctx := op_context.OpContext[op_context.Context](sctx)
	count, err := op_context.DB(ctx).FindWithFilter(sctx, filter, docs)
	if err != nil {
		ctx.Logger().Error("failed to read oplog", err, logger.Fields{"oplog": utils.ObjectTypeName(docs)})
	}
	return count, err
}

func MakeOplogController() oplog.OplogController {
	return &OplogControllerDb{}
}
