package db

import (
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/utils"
)

type MonthPartition struct {
	common.ObjectBase
	Table string      `gorm:"index;uniqueIndex:u_month_partition"`
	Month utils.Month `gorm:"index;uniqueIndex:u_month_partition"`
}
