package pool

import (
	"github.com/evgeniums/evgo/pkg/common"
)

type PoolServiceAssociation interface {
	common.Object
	Pool() string
	Role() string
	Service() string
}

type PoolId struct {
	POOL_ID string `gorm:"index;index:,unique,composite:u" json:"pool_id"`
}

type WithRole struct {
	ROLE string `gorm:"index;index:,unique,composite:u" json:"role" validate:"required,alphanum_" vmessage:"Role name can contain only digits and letters"`
}

type PoolServiceAssociationCmd struct {
	WithRole
	SERVICE_ID string `gorm:"index" json:"service_id" validate:"required" vmessage:"Service ID can not be empty"`
}

type PoolServiceAssociationEssentials struct {
	PoolId
	PoolServiceAssociationCmd
}

type PoolServiceAssociationBase struct {
	common.ObjectBase
	PoolServiceAssociationEssentials
}

func (p *PoolServiceAssociationEssentials) Pool() string {
	return p.POOL_ID
}

func (p *WithRole) Role() string {
	return p.ROLE
}

func (p *PoolServiceAssociationCmd) Service() string {
	return p.SERVICE_ID
}

func (PoolServiceAssociationBase) TableName() string {
	return "pool_service_associations"
}

type PoolBindingServiceFields struct {
	common.IDBase
	PoolServiceBaseEssentials
}

type PoolBindingPoolFields struct {
	common.IDBase
	common.WithUniqueNameBase
}

type PoolServiceBinding struct {
	common.ObjectBase   `source:"pool_service_associations"`
	WithRole            `source:"pool_service_associations"`
	PoolServiceBaseData `source:"pool_services"`
	PoolId              string `json:"pool_id" source:"pools.id" display:"Pool ID"`
	PoolName            string `json:"pool_name" source:"pools.name" display:"Pool"`
	ServiceId           string `json:"service_id" source:"pool_services.id" display:"Service ID"`
	ServiceName         string `json:"service_name" source:"pool_services.name" display:"Service"`
}

func (s *PoolServiceBinding) Name() string {
	return s.ServiceName
}

func (s *PoolServiceBinding) SetName(name string) {
	s.ServiceName = name
}
