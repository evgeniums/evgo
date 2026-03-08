package multitenancy

import (
	"context"
	"net/http"

	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_subscriber"
	"github.com/evgeniums/evgo/pkg/utils"
)

const (
	OpAdd             string = "add"
	OpDelete          string = "delete"
	OpActivate        string = "activate"
	OpDeactivate      string = "deactivate"
	OpSetPath         string = "set_path"
	OpSetShadowPath   string = "set_shadow_path"
	OpSetRole         string = "set_role"
	OpSetCustomer     string = "set_customer"
	OpChangePoolOrDb  string = "change_pool_or_db"
	OpAddIpAddress    string = "add_ip_address"
	OpDeleteIpAddress string = "delete_ip_address"
	OpSetDbRole       string = "set_db_role"
	OpSetPathBlocked  string = "set_path_blocked"
)

const (
	TENANCY_DATABASE_ROLE string = "tenancy_db"
)

type TenancyBlockPathMode string

const (
	TenancyBlockPathModeDefault TenancyBlockPathMode = "default"
	TenancyBlockPathModeShadow  TenancyBlockPathMode = "shadow"
	TenancyBlockPathModeBoth    TenancyBlockPathMode = "both"
)

const (
	ErrorCodeTenancyConflictRole           = "tenancy_conflict_role"
	ErrorCodeTenancyConflictPath           = "tenancy_conflict_path"
	ErrorCodeTenancyNotFound               = "tenancy_not_found"
	ErrorCodeTenancyDbInitializationFailed = "tenancy_db_initialization_failed"
	ErrorCodeForeignDatabase               = "foreign_tenancy_database"
	ErrorCodeNoDbserviceInPool             = "no_db_service_in_pool"
)

var ErrorDescriptions = map[string]string{
	ErrorCodeTenancyNotFound:               "Tenancy not found",
	ErrorCodeTenancyConflictRole:           "Tenancy with such role already exists for that customer",
	ErrorCodeTenancyConflictPath:           "Tenancy with such path already exists in that pool",
	ErrorCodeTenancyDbInitializationFailed: "Failed to initialize tenancy database",
	ErrorCodeForeignDatabase:               "Database does not belong to this tenancy",
	ErrorCodeNoDbserviceInPool:             "Pool does not contain service for tenancy database",
}

var ErrorHttpCodes = map[string]int{
	ErrorCodeTenancyNotFound:               http.StatusNotFound,
	ErrorCodeNoDbserviceInPool:             http.StatusInternalServerError,
	ErrorCodeTenancyDbInitializationFailed: http.StatusInternalServerError,
	ErrorCodeForeignDatabase:               http.StatusInternalServerError,
}

type Multitenancy interface {

	// Check if multiple tenancies are enabled
	IsMultiTenancy() bool

	// Get all tenancies
	Tenancies() []Tenancy

	// Find tenancy by ID.
	Tenancy(id string) (Tenancy, error)

	// Find tenancy by path.
	TenancyByPath(path string) (Tenancy, error)

	// Find tenancy by shadow path.
	TenancyByShadowPath(path string) (Tenancy, error)

	// Load tenancy.
	LoadTenancy(sctx context.Context, id string) (Tenancy, error)

	// Unload tenancy.
	UnloadTenancy(id string)

	// Create tenancy
	CreateTenancy(sctx context.Context, data *TenancyData) (*TenancyItem, error)

	// Get tenancy controller.
	TenancyController() TenancyController

	// Check if ip address is in the list of tenancy addresses.
	HasIpAddressByPath(path string, ipAddress string, tag string) bool

	// Close tenancies, e.g. close tenancy databases.
	Close()
}

func IsMultiTenancy(m Multitenancy) bool {
	if m == nil {
		return false
	}
	return m.IsMultiTenancy()
}

type PubsubNotification struct {
	Tenancy   string `json:"tenancy"`
	Operation string `json:"operation"`
}

const PubsubTopicName = "tenancy"

type PubsubTopic struct {
	*pubsub_subscriber.TopicBase[*PubsubNotification]
}

func NewPubsubNotification() *PubsubNotification {
	return &PubsubNotification{}
}

type TenancyController interface {
	generic_error.ErrorsExtender
	Add(sctx context.Context, tenancy *TenancyData) (*TenancyItem, error)
	Find(sctx context.Context, id string, idIsDisplay ...bool) (*TenancyItem, error)
	List(sctx context.Context, filter *db.Filter) ([]*TenancyItem, int64, error)

	Exists(sctx context.Context, fields db.Fields) (bool, error)
	Delete(sctx context.Context, id string, withDb bool, idIsDisplay ...bool) error

	SetPath(sctx context.Context, id string, path string, idIsDisplay ...bool) error
	SetShadowPath(sctx context.Context, id string, path string, idIsDisplay ...bool) error
	SetCustomer(sctx context.Context, id string, customerId string, idIsDisplay ...bool) error
	SetRole(sctx context.Context, id string, role string, idIsDisplay ...bool) error
	ChangePoolOrDb(sctx context.Context, id string, poolId string, dbName string, idIsDisplay ...bool) error
	Activate(sctx context.Context, id string, idIsDisplay ...bool) error
	Deactivate(sctx context.Context, id string, idIsDisplay ...bool) error
	SetDbRole(sctx context.Context, id string, dbRole string, idIsDisplay ...bool) error

	SetPathBlocked(sctx context.Context, id string, blocked bool, mode TenancyBlockPathMode, idIsDisplay ...bool) error

	ListIpAddresses(sctx context.Context, filter *db.Filter) ([]*TenancyIpAddressItem, int64, error)
	DeleteIpAddress(sctx context.Context, id string, ipAddress string, tag string, idIsDisplay ...bool) error
	AddIpAddress(sctx context.Context, id string, ipAddress string, tag string, idIsDisplay ...bool) error
}

func TenancyId(ctrl TenancyController, sctx context.Context, id string, idIsDisplay ...bool) (string, *TenancyItem, error) {

	useDisplay := utils.OptionalArg(false, idIsDisplay...)

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("TenancyId", logger.Fields{"tenancy": id, "use_display": useDisplay})
	defer ctx.TraceOutMethod()

	// return ID as is if it is not display format
	if !useDisplay {
		return id, nil, nil
	}

	// parse id
	customerLogin, role, vErr := ParseTenancyDisplay(id)
	if vErr != nil {
		c.SetMessage("failed to parse display")
		ctx.SetGenericError(vErr.GenericError())
		c.SetError(vErr)
		return "", nil, vErr.GenericError()
	}

	// find tenancy by login and role
	filter := db.NewFilter()
	filter.AddField("customers.login", customerLogin)
	filter.AddField("role", role)
	filter.Limit = 1
	tenancies, _, err := ctrl.List(sctx, filter)
	if err != nil {
		c.SetMessage("failed to list tenancies")
		return "", nil, c.SetError(err)
	}
	if len(tenancies) == 0 {
		ctx.SetGenericErrorCode(ErrorCodeTenancyNotFound)
		return "", nil, c.SetError(ctx.GenericError())
	}
	tenancy := tenancies[0]

	// done
	return tenancy.GetID(), tenancy, nil
}

func FindTenancy(ctrl TenancyController, sctx context.Context, id string, idIsDisplay ...bool) (*TenancyItem, error) {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("FindTenancy")
	defer ctx.TraceOutMethod()

	// adjust ID
	id, tenancy, err := TenancyId(ctrl, sctx, id, idIsDisplay...)
	if err != nil {
		return nil, c.SetError(err)
	}

	// maybe done
	if tenancy != nil {
		return tenancy, nil
	}

	// find tenancy
	filter := db.NewFilter()
	filter.AddField("tenancies.id", id)
	filter.Limit = 1
	tenancies, _, err := ctrl.List(sctx, filter)
	if err != nil {
		return nil, c.SetError(err)
	}
	if len(tenancies) == 0 {
		ctx.SetGenericErrorCode(ErrorCodeTenancyNotFound)
		return nil, c.SetError(ctx.GenericError())
	}
	tenancy = tenancies[0]

	// done
	return tenancy, nil
}

func ListTenancyIpAddresses(ctrl TenancyController, sctx context.Context, id string, filter *db.Filter, idIsDisplay ...bool) ([]*TenancyIpAddressItem, int64, error) {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("ListTenancyIpAddresses")
	defer ctx.TraceOutMethod()

	// find out tenancy ID
	tenancyId, _, err := TenancyId(ctrl, sctx, id, idIsDisplay...)
	if err != nil {
		c.SetMessage("failed to find out tenancy ID")
		return nil, 0, c.SetError(err)
	}

	// prepare filter
	f := filter
	if f == nil {
		f = db.NewFilter()
	} else {
		defer delete(filter.Fields, "tenancies.id")
	}
	f.AddField("tenancies.id", tenancyId)
	items, count, err := ctrl.ListIpAddresses(sctx, f)
	if err != nil {
		c.SetMessage("failed to list IP addresses")
		return nil, 0, c.SetError(err)
	}

	// done
	return items, count, nil
}
