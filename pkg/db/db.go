package db

import (
	"context"
	"sync"
	"time"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

const (
	SORT_ASC  string = "ASC"
	SORT_DESC string = "DESC"
)

type Fields = map[string]interface{}

func IsFieldSet(f Fields, key string) bool {
	_, ok := f[key]
	return ok
}

type DBConfig struct {
	DB_PROVIDER     string `gorm:"index"`
	DB_HOST         string `gorm:"index"`
	DB_PORT         uint16 `gorm:"index"`
	DB_NAME         string `gorm:"index"`
	DB_USER         string `gorm:"index"`
	DB_PASSWORD     string `mask:"true"`
	DB_EXTRA_CONFIG string
	DB_DSN          string
}

type DBHandlers interface {
	FindByField(sctx context.Context, field string, value interface{}, obj interface{}, dest ...interface{}) (found bool, err error)
	FindByFields(sctx context.Context, fields Fields, obj interface{}, dest ...interface{}) (found bool, err error)
	FindWithFilter(sctx context.Context, filter *Filter, docs interface{}, dest ...interface{}) (int64, error)
	FindForUpdate(sctx context.Context, fields Fields, obj interface{}) (bool, error)
	FindForShare(sctx context.Context, fields Fields, obj interface{}) (bool, error)
	Exists(sctx context.Context, filter *Filter, doc interface{}) (bool, error)

	Create(sctx context.Context, obj interface{}) error
	CreateDup(sctx context.Context, obj interface{}, ignoreConflict ...bool) (bool, error)

	Delete(sctx context.Context, obj common.Object) error
	DeleteByField(sctx context.Context, field string, value interface{}, model interface{}) error
	DeleteByFields(sctx context.Context, fields Fields, obj interface{}) error

	RowsWithFilter(sctx context.Context, filter *Filter, obj interface{}) (Cursor, error)
	AllRows(sctx context.Context, obj interface{}) (Cursor, error)

	Update(sctx context.Context, obj interface{}, filter Fields, fields Fields) error
	UpdateAll(sctx context.Context, obj interface{}, newFields Fields) error
	UpdateWithFilter(sctx context.Context, obj interface{}, filter *Filter, newFields Fields) error

	Join(sctx context.Context, joinConfig *JoinQueryConfig, filter *Filter, dest interface{}) (int64, error)

	Joiner() Joiner

	CreateDatabase(sctx context.Context, dbName string) error
	MakeExpression(expr string, args ...interface{}) interface{}

	Sum(sctx context.Context, groupFields []string, sumFields []string, filter *Filter, model interface{}, dest ...interface{}) (int64, error)

	Transaction(handler TransactionHandler) error
	EnableDebug(bool)
}

type Transaction interface {
	DBHandlers
}

type TransactionHandler = func(tx Transaction) error

type Cursor interface {
	Next(sctx context.Context) (bool, error)
	Close(sctx context.Context) error
	Scan(sctx context.Context, obj interface{}) error
}

type DB interface {
	WithFilterParser

	ID() string
	Clone() DB

	InitWithConfig(sctx context.Context, vld validator.Validator, cfg *DBConfig) error

	DBHandlers

	EnableVerboseErrors(bool)

	AutoMigrate(sctx context.Context, models []interface{}) error
	MigrateDropIndex(sctx context.Context, model interface{}, indexName string) error

	PartitionedMonthAutoMigrate(sctx context.Context, models []interface{}) error
	PartitionedMonthsDetach(sctx context.Context, table string, months []utils.Month) error
	PartitionedMonthsDelete(sctx context.Context, table string, months []utils.Month) error

	NativeHandler() interface{}

	Close()
}

type WithDB interface {
	Db() DB
}

type WithDBBase struct {
	db DB
}

func (w *WithDBBase) Db() DB {
	return w.db
}

func (w *WithDBBase) Init(db DB) {
	w.db = db
}

func Update(db DBHandlers, sctx context.Context, obj common.Object, fields Fields) error {
	f := utils.CopyMap(fields)
	f["updated_at"] = time.Now()
	return db.Update(sctx, obj, nil, f)
}

func UpdateMulti(db DBHandlers, sctx context.Context, obj interface{}, filter Fields, fields Fields) error {
	f := utils.CopyMap(fields)
	f["updated_at"] = time.Now()
	return db.Update(sctx, obj, filter, f)
}

func UpdateAll(db DBHandlers, sctx context.Context, obj interface{}, fields Fields) error {
	f := utils.CopyMap(fields)
	f["updated_at"] = time.Now()
	return db.UpdateAll(sctx, obj, f)
}

func UpdateWithFilter(db DBHandlers, sctx context.Context, obj interface{}, filter *Filter, fields Fields) error {
	f := utils.CopyMap(fields)
	f["updated_at"] = time.Now()
	return db.UpdateWithFilter(sctx, obj, filter, f)
}

type AllDatabases struct {
	databases map[string]DB
}

var all *AllDatabases
var dbMutex sync.Mutex

func Databases() *AllDatabases {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	if all == nil {
		all = &AllDatabases{}
		all.databases = make(map[string]DB)
	}
	return all
}

func (a *AllDatabases) Register(db DB) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	a.databases[db.ID()] = db
}

func (a *AllDatabases) Unregister(db DB) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	delete(a.databases, db.ID())
}

func (a *AllDatabases) CloseAll() {
	dbMutex.Lock()
	databases := utils.AllMapValues(a.databases)
	dbMutex.Unlock()
	for _, db := range databases {
		db.Close()
	}
}
