package work_schedule

import (
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pubsub/pool_pubsub"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_subscriber"
)

type PoolWorkNotificationHandler[T Work] struct {
	pubsub_subscriber.SubscriberClientBase

	tenancies  multitenancy.Multitenancy
	controller *WorkSchedule[T]
}

func (p *PoolWorkNotificationHandler[T]) Handle(ctx op_context.Context, msg *PubsubWork[T]) error {

	c := ctx.TraceInMethod("PoolWorkNotificationHandler.Handle")
	defer ctx.TraceOutMethod()

	ctx.SetLoggerField("pool_work_mode", msg.Mode)
	ctx.SetLoggerField("pool_work_id", msg.Work.GetReferenceId())
	ctx.SetLoggerField("pool_work_type", msg.Work.GetReferenceType())

	var tenancy multitenancy.Tenancy
	var err error
	if msg.Tenancy != "" {
		tenancy, err = p.tenancies.Tenancy(msg.Tenancy)
		if err != nil {
			return c.SetError(err)
		}
	}

	err = p.controller.InvokeWork(ctx, msg.Work, msg.Mode, tenancy)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

type PoolWorkSubscriber[T Work] struct {
	topic   *PubsubTopic[T]
	handler *PoolWorkNotificationHandler[T]
}

func NewPoolSubscriber[T Work](tenancies multitenancy.Multitenancy, controller *WorkSchedule[T], name string) *PoolWorkSubscriber[T] {
	p := &PoolWorkSubscriber[T]{}
	p.handler = &PoolWorkNotificationHandler[T]{
		tenancies:  tenancies,
		controller: controller,
	}
	p.handler.Init(name)
	p.topic = &PubsubTopic[T]{}
	return p
}

func (p *PoolWorkSubscriber[T]) Init(ctx op_context.Context, pubsub pool_pubsub.PoolPubsub, topicName string) error {

	c := ctx.TraceInMethod("PoolWorkSubscriber.Init")
	defer ctx.TraceOutMethod()

	p.topic.TopicBase = pubsub_subscriber.New(topicName, MakePubsubWork[T])
	_, err := pubsub.SubscribeSelfPool(ctx, p.topic)
	if err != nil {
		c.SetError(err)
		return ctx.Logger().PushFatalStack("failed to subscribe to pubsub notifications in self pool", err)
	}
	p.topic.Subscribe(p.handler)

	return nil
}
