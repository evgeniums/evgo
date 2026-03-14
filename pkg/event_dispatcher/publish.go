package event_dispatcher

import (
	"context"
)

func (d *DispatcherBase) Publish(ctx context.Context, event Event) error {
	return d.mq.Publish(ctx, event.Key(), event)
}
