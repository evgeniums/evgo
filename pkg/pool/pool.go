package pool

import (
	"errors"

	"github.com/evgeniums/evgo/pkg/common"
)

type Pool interface {
	common.Object
	common.WithUniqueName
	common.WithLongName
	common.WithDescription
	common.WithActive

	Service(role string) (*PoolServiceBinding, error)
	SetServices([]*PoolServiceBinding)

	ServiceByName(name string) (*PoolServiceBinding, error)
}

type PoolBaseData struct {
	common.WithUniqueNameBase
	common.WithLongNameBase
	common.WithDescriptionBase
	common.WithActiveBase
}

func (p *PoolBaseData) Fill(pool Pool) {
	p.SetName(pool.Name())
	p.SetActive(pool.IsActive())
	p.SetDescription(pool.Description())
	p.SetLongName(pool.LongName())
}

type PoolBaseEssentials struct {
	common.ObjectBase
	PoolBaseData
}

type PoolBase struct {
	PoolBaseEssentials
	Services map[string]*PoolServiceBinding `gorm:"-:all" json:"-"`
}

func NewPool() *PoolBase {
	p := &PoolBase{}
	InitPool(p)
	return p
}

func InitPool(p *PoolBase) {
	p.Init()
	p.Services = make(map[string]*PoolServiceBinding)
}

func (PoolBase) TableName() string {
	return "pools"
}

func (p *PoolBase) Service(role string) (*PoolServiceBinding, error) {

	service, ok := p.Services[role]
	if !ok {
		return nil, errors.New("service not found")
	}

	return service, nil
}

func (p *PoolBase) ServiceByName(name string) (*PoolServiceBinding, error) {

	for _, service := range p.Services {
		if service.ServiceName == name {
			return service, nil
		}
	}

	return nil, errors.New("service not found")
}

func (p *PoolBase) SetServices(services []*PoolServiceBinding) {
	p.Services = make(map[string]*PoolServiceBinding)
	for _, service := range services {
		p.Services[service.Role()] = service
	}
}
