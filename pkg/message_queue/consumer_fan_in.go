package message_queue

type ConsumerFanIn[K comparable, M Message[K]] interface {
	FeederFanIn[M]
	AddConsumer(consumer Consumer[K, M])
	RemoveConsumer(consumer Consumer[K, M])
}

type ConsumerFanInBase[K comparable, M Message[K]] struct {
	FeederFanInBase[M]
}

func NewConsumerFanIn[K comparable, M Message[K]]() *ConsumerFanInBase[K, M] {
	f := &ConsumerFanInBase[K, M]{}
	f.construct()
	return f
}

func (f *ConsumerFanInBase[K, M]) AddConsumer(consumer Consumer[K, M]) {
	f.AddFeeder(consumer.Feeder())
}

func (f *ConsumerFanInBase[K, M]) RemoveConsumer(consumer Consumer[K, M]) {
	f.RemoveFeeder(consumer.Feeder())
}
