package db

import (
	"context"
	"sync"

	"github.com/evgeniums/evgo/pkg/utils"
)

type JoinQuery interface {
	Join(sctx context.Context, filter *Filter, dest interface{}) (int64, error)
}

type Joiner interface {
	Join(model interface{}, field string) JoinBegin
}

type JoinBegin interface {
	On(model interface{}, field string) JoinEnd
}

type JoinEnd interface {
	Joiner
	Sum(groupFields []string, sumFields []string) JoinEnd
	Destination(dst interface{}) (JoinQuery, error)
}

type JoinTableData struct {
	Model interface{}
}

type JoinTableBase struct {
	JoinTableData
}

func (j *JoinTableBase) Model() interface{} {
	return j.JoinTableData.Model
}

type JoinPairData struct {
	LeftField  string
	RightField string
}

type JoinPairBase struct {
	JoinPairData
}

func (j *JoinPairBase) LeftField() string {
	return j.JoinPairData.LeftField
}

func (j *JoinPairBase) RightField() string {
	return j.JoinPairData.RightField
}

type JoinQueryData struct {
	destination interface{}
}

type JoinQueryBase struct {
	JoinQueryData
}

func (j *JoinQueryBase) Destination() interface{} {
	return j.JoinQueryData.destination
}

func (j *JoinQueryBase) SetDestination(destination interface{}) {
	j.JoinQueryData.destination = destination
}

type JoinQueryBuilder = func() (JoinQuery, error)

type JoinQueryConfig struct {
	Builder JoinQueryBuilder
	Name    string
	Nocache bool
}

func NewJoin(builder JoinQueryBuilder, name string, nocache ...bool) *JoinQueryConfig {
	return &JoinQueryConfig{Builder: builder, Name: name, Nocache: utils.OptionalArg(false, nocache...)}
}

type JoinQueries struct {
	mutex sync.Mutex
	cache map[string]JoinQuery
}

func NewJoinQueries() *JoinQueries {
	j := &JoinQueries{}
	j.cache = make(map[string]JoinQuery)
	return j
}

func (j *JoinQueries) FindOrCreate(config *JoinQueryConfig) (JoinQuery, error) {
	var err error
	j.mutex.Lock()
	q, ok := j.cache[config.Name]
	j.mutex.Unlock()
	if !ok || config.Nocache {
		q, err = config.Builder()
		if err != nil {
			return nil, err
		}
	}
	j.mutex.Lock()
	if !config.Nocache {
		j.cache[config.Name] = q
	}
	j.mutex.Unlock()
	return q, nil
}
