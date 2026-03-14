package default_event_dispatcher

import (
	"context"

	"github.com/evgeniums/evgo/pkg/event_dispatcher"
)

func (d *DispatcherBase) Subscribe(sctx context.Context, key event_dispatcher.EventKey) (event_dispatcher.EventSubscriber, error) {
	subscriber, err := d.MakeSubscriber()
	if err != nil {
		return nil, err
	}
	return subscriber, subscriber.Subscribe(sctx, d.mq, key)
}
