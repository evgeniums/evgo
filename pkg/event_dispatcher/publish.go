package event_dispatcher

import (
	"context"
)

func (d *DispatcherBase) Publish(sctx context.Context, event EventKey) error {
	return d.Publish(sctx, event)
}
