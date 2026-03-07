package api

import "github.com/evgeniums/evgo/pkg/db"

type Query interface {
	Query() string
	SetQuery(q string)
}

type WithDbQuery struct {
	Query string `json:"query"`
}

type DbQuery struct {
	WithDbQuery
}

func (d *DbQuery) Query() string {
	return d.WithDbQuery.Query
}

func (d *DbQuery) SetQuery(q string) {
	d.WithDbQuery.Query = q
}

func NewDbQuery(filter *db.Filter) *DbQuery {
	cmd := &DbQuery{}
	if filter != nil {
		cmd.SetQuery(filter.ToQueryString())
	}
	return cmd
}

type WithGroupBy struct {
	GroupBy []string `json:"group_by"`
}

func (w *WithGroupBy) Groups() []string {
	return w.GroupBy
}

type QueryWithGroupBy struct {
	*DbQuery
	WithGroupBy
}
