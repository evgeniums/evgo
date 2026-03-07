package pool_api

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/pool"
)

type PoolResponse struct {
	api.ResponseBase
	*pool.PoolBase
}

type ServiceResponse struct {
	api.ResponseBase
	*pool.PoolServiceBase
}

type ListServicesResponse = api.ResponseList[*pool.PoolServiceBase]

type ListPoolsResponse = api.ResponseList[*pool.PoolBase]

type ListServicePoolsResponse = api.ResponseList[*pool.PoolServiceBinding]

var (
	UpdateService             = func() api.Operation { return api.UpdatePartial("update_service") }
	UpdatePool                = func() api.Operation { return api.UpdatePartial("update_pool") }
	RemoveServiceFromPool     = func() api.Operation { return api.Unbind("remove_service_from_pool") }
	RemoveServiceFromAllPools = func() api.Operation { return api.Unbind("remove_service_from_all_pools") }
	RemoveAllServicesFromPool = func() api.Operation { return api.Unbind("remove_all_services_from_pool") }
	ListServices              = func() api.Operation { return api.List("list_services") }
	ListServicePools          = func() api.Operation { return api.List("list_service_pools") }
	FindService               = func() api.Operation { return api.Find("find_service") }
	FindPool                  = func() api.Operation { return api.Find("find_pool") }
	DeleteService             = func() api.Operation { return api.Delete("delete_service") }
	DeletePool                = func() api.Operation { return api.Delete("delete_pool") }
	AddService                = func() api.Operation { return api.Add("add_service") }
	AddServiceToPool          = func() api.Operation { return api.Bind("add_service_to_pool") }
	AddPool                   = func() api.Operation { return api.Add("add_pool") }
	ListPools                 = func() api.Operation { return api.List("list_pools") }
	ListPoolServices          = func() api.Operation { return api.List("list_pool_services") }
)
