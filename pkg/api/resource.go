package api

import (
	"strings"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/utils"
)

type ResourceId interface {
	Type() string
	Value() string
	SetType(val string)
	SetValue(val string)
}

type ResourceIds interface {
	HasId(name string) bool
	GetId(name string) ResourceId
	SetId(id ResourceId)
	UnsetId(name string)
	Clear()
}

type Resource interface {
	Host() string
	SetHost(val string)

	Type() string
	Id() string
	SetId(val string, rebuild ...bool)

	ResourceId() ResourceId
	SetResourceId(val ResourceId, rebuild ...bool)

	HasId() bool
	SetHasId(val bool)

	IsService() bool
	SetSetvice(val bool)
	IsServicePart() bool
	ServiceResource() Resource

	IsTenancy() bool
	IsInTenancy() bool
	TenancyResourceInPath() Resource

	SetParent(parent Resource, rebuild ...bool)
	Parent() Resource
	AddChild(resource Resource, rebuild ...bool)
	AddChildren(resources ...Resource)
	Children() []Resource

	AddOperation(operation Operation, getter ...bool)
	AddOperations(operations ...Operation)
	Operations() []Operation
	Getter() Operation
	EachOperation(handler func(operation Operation) error, recursive ...bool) error
	RemoveOperations(accessType access_control.AccessType)
	RemoveOperation(name string)
	ReplaceOperation(op Operation)

	PathPrototype() string
	ParameterSubstitution() string
	ActualPath() string
	FullPathPrototype() string
	FullActualPath() string
	FullActualTenancyPath(tenancyId string) string
	ServicePathPrototype() string
	ServiceActualPath() string
	BuildActualPath(actualResourceIds ResourceIds, service ...bool) string
	FillActualPaths(actualResourceIds ResourceIds)
	ResetIds()

	RebuildPaths()

	Chain() []Resource
	ChainResourceId(resourceType string) string
	ChainHasResourceType(resourceType string) bool

	Clone(withOperations bool) Resource
	CloneChain(withOperations bool) Resource
}

type ResourceIdBase struct {
	typ   string
	value string
}

func NewResourceIdBase(typ string, value ...string) ResourceId {
	r := &ResourceIdBase{typ: typ}
	if len(value) > 0 {
		r.value = value[0]
	}
	return r
}

func (r *ResourceIdBase) Type() string {
	return r.typ
}

func (r *ResourceIdBase) Value() string {
	return r.value
}

func (r *ResourceIdBase) SetType(typ string) {
	r.typ = typ
}

func (r *ResourceIdBase) SetValue(val string) {
	r.value = val
}

type ResourceConfig struct {
	Host    string
	HasId   bool
	Service bool
	// Id        string
	Tenancy   bool
	IdWrapper ResourceId
}

type ResourceBase struct {
	ResourceConfig
	pathPrototype        string
	actualPath           string
	fullPathPrototype    string
	fullActualPath       string
	servicePathPrototype string
	serviceActualPath    string
	parent               Resource
	children             []Resource
	operations           []Operation
	getter               Operation

	parameterSubstitution string

	inTenancy       bool
	tenancyResource Resource
}

func NewResource(resourceType string, config ...ResourceConfig) *ResourceBase {
	r := &ResourceBase{}
	r.Init(resourceType, config...)
	return r
}

func NewTenancyResource(tenancyResourceName ...string) *ResourceBase {

	resourceType := utils.OptionalArg("tenancy", tenancyResourceName...)

	parent := NewResource(resourceType)
	r := NewResource(resourceType, ResourceConfig{HasId: true, Tenancy: true})
	parent.AddChild(r)
	return r
}

func (r *ResourceBase) Init(resourceType string, config ...ResourceConfig) {
	r.children = make([]Resource, 0)
	r.operations = make([]Operation, 0)
	r.ResourceConfig = utils.OptionalArg(ResourceConfig{}, config...)
	r.inTenancy = r.ResourceConfig.Tenancy
	if r.ResourceConfig.Tenancy {
		r.tenancyResource = r
	}
	r.parameterSubstitution = utils.ConcatStrings(":", resourceType)
	if r.ResourceConfig.IdWrapper == nil {
		r.ResourceConfig.IdWrapper = &ResourceIdBase{typ: resourceType}
	} else {
		r.ResourceConfig.IdWrapper.SetType(resourceType)
	}

	r.RebuildPaths()
}

func (r *ResourceBase) RebuildPaths() {

	if r.HasId() {
		r.pathPrototype = utils.ConcatStrings("/:", r.Type())
		if r.Tenancy && r.Id() == "" {
			r.actualPath = r.pathPrototype
		} else {
			r.actualPath = utils.ConcatStrings("/", r.Id())
		}
	} else {
		r.pathPrototype = utils.ConcatStrings("/", r.Type())
		r.actualPath = r.pathPrototype
	}

	r.fullPathPrototype = r.pathPrototype
	r.fullActualPath = r.actualPath

	if r.IsService() {
		r.servicePathPrototype = r.pathPrototype
		r.serviceActualPath = r.actualPath
	}

	r.inTenancy = r.isInTenancy()

	parent := r.Parent()
	if parent != nil {
		r.fullPathPrototype = utils.ConcatStrings(parent.FullPathPrototype(), r.pathPrototype)
		r.fullActualPath = utils.ConcatStrings(parent.FullActualPath(), r.actualPath)
		if r.IsServicePart() && !r.IsService() {
			r.servicePathPrototype = utils.ConcatStrings(parent.ServicePathPrototype(), r.pathPrototype)
			r.serviceActualPath = utils.ConcatStrings(parent.ServiceActualPath(), r.actualPath)
		}
		if r.inTenancy && !r.IsTenancy() {
			r.tenancyResource = parent.TenancyResourceInPath()
		}
	}

	for _, child := range r.children {
		child.RebuildPaths()
	}
}

func (r *ResourceBase) TenancyResourceInPath() Resource {
	return r.tenancyResource
}

func (r *ResourceBase) ParameterSubstitution() string {
	return r.parameterSubstitution
}

func (r *ResourceBase) PathPrototype() string {
	return r.pathPrototype
}

func (r *ResourceBase) ActualPath() string {
	return r.actualPath
}

func (r *ResourceBase) FullPathPrototype() string {
	return r.fullPathPrototype
}

func (r *ResourceBase) FullActualPath() string {
	return r.fullActualPath
}

func (r *ResourceBase) FullActualTenancyPath(tenancyPath string) string {
	if !r.IsInTenancy() {
		return r.FullActualPath()
	}

	if r.TenancyResourceInPath() == nil {
		panic("Tenancy resource must not be nil")
	}

	actualPath := r.fullActualPath
	tenancyParameter := r.TenancyResourceInPath().ParameterSubstitution()
	path := strings.ReplaceAll(actualPath, tenancyParameter, tenancyPath)
	return path
}

func (r *ResourceBase) ServicePathPrototype() string {
	return r.servicePathPrototype
}

func (r *ResourceBase) Host() string {
	if r.ResourceConfig.Host == "" && r.Parent() != nil {
		return r.Parent().Host()
	}
	return r.ResourceConfig.Host
}

func (r *ResourceBase) SetHost(val string) {
	r.ResourceConfig.Host = val
}

func (r *ResourceBase) ServiceActualPath() string {
	return r.serviceActualPath
}

func (r *ResourceBase) Type() string {
	return r.IdWrapper.Type()
}

func (r *ResourceBase) Id() string {
	return r.IdWrapper.Value()
}

func (r *ResourceBase) updateId(rebuild ...bool) {
	if r.IdWrapper.Value() != "" {
		r.SetHasId(true)
	}
	if utils.OptionalArg(true, rebuild...) {
		r.RebuildPaths()
	}
}

func (r *ResourceBase) SetId(val string, rebuild ...bool) {
	r.IdWrapper.SetValue(val)
	r.updateId(rebuild...)
}

func (r *ResourceBase) SetResourceId(val ResourceId, rebuild ...bool) {
	r.IdWrapper = val
	r.updateId(rebuild...)
}

func (r *ResourceBase) ResourceId() ResourceId {
	return r.IdWrapper
}

func (r *ResourceBase) HasId() bool {
	return r.ResourceConfig.HasId
}

func (r *ResourceBase) SetHasId(val bool) {
	r.ResourceConfig.HasId = val
	r.RebuildPaths()
}

func (r *ResourceBase) IsService() bool {
	return r.ResourceConfig.Service
}

func (r *ResourceBase) IsServicePart() bool {
	if r.IsService() {
		return true
	}
	if r.Parent() != nil {
		return r.parent.IsServicePart()
	}
	return false
}

func (r *ResourceBase) IsTenancy() bool {
	return r.ResourceConfig.Tenancy
}

func (r *ResourceBase) isInTenancy() bool {
	if r.IsTenancy() {
		return true
	}
	if r.Parent() != nil {
		return r.parent.IsInTenancy()
	}
	return false
}

func (r *ResourceBase) IsInTenancy() bool {
	return r.inTenancy
}

func (r *ResourceBase) ServiceResource() Resource {
	if r.IsService() {
		return r
	}
	if r.Parent() != nil {
		return r.parent.ServiceResource()
	}
	return nil
}

func (r *ResourceBase) SetSetvice(val bool) {
	r.ResourceConfig.Service = val
	r.RebuildPaths()
}

func (r *ResourceBase) SetParent(parent Resource, rebuild ...bool) {
	r.parent = parent
	if utils.OptionalArg(true, rebuild...) {
		r.RebuildPaths()
	}
}

func (r *ResourceBase) Parent() Resource {
	return r.parent
}

func (r *ResourceBase) AddChild(child Resource, rebuild ...bool) {
	r.children = append(r.children, child)
	child.SetParent(r, rebuild...)
}

func (r *ResourceBase) AddChildren(resources ...Resource) {
	for _, child := range resources {
		r.AddChild(child)
	}
}

func (r *ResourceBase) Children() []Resource {
	return r.children
}

func (r *ResourceBase) RemoveOperations(accessType access_control.AccessType) {

	i := 0
	for _, op := range r.operations {
		if op.AccessType() != accessType {
			r.operations[i] = op
			i++
		}
	}
	for j := i; j < len(r.operations); j++ {
		r.operations[j] = nil
	}
	r.operations = r.operations[:i]
}

func (r *ResourceBase) RemoveOperation(name string) {

	i := 0
	for _, op := range r.operations {
		if op.Name() != name {
			r.operations[i] = op
			i++
		}
	}
	for j := i; j < len(r.operations); j++ {
		r.operations[j] = nil
	}
	r.operations = r.operations[:i]
}

func (r *ResourceBase) ReplaceOperation(op Operation) {
	r.RemoveOperations(op.AccessType())
	r.AddOperation(op)
}

func (r *ResourceBase) AddOperation(operation Operation, getter ...bool) {
	r.operations = append(r.operations, operation)
	if utils.OptionalArg(false, getter...) {
		r.getter = operation
	}
	operation.SetResource(r)
}

func (r *ResourceBase) AddOperations(operations ...Operation) {
	for _, op := range operations {
		r.AddOperation(op)
	}
}

func (r *ResourceBase) Getter() Operation {
	return r.getter
}

func (r *ResourceBase) Operations() []Operation {
	return r.operations
}

func (r *ResourceBase) EachOperation(handler func(operation Operation) error, recursive ...bool) error {

	for _, operation := range r.operations {
		err := handler(operation)
		if err != nil {
			return err
		}
	}

	if utils.OptionalArg(true, recursive...) {
		for _, child := range r.children {
			err := child.EachOperation(handler)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *ResourceBase) Chain() []Resource {
	chain := make([]Resource, 0)

	for parent := r.Parent(); parent != nil; parent = parent.Parent() {
		chain = append([]Resource{parent}, chain...)
	}

	return chain
}

func (r *ResourceBase) ChainResourceId(resourceType string) string {
	if r.Type() == resourceType {
		return r.Id()
	}
	for parent := r.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Type() == resourceType {
			return parent.Id()
		}
	}
	return ""
}

func (r *ResourceBase) BuildActualPath(actualResourceIds ResourceIds, service ...bool) string {

	servicePath := utils.OptionalArg(false, service...)
	if servicePath && !r.IsServicePart() {
		return ""
	}

	path := "/"
	if r.HasId() {
		id := actualResourceIds.GetId(r.Type())
		if id != nil {
			path = utils.ConcatStrings("/", id.Value())
		}
	} else {
		path = utils.ConcatStrings("/", r.Type())
	}

	parent := r.Parent()
	if parent != nil {
		parentPath := parent.BuildActualPath(actualResourceIds, service...)
		if parentPath != "" {
			path = utils.ConcatStrings(parentPath, path)
		}
	}

	return path
}

func (r *ResourceBase) FillActualPaths(actualResourceIds ResourceIds) {

	if r.HasId() {
		selfId := actualResourceIds.GetId(r.Type())
		if selfId != nil {
			r.SetId(selfId.Value(), false)
		}
	}

	parent := r.Parent()
	if parent != nil {
		parent.FillActualPaths(actualResourceIds)
	} else {
		r.RebuildPaths()
	}
}

func (r *ResourceBase) ResetIds() {
	if r.HasId() {
		r.SetId("", false)
	}
	parent := r.Parent()
	if parent != nil {
		parent.ResetIds()
	} else {
		r.RebuildPaths()
	}
}

func (r *ResourceBase) Clone(withOperations bool) Resource {
	cfg := r.ResourceConfig
	cfg.IdWrapper = nil
	resource := NewResource(r.Type(), cfg)
	if r.HasId() {
		resource.SetId(r.Id())
	}
	if withOperations {
		r.EachOperation(func(operation Operation) error {
			op := NewOperation(operation.Name(), operation.AccessType(), operation.TestOnly())
			resource.AddOperation(op)
			return nil
		})
	}
	return resource
}

func (r *ResourceBase) CloneChain(withOperations bool) Resource {
	resource := NewResource(r.Type(), r.ResourceConfig)
	if r.HasId() {
		resource.SetId(r.Id())
	}

	var topResource Resource
	topResource = resource
	for parent := r.Parent(); parent != nil; parent = parent.Parent() {
		parentClone := parent.Clone(withOperations)
		parentClone.AddChild(topResource, false)
		topResource = parentClone
	}
	topResource.RebuildPaths()

	return resource
}

func (r *ResourceBase) ChainHasResourceType(resourceType string) bool {
	if r.Type() == resourceType {
		return true
	}
	if r.parent != nil {
		return r.parent.ChainHasResourceType(resourceType)
	}
	return false
}

func GroupResource(resourceType string) Resource {
	r := NewResource(resourceType)
	return r
}

func NamedResource(resourceType string) Resource {
	cfg := ResourceConfig{
		HasId: true,
	}
	r := NewResource(resourceType, cfg)

	group := GroupResource(resourceType)
	group.AddChild(r)

	return r
}

func PrepareCollectionAndNameResource(typeName string) (serviceName string, collectionResource Resource, objectResource Resource) {

	serviceName = utils.ConcatStrings(typeName, "s")
	objectResource = NamedResource(typeName)
	collectionResource = objectResource.Parent()

	return
}

func AddChildResource(resource Resource, childName string) Resource {
	child := NewResource(childName)
	resource.AddChild(child)
	return child
}

type ResourceIdsBase struct {
	ids map[string]ResourceId
}

func NewResourceIdsBase() *ResourceIdsBase {
	return &ResourceIdsBase{ids: make(map[string]ResourceId)}
}

func (r *ResourceIdsBase) HasId(name string) bool {
	_, ok := r.ids[name]
	return ok
}

func (r *ResourceIdsBase) GetId(name string) ResourceId {
	val := r.ids[name]
	return val
}

func (r *ResourceIdsBase) SetId(val ResourceId) {
	r.ids[val.Type()] = val
}

func (r *ResourceIdsBase) UnsetId(name string) {
	delete(r.ids, name)
}

func (r *ResourceIdsBase) Clear() {
	clear(r.ids)
}
