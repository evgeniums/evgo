package api_server

import (
	"context"
)

type QueueProducer interface {
	Produce(ctx context.Context) <-chan interface{}
	Push(ctx context.Context, object interface{})
	Close()
}

type QueueProducerBase struct {
	ch chan interface{}
}

func (p *QueueProducerBase) Produce(ctx context.Context) <-chan interface{} {
	return p.ch
}

func (p *QueueProducerBase) Push(ctx context.Context, object interface{}) {
	p.ch <- object
}

func (p *QueueProducerBase) Close() {
	close(p.ch)
}

type QueueProducerKey struct{}

func WrapQueueContext(ctx context.Context, q QueueProducer) context.Context {
	newCtx := context.WithValue(ctx, QueueProducerKey{}, q)
	return newCtx
}

func MakeQueueContext(q QueueProducer) context.Context {
	ctx := context.WithValue(context.Background(), QueueProducerKey{}, q)
	return ctx
}

func QueueContext(ctx context.Context) QueueProducer {
	v, _ := ctx.Value(QueueProducerKey{}).(QueueProducer)
	return v
}
