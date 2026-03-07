package api_server

import "github.com/evgeniums/evgo/pkg/api"

type EnumEntry struct {
	Value   string `json:"value"`
	Display string `json:"display"`
}

type FieldEnums = map[string][]*EnumEntry

func EnumMap(enumMap map[string]string) []*EnumEntry {
	enums := make([]*EnumEntry, 0, len(enumMap))
	for value, display := range enumMap {
		enums = append(enums, &EnumEntry{value, display})
	}
	return enums
}

func EnumList(enumList []string) []*EnumEntry {
	enums := make([]*EnumEntry, 0, len(enumList))
	for _, value := range enumList {
		enums = append(enums, &EnumEntry{value, value})
	}
	return enums
}

type EnumGetter func(request Request) ([]*EnumEntry, error)

type DynamicTableField struct {
	Field      string       `json:"field"`
	Type       string       `json:"type"`
	Index      bool         `json:"index"`
	Display    string       `json:"display"`
	Enum       []*EnumEntry `json:"enum,omitempty"`
	Visible    bool         `json:"visible"`
	Money      bool         `json:"money"`
	EnumGetter EnumGetter   `json:"-"`
}

type DynamicTable struct {
	api.ResponseStub
	Columns              []*DynamicTableField `json:"columns"`
	DefaultSortColumn    string               `json:"default_sort_column"`
	DefaultSortDirection string               `json:"default_sort_direction"`
}

type DynamicTableQuery struct {
	Path string `json:"path" validate:"required" vmessage:"Invalid table path."`
}

type DynamicTableConfig struct {
	Operation            api.Operation
	Model                interface{}
	Displays             map[string]string
	ColumnsOrder         []string
	DefaultSortField     string
	DefaultSortDirection string
	EnumGetters          map[string]EnumGetter
	Enums                FieldEnums
	VisibleColumns       []string
}

type DynamicFieldTranslator interface {
	Tr(field *DynamicTableField, tableName ...string) (string, bool)
}

type DynamicTables interface {
	AddTable(table *DynamicTableConfig) error
	Table(request Request, path string) (*DynamicTable, error)
	SetTranslator(translator DynamicFieldTranslator)
}
