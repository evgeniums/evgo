package tenancy_api

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/multitenancy"
)

const ServiceName string = "tenancies"
const TenancyResource string = "tenancy"
const IpAddressResource string = "ip-address"

type IpAddressCmd = multitenancy.IpAddressCmd

type TenancyResponse struct {
	api.ResponseBase
	*multitenancy.TenancyItem
}

type ListTenanciesResponse = api.ResponseList[*multitenancy.TenancyItem]
type ListIpAddressesResponse = api.ResponseList[*multitenancy.TenancyIpAddressItem]

type DeleteTenancyCmd struct {
	WithDatabase bool `json:"with_database"`
}

var (
	List            = func() api.Operation { return api.List("list_tenancies") }
	Add             = func() api.Operation { return api.Add("add_tenancy") }
	Find            = func() api.Operation { return api.Find("find_tenancy") }
	Exists          = func() api.Operation { return api.Exists("tenancy_exists") }
	Delete          = func() api.Operation { return api.Delete("delete_tenancy") }
	SetActive       = func() api.Operation { return api.Update("set_tenancy_active") }
	SetPath         = func() api.Operation { return api.Update("set_tenancy_path") }
	SetShadowPath   = func() api.Operation { return api.Update("set_tenancy_shadow_path") }
	SetRole         = func() api.Operation { return api.Update("set_tenancy_role") }
	SetCustomer     = func() api.Operation { return api.Update("set_tenancy_customer") }
	ChangePoolOrDb  = func() api.Operation { return api.UpdatePartial("change_tenancy_pool_or_db") }
	AddIpAddress    = func() api.Operation { return api.Add("add_tenancy_ip_address") }
	DeleteIpAddress = func() api.Operation { return api.Update("delete_tenancy_ip_address") }
	ListIpAddresses = func() api.Operation { return api.List("list_tenancy_ip_address") }
	SetDbRole       = func() api.Operation { return api.Update("set_db_role") }
	SetPathBlocked  = func() api.Operation { return api.UpdatePartial("set_path_blocked") }
)
