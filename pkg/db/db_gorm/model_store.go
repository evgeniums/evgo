package db_gorm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/evgeniums/evgo/pkg/db"
	"gorm.io/gorm/schema"
)

type FieldDescriptor struct {
	Json       string
	DbTable    string
	DbField    string
	FullDbName string
	Index      bool
	Schema     *schema.Field
}

type ModelDescriptor struct {
	Sample     interface{}
	Schema     *schema.Schema
	FieldsJson map[string]*FieldDescriptor
}

func (d *ModelDescriptor) FindJsonField(json string) (*FieldDescriptor, error) {
	f, ok := d.FieldsJson[json]
	if !ok {
		return nil, errors.New("field not found")
	}
	return f, nil
}

func (d *ModelDescriptor) FieldsReady() bool {
	return d.FieldsJson != nil
}

func IsIndexField(field *schema.Field) bool {
	// NOTE If field is additional part of composite index only then searching with that index can be slow
	return field.PrimaryKey || field.TagSettings["INDEX"] != "" || field.TagSettings["UNIQUEINDEX"] != ""
}

func (d *ModelDescriptor) ParseFields() error {
	d.FieldsJson = make(map[string]*FieldDescriptor)

	// first run with plain list
	for _, field := range d.Schema.Fields {

		if field.IgnoreMigration {
			continue
		}

		fd := &FieldDescriptor{Schema: field}
		fd.Json = field.Tag.Get("json")
		if fd.Json == "-" {
			continue
		}
		if fd.Json == "" {
			fd.Json = field.DBName
		} else {
			parts := strings.Split(fd.Json, ",")
			if len(parts) == 2 {
				// remove "omitempty" property
				fd.Json = parts[0]
			}
		}
		fd.FullDbName = field.Tag.Get("source")
		if fd.FullDbName == "" {
			fd.FullDbName = fmt.Sprintf("%s.%s", d.Schema.Table, field.DBName)
		}
		parts := strings.Split(fd.FullDbName, ".")
		if len(parts) == 2 {
			fd.DbTable = parts[0]
			fd.DbField = parts[1]
		} else {
			fd.DbTable = fd.FullDbName
			fd.DbField = field.DBName
			fd.FullDbName = fmt.Sprintf("%s.%s", fd.DbTable, fd.DbField)
		}
		fd.Index = IsIndexField(field)

		d.FieldsJson[fd.Json] = fd
		d.FieldsJson[fd.FullDbName] = fd
	}

	// second run, find sources of embedded structs
	embeddedSources := make(map[string]string)
	for i := 0; i < d.Schema.ModelType.NumField(); i++ {
		field := d.Schema.ModelType.Field(i)
		if field.Type.Kind() == reflect.Struct && field.Anonymous {
			name := field.Type.Name()
			source := field.Tag.Get("source")
			if source != "" {
				embeddedSources[name] = source
			}
		}
	}

	// third run - replace field sources
	for _, fd := range d.FieldsJson {
		if fd.Schema.OwnerSchema != nil {
			sourceTable, ok := embeddedSources[fd.Schema.OwnerSchema.Name]
			if ok {
				fd.DbTable = sourceTable
				fd.FullDbName = fmt.Sprintf("%s.%s", fd.DbTable, fd.DbField)
			}
		}
	}

	// done
	return nil
}

type ModelStore struct {
	mutex       sync.Mutex
	descriptors map[string]*ModelDescriptor
	schemaCache *sync.Map
	schemaNamer schema.Namer
}

var GlobalModelStore *ModelStore

func NewModelStore(global bool) *ModelStore {

	m := &ModelStore{}
	m.descriptors = make(map[string]*ModelDescriptor)
	m.schemaCache = &sync.Map{}
	m.schemaNamer = &schema.NamingStrategy{}
	if global && GlobalModelStore == nil {
		GlobalModelStore = m
		db.SetGlobalModelStore(m)
	}
	return m
}

func NewModelDescriptor(model interface{}, cacheStore *sync.Map, namer schema.Namer) *ModelDescriptor {
	d := &ModelDescriptor{Sample: model}
	var err error
	d.Schema, err = schema.Parse(model, cacheStore, namer)
	if err != nil {
		panic(fmt.Sprintf("invalid model: %s", err))
	}
	return d
}

func (m *ModelStore) RegisterModel(model interface{}) {

	d := NewModelDescriptor(model, m.schemaCache, m.schemaNamer)

	m.mutex.Lock()
	m.descriptors[d.Schema.Table] = d
	m.mutex.Unlock()
}

func (m *ModelStore) RegisterModels(models []interface{}) {
	for _, model := range models {
		m.RegisterModel(model)
	}
}

func (m *ModelStore) FindModel(name string) interface{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	d, ok := m.descriptors[name]
	if !ok {
		return nil
	}
	return d.Sample
}

func (m *ModelStore) AllModels() []interface{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	models := make([]interface{}, len(m.descriptors))
	i := 0
	for _, descriptor := range m.descriptors {
		models[i] = descriptor.Sample
		i++
	}
	return models
}

func (m *ModelStore) FindDescriptor(name string) *ModelDescriptor {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	d, ok := m.descriptors[name]
	if !ok {
		return nil
	}
	return d
}

func (m *ModelStore) ParseModelFields(descriptor *ModelDescriptor) error {

	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := descriptor.ParseFields()
	if err != nil {
		return err
	}

	return nil
}
