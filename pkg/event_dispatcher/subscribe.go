package event_dispatcher

import (
	"context"
)

func (d *DispatcherBase) Subscribe(sctx context.Context, key EventKey) (EventSubscriber, error) {
	subscriber, err := d.MakeSubscriber()
	if err != nil {
		return nil, err
	}
	return subscriber, subscriber.Subscribe(sctx, d.mq, key)
}
