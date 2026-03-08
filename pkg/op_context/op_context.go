package op_context

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/cache"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/oplog"
	"github.com/evgeniums/evgo/pkg/utils"
)

const AutoUserType string = "auto"

type CallContext interface {
	Method() string
	Error() error
	Message() string

	SetError(err error) error
	SetErrorStr(err string) error
	SetMessage(msg string)

	SetLoggerField(name string, value interface{})
	AddLoggerFields(fields logger.Fields)
	UnsetLoggerField(name string)
	LoggerFields() logger.Fields

	logger.WithLogger
}

type Origin interface {
	App() string
	SetApp(string)
	Name() string
	SetName(string)
	Source() string
	SetSource(string)
	SessionClient() string
	SetSessionClient(string)
	User() string
	SetUser(string)
	UserType() string
	SetUserType(string)
}

type Context interface {
	app_context.WithApp
	common.WithName
	logger.WithLogger
	db.WithDB

	MainDB() db.DB
	MainLogger() logger.Logger

	DbTransaction() db.Transaction
	SetDbTransaction(tx db.Transaction)
	ClearDbTransaction()
	SetOverrideDb(db db.DBHandlers)
	OverrideDb() db.DBHandlers

	Cache() cache.Cache
	SetCache(cache.Cache)

	ErrorManager() generic_error.ErrorManager
	SetErrorManager(manager generic_error.ErrorManager)

	MakeGenericError(code string) generic_error.Error

	SetID(id string)
	ID() string

	TraceInMethod(methodName string, fields ...logger.Fields) CallContext
	TraceOutMethod()

	SetGenericError(err generic_error.Error, override ...bool)
	GenericError() generic_error.Error
	SetGenericErrorCode(code string, override ...bool)

	Tr(phrase string) string

	SetLoggerField(name string, value interface{})
	AddLoggerFields(fields logger.Fields)
	LoggerFields() logger.Fields
	UnsetLoggerField(name string)

	SetErrorAsWarn(enable bool)

	Oplog(o oplog.Oplog)
	SetOplogHandler(handler OplogHandler)
	OplogHandler() OplogHandler
	SetOplogWriter(writer oplog.OplogController)
	OplogWriter() oplog.OplogController

	SetOrigin(o Origin)
	Origin() Origin

	ClearError()
	Reset()
	DumpLog(successMessage ...string)
	Close(sctx context.Context, successMessage ...string)
}

func ExecDbTransaction(sctx context.Context, handler func() error) error {

	ctx := OpContext[Context](sctx)
	if ctx.DbTransaction() != nil {
		return errors.New("nested transactions not supported")
	}

	h := func(tx db.Transaction) error {

		ctx.SetDbTransaction(tx)
		defer ctx.ClearDbTransaction()

		return handler()
	}
	return DB(ctx).Transaction(h)
}

type WithCtx interface {
	Ctx() Context
}

type CallContextBuilder = func(methodName string, parentLogger logger.Logger, fields ...logger.Fields) CallContext

type OplogHandler = func() oplog.OplogController

func DB(c Context, forceMainDb ...bool) db.DBHandlers {
	if c.DbTransaction() != nil {
		return c.DbTransaction()
	}
	if c.OverrideDb() != nil {
		return c.OverrideDb()
	}
	if utils.OptionalArg(false, forceMainDb...) {
		return c.MainDB()
	}
	return c.Db()
}

type OpContextKey struct{}

func WrapOpContext(ctx context.Context, opContext Context) context.Context {
	newCtx := context.WithValue(ctx, OpContextKey{}, opContext)
	newCtx = logger.WrapOpContext(newCtx, opContext)
	return newCtx
}

func MakeOpContext(opContext Context) context.Context {
	newCtx := context.WithValue(context.Background(), OpContextKey{}, opContext)
	newCtx = logger.WrapOpContext(newCtx, opContext)
	return newCtx
}

func OpContext[T Context](ctx context.Context) T {
	v, _ := ctx.Value(OpContextKey{}).(T)
	return v
}
