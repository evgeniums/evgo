package event_dispatcher

import (
	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/evgeniums/evgo/pkg/utils"
)

const MaxSelectors int = 6

type Message struct {
	MessageType string
	Message     any
}

type EventKey struct {
	selectors [MaxSelectors]utils.OptString
}

func (k EventKey) Key() EventKey {
	return k
}

func (k EventKey) GetSelectors() []message_queue.Optional[string] {
	return k.selectors[:]
}

func (k EventKey) SetSelector(i int, selector string) {
	k.selectors[i] = utils.Opt(selector)
}

func (k EventKey) UnsetSelector(i int, selector string) {
	k.selectors[i] = utils.NullString()
}

func (k EventKey) Length() int {
	return MaxSelectors
}

func (k EventKey) GetSelector(i int) (string, bool) {
	if i >= k.Length() {
		return "", false
	}
	v := k.selectors[i]
	return v.Value, v.IsSet
}

type Event struct {
	EventKey
	Message
	Parameters map[string]string
}
