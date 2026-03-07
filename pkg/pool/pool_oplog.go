package pool

import "github.com/evgeniums/evgo/pkg/oplog"

type OpLogPool struct {
	oplog.OplogBase
	PoolName    string `gorm:"index" json:"pool_name"`
	PoolId      string `gorm:"index" json:"pool_id"`
	ServiceName string `gorm:"index" json:"service_name"`
	ServiceId   string `gorm:"index" json:"service_id"`
	Role        string `gorm:"index" json:"role"`
}
