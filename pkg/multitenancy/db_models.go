package multitenancy

import "github.com/evgeniums/evgo/pkg/common"

type TenancyMeta struct {
	common.ObjectBase
}

func DbModels() []interface{} {
	return []interface{}{&TenancyDb{}, &TenancyIpAddress{}, &OpLogTenancy{}}
}

func DbInternalModels() []interface{} {
	return []interface{}{&TenancyMeta{}}
}

type TenancyDbModels struct {
	DbModels            []interface{}
	PartitionedDbModels []interface{}
}

func QueryDbModels() []interface{} {
	return []interface{}{&TenancyItem{}, &OpLogTenancy{}}
}
