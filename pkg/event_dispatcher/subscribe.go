package event_dispatcher

import (
	"context"
)

func (d *DispatcherBase) Subscribe(sctx context.Context, key EventKey, subscriber EventSubscriber) error {
	return subscriber.Subscribe(sctx, d.mq, key)
}
