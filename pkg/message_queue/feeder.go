package message_queue

const DEFAULT_FEEDER_CHANNEL_DEPTH = 10

type MessageProvider interface {
	Next()
}

type Feeder[V any] interface {
	TypedChannel() <-chan V
	Channel() <-chan any
	Push(object V) bool
	Close()
	Next()
}

type FeederConfig struct {
	FEEDER_TYPED_CHANNEL bool
	FEEDER_CHANNEL_DEPTH int `default:"10"`
}

type FeederBase[V any] struct {
	ch       chan any
	chV      chan V
	provider MessageProvider
}

func NewFeeder[V any](provider MessageProvider, config ...*FeederConfig) *FeederBase[V] {
	q := &FeederBase[V]{provider: provider}

	if len(config) != 0 {
		cfg := config[0]
		if cfg.FEEDER_TYPED_CHANNEL {
			q.chV = make(chan V, cfg.FEEDER_CHANNEL_DEPTH)
		} else {
			q.ch = make(chan any, cfg.FEEDER_CHANNEL_DEPTH)
		}
	} else {
		q.ch = make(chan any, DEFAULT_FEEDER_CHANNEL_DEPTH)
	}

	return q
}

func (p *FeederBase[V]) Channel() <-chan any {
	return p.ch
}

func (p *FeederBase[V]) TypedChannel() <-chan V {
	return p.chV
}

func (p *FeederBase[V]) Push(object V) bool {
	if p.ch != nil {
		select {
		case p.ch <- object:
			return true
		default:
			return false
		}
	}

	if p.chV != nil {
		select {
		case p.chV <- object:
			return true
		default:
			return false
		}
	}

	return false
}

func (p *FeederBase[V]) Close() {
	close(p.ch)
}

func (p *FeederBase[V]) Next() {
	p.provider.Next()
}
