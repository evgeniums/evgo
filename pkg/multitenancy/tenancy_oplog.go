package multitenancy

import "github.com/evgeniums/evgo/pkg/oplog"

type OpLogTenancy struct {
	oplog.OplogBase
	Customer        string `gorm:"index" json:"customer"`
	TenancyId       string `gorm:"index" json:"tenancy_id"`
	Role            string `gorm:"index" json:"role"`
	Path            string `gorm:"index" json:"path"`
	ShadowPath      string `gorm:"index" json:"shadow_path"`
	DbName          string `gorm:"index" json:"db_name"`
	Pool            string `gorm:"index" json:"pool"`
	IpAddress       string `gorm:"index" json:"ip_address"`
	IpAddressTag    string `gorm:"index" json:"ip_address_tag"`
	DbRole          string `gorm:"index" json:"db_role"`
	BlockPath       bool   `gorm:"index" json:"block_path"`
	BlockShadowPath bool   `gorm:"index" json:"block_shadow_path"`
}
