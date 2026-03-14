package event_dispatcher

import "github.com/evgeniums/evgo/pkg/message_queue"

const MaxSelectors int = 7

type Message struct {
	MessageType string
	Message     any
}

type EventKey struct {
	Subject      string
	SubjectType  string
	SubjectTopic string
	Service      string
	Name         string
	Object       string
	ObjectTopic  string
}

func (k EventKey) Key() EventKey {
	return k
}

func (k EventKey) GetSelectors() []message_queue.Optional[string] {
	selectors := make([]message_queue.Optional[string], 5)

	i := 0
	addSelector := func(value string) {
		if value == "" {
			selectors[i] = message_queue.None[string]()
		} else {
			selectors[i] = message_queue.Some[string](value)
		}
		i++
	}

	addSelector(k.Subject)
	addSelector(k.SubjectType)
	addSelector(k.SubjectTopic)
	addSelector(k.Service)
	addSelector(k.Name)
	addSelector(k.Object)
	addSelector(k.ObjectTopic)

	return selectors
}

type Event struct {
	EventKey
	Message
}
