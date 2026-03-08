package pubsub_inmem

import (
	"context"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/message"
	"github.com/evgeniums/evgo/pkg/pubsub"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_subscriber"
)

const Provider string = "inmem"

type PubsubInmem struct {
	pubsub_subscriber.SubscriberBase
	pubsub.PublisherBase
}

func New(app app_context.Context, serializer ...message.Serializer) *PubsubInmem {
	p := &PubsubInmem{}
	p.PublisherBase.Construct(serializer...)
	p.SubscriberBase.Construct(app, serializer...)
	return p
}

func (p *PubsubInmem) Shutdown(ctx context.Context) error {
	return nil
}

func (p *PubsubInmem) Subscribe(topic pubsub_subscriber.Topic) (string, error) {
	return p.AddTopic(topic)
}

func (p *PubsubInmem) Unsubscribe(topicName string, subscriptionId ...string) {
	p.DeleteTopic(topicName, subscriptionId...)
}

func (p *PubsubInmem) Publish(topicName string, obj interface{}) error {

	msg, err := p.Serialize(obj)
	if err != nil {
		return err
	}

	opCtx, sctx := p.NewOpContext(topicName)
	defer opCtx.Close(sctx)

	p.Handle(sctx, topicName, msg)

	return nil
}
