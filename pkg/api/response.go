package api

import (
	"github.com/evgeniums/evgo/pkg/common"
)

const (
	TargetSelf   = "self"
	TargetParent = "parent"
	TargetChild  = "parent"
)

type Response = interface{}

type ResponseStub struct {
}

type ResponseBase struct {
	HateoasLinksContainer
}

type ResponseStatus struct {
	ResponseStub
	Status string `json:"status"`
}

type ResponseCount struct {
	Count int64 `json:"count,omitempty"`
}

type ResponseExists struct {
	ResponseStub
	Exists bool `json:"exists"`
}

type ResponseListI interface {
	ItemCount() int
	ItemId(index int) string
	MakeItemLinks()
	SetItemLink(index int, link *HateoasLinksItem)
}

type ResponseList[T common.WithID] struct {
	ResponseCount
	Items []T `json:"items"`

	ResponseBase
	ItemLinks []*HateoasLinksItem `json:"_item_links,omitempty"`
}

func (r *ResponseList[T]) ItemCount() int {
	return len(r.Items)
}

func (r *ResponseList[T]) ItemId(index int) string {
	return r.Items[index].GetID()
}

func (r *ResponseList[T]) MakeItemLinks() {
	r.ItemLinks = make([]*HateoasLinksItem, r.ItemCount())
}

func (r *ResponseList[T]) SetItemLink(index int, link *HateoasLinksItem) {
	r.ItemLinks[index] = link
}
