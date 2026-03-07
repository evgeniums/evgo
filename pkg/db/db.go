package db

import (
	"sync"
	"time"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/logger"
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
	FindByField(ctx logger.WithLogger, field string, value interface{}, obj interface{}, dest ...interface{}) (found bool, err error)
	FindByFields(ctx logger.WithLogger, fields Fields, obj interface{}, dest ...interface{}) (found bool, err error)
	FindWithFilter(ctx logger.WithLogger, filter *Filter, docs interface{}, dest ...interface{}) (int64, error)
	FindForUpdate(ctx logger.WithLogger, fields Fields, obj interface{}) (bool, error)
	FindForShare(ctx logger.WithLogger, fields Fields, obj interface{}) (bool, error)
	Exists(ctx logger.WithLogger, filter *Filter, doc interface{}) (bool, error)

	Create(ctx logger.WithLogger, obj interface{}) error
	CreateDup(ctx logger.WithLogger, obj interface{}, ignoreConflict ...bool) (bool, error)

	Delete(ctx logger.WithLogger, obj common.Object) error
	DeleteByField(ctx logger.WithLogger, field string, value interface{}, model interface{}) error
	DeleteByFields(ctx logger.WithLogger, fields Fields, obj interface{}) error

	RowsWithFilter(ctx logger.WithLogger, filter *Filter, obj interface{}) (Cursor, error)
	AllRows(ctx logger.WithLogger, obj interface{}) (Cursor, error)

	Update(ctx logger.WithLogger, obj interface{}, filter Fields, fields Fields) error
	UpdateAll(ctx logger.WithLogger, obj interface{}, newFields Fields) error
	UpdateWithFilter(ctx logger.WithLogger, obj interface{}, filter *Filter, newFields Fields) error

	Join(ctx logger.WithLogger, joinConfig *JoinQueryConfig, filter *Filter, dest interface{}) (int64, error)

	Joiner() Joiner

	CreateDatabase(ctx logger.WithLogger, dbName string) error
	MakeExpression(expr string, args ...interface{}) interface{}

	Sum(ctx logger.WithLogger, groupFields []string, sumFields []string, filter *Filter, model interface{}, dest ...interface{}) (int64, error)

	Transaction(handler TransactionHandler) error
	EnableDebug(bool)
}

type Transaction interface {
	DBHandlers
}

type TransactionHandler = func(tx Transaction) error

type Cursor interface {
	Next(ctx logger.WithLogger) (bool, error)
	Close(ctx logger.WithLogger) error
	Scan(ctx logger.WithLogger, obj interface{}) error
}

type DB interface {
	WithFilterParser

	ID() string
	Clone() DB

	InitWithConfig(ctx logger.WithLogger, vld validator.Validator, cfg *DBConfig) error

	DBHandlers

	EnableVerboseErrors(bool)

	AutoMigrate(ctx logger.WithLogger, models []interface{}) error
	MigrateDropIndex(ctx logger.WithLogger, model interface{}, indexName string) error

	PartitionedMonthAutoMigrate(ctx logger.WithLogger, models []interface{}) error
	PartitionedMonthsDetach(ctx logger.WithLogger, table string, months []utils.Month) error
	PartitionedMonthsDelete(ctx logger.WithLogger, table string, months []utils.Month) error

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

func Update(db DBHandlers, ctx logger.WithLogger, obj common.Object, fields Fields) error {
	f := utils.CopyMap(fields)
	f["updated_at"] = time.Now()
	return db.Update(ctx, obj, nil, f)
}

func UpdateMulti(db DBHandlers, ctx logger.WithLogger, obj interface{}, filter Fields, fields Fields) error {
	f := utils.CopyMap(fields)
	f["updated_at"] = time.Now()
	return db.Update(ctx, obj, filter, f)
}

func UpdateAll(db DBHandlers, ctx logger.WithLogger, obj interface{}, fields Fields) error {
	f := utils.CopyMap(fields)
	f["updated_at"] = time.Now()
	return db.UpdateAll(ctx, obj, f)
}

func UpdateWithFilter(db DBHandlers, ctx logger.WithLogger, obj interface{}, filter *Filter, fields Fields) error {
	f := utils.CopyMap(fields)
	f["updated_at"] = time.Now()
	return db.UpdateWithFilter(ctx, obj, filter, f)
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
