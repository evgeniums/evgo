package console_tool

import (
	"errors"
	"fmt"
	"os"

	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/jessevdk/go-flags"
)

type CommandGroup interface {
	Name() string
	Description() string
	InvokeInTenancy() bool
}

type CommandGroupBase struct {
	GroupName        string
	GroupDescription string
}

func NewCommandGroup(groupName string, groupDescription string) *CommandGroupBase {
	c := &CommandGroupBase{}
	c.GroupName = groupName
	c.GroupDescription = groupDescription
	return c
}

func (c *CommandGroupBase) Name() string {
	return c.GroupName
}

func (c *CommandGroupBase) Description() string {
	return c.GroupDescription
}

func (c *CommandGroupBase) InvokeInTenancy() bool {
	return false
}

type HandlerBuilder[T CommandGroup] func() Handler[T]

type Commands[T CommandGroup] struct {
	*CommandGroupBase
	Self          T
	ExtraHandlers []HandlerBuilder[T]
	InTenancy     bool
}

func (c *Commands[T]) AddHandlers(handlers ...HandlerBuilder[T]) {
	c.ExtraHandlers = append(c.ExtraHandlers, handlers...)
}

func (c Commands[T]) AddCommand(parent *flags.Command, ctxBuilder ContextBulder, group T) {
	for _, handler := range c.ExtraHandlers {
		AddCommand(parent, ctxBuilder, group, handler)
	}
}

func (c *Commands[T]) Handlers(ctxBuilder ContextBulder, parser *flags.Parser) *flags.Command {

	commandGroup, err := parser.AddCommand(c.Name(), c.Description(), "", &Dummy{})
	if err != nil {
		fmt.Printf("failed to add %s group: %s", c.Name(), err)
		os.Exit(1)
	}
	// fmt.Printf("added group: %s\n", c.Name())

	for _, handler := range c.ExtraHandlers {
		// fmt.Printf("adding handler %d for group: %s\n", i, c.Name())
		AddCommand(commandGroup, ctxBuilder, c.Self, handler)
	}

	return commandGroup
}

func (c *Commands[T]) SubHandlers(ctxBuilder ContextBulder, parent *flags.Command) *flags.Command {

	commandGroup, err := parent.AddCommand(c.Name(), c.Description(), "", &Dummy{})
	if err != nil {
		fmt.Printf("failed to add %s group: %s", c.Name(), err)
		os.Exit(1)
	}

	for _, handler := range c.ExtraHandlers {
		AddCommand(commandGroup, ctxBuilder, c.Self, handler)
	}

	return commandGroup
}

func (c *Commands[T]) Description() string {
	return c.CommandGroupBase.Description()
}

func (c *Commands[T]) Name() string {
	return c.CommandGroupBase.Name()
}

func (c *Commands[T]) InvokeInTenancy() bool {
	return c.InTenancy
}

func (c *Commands[T]) Construct(self T, name string, description string) {
	c.Self = self
	c.CommandGroupBase = NewCommandGroup(name, description)
	c.ExtraHandlers = make([]HandlerBuilder[T], 0)
}

//-------------------------------------------------

func AddCommand[T CommandGroup](parent *flags.Command, ctxBuilder ContextBulder, group T, makeHandler HandlerBuilder[T]) {

	handler := makeHandler()
	handler.Construct(ctxBuilder, group)

	// fmt.Printf("adding command %s to group %s\n", handler.HandlerName(), group.Name())

	_, err := parent.AddCommand(handler.HandlerName(), handler.HandlerDescription(), "", handler)
	if err != nil {
		fmt.Printf("failed to add command %s to group %s: %s", handler.HandlerName(), group.Name(), err)
		os.Exit(1)
	}

	// fmt.Printf("added command %s to group %s\n", handler.HandlerName(), group.Name())
}

//-------------------------------------------------

type Handler[T CommandGroup] interface {
	CtxBuilder() ContextBulder
	HandlerGroup() T
	Construct(ctxBuilder ContextBulder, group T)
	HandlerName() string
	HandlerDescription() string
	Data() interface{}
}

type HandlerBaseHolder[T CommandGroup] struct {
	CtxBuilder  ContextBulder `no-flag:"true"`
	Group       T             `no-flag:"true"`
	Name        string        `no-flag:"true"`
	Description string        `no-flag:"true"`
}

type HandlerBase[T CommandGroup] struct {
	HandlerBaseHolder[T]
}

func (h *HandlerBase[T]) Construct(ctxBuilder ContextBulder, group T) {
	h.HandlerBaseHolder.CtxBuilder = ctxBuilder
	h.HandlerBaseHolder.Group = group
}

func (h *HandlerBase[T]) Init(name string, description string) {
	h.HandlerBaseHolder.Name = name
	h.HandlerBaseHolder.Description = description
}

func (h *HandlerBase[T]) CtxBuilder() ContextBulder {
	return h.HandlerBaseHolder.CtxBuilder
}

func (h *HandlerBase[T]) HandlerGroup() T {
	return h.HandlerBaseHolder.Group
}

func (h *HandlerBase[T]) HandlerName() string {
	return h.HandlerBaseHolder.Name
}

func (h *HandlerBase[T]) HandlerDescription() string {
	return h.HandlerBaseHolder.Description
}

func (h *HandlerBase[T]) Context(opData interface{}) (multitenancy.TenancyContext, error) {
	ctx := h.HandlerBaseHolder.CtxBuilder(h.Group.Name(), h.HandlerBaseHolder.Name)
	if h.Group.InvokeInTenancy() && ctx.GetTenancy() == nil {
		return ctx, errors.New("THIS COMMAND MUST BE INVOKED IN TENANCY")
	}
	if !h.Group.InvokeInTenancy() && ctx.GetTenancy() != nil {
		return ctx, errors.New("THIS COMMAND MUST BE INVOKED NOT IN TENANCY")
	}
	err := ctx.App().Validator().Validate(opData)
	return ctx, err
}

func (h *HandlerBase[T]) Data() interface{} {
	return &Dummy{}
}

type QueryData struct {
	Query string `long:"query" description:"Query to filter items in response"`
}

type GroupByData struct {
	GroupBy []string `long:"groupby" description:"Fields to group by, can be specified multiple times"`
}

type QueryWithGroupBy struct {
	QueryData
	GroupByData
}
