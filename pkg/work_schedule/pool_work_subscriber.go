package work_schedule

import (
	"context"

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

func (p *PoolWorkNotificationHandler[T]) Handle(sctx context.Context, msg *PubsubWork[T]) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolWorkNotificationHandler.Handle")
	defer ctx.TraceOutMethod()

	ctx.SetLoggerField("pool_work_mode", msg.Mode)
	ctx.SetLoggerField("pool_work_no_db", msg.NoDb)

	var tenancy multitenancy.Tenancy
	var err error
	if msg.Tenancy != "" && p.tenancies != nil {
		tenancy, err = p.tenancies.Tenancy(msg.Tenancy)
		if err != nil {
			return c.SetError(err)
		}
	}

	if msg.NoDb {
		// no-db path: work payload is in the message; run directly without a DB lease
		ctx.SetLoggerField("pool_work_id", msg.Work.GetReferenceId())
		ctx.SetLoggerField("pool_work_type", msg.Work.GetReferenceType())
		err = p.controller.InvokeWork(sctx, msg.Work, msg.Mode, tenancy)
		if err != nil {
			return c.SetError(err)
		}
		return nil
	}

	// durable nudge path: claim due works from the DB (the lease is the authority)
	// and enqueue each one into the local worker pool
	works, err := p.controller.ClaimDueWorks(sctx)
	if err != nil {
		return c.SetError(err)
	}
	for _, work := range works {
		p.controller.enqueuWork(sctx, work, tenancy)
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

// HandleMessage delivers a PubsubWork message directly to the subscriber's
// handler, bypassing the pubsub transport. Intended for testing.
func (p *PoolWorkSubscriber[T]) HandleMessage(sctx context.Context, msg *PubsubWork[T]) error {
	return p.handler.Handle(sctx, msg)
}

func (p *PoolWorkSubscriber[T]) Init(sctx context.Context, pubsub pool_pubsub.PoolPubsub, topicName string) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolWorkSubscriber.Init")
	defer ctx.TraceOutMethod()

	p.topic.TopicBase = pubsub_subscriber.New(topicName, MakePubsubWork[T])
	_, err := pubsub.SubscribeSelfPool(sctx, p.topic)
	if err != nil {
		c.SetError(err)
		return ctx.Logger().PushFatalStack("failed to subscribe to pubsub notifications in self pool", err)
	}
	p.topic.Subscribe(p.handler)

	return nil
}
