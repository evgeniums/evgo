package multitenancy

import (
	"errors"
	"strings"

	"github.com/evgeniums/go-utils/pkg/app_context"
	"github.com/evgeniums/go-utils/pkg/cache"
	"github.com/evgeniums/go-utils/pkg/common"
	"github.com/evgeniums/go-utils/pkg/db"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/op_context/default_op_context"
	"github.com/evgeniums/go-utils/pkg/pool"
	"github.com/evgeniums/go-utils/pkg/pool/app_with_pools"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/evgeniums/go-utils/pkg/validator"
)

type TenancyPath interface {
	common.WithID
	Path() string
}

type Tenancy interface {
	common.Object
	common.WithActive
	common.WithDescription

	Path() string
	ShadowPath() string
	CustomerId() string
	CustomerDisplay() string
	Role() string
	DbName() string
	DbRole() string
	PoolId() string

	Db() db.DB
	Pool() pool.Pool
	Cache() cache.Cache

	IsBlockedPath() bool
	IsBlockedShadowPath() bool
}

type WithPath struct {
	PATH        string `json:"path" gorm:"uniqueIndex" validate:"omitempty,alphanum_" vmessage:"Path must be alhanumeric" long:"path" description:"Path of tenancy resource in API service"`
	SHADOW_PATH string `json:"shadow_path" gorm:"index" validate:"omitempty,alphanum_" vmessage:"Shadow path must be alhanumeric" long:"shadow_path" description:"Shadow path of tenancy resource in API service"`
}

type WithRole struct {
	ROLE string `json:"role" gorm:"index;index:,unique,composite:u" validate:"required,alphanum_" vmessage:"Role must be alphanumeric not empty" long:"role" description:"Role of this tenancy for customer, each tenancy must have unique role per customer" required:"true"`
}

type WithCustomerId struct {
	CUSTOMER_ID string `json:"customer_id" gorm:"index;index:,unique,composite:u" validate:"required,alphanum_|email" vmessage:"Invalid customer ID" long:"customer" description:"ID or name of a customer that will own the tenancy" required:"true" display:"Customer ID"`
}

type WithPoolAndDb struct {
	POOL_ID string `json:"pool_id" gorm:"index" validate:"omitempty,alphanum" vmessage:"Pool ID must be alhanumeric" long:"pool" description:"Name or ID of a pool this tenancy belongs to" display:"Pool ID"`
	DBNAME  string `json:"dbname" gorm:"index;column:dbname" validate:"omitempty,alphanum_" vmessage:"Database name must be alhanumeric" long:"dbname" description:"Name of tenancy's database, if empty then will be generated automatically" display:"Database name"`
}

type WithDbRole struct {
	DB_ROLE string `json:"db_role" gorm:"index;column:db_role" validate:"omitempty,alphanum_" vmessage:"Database role must be alhanumeric" long:"db_role" description:"Role of database service in the pool" display:"Database service role"`
}

type WithBlockPath struct {
	BLOCK_PATH        bool `json:"block_path" gorm:"index;column:block_path" display:"Block tenancy path"`
	BLOCK_SHADOW_PATH bool `json:"block_shadow_path" gorm:"index;column:block_shadow_path" display:"Block shadow tenancy path"`
}

type BlockPathCmd struct {
	Block bool                 `json:"block" long:"block" description:"Block or unblock tenancy path"`
	Mode  TenancyBlockPathMode `json:"mode" long:"mode" description:"Mode: default | shadow | both"`
}

func (t *WithBlockPath) IsBlockedPath() bool {
	return t.BLOCK_PATH
}

func (t *WithBlockPath) IsBlockedShadowPath() bool {
	return t.BLOCK_SHADOW_PATH
}

type TenancyData struct {
	common.WithDescriptionBase
	WithPath
	WithRole
	WithCustomerId
	WithPoolAndDb
	WithDbRole
	WithBlockPath
}

func (t *WithCustomerId) CustomerId() string {
	return t.CUSTOMER_ID
}

func (t *WithRole) Role() string {
	return t.ROLE
}

func (t *WithPath) Path() string {
	return t.PATH
}

func (t *WithPath) SetPath(path string) {
	t.PATH = path
}

func (t *WithPath) ShadowPath() string {
	return t.SHADOW_PATH
}

func (t *WithPath) SetShadowPath(path string) {
	t.SHADOW_PATH = path
}

func (t *WithPoolAndDb) PoolId() string {
	return t.POOL_ID
}

func (t *WithPoolAndDb) DbName() string {
	return t.DBNAME
}

func (t *WithDbRole) DbRole() string {
	return t.DB_ROLE
}

func (t *WithDbRole) SetDbRole(role string) {
	t.DB_ROLE = role
}

func TenancyDisplay(t Tenancy) string {

	if t.CustomerDisplay() != "" {
		return utils.ConcatStrings(t.CustomerDisplay(), "/", t.Role())
	}

	if t.GetID() == "" {
		return t.Path()
	}

	return t.GetID()
}

func TenancySelector(customer string, role string) string {
	return utils.ConcatStrings(customer, "/", role)
}

func ParseTenancyDisplay(display string) (string, string, *validator.ValidationError) {
	parts := strings.Split(display, "/")
	if len(parts) != 2 {
		err := &validator.ValidationError{Message: "invalid format of tenancy ID"}
		return "", "", err
	}
	return parts[0], parts[1], nil
}

type TenancyDb struct {
	common.ObjectBase
	common.WithActiveBase
	TenancyData
}

func (TenancyDb) TableName() string {
	return "tenancies"
}

type TenancyItem struct {
	TenancyDb     `source:"tenancies"`
	CustomerLogin string `json:"customer_login" source:"customers.login" gorm:"index" display:"Customer"`
	PoolName      string `json:"pool_name" source:"pools.name" gorm:"index" display:"Pool"`
}

func (TenancyItem) TableName() string {
	return "tenancy_items"
}

func (t *TenancyItem) CustomerDisplay() string {
	return t.CustomerLogin
}

func CheckTenancyDatabase(ctx op_context.Context, database db.DB, tenancyId string) error {

	c := ctx.TraceInMethod("CheckTenancyDatabase", logger.Fields{"tenancy_id": tenancyId})
	defer ctx.TraceOutMethod()

	filter := db.NewFilter()
	filter.AddField("id", tenancyId)
	exists, err := database.Exists(ctx, filter, &TenancyMeta{})
	if err != nil {
		return c.SetError(err)
	}
	if !exists {
		ctx.SetGenericErrorCode(ErrorCodeForeignDatabase)
		return c.SetError(errors.New("database does not belong to this tenancy"))
	}
	return nil
}

type WithTenancy interface {
	GetTenancy() Tenancy
}

type TenancyContext interface {
	app_with_pools.Context
	WithTenancy
}

func ContextTenancyId(ctx TenancyContext) string {
	if ctx.GetTenancy() == nil {
		return ""
	}
	return ctx.GetTenancy().GetID()
}

func ContextTenancyPath(ctx TenancyContext) string {
	if ctx.GetTenancy() == nil {
		return ""
	}
	return ctx.GetTenancy().Path()
}

func ContextTenancyDisplay(ctx TenancyContext) string {
	if ctx.GetTenancy() == nil {
		return ""
	}
	return TenancyDisplay(ctx.GetTenancy())
}

func ContextTenancyShadowPath(ctx TenancyContext) string {
	if ctx.GetTenancy() == nil {
		return ""
	}
	return ctx.GetTenancy().ShadowPath()
}

type TenancyContextBase struct {
	app_with_pools.ContextBase
	Tenancy Tenancy
}

func (u *TenancyContextBase) GetTenancy() Tenancy {
	return u.Tenancy
}

func (u *TenancyContextBase) SetTenancy(tenancy Tenancy) {
	u.Tenancy = tenancy
	u.SetLoggerField("tenancy", TenancyDisplay(tenancy))
	if tenancy.Cache() != nil {
		u.SetCache(tenancy.Cache())
	}
	if u.OplogHandler() != nil && u.OplogWriter() == nil {
		u.SetOplogWriter(u.OplogHandler()(u))
	}
}

func (u *TenancyContextBase) Db() db.DB {
	t := u.GetTenancy()
	if t != nil && t.Db() != nil {
		return t.Db()
	}
	return u.ContextBase.Db()
}

func (u *TenancyContextBase) Pool() pool.Pool {
	t := u.GetTenancy()
	if t != nil && t.Pool() != nil {
		return t.Pool()
	}
	return u.ContextBase.Pool()
}

func NewContext(fromCtx ...op_context.Context) *TenancyContextBase {
	c := &TenancyContextBase{}
	c.Construct(fromCtx...)
	return c
}

func NewInitContext(app app_context.Context, log logger.Logger, db db.DB) *TenancyContextBase {
	c := default_op_context.NewContext()
	c.Init(app, log, db)
	t := NewContext(c)
	return t
}

type IpAddressCmd struct {
	Ip  string `json:"ip" validate:"required,ip" vmessage:"Invalid IP address" long:"ip" description:"IP address" required:"true"`
	Tag string `json:"tag" validate:"required,alphanum_" vmessage:"Invalid tag" long:"tag" description:"Tag for IP address group" required:"true"`
}

type TenancyIpAddress struct {
	common.ObjectBase
	TenancyId string `gorm:"index" json:"tenancy_id"`
	Ip        string `gorm:"index" json:"ip"`
	Tag       string `gorm:"index" json:"tag"`
}

type TenancyIpAddressItem struct {
	TenancyIpAddress `source:"tenancy_ip_addresses"`
	PoolName         string `json:"pool_name" source:"pools.name" gorm:"index" display:"Pool"`
	CustomerLogin    string `json:"customer_login" source:"customers.login" gorm:"index" display:"Customer"`
	TenancyRole      string `json:"tenancy_role" source:"tenancies.role" gorm:"index" display:"Tenancy role"`
}

func CloseTenancyDb(tenancy Tenancy) {
	if tenancy.Db() != nil {
		tenancy.Db().Close()
	}
}
