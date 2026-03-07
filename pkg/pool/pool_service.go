package pool

import (
	"github.com/evgeniums/evgo/pkg/common"
)

type Secrets interface {
	Secret1() string
	Secret2() string
}

type ServiceConfig interface {
	Provider() string
	PublicHost() string
	PublicPort() uint16
	PublicUrl() string
	PrivateHost() string
	PrivatePort() uint16
	PrivateUrl() string
	Parameter1() string
	Parameter2() string
	Parameter3() string
	Parameter1Name() string
	Parameter2Name() string
	Parameter3Name() string
	User() string
	IpAddress() string
	ApiVersion() string
	PathPrefix() string
}

type PoolServiceEssentials interface {
	common.WithLongName
	common.WithDescription
	common.WithActive
	common.WithType
	common.WithRefId
	ServiceConfig
}

type PoolService interface {
	common.Object
	common.WithUniqueName
	PoolServiceEssentials
	Secrets
}

var NilService PoolService

type SecretsBase struct {
	SECRET1 string `json:"secret1" long:"secret1" description:"If set then sak for secret 1"`
	SECRET2 string `json:"secret2" long:"secret2" description:"If set then ask for secret 2"`
}

func (s *SecretsBase) Secret1() string {
	return s.SECRET1
}

func (s *SecretsBase) Secret2() string {
	return s.SECRET2
}

type ServiceConfigBase struct {
	PROVIDER        string `gorm:"index;column:provider" json:"provider" long:"provider" description:"Service provider" required:"true"`
	PUBLIC_HOST     string `gorm:"index" json:"public_host" long:"public_host" description:"Public host of the service (optional)"`
	PUBLIC_PORT     uint16 `gorm:"index" json:"public_port" long:"public_port" description:"Public port of the service (optional)"`
	PUBLIC_URL      string `gorm:"index" json:"public_url" long:"public_url" description:"Public url of the service (optional)" display:"Public URL"`
	PRIVATE_HOST    string `gorm:"index" json:"private_host" long:"private_host" description:"Private host of the service (optional)"`
	PRIVATE_PORT    uint16 `gorm:"index" json:"private_port" long:"private_port" description:"Private port of the service (optional)"`
	PRIVATE_URL     string `gorm:"index" json:"private_url" long:"private_url" description:"Private URL of the service (optional)" display:"Private URL"`
	USER            string `gorm:"index" json:"user" long:"user" description:"User for login to the service (optional)"`
	DB_NAME         string `gorm:"index;column:db_name" json:"db_name" long:"db_name" description:"Name of database (optional)" display:"Database"`
	PARAMETER1      string `gorm:"index;column:parameter1" json:"parameter1" long:"parameter1" description:"Generic parameter1 of the service (optional)"`
	PARAMETER2      string `gorm:"index;column:parameter2" json:"parameter2" long:"parameter2" description:"Generic parameter2 of the service (optional)"`
	PARAMETER3      string `gorm:"index;column:parameter3" json:"parameter3" long:"parameter3" description:"Generic parameter3 of the service (optional)"`
	PARAMETER1_NAME string `gorm:"index;column:parameter1_name" json:"parameter1_name" long:"parameter1_name" description:"Name of generic parameter1 of the service (optional)"`
	PARAMETER2_NAME string `gorm:"index;column:parameter2_name" json:"parameter2_name" long:"parameter2_name" description:"Name of generic parameter2 of the service (optional)"`
	PARAMETER3_NAME string `gorm:"index;column:parameter3_name" json:"parameter3_name" long:"parameter3_name" description:"Name of generic parameter3 of the service (optional)"`
	IP_ADDRESS      string `gorm:"index" json:"ip_address" long:"ip_address" description:"IP address of the service (optional)"`
	API_VERSION     string `gorm:"index" json:"api_version" long:"api_version" default:"1.0.0" description:"API version of the service (optional)"`
	PATH_PREFIX     string `gorm:"index" json:"path_prefix" long:"path_prefix" default:"/api" description:"URL path prefix of the service (optional)"`
}

func (s *ServiceConfigBase) User() string {
	return s.USER
}

func (s *ServiceConfigBase) DbName() string {
	return s.DB_NAME
}

func (s *ServiceConfigBase) IpAddress() string {
	return s.IP_ADDRESS
}

func (s *ServiceConfigBase) PathPrefix() string {
	return s.PATH_PREFIX
}

func (s *ServiceConfigBase) ApiVersion() string {
	return s.API_VERSION
}

func (s *ServiceConfigBase) PublicHost() string {
	return s.PUBLIC_HOST
}

func (s *ServiceConfigBase) PublicPort() uint16 {
	return s.PUBLIC_PORT
}

func (s *ServiceConfigBase) PublicUrl() string {
	return s.PUBLIC_URL
}

func (s *ServiceConfigBase) PrivateHost() string {
	if s.PRIVATE_HOST == "" {
		return s.IP_ADDRESS
	}
	return s.PRIVATE_HOST
}

func (s *ServiceConfigBase) PrivatePort() uint16 {
	return s.PRIVATE_PORT
}

func (s *ServiceConfigBase) PrivateUrl() string {
	return s.PRIVATE_URL
}

func (s *ServiceConfigBase) Parameter1() string {
	return s.PARAMETER1
}

func (s *ServiceConfigBase) Parameter2() string {
	return s.PARAMETER2
}

func (s *ServiceConfigBase) Parameter3() string {
	return s.PARAMETER3
}

func (s *ServiceConfigBase) Parameter1Name() string {
	return s.PARAMETER1_NAME
}

func (s *ServiceConfigBase) Parameter2Name() string {
	return s.PARAMETER2_NAME
}

func (s *ServiceConfigBase) Parameter3Name() string {
	return s.PARAMETER3_NAME
}

func (s *ServiceConfigBase) Provider() string {
	return s.PROVIDER
}

type PoolServiceBaseData struct {
	common.WithLongNameBase
	common.WithDescriptionBase
	common.WithActiveBase
	common.WithTypeBase
	common.WithRefIdBase
	ServiceConfigBase
	SecretsBase
}

type PoolServiceBaseEssentials struct {
	PoolServiceBaseData
	common.WithUniqueNameBase
}

type PoolServiceBase struct {
	common.ObjectBase
	PoolServiceBaseEssentials
}

func NewService() *PoolServiceBase {
	s := &PoolServiceBase{}
	InitService(s)
	return s
}

func InitService(s *PoolServiceBase) {
	s.Init()
}

func (PoolServiceBase) TableName() string {
	return "pool_services"
}
