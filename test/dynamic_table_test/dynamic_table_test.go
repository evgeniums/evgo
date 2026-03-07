package dynamic_table_test

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/api/api_server/dynamic_table_gorm"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	DateField  utils.Date
	MonthField utils.Month
	Amount     int `money:"true" json:"amount" gorm:"index"`
}

func TestDynamicTable(t *testing.T) {

	dt := dynamic_table_gorm.New()

	resource := api.NewResource("test")
	op := api.Get("test_op")
	resource.AddOperation(op)

	cfg := &api_server.DynamicTableConfig{Model: &TestStruct{}, Operation: op}
	err := dt.AddTable(cfg)
	require.NoError(t, err)

	dtRes, ok := dt.FindTable(resource.ServicePathPrototype())
	require.True(t, ok)
	require.NotNil(t, dtRes)
	test_utils.DumpObject(t, dtRes.DynamicTable, "Table config")
}
