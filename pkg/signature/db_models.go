package signature

import (
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/user"
)

type MessageSignature struct {
	common.ObjectWithMonthBase
	auth.WithUserBase
	Context    string `gorm:"index;index:,unique,composite:u_month"`
	Operation  string `gorm:"index"`
	Algorithm  string `gorm:"index"`
	Message    string
	Signature  string
	ExtraData  string `gorm:"index"`
	PubKeyHash string `gorm:"index"`
}

type OpLogPubKey struct {
	user.OpLogUser
	KeyId   string `gorm:"index" json:"key_id"`
	KeyHash string `gorm:"index" json:"key_hash"`
}

func DbModels() []interface{} {
	return []interface{}{&MessageSignature{}, &OpLogPubKey{}}
}

func QueryDbModels() []interface{} {
	return []interface{}{&OpLogPubKey{}, &MessageSignature{}}
}
