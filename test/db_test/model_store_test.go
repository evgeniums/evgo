package db_test

import (
	"sync"
	"testing"

	"github.com/evgeniums/evgo/pkg/db/db_gorm"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

type TestStruct struct {
	DateField  utils.Date
	MonthField utils.Month
	Amount     int `money:"true" json:"amount"`
}

func TestModelDescriptor(t *testing.T) {

	cache := &sync.Map{}
	namer := &schema.NamingStrategy{}

	descr := db_gorm.NewModelDescriptor(&pool.PoolServiceBinding{}, cache, namer)
	require.NotNil(t, descr)

	err := descr.ParseFields()
	assert.NoError(t, err)
}

func TestModelDateMonth(t *testing.T) {

	cache := &sync.Map{}
	namer := &schema.NamingStrategy{}

	descr := db_gorm.NewModelDescriptor(&TestStruct{}, cache, namer)
	require.NotNil(t, descr)

	err := descr.ParseFields()
	require.NoError(t, err)

	for _, field := range descr.Schema.Fields {
		t.Logf("field %s type %s, money=%v", field.Name, field.FieldType, field.Tag.Get("money") != "")
	}
}
