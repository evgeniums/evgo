package api

import "github.com/evgeniums/evgo/pkg/access_control"

const PackageName = "evgo"

func OperationAsResource(sampleResource Resource, resourceName string, resourceId string, op Operation) Operation {
	opResource := NewResource(resourceName)
	opResource.AddOperation(op)
	namedResource := sampleResource.CloneChain(false)
	namedResource.SetId(resourceId)
	namedResource.AddChild(opResource)
	return op
}

func NamedResourceOperation(sampleResource Resource, resourceId string, op Operation) Operation {
	namedResource := sampleResource.CloneChain(false)
	namedResource.SetId(resourceId)
	namedResource.AddOperation(op)
	return op
}

func NamedSubresource(sampleResource Resource, resourceId string, subResourceName string, subResourceId string) Resource {
	subResource := NewResource(subResourceName)
	subResource.SetId(subResourceId)
	namedResource := sampleResource.CloneChain(false)
	namedResource.SetId(resourceId)
	namedResource.AddChild(subResource)
	return subResource
}

func NamedSubresourceOperation(sampleResource Resource, resourceId string, subResourceName string, subResourceId string, op Operation) Operation {
	r := NamedSubresource(sampleResource, resourceId, subResourceName, subResourceId)
	r.AddOperation(op)
	return op
}

func Subresource(sampleResource Resource, resourceId string, subResourceName string) Resource {
	subResource := NewResource(subResourceName)
	namedResource := sampleResource.CloneChain(false)
	namedResource.SetId(resourceId)
	namedResource.AddChild(subResource)
	return subResource
}

func SubresourceOperation(sampleResource Resource, resourceId string, subResourceName string, op Operation) Operation {
	subResource := Subresource(sampleResource, resourceId, subResourceName)
	subResource.AddOperation(op)
	return op
}

func NewResourceWithNamedSubresource(resourceName string, subResourceName string, op ...Operation) Resource {
	subResource := NamedResource(subResourceName)
	resource := NewResource(resourceName)
	resource.AddChild(subResource.Parent())
	if len(op) != 0 {
		subResource.AddOperation(op[0])
	}
	return resource
}

func NewResourceWithOp(resourceName string, op Operation) Resource {
	resource := NewResource(resourceName)
	resource.AddOperation(op)
	return resource
}

func Post(name string) Operation {
	return NewOperation(name, access_control.Post)
}

func Create(name string) Operation {
	return NewOperation(name, access_control.Create)
}

func Add(name string) Operation {
	return NewOperation(name, access_control.Create)
}

func Find(name string) Operation {
	return NewOperation(name, access_control.Get)
}

func Get(name string) Operation {
	return NewOperation(name, access_control.Get)
}

func Read(name string) Operation {
	return NewOperation(name, access_control.Get)
}

func Exists(name string) Operation {
	return NewOperation(name, access_control.Get)
}

func Check(name string) Operation {
	return NewOperation(name, access_control.Get)
}

func List(name string) Operation {
	return NewOperation(name, access_control.Read)
}

func Delete(name string) Operation {
	return NewOperation(name, access_control.Delete)
}

func Update(name string) Operation {
	return NewOperation(name, access_control.Put)
}

func UpdatePartial(name string) Operation {
	return NewOperation(name, access_control.Patch)
}

type UpdateCmd struct {
	Fields map[string]interface{} `json:"field"`
}

func Bind(name string) Operation {
	return NewOperation(name, access_control.Create)
}

func Unbind(name string) Operation {
	return NewOperation(name, access_control.Delete)
}
