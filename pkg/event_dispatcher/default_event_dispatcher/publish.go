package default_event_dispatcher

import (
	"context"

	"github.com/evgeniums/evgo/pkg/event_dispatcher"
)

func (d *DispatcherBase) Publish(ctx context.Context, event *event_dispatcher.Event) error {
	return d.mq.Publish(ctx, event.Key(), event_dispatcher.EventWrapper{Event: event, Context: ctx})
}
