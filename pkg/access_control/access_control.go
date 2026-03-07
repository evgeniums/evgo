package access_control

import (
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

// Interface for access controllers.
type AccessControl interface {
	CheckAccess(ctx op_context.Context, resource Resource, subject Subject, accessType AccessType) bool
	DefaultAccess() Access
	SetDefaultAccess(Access)
}

// Base implementation of access controller.
type AccessControlBase struct {
	acl             Acl
	resourceManager ResourceManager
	defaultAccess   Access
}

func NewAccessControl(acl Acl, resourceManager ResourceManager, defaultAccess ...Access) *AccessControlBase {
	a := &AccessControlBase{}
	a.Init(acl, resourceManager, defaultAccess...)
	return a
}

func (a *AccessControlBase) Init(acl Acl, resourceManager ResourceManager, defaultAccess ...Access) {
	a.acl = acl
	a.resourceManager = resourceManager
	a.defaultAccess = utils.OptionalArg[Access](&AccessBase{}, defaultAccess...)
}

func (a *AccessControlBase) CheckAccess(ctx op_context.Context, resource Resource, subject Subject, accessType AccessType) (bool, error) {

	ctx.TraceInMethod("AccessControl.CheckAccess")
	defer ctx.TraceOutMethod()

	// check owner access
	if resource.IsOwner(subject) {
		// if owner access is allowed then finish checking, otherwise go further and use ACL rules to check access
		if resource.OwnerAccess().Check(accessType) {
			// TODO log used rule
			return true, nil
		}
	}

	// init with default access
	allow := a.defaultAccess.Check(accessType)
	ruleFound := false

	// collect tags of the resource using tags of all resources in the path
	tags := make(map[int][]string, 0)
	paths := resource.Paths()
	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		resourceTags, err := a.resourceManager.ResourceTags(ctx, path)
		if err != nil {
			return false, err
		}
		tags[i] = resourceTags
	}

	// look for rules for all resource paths starting from the most specific path
	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		// look for rule per each tag starting from the tags for the most specific path
		for j := i; j >= 0; j-- {
			resourceTags := tags[j]
			// if rule for tag is not found then look for the resource without tags
			resourceTags = append(resourceTags, "")
			for _, tag := range resourceTags {
				// look for a rule for any subject's role
				for _, role := range subject.Roles() {
					rule, err := a.acl.FindRule(ctx, path, tag, role)
					if err != nil {
						return false, err
					}
					if rule != nil {
						ruleFound = true
						allow = rule.Access().Check(accessType)
						// TODO log used rule
						break
					}
				}
				if ruleFound {
					break
				}
			}
			if ruleFound {
				break
			}
		}
		if ruleFound {
			break
		}
	}

	// TODO log detected access
	return allow, nil
}

func (a *AccessControlBase) DefaultAccess() Access {
	return a.defaultAccess
}

func (a *AccessControlBase) SetDefaultAccess(access Access) {
	a.defaultAccess = access
}
