package work_schedule

import (
	"context"

	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pubsub/pool_pubsub"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_subscriber"
	"github.com/evgeniums/evgo/pkg/utils"
)

type PubsubWork[T Work] struct {
	Work    T        `json:"work"`
	Mode    PostMode `json:"mode"`
	Tenancy string   `json:"tenancy"`
}

type PubsubTopic[T Work] struct {
	*pubsub_subscriber.TopicBase[*PubsubWork[T]]
}

func MakePubsubWork[T Work]() *PubsubWork[T] {
	return &PubsubWork[T]{}
}

func NewPubsubWork[T Work](work T, postMode PostMode, tenancy ...multitenancy.Tenancy) *PubsubWork[T] {
	w := &PubsubWork[T]{Work: work, Mode: postMode}
	t := utils.OptionalArg(nil, tenancy...)
	if t != nil {
		w.Tenancy = t.GetID()
	}
	return w
}

type PoolWorkPublisher[T Work] struct {
	pubsub    pool_pubsub.PoolPubsub
	topicName string
}

func NewPoolWorkPublisher[T Work](pubsub pool_pubsub.PoolPubsub, topicName string) *PoolWorkPublisher[T] {
	p := &PoolWorkPublisher[T]{pubsub: pubsub, topicName: topicName}
	return p
}

func (p *PoolWorkPublisher[T]) InvokeWork(sctx context.Context, work T, postMode PostMode, tenancy ...multitenancy.Tenancy) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("PoolWorkPublisher.InvokeWork")
	defer ctx.TraceOutMethod()

	c.SetLoggerField("pool_topic_name", p.topicName)
	c.Logger().Debug("publish work to self pool")

	msg := NewPubsubWork(work, postMode, tenancy...)
	err := p.pubsub.PublishSelfPool(p.topicName, msg)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}
