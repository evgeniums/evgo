package dynamic_table_gorm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/db/db_gorm"
	"github.com/evgeniums/evgo/pkg/logger"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm/schema"
)

type Table struct {
	*api_server.DynamicTable
	Schema *schema.Schema
}

type DynamicTablesGorm struct {
	mutex      sync.RWMutex
	tables     map[string]*Table
	translator api_server.DynamicFieldTranslator

	schemaCache *sync.Map
	schemaNamer schema.Namer
}

func New(translator ...api_server.DynamicFieldTranslator) *DynamicTablesGorm {

	d := &DynamicTablesGorm{}
	d.tables = make(map[string]*Table)
	d.schemaCache = &sync.Map{}
	d.schemaNamer = &schema.NamingStrategy{}

	if len(translator) != 0 {
		d.translator = translator[0]
	}

	return d
}

func (d *DynamicTablesGorm) SetTranslator(translator api_server.DynamicFieldTranslator) {
	d.translator = translator
}

func (d *DynamicTablesGorm) FindTable(path string) (*Table, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	t, ok := d.tables[path]
	return t, ok
}

func (d *DynamicTablesGorm) Table(request api_server.Request, path string) (*api_server.DynamicTable, error) {

	// setup
	c := request.TraceInMethod("DynamicTable.Table", logger.Fields{"path": path})
	request.TraceOutMethod()

	// find table
	t, ok := d.FindTable(path)
	if !ok {
		return nil, c.SetErrorStr("unknown table")
	}
	c.SetLoggerField("table", t.Schema.Table)

	// clone result from stored table
	result := &api_server.DynamicTable{}
	*result = *t.DynamicTable

	// process fields
	for _, field := range result.Columns {

		// translate field's display
		if d.translator != nil {
			d.translator.Tr(field, t.Schema.Table)
		}

		// fill field's enum list
		if len(field.Enum) == 0 && field.EnumGetter != nil {
			enum, err := field.EnumGetter(request)
			if err != nil {
				c.Logger().Warn("failed to translate field", db.Fields{"field": field.Field})
				continue
			}
			field.Enum = enum
		}
	}

	// done
	return result, nil
}

func FieldDisplay(field *schema.Field, name string, explicits map[string]string) string {

	// check if there is a tag for display
	display := field.Tag.Get("display")

	// check if display is explicitly set for field name
	if display == "" {
		var ok bool
		display, ok = explicits[name]
		if ok {
			return display
		}
	}

	// construct display from field name by replacing _ with spaces and transforming first letter to upper case
	if display == "" {
		display = cases.Title(language.Und, cases.NoLower).String(name)
		display = strings.Replace(display, "_", " ", -1)
	}

	// check if there is override in map of explicits
	override, ok := explicits[display]
	if ok {
		return override
	}

	// done
	return display
}

func (d *DynamicTablesGorm) AddTable(table *api_server.DynamicTableConfig) error {

	// create table and set default fields
	t := &Table{}
	t.DynamicTable = &api_server.DynamicTable{}
	t.Columns = make([]*api_server.DynamicTableField, 0)
	t.DefaultSortDirection = table.DefaultSortDirection
	if t.DefaultSortDirection == "" {
		t.DefaultSortDirection = db.SORT_DESC
	}
	t.DefaultSortColumn = table.DefaultSortField
	if t.DefaultSortColumn == "" {
		t.DefaultSortColumn = "created_at"
	}

	// parse model's schema
	s, err := schema.Parse(table.Model, d.schemaCache, d.schemaNamer)
	if err != nil {
		return fmt.Errorf("invalid model: %s", err)
	}

	visibleFields := make(map[string]bool)
	for _, visibleField := range table.VisibleColumns {
		visibleFields[visibleField] = true
	}

	// process fields
	defaultOrder := make([]string, 0, len(s.Fields))
	fields := make(map[string]*api_server.DynamicTableField)
	for _, field := range s.Fields {
		tableField := &api_server.DynamicTableField{}

		// set field name
		tableField.Field = field.Tag.Get("json")
		if tableField.Field == "" {
			tableField.Field = field.DBName
		}
		tableField.Field = strings.Replace(tableField.Field, ",omitempty", "", -1)

		// set field visible
		tableField.Visible = len(visibleFields) == 0 || visibleFields[tableField.Field]

		// set field display
		tableField.Display = FieldDisplay(field, tableField.Field, table.Displays)

		// set index flag
		tableField.Index = db_gorm.IsIndexField(field)

		// set type
		reflectType := field.FieldType.String()
		if reflectType == "utils.Date" {
			tableField.Type = "date"
		} else if reflectType == "utils.Month" {
			tableField.Type = "month"
		} else {
			tableField.Type = string(field.DataType)
		}

		// set money type
		tableField.Money = field.Tag.Get("money") != ""

		// set enum
		tableField.Enum = table.Enums[tableField.Field]

		// set enum getter
		tableField.EnumGetter = table.EnumGetters[tableField.Field]

		// add to map
		fields[tableField.Field] = tableField
		defaultOrder = append(defaultOrder, tableField.Field)
	}

	// try to order fields using column order argument
	for i := 0; i < len(table.ColumnsOrder); i++ {
		fieldName := table.ColumnsOrder[i]
		field, ok := fields[fieldName]
		if !ok {
			return fmt.Errorf("unknown field %s in column order", fieldName)
		}
		t.Columns = append(t.Columns, field)
		delete(fields, fieldName)
	}

	// sort the rest fields by default order
	for i := 0; i < len(defaultOrder); i++ {
		fieldName := defaultOrder[i]
		field, ok := fields[fieldName]
		if !ok {
			continue
		}
		t.Columns = append(t.Columns, field)
		delete(fields, fieldName)
	}

	// add table to store
	t.Schema = s
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.tables[table.Operation.Resource().ServicePathPrototype()] = t

	// done
	return nil
}
