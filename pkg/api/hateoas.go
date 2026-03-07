package api

import (
	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/utils"
)

type HateoasLink struct {
	Target     string `json:"target,omitempty"`
	Operation  string `json:"operation,omitempty"`
	HttpMethod string `json:"http_method,omitempty"`
	Host       string `json:"host,omitempty"`
	Path       string `json:"path,omitempty"`
}

type WithHateoasLinks interface {
	SetHateoasLinks([]*HateoasLink)
}

type ObjectWithHateoasLinks interface {
	common.WithID
	WithHateoasLinks
}

type HateoasLinksStub struct {
}

func (h *HateoasLinksStub) SetHateoasLinks(links []*HateoasLink) {
}

type HateoasLinksContainer struct {
	HateoasLinks []*HateoasLink `json:"_links,omitempty" gorm:"-:all"`
}

func (h *HateoasLinksContainer) SetHateoasLinks(links []*HateoasLink) {
	h.HateoasLinks = links
}

type HateoasLinksItem struct {
	common.IDBase
	HateoasLinksContainer
}

func MakeHateoasLinks(resource Resource, withTestOps ...bool) []*HateoasLink {

	withTest := utils.OptionalArg(false, withTestOps...)
	links := make([]*HateoasLink, 0)

	addLink := func(host string, path string, target string, operation Operation) {

		if operation == nil {
			return
		}
		if operation.TestOnly() && !withTest {
			return
		}

		link := &HateoasLink{}
		link.Host = host
		link.HttpMethod = access_control.Access2HttpMethod(operation.AccessType())
		link.Path = path
		link.Operation = operation.Name()
		link.Target = target

		links = append(links, link)
	}

	// add self operations
	path := resource.FullActualPath()
	selfHost := resource.Host()
	resource.EachOperation(func(operation Operation) error { addLink(selfHost, path, TargetSelf, operation); return nil }, false)

	addGetter := func(target string, r Resource) {
		addLink(r.Host(), r.ActualPath(), target, r.Getter())
	}

	// add children getters
	for _, child := range resource.Children() {
		addGetter(TargetChild, child)
	}

	// add parent getter
	parent := resource.Parent()
	if parent != nil {
		addGetter(TargetParent, parent)
	}

	// done
	return links
}

func InjectHateoasLinksToObject(resource Resource, withLinks WithHateoasLinks) {
	withLinks.SetHateoasLinks(MakeHateoasLinks(resource))
}

func HateoasList(response ResponseListI, parentResource Resource, resourceType string) {

	count := response.ItemCount()
	if count == 0 {
		return
	}

	sampleResource := NamedResource(resourceType)
	epResource := parentResource.CloneChain(true)
	epResource.AddChild(sampleResource)

	response.MakeItemLinks()

	for i := 0; i < count; i++ {

		obj := &HateoasLinksItem{}
		obj.SetID(response.ItemId(i))

		sampleResource.SetId(obj.GetID())
		InjectHateoasLinksToObject(sampleResource, obj)

		response.SetItemLink(i, obj)
	}
}
